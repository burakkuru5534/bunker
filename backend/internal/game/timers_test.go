package game

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
	"github.com/burakkuru5534/bunker/backend/internal/room"
)

type call struct {
	gameID  string
	version int64
	at      time.Time
}

// fakeSource: her çağrıyı kaydeder; davranış fn ile belirlenir.
type fakeSource struct {
	mu     sync.Mutex
	calls  []call
	fn     func(gameID string, expected int64, n int) (TimeoutResult, error)
	active []Deadline
	listFn func(n int) ([]Deadline, error)
	lists  int
	block  chan struct{} // doluysa Timeout bunu bekler
}

func (f *fakeSource) Timeout(ctx context.Context, gameID string, expected int64) (TimeoutResult, error) {
	f.mu.Lock()
	f.calls = append(f.calls, call{gameID, expected, time.Now()})
	n := len(f.calls)
	fn, block := f.fn, f.block
	f.mu.Unlock()
	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return TimeoutResult{}, ctx.Err()
		}
	}
	return fn(gameID, expected, n)
}

func (f *fakeSource) ActiveDeadlines(context.Context) ([]Deadline, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lists++
	if f.listFn != nil {
		return f.listFn(f.lists)
	}
	return f.active, nil
}

func (f *fakeSource) snapshot() []call {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]call(nil), f.calls...)
}

func quietTimers(src timeoutSource) *Timers {
	t := NewTimers(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	t.retryBase, t.retryMax = 10*time.Millisecond, 40*time.Millisecond
	t.Bind(src)
	return t
}

// waitFor, koşul sağlanana kadar (en fazla 2 sn) bekler.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("zaman aşımı: %s", what)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func finished(string, int64, int) (TimeoutResult, error) {
	return TimeoutResult{Advanced: true, Finished: true}, nil
}

func TestTimersFireAtDeadline(t *testing.T) {
	src := &fakeSource{fn: finished}
	tm := quietTimers(src)
	defer tm.Stop(context.Background())
	at := time.Now().Add(60 * time.Millisecond)
	tm.Schedule("g1", 3, at)
	waitFor(t, "tetikleme", func() bool { return len(src.snapshot()) == 1 })
	c := src.snapshot()[0]
	if c.gameID != "g1" || c.version != 3 || c.at.Before(at) {
		t.Fatalf("çağrı %+v (erken: %v)", c, c.at.Before(at))
	}
	waitFor(t, "bırakma", func() bool { return tm.Pending() == 0 })
}

func TestTimersVersionOrdering(t *testing.T) {
	src := &fakeSource{fn: finished}
	tm := quietTimers(src)
	defer tm.Stop(context.Background())
	soon := time.Now().Add(40 * time.Millisecond)
	tm.Schedule("g", 5, soon)
	tm.Schedule("g", 4, time.Now()) // sırasız gelen eski olay: yok sayılmalı
	waitFor(t, "tetikleme", func() bool { return len(src.snapshot()) == 1 })
	if c := src.snapshot()[0]; c.version != 5 || c.at.Before(soon) {
		t.Fatalf("eski sürüm planı ezdi: %+v", c)
	}
	// Daha yeni sürüm öncekinin yerini alır.
	tm.Schedule("h", 1, time.Now().Add(30*time.Millisecond))
	tm.Schedule("h", 2, time.Now().Add(60*time.Millisecond))
	waitFor(t, "h tetiklendi", func() bool { return len(src.snapshot()) == 2 })
	time.Sleep(60 * time.Millisecond)
	if calls := src.snapshot(); len(calls) != 2 || calls[1].version != 2 {
		t.Fatalf("çağrılar %+v", calls)
	}
}

func TestTimersRetryThenReschedule(t *testing.T) {
	next := time.Now().Add(50 * time.Millisecond)
	src := &fakeSource{fn: func(_ string, expected int64, n int) (TimeoutResult, error) {
		switch {
		case n <= 2:
			return TimeoutResult{}, errors.New("veritabanı geçici olarak yok")
		case expected == 1:
			return TimeoutResult{Advanced: true, Version: 2, Deadline: next}, nil
		default:
			return TimeoutResult{Advanced: true, Finished: true, Version: 3}, nil
		}
	}}
	tm := quietTimers(src)
	defer tm.Stop(context.Background())
	tm.Schedule("g", 1, time.Now())
	waitFor(t, "bitiş", func() bool { return tm.Pending() == 0 })
	calls := src.snapshot()
	if len(calls) != 4 || calls[0].version != 1 || calls[1].version != 1 || calls[2].version != 1 || calls[3].version != 2 {
		t.Fatalf("çağrılar %+v", calls)
	}
	if gap := calls[2].at.Sub(calls[1].at); gap < 20*time.Millisecond {
		t.Fatalf("artan bekleme yok: %v", gap)
	}
	if calls[3].at.Before(next) {
		t.Fatal("yeni bitişten önce tetiklendi")
	}
}

func TestTimersEarlyAndStaleAndNotFound(t *testing.T) {
	later := time.Now().Add(40 * time.Millisecond)
	src := &fakeSource{fn: func(gameID string, expected int64, n int) (TimeoutResult, error) {
		switch gameID {
		case "early": // saat kayması: aynı sürüm, bitiş gelecekte
			if n == 1 {
				return TimeoutResult{Version: expected, Deadline: later}, nil
			}
			return TimeoutResult{Finished: true}, nil
		case "gone":
			return TimeoutResult{}, errGameNotFound
		}
		return TimeoutResult{Finished: true}, nil
	}}
	tm := quietTimers(src)
	defer tm.Stop(context.Background())
	tm.Schedule("early", 1, time.Now())
	waitFor(t, "erken tetikleme yeniden planlandı", func() bool { return len(src.snapshot()) == 2 })
	if src.snapshot()[1].at.Before(later) {
		t.Fatal("yeniden planlama bitişten önce tetiklendi")
	}
	tm.Schedule("gone", 1, time.Now())
	waitFor(t, "olmayan oyun bırakıldı", func() bool { return tm.Pending() == 0 })
}

func TestTimersPublisherObservesEvents(t *testing.T) {
	src := &fakeSource{fn: finished}
	tm := quietTimers(src)
	defer tm.Stop(context.Background())
	var forwarded []room.Event
	pub := tm.Publisher(publisherFunc(func(e room.Event) { forwarded = append(forwarded, e) }))
	at := time.Now().Add(time.Hour)
	pub.Publish(room.Event{GameID: "g", Type: engine.EventPhaseChanged, Version: 2,
		Payload: map[string]any{"phase": "REVEAL", "phase_deadline_at": at}})
	pub.Publish(room.Event{Type: "room.ready_changed"}) // oyun dışı: dokunulmaz
	if tm.Pending() != 1 || len(forwarded) != 2 {
		t.Fatalf("planlı %d, iletilen %d", tm.Pending(), len(forwarded))
	}
	pub.Publish(room.Event{GameID: "g", Type: engine.EventFinished, Version: 9})
	if tm.Pending() != 0 || len(forwarded) != 3 {
		t.Fatal("game.finished planı bırakmadı")
	}
	// nil zamanlayıcı: yayıncı aynen geçer.
	var nilTimers *Timers
	if nilTimers.Publisher(nil) == nil {
		t.Fatal("nil yayıncı")
	}
	nilTimers.Start()
	nilTimers.Schedule("x", 1, time.Now())
	if err := nilTimers.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestTimersRecoveryRetries(t *testing.T) {
	past := time.Now().Add(-time.Minute)
	src := &fakeSource{fn: finished, listFn: func(n int) ([]Deadline, error) {
		if n < 3 {
			return nil, errors.New("açılışta DB hazır değil")
		}
		return []Deadline{{"a", 4, past}, {"b", 7, time.Now().Add(time.Hour)}}, nil
	}}
	tm := quietTimers(src)
	defer tm.Stop(context.Background())
	tm.Start()
	waitFor(t, "geçmiş bitiş hemen işlendi", func() bool { return len(src.snapshot()) == 1 })
	if c := src.snapshot()[0]; c.gameID != "a" || c.version != 4 {
		t.Fatalf("çağrı %+v", c)
	}
	waitFor(t, "b planlı kaldı", func() bool { return tm.Pending() == 1 })
}

func TestTimersStopWaitsAndBlocksNewFires(t *testing.T) {
	src := &fakeSource{fn: finished, block: make(chan struct{})}
	tm := quietTimers(src)
	tm.Schedule("running", 1, time.Now())
	tm.Schedule("later", 1, time.Now().Add(50*time.Millisecond))
	waitFor(t, "çalışan tetikleme", func() bool { return len(src.snapshot()) == 1 })

	// Çalışan çağrı bloklu: kısa süreli Stop zaman aşımına uğrar ama
	// bağlamı iptal ettiği için çağrı biter; ikinci Stop başarılı olur.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := tm.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	if n := len(src.snapshot()); n != 1 {
		t.Fatalf("durdurulduktan sonra %d çağrı", n)
	}
	tm.Schedule("x", 1, time.Now())
	if tm.Pending() != 0 {
		t.Fatal("durdurulmuş zamanlayıcı plan kabul etti")
	}
}
