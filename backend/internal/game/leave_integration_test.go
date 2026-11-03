package game

import (
	"fmt"
	"sync"
	"testing"

	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
	"github.com/burakkuru5534/bunker/backend/internal/room"
)

func (a *api) leaveRoom(l lobby, user string) {
	a.t.Helper()
	if r := a.do("POST", "/v1/rooms/"+l.roomID+"/leave", l.tokenOf(user), ""); r.code != 204 {
		a.t.Fatalf("ayrıl: %d %s", r.code, r.raw)
	}
}

func TestIntegrationLeaveDuringGameOnTurn(t *testing.T) {
	a := newAPI(t, true)
	l, gameID := a.startGame(5, 1)
	s := a.advance(gameID, nil) // REVEAL tur 1
	turn := s.Players[s.TurnSeat].UserID
	before := len(a.events.all())
	a.leaveRoom(l, turn)

	var (
		status string
		round  *int16
	)
	if err := a.env.DB.Pool().QueryRow(t.Context(), `select status, eliminated_round from game_participants where game_id = $1 and user_id = $2`,
		gameID, turn).Scan(&status, &round); err != nil {
		t.Fatal(err)
	}
	if status != "left" || round == nil || *round != 1 {
		t.Fatalf("katılımcı %s %v", status, round)
	}
	if n := a.scalar(`select count(*) from player_cards where game_id = $1 and user_id = $2 and reveal_method = 'elimination'`, gameID, turn).(int64); n != 5 {
		t.Fatalf("%d kart açıldı", n)
	}
	next := a.load(gameID)
	if next.Phase != engine.PhaseReveal || next.TurnSeat == s.TurnSeat || next.Version != s.Version+1 {
		t.Fatalf("sıra geçmedi: %s koltuk %d", next.Phase, next.TurnSeat)
	}
	if a.scalar(`select count(*) from round_results where game_id = $1`, gameID).(int64) != 0 ||
		a.scalar(`select count(*) from game_actions where game_id = $1 and type = 'leave' and actor_id = $2`, gameID, turn).(int64) != 1 {
		t.Fatal("ayrılma kaydı yanlış")
	}
	evs := a.events.all()[before:]
	var types []string
	for _, e := range evs {
		types = append(types, e.Type)
	}
	// Koltuklar rastgele: sırası gelen oda sahibiyse sahiplik de devredilir.
	want := []string{room.EventMemberLeft}
	if turn == l.users[0] {
		want = append(want, room.EventHostChanged)
	}
	want = append(want, engine.EventPlayerEliminated, engine.EventPhaseChanged)
	if fmt.Sprint(types) != fmt.Sprint(want) {
		t.Fatalf("olaylar %v, beklenen %v", types, want)
	}
	el := evs[len(want)-2]
	if el.Version != next.Version || el.RoomID != l.roomID || el.Payload.(map[string]any)["method"] != "left" {
		t.Fatalf("eleme olayı %+v", el)
	}
	checkEliminatedEvent(t, a, gameID, el, turn)

	// Ayrılan izleyici olarak döner; oyunda eylem yapamaz.
	code := a.scalar(`select invite_code from rooms where id = $1`, l.roomID).(string)
	if r := a.do("POST", "/v1/rooms/join", l.tokenOf(turn), `{"invite_code":"`+code+`"}`); r.code != 200 {
		t.Fatalf("geri dönüş: %d %s", r.code, r.raw)
	}
	if role := a.scalar(`select role from room_members where room_id = $1 and user_id = $2`, l.roomID, turn); role != "spectator" {
		t.Fatalf("rol %v", role)
	}
	if r := a.act(gameID, l.tokenOf(turn), newActionID(), "discussion_ready", `{}`); errCode(r) != "INVALID_PHASE" && errCode(r) != "NOT_ALIVE" {
		t.Fatalf("izleyici eylemi: %d %s", r.code, r.raw)
	}
	// İzleyici (zaten hayatta değil) tekrar ayrılırsa oyun değişmez.
	v := a.load(gameID).Version
	a.leaveRoom(l, turn)
	if a.load(gameID).Version != v {
		t.Fatal("hayatta olmayanın ayrılması oyunu değiştirdi")
	}
}

func TestIntegrationLeaveEndsGame(t *testing.T) {
	a := newAPI(t, true)
	l, gameID := a.startGame(4, 2)
	a.advance(gameID, nil)
	a.leaveRoom(l, l.users[1])
	if a.load(gameID).Phase == engine.PhaseFinished {
		t.Fatal("3 hayatta: bitmemeli")
	}
	before := len(a.events.all())
	a.leaveRoom(l, l.users[3])
	s := a.load(gameID)
	if s.Phase != engine.PhaseFinished || s.EndReason != engine.EndCapacityReached || len(s.Winners()) != 2 {
		t.Fatalf("bitiş: %s %s", s.Phase, s.EndReason)
	}
	if a.scalar(`select status from rooms where id = $1`, l.roomID) != "finished" {
		t.Fatal("oda bitmedi")
	}
	if n := a.scalar(`select count(*) from game_participants where game_id = $1 and is_winner`, gameID).(int64); n != 2 {
		t.Fatalf("%d kazanan", n)
	}
	finished := a.eventsSince(before, engine.EventFinished)
	if len(finished) != 1 || finished[0].Payload.(Result).EndReason != "capacity_reached" {
		t.Fatalf("game.finished %+v", finished)
	}
	waitFor(t, "zamanlayıcı bıraktı", func() bool { return a.timers.Pending() == 0 })
	// Ayrılan da katılımcıdır: sonucu görür.
	if r := a.do("GET", "/v1/games/"+gameID+"/result", l.tokenOf(l.users[1]), ""); r.code != 200 {
		t.Fatalf("ayrılanın sonucu: %d", r.code)
	}
	// Bitmiş odada ayrılma oyunu etkilemez.
	v := s.Version
	a.leaveRoom(l, l.users[0])
	if a.load(gameID).Version != v {
		t.Fatal("bitmiş oyun değişti")
	}
}

func TestIntegrationLeaveCompletesVoting(t *testing.T) {
	a := newAPI(t, true)
	l, gameID := a.startGame(5, 1)
	a.toPhase(gameID, engine.PhaseVoting)
	// u4 dışında herkes oy verdi (u3'e 2 oy, u4'e 1 oy); u4 ayrılınca sayım hemen.
	a.voteAll(l, gameID, map[int]int{0: 3, 1: 3, 2: 4, 3: 0})
	a.leaveRoom(l, l.users[4])
	s := a.load(gameID)
	if s.Phase != engine.PhaseElimination || s.Results[0].EliminatedID != l.users[3] {
		t.Fatalf("sayım: %s %+v", s.Phase, s.Results)
	}
	for _, te := range s.Results[0].Tally {
		if te.UserID == l.users[4] {
			t.Fatal("ayrılan aday sayımda")
		}
	}
}

// Sırası gelen oyuncunun kart açması ile ayrılması aynı anda: hangisi önce
// gelirse gelsin durum tutarlı, 500 yok.
func TestIntegrationLeaveConcurrent(t *testing.T) {
	a := newAPI(t, true)
	for range 5 {
		l, gameID := a.startGame(6, 2)
		s := a.advance(gameID, nil)
		turn := s.Players[s.TurnSeat].UserID
		others := []string{}
		for _, p := range s.Players {
			if p.UserID != turn {
				others = append(others, p.UserID)
			}
		}
		var wg sync.WaitGroup
		var revealCode int
		wg.Add(4)
		go func() {
			defer wg.Done()
			revealCode = a.act(gameID, l.tokenOf(turn), newActionID(), "reveal_card", `{"category":"profession"}`).code
		}()
		go func() { defer wg.Done(); a.leaveRoom(l, turn) }()
		for _, u := range others[:2] {
			go func() { defer wg.Done(); a.leaveRoom(l, u) }()
		}
		wg.Wait()
		if revealCode != 200 && revealCode != 409 {
			t.Fatalf("kart açma: %d", revealCode)
		}
		st := a.load(gameID)
		if st.AliveCount() != 3 || st.Phase != engine.PhaseReveal && st.Phase != engine.PhaseDiscussion {
			t.Fatalf("durum: %d hayatta, %s", st.AliveCount(), st.Phase)
		}
		if st.Phase == engine.PhaseReveal && !st.Players[st.TurnSeat].Alive() {
			t.Fatal("sıra ayrılmış oyuncuda")
		}
		if n := a.scalar(`select count(*) from game_actions where game_id = $1 and type = 'leave'`, gameID).(int64); n != 3 {
			t.Fatalf("%d ayrılma kaydı", n)
		}
	}
}

// Oda sahibi oyunda ayrılırsa oyunda özel yetkisi olmadığı için oyun sürer,
// sahiplik devredilir (game-design §4).
func TestIntegrationHostLeavesDuringGame(t *testing.T) {
	a := newAPI(t, true)
	l, gameID := a.startGame(5, 1)
	a.advance(gameID, nil)
	before := len(a.events.all())
	a.leaveRoom(l, l.users[0])
	if a.load(gameID).Phase == engine.PhaseFinished {
		t.Fatal("oyun bitmemeli")
	}
	host := a.scalar(`select host_user_id::text from rooms where id = $1`, l.roomID)
	if host == l.users[0] || len(a.eventsSince(before, room.EventHostChanged)) != 1 ||
		len(a.eventsSince(before, engine.EventPlayerEliminated)) != 1 {
		t.Fatalf("sahiplik: %v", host)
	}
}
