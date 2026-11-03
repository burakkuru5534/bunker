package game

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
	"github.com/burakkuru5534/bunker/backend/internal/room"
)

// timeoutSource, zamanlayıcının kullandığı servis yüzüdür (testlerde sahte).
type timeoutSource interface {
	Timeout(ctx context.Context, gameID string, expected int64) (TimeoutResult, error)
	ActiveDeadlines(ctx context.Context) ([]Deadline, error)
}

// Timers, aktif oyunların aşama bitişlerini bellekte izler (ADR-0005).
// Kaynak gerçek PostgreSQL'dir: bellekteki her şey açılışta
// ActiveDeadlines'tan yeniden kurulur. Planlama, olay akışından beslenir
// (game.phase_changed → planla, game.finished → bırak); böylece başlatma,
// oyuncu eylemi ve süre dolumu aynı yoldan geçer.
//
// Tek instance varsayımıyla çalışır (ADR-0003/0011). İkinci bir instance
// aynı oyunu işlemeye kalksa bile satır kilidi ve beklenen sürüm sayesinde
// süre dolumu bir kez uygulanır.
type Timers struct {
	logger *slog.Logger
	src    timeoutSource

	// Yeniden deneme: veritabanı hatasında artan bekleme.
	retryBase, retryMax time.Duration
	callTimeout         time.Duration

	mu      sync.Mutex
	games   map[string]*timerEntry
	stopped bool
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

type timerEntry struct {
	version int64
	at      time.Time
	attempt int
	timer   *time.Timer
}

// NewTimers, zamanlayıcıyı kurar. Bind ile servis bağlanmadan Start
// çağrılmamalıdır.
func NewTimers(logger *slog.Logger) *Timers {
	if logger == nil {
		logger = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Timers{
		logger: logger, games: map[string]*timerEntry{}, ctx: ctx, cancel: cancel,
		retryBase: time.Second, retryMax: 30 * time.Second, callTimeout: 10 * time.Second,
	}
}

// Bind, süre dolumlarını işleyecek servisi bağlar.
func (t *Timers) Bind(src timeoutSource) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.src = src
}

// Publisher, olayları next'e iletirken oyun bitiş zamanlarını izleyen bir
// yayıncı döndürür. t nil ise next aynen döner.
func (t *Timers) Publisher(next room.Publisher) room.Publisher {
	if next == nil {
		next = room.NoPublisher{}
	}
	if t == nil {
		return next
	}
	return publisherFunc(func(e room.Event) {
		t.observe(e)
		next.Publish(e)
	})
}

type publisherFunc func(room.Event)

func (f publisherFunc) Publish(e room.Event) { f(e) }

func (t *Timers) observe(e room.Event) {
	if e.GameID == "" {
		return
	}
	switch e.Type {
	case engine.EventPhaseChanged:
		payload, _ := e.Payload.(map[string]any)
		if at, ok := payload["phase_deadline_at"].(time.Time); ok && !at.IsZero() {
			t.Schedule(e.GameID, e.Version, at)
		}
	case engine.EventFinished:
		t.Cancel(e.GameID)
	}
}

// Schedule, oyunun süre dolumunu planlar. Daha eski bir sürüm yok sayılır
// (olaylar sırasız gelebilir); aynı veya yeni sürüm öncekinin yerini alır.
func (t *Timers) Schedule(gameID string, version int64, at time.Time) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stopped {
		return
	}
	if cur, ok := t.games[gameID]; ok {
		if version < cur.version {
			return
		}
		cur.timer.Stop()
	}
	t.arm(gameID, &timerEntry{version: version, at: at}, time.Until(at))
}

// arm, kilit altında çağrılır.
func (t *Timers) arm(gameID string, e *timerEntry, d time.Duration) {
	version := e.version
	e.timer = time.AfterFunc(max(d, 0), func() { t.fire(gameID, version) })
	t.games[gameID] = e
}

// Cancel, oyunu izlemeyi bırakır.
func (t *Timers) Cancel(gameID string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if cur, ok := t.games[gameID]; ok {
		cur.timer.Stop()
		delete(t.games, gameID)
	}
}

// Pending, izlenen oyun sayısıdır.
func (t *Timers) Pending() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.games)
}

func (t *Timers) fire(gameID string, version int64) {
	t.mu.Lock()
	cur, ok := t.games[gameID]
	if t.stopped || !ok || cur.version != version || t.src == nil {
		t.mu.Unlock()
		return
	}
	t.wg.Add(1) // stopped kilit altında denetlendi: Stop'un Wait'inden önce
	src := t.src
	t.mu.Unlock()
	defer t.wg.Done()

	ctx, cancel := context.WithTimeout(t.ctx, t.callTimeout)
	res, err := src.Timeout(ctx, gameID, version)
	cancel()

	t.mu.Lock()
	defer t.mu.Unlock()
	cur, ok = t.games[gameID]
	if t.stopped || !ok || cur.version != version {
		return // bu arada yenisi planlandı veya bırakıldı
	}
	switch {
	case isGameNotFound(err):
		cur.timer.Stop()
		delete(t.games, gameID)
	case err != nil:
		cur.attempt++
		wait := min(t.retryBase<<min(cur.attempt-1, 10), t.retryMax)
		t.logger.Warn("süre dolumu işlenemedi; yeniden denenecek", "game_id", gameID,
			"version", version, "attempt", cur.attempt, "retry_in", wait.String(), "error", err)
		t.arm(gameID, cur, wait)
	case res.Finished:
		delete(t.games, gameID)
	case res.Version >= version && !res.Deadline.IsZero():
		// İlerledi (olay zaten planladı; aynı değerle yenilemek zararsız),
		// erken tetiklendi (aynı sürüm, gelecekteki bitiş) veya eskimişti
		// (daha yeni sürüm): her durumda güncel bitişe planla.
		t.arm(gameID, &timerEntry{version: res.Version, at: res.Deadline}, time.Until(res.Deadline))
	default:
		delete(t.games, gameID)
	}
}

// Start, açılış kurtarmasını arka planda yapar: bitmemiş oyunlar yüklenir,
// süresi geçmiş olanlar hemen, diğerleri zamanında işlenir. Veritabanı
// hatasında artan beklemeyle yeniden dener.
func (t *Timers) Start() {
	if t == nil {
		return
	}
	t.mu.Lock()
	if t.stopped || t.src == nil {
		t.mu.Unlock()
		return
	}
	t.wg.Add(1)
	src := t.src
	t.mu.Unlock()
	go func() {
		defer t.wg.Done()
		wait := t.retryBase
		for {
			ctx, cancel := context.WithTimeout(t.ctx, t.callTimeout)
			list, err := src.ActiveDeadlines(ctx)
			cancel()
			if err == nil {
				for _, d := range list {
					t.Schedule(d.GameID, d.Version, d.At)
				}
				t.logger.Info("zamanlayıcı kurtarması tamam", "active_games", len(list))
				return
			}
			t.logger.Warn("zamanlayıcı kurtarması başarısız; yeniden denenecek", "retry_in", wait.String(), "error", err)
			select {
			case <-t.ctx.Done():
				return
			case <-time.After(wait):
			}
			wait = min(wait*2, t.retryMax)
		}
	}()
}

// Stop, yeni tetiklemeleri durdurur ve çalışanların bitmesini ctx süresince
// bekler. Bellekteki planlar atılır; veritabanındaki bitiş zamanları
// sonraki açılışta işlenir.
func (t *Timers) Stop(ctx context.Context) error {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	t.stopped = true
	for id, e := range t.games {
		e.timer.Stop()
		delete(t.games, id)
	}
	t.mu.Unlock()
	t.cancel()
	done := make(chan struct{})
	go func() { t.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
