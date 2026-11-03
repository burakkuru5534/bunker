package game

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
)

// fast, testte oyunu saniyeler içinde bitiren süreler.
var fast = engine.Durations{
	ScenarioIntro: 20 * time.Millisecond, RevealPerPlayer: 15 * time.Millisecond,
	DiscussionSmall: 20 * time.Millisecond, DiscussionLarge: 25 * time.Millisecond,
	Voting: 20 * time.Millisecond, TieBreak: 15 * time.Millisecond, Elimination: 15 * time.Millisecond,
	LargeTableAtLeast: 7,
}

func (a *api) migrateExec(sql string, args ...any) {
	a.t.Helper()
	if _, err := a.env.Migrate.Exec(context.Background(), sql, args...); err != nil {
		a.t.Fatalf("%s: %v", sql, err)
	}
}

// expire, aşamanın bitişini geçmişe çeker (instance uyurken süre dolmuş gibi).
func (a *api) expire(gameID string) {
	a.migrateExec(`update games set phase_started_at = now() - interval '2 minutes',
		phase_deadline_at = now() - interval '1 minute' where id = $1`, gameID)
}

func (a *api) timeoutRows(gameID string) int64 {
	return a.scalar(`select count(*) from game_actions where game_id = $1 and type = 'phase_timeout' and actor_id is null`, gameID).(int64)
}

// Zamanlayıcı açıkken oyun hiçbir müdahale olmadan sonuna kadar oynanır:
// otomatik kart açma, oy yokken kura, eleme, bitiş.
func TestIntegrationTimersPlayWholeGame(t *testing.T) {
	a := newAPI(t, true)
	a.starter.engine, a.svc.engine = engine.New(fast), engine.New(fast)
	_, gameID := a.startGame(6, 2)
	waitForDB := func(what string, cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(15 * time.Second)
		for !cond() {
			if time.Now().After(deadline) {
				t.Fatalf("zaman aşımı: %s (aşama %v)", what, a.scalar(`select phase from games where id = $1`, gameID))
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	waitForDB("oyun bitti", func() bool { return a.scalar(`select phase from games where id = $1`, gameID) == "finished" })
	s := a.load(gameID)
	if s.EndReason != engine.EndCapacityReached || len(s.Winners()) != 2 || len(s.Results) != 4 {
		t.Fatalf("bitiş: %s kazanan %d sonuç %d", s.EndReason, len(s.Winners()), len(s.Results))
	}
	// Her sürüm artışı bir süre dolumu kaydıdır (oyuncu eylemi yok); başlangıç sürümü 1.
	if n := a.timeoutRows(gameID); n != s.Version-1 {
		t.Fatalf("%d süre dolumu kaydı, sürüm %d", n, s.Version)
	}
	auto := a.scalar(`select count(*) from player_cards where game_id = $1 and reveal_method = 'timeout'`, gameID).(int64)
	if auto == 0 {
		t.Fatal("otomatik açılan kart yok")
	}
	waitForDB("plan bırakıldı", func() bool { return a.timers.Pending() == 0 })
	finishedEvents := 0
	for _, e := range a.events.all() {
		if e.Type == engine.EventFinished && e.GameID == gameID {
			finishedEvents++
		}
	}
	if finishedEvents != 1 {
		t.Fatalf("%d game.finished", finishedEvents)
	}
}

// Açılış kurtarması: sunucu kapalıyken süresi dolan aşama yeni zamanlayıcı
// başlatılınca işlenir; gelecekteki bitiş planlanır.
func TestIntegrationTimersRecovery(t *testing.T) {
	a := newAPI(t, true)
	_, overdueGame := a.startGame(4, 1)
	_, futureGame := a.startGame(5, 1)
	a.expire(overdueGame)
	_ = a.timers.Stop(context.Background()) // eski "instance" kapandı

	fresh := NewTimers(a.logger)
	defer fresh.Stop(context.Background())
	a.svc.publisher = fresh.Publisher(a.events)
	fresh.Bind(a.svc)
	fresh.Start()

	waitFor(t, "süresi geçmiş oyun ilerledi", func() bool {
		return a.scalar(`select state_version from games where id = $1`, overdueGame).(int64) == 2
	})
	s := a.load(overdueGame)
	if s.Phase != engine.PhaseReveal || s.Round != 1 || a.timeoutRows(overdueGame) != 1 {
		t.Fatalf("kurtarılan: %s r%d", s.Phase, s.Round)
	}
	if !s.Deadline.After(time.Now()) {
		t.Fatal("yeni aşama geçmişten başladı (süre 'şimdi'den sayılmalı)")
	}
	if a.scalar(`select state_version from games where id = $1`, futureGame).(int64) != 1 {
		t.Fatal("süresi dolmamış oyun ilerletildi")
	}
	waitFor(t, "iki oyun planlı", func() bool { return fresh.Pending() == 2 })
}

// Süre dolumunun anlamı: eski sürüm, erken çağrı ve bitmiş oyun yazmaz.
func TestIntegrationTimeoutSemantics(t *testing.T) {
	a := newAPI(t, true)
	_, gameID := a.startGame(4, 1)
	ctx := context.Background()
	res, err := a.svc.Timeout(ctx, gameID, 1) // süre dolmadı
	if err != nil || res.Advanced || res.Version != 1 || res.Deadline.IsZero() {
		t.Fatalf("erken: %+v %v", res, err)
	}
	a.expire(gameID)
	res, err = a.svc.Timeout(ctx, gameID, 0) // eski sürüm
	if err != nil || res.Advanced || res.Version != 1 {
		t.Fatalf("eski sürüm: %+v %v", res, err)
	}
	if a.timeoutRows(gameID) != 0 {
		t.Fatal("işlenmemesi gereken süre dolumu kaydedildi")
	}
	res, err = a.svc.Timeout(ctx, gameID, 1)
	if err != nil || !res.Advanced || res.Version != 2 || res.Finished {
		t.Fatalf("işlendi: %+v %v", res, err)
	}
	if _, err := a.svc.Timeout(ctx, newUUID(), 1); !isGameNotFound(err) {
		t.Fatalf("olmayan oyun: %v", err)
	}
	a.migrateExec(`update games set phase = 'finished', phase_deadline_at = null, turn_seat = null,
		finished_at = now(), end_reason = 'abandoned' where id = $1`, gameID)
	if res, err := a.svc.Timeout(ctx, gameID, 2); err != nil || !res.Finished || res.Advanced {
		t.Fatalf("bitmiş: %+v %v", res, err)
	}
}

// Aynı süre dolumu aynı anda birden fazla kez tetiklenirse tek kez işlenir.
func TestIntegrationTimeoutConcurrent(t *testing.T) {
	a := newAPI(t, true)
	_, gameID := a.startGame(6, 2)
	a.expire(gameID)
	var wg sync.WaitGroup
	advanced := make([]bool, 8)
	for i := range advanced {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := a.svc.Timeout(context.Background(), gameID, 1)
			if err != nil {
				t.Error(err)
			}
			advanced[i] = res.Advanced
		}()
	}
	wg.Wait()
	n := 0
	for _, ok := range advanced {
		if ok {
			n++
		}
	}
	if n != 1 || a.timeoutRows(gameID) != 1 || a.scalar(`select state_version from games where id = $1`, gameID).(int64) != 2 {
		t.Fatalf("%d ilerleme, %d kayıt", n, a.timeoutRows(gameID))
	}
}

// Süresi geçmiş aşamada gelen eylem: önce süre dolumu işlenip kaydedilir,
// eylem güncel duruma uygulanır; eylem reddedilse de süre dolumu kalır.
func TestIntegrationActOnOverduePhase(t *testing.T) {
	a := newAPI(t, true)
	l, gameID := a.startGame(5, 1)
	a.expire(gameID) // SCENARIO_INTRO süresi doldu, zamanlayıcı işlemedi
	s := a.load(gameID)
	first := s.Players[0].UserID // tur 1 koltuk 0
	r := a.act(gameID, l.tokenOf(first), newActionID(), "reveal_card", `{"category":"profession"}`)
	if r.code != 200 || r.body["state_version"] != float64(3) {
		t.Fatalf("süre dolumu + açma: %d %s", r.code, r.raw)
	}
	if a.timeoutRows(gameID) != 1 {
		t.Fatal("süre dolumu kaydı yok")
	}

	// Şimdi koltuk 1'in süresi geçmiş: süre dolumu onun kartını otomatik açar,
	// sıra koltuk 2'ye geçer; koltuk 1'in kendi açma isteği NOT_YOUR_TURN alır
	// ama otomatik açma geri alınmaz.
	a.expire(gameID)
	second := s.Players[1].UserID
	r = a.act(gameID, l.tokenOf(second), newActionID(), "reveal_card", `{"category":"profession"}`)
	if r.code != 409 || errCode(r) != "NOT_YOUR_TURN" {
		t.Fatalf("geç kalan açma: %d %s", r.code, r.raw)
	}
	if m := a.scalar(`select reveal_method from player_cards where game_id = $1 and user_id = $2 and category = 'profession'`,
		gameID, second); m != "timeout" {
		t.Fatalf("otomatik açma kayboldu: %v", m)
	}
	if v := a.scalar(`select state_version from games where id = $1`, gameID).(int64); v != 4 || a.timeoutRows(gameID) != 2 {
		t.Fatalf("sürüm %d", v)
	}
}
