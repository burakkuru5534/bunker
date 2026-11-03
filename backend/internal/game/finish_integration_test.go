package game

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
	"github.com/burakkuru5534/bunker/backend/internal/room"
)

// waitFinished, zamanlayıcının (kısa sürelerle) oyunu bitirmesini bekler.
func (a *api) waitFinished(gameID string) {
	a.t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for a.scalar(`select phase from games where id = $1`, gameID) != "finished" {
		if time.Now().After(deadline) {
			a.t.Fatal("oyun bitmedi")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestIntegrationGameFinishAndResult(t *testing.T) {
	a := newAPI(t, true)
	a.starter.engine, a.svc.engine = engine.New(fast), engine.New(fast)
	l, gameID := a.startGame(5, 2)
	if r := a.do("GET", "/v1/games/"+gameID+"/result", l.tokens[0], ""); r.code != 409 || errCode(r) != "INVALID_PHASE" {
		t.Fatalf("sürerken sonuç: %d %s", r.code, r.raw)
	}
	roomVersion := a.scalar(`select version from rooms where id = $1`, l.roomID).(int64)
	a.waitFinished(gameID)

	// Oda "finished", sürüm bir arttı.
	if a.scalar(`select status from rooms where id = $1`, l.roomID) != "finished" ||
		a.scalar(`select version from rooms where id = $1`, l.roomID).(int64) != roomVersion+1 {
		t.Fatal("oda bitmiş duruma geçmedi")
	}
	g := a.do("GET", "/v1/rooms/"+l.roomID, l.tokens[1], "")
	if g.body["status"] != "finished" || g.body["current_game_id"] != gameID {
		t.Fatalf("oda görünümü: %s", g.raw)
	}

	// Sonuç: katılımcı görür; yabancı / olmayan / bozuk aynı 404.
	r := a.do("GET", "/v1/games/"+gameID+"/result", l.tokens[3], "")
	if r.code != 200 {
		t.Fatalf("sonuç: %d %s", r.code, r.raw)
	}
	var res Result
	if err := json.Unmarshal([]byte(r.raw), &res); err != nil {
		t.Fatal(err)
	}
	if res.GameID != gameID || res.EndReason != "capacity_reached" || res.Capacity != 2 || res.Scenario.Title == "" ||
		len(res.Players) != 5 || len(res.Rounds) != 3 {
		t.Fatalf("sonuç %+v", res)
	}
	winners := 0
	for i, p := range res.Players {
		if len(p.Cards) != 5 || len(p.Nickname) < 3 {
			t.Fatalf("oyuncu %+v", p)
		}
		for j, c := range p.Cards {
			if c.Category != string(engine.Categories[j]) || c.Text != a.cardText(gameID, p.UserID, engine.Categories[j]) {
				t.Fatalf("kart %d/%d %+v", i, j, c)
			}
		}
		if p.IsWinner {
			winners++
			if p.Status != "alive" || p.EliminatedRound != nil {
				t.Fatalf("kazanan %+v", p)
			}
		} else if p.Status != "eliminated" || p.EliminatedRound == nil {
			t.Fatalf("elenen %+v", p)
		}
	}
	if winners != 2 {
		t.Fatalf("%d kazanan", winners)
	}
	for i, rd := range res.Rounds {
		if rd.Round != i+1 || rd.EliminatedUserID == nil || rd.Method == "" || len(rd.Tally) == 0 {
			t.Fatalf("tur %+v", rd)
		}
	}
	if strings.Contains(r.raw, "card_id") || strings.Contains(r.raw, "balance") || strings.Contains(r.raw, "target") {
		t.Fatalf("sonuçta gizli alan: %s", r.raw)
	}
	outsider, _ := a.guest("Yabanci")
	if r := a.do("GET", "/v1/games/"+gameID+"/result", outsider, ""); r.code != 404 {
		t.Fatalf("yabancı: %d", r.code)
	}
	for _, id := range []string{newUUID(), "bozuk"} {
		if r := a.do("GET", "/v1/games/"+id+"/result", l.tokens[0], ""); r.code != 404 {
			t.Fatalf("%s: %d", id, r.code)
		}
	}

	// game.finished olayı tam olarak GET /result yanıtı.
	var finished []room.Event
	for _, e := range a.events.all() {
		if e.Type == engine.EventFinished && e.GameID == gameID {
			finished = append(finished, e)
		}
	}
	if len(finished) != 1 || finished[0].To != "" || finished[0].RoomID != l.roomID {
		t.Fatalf("game.finished %+v", finished)
	}
	evJSON, _ := json.Marshal(finished[0].Payload)
	resJSON, _ := json.Marshal(res)
	if string(evJSON) != string(resJSON) {
		t.Fatalf("olay ve sonuç farklı\nolay:  %s\nsonuç: %s", evJSON, resJSON)
	}
}

func TestIntegrationRematch(t *testing.T) {
	a := newAPI(t, true)
	a.starter.engine, a.svc.engine = engine.New(fast), engine.New(fast)
	l, gameID := a.startGame(4, 1)
	if r := a.do("POST", "/v1/rooms/"+l.roomID+"/rematch", l.tokens[0], ""); r.code != 409 || errCode(r) != "INVALID_PHASE" {
		t.Fatalf("oyun sürerken: %d %s", r.code, r.raw)
	}
	a.waitFinished(gameID)
	if r := a.do("POST", "/v1/rooms/"+l.roomID+"/rematch", l.tokens[1], ""); r.code != 403 || errCode(r) != "FORBIDDEN" {
		t.Fatalf("sahip değil: %d %s", r.code, r.raw)
	}
	outsider, _ := a.guest("Yabanci")
	if r := a.do("POST", "/v1/rooms/"+l.roomID+"/rematch", outsider, ""); r.code != 404 {
		t.Fatalf("üye değil: %d", r.code)
	}
	// Biri oyundan sonra ayrılıp izleyici olarak dönmüş olsun: tekrar oynada oyuncu olur.
	a.migrateExec(`update room_members set role = 'spectator', is_ready = false where room_id = $1 and user_id = $2`, l.roomID, l.users[3])
	before := len(a.events.all())
	r := a.do("POST", "/v1/rooms/"+l.roomID+"/rematch", l.tokens[0], "")
	if r.code != 200 || r.body["status"] != "lobby" || r.body["current_game_id"] != nil {
		t.Fatalf("tekrar oyna: %d %s", r.code, r.raw)
	}
	for _, m := range r.body["members"].([]any) {
		mm := m.(map[string]any)
		if mm["role"] != "player" || mm["is_ready"] != false {
			t.Fatalf("üye %+v", mm)
		}
	}
	version := a.scalar(`select version from rooms where id = $1`, l.roomID).(int64)
	if r.body["version"] != float64(version) {
		t.Fatalf("yanıt sürümü %v, oda %d", r.body["version"], version)
	}
	evs := a.eventsSince(before, room.EventReturnedToLobby)
	if len(evs) != 1 || evs[0].Version != version {
		t.Fatalf("returned_to_lobby %+v", evs)
	}
	view := evs[0].Payload.(map[string]any)["room"].(room.View)
	if view.Version != version || view.Status != room.StatusLobby || len(view.Members) != 4 {
		t.Fatalf("olay görünümü %+v", view)
	}
	// İkinci kez: artık lobide.
	if r := a.do("POST", "/v1/rooms/"+l.roomID+"/rematch", l.tokens[0], ""); errCode(r) != "INVALID_PHASE" {
		t.Fatalf("ikinci: %d %s", r.code, r.raw)
	}
	// Yeni oyun aynı odada başlar; eski sonuç okunmaya devam eder.
	for _, tok := range l.tokens {
		a.do("POST", "/v1/rooms/"+l.roomID+"/ready", tok, `{"ready":true}`)
	}
	r = a.start(l, l.tokens[0])
	if r.code != 201 || r.body["game_id"] == gameID {
		t.Fatalf("yeni oyun: %d %s", r.code, r.raw)
	}
	if r := a.do("GET", "/v1/games/"+gameID+"/result", l.tokens[2], ""); r.code != 200 {
		t.Fatalf("eski sonuç: %d", r.code)
	}
	if n := a.scalar(`select count(*) from games where room_id = $1`, l.roomID).(int64); n != 2 {
		t.Fatalf("%d oyun", n)
	}
}

// Kilit sırası oda → oyun: oda satırı başka bir işlemde kilitliyken oyun
// eylemi oyunu kilitlemeden önce odayı bekler (deadlock önlemi).
func TestIntegrationActionLocksRoomFirst(t *testing.T) {
	a := newAPI(t, true)
	l, gameID := a.startGame(4, 1)
	s := a.advance(gameID, nil)
	ctx := t.Context()
	tx, err := a.env.Migrate.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `select 1 from rooms where id = $1 for update`, l.roomID); err != nil {
		t.Fatal(err)
	}
	done := make(chan resp, 1)
	go func() {
		done <- a.act(gameID, l.tokenOf(s.Players[s.TurnSeat].UserID), newActionID(), "reveal_card", `{"category":"profession"}`)
	}()
	select {
	case r := <-done:
		t.Fatalf("eylem oda kilidini beklemedi: %d %s", r.code, r.raw)
	case <-time.After(300 * time.Millisecond):
	}
	// Bu sırada oyun satırı kilitli olmamalı (eylem henüz oyuna ulaşmadı).
	var free bool
	if err := a.env.Migrate.QueryRow(ctx, `select exists (select 1 from games where id = $1 for update skip locked)`, gameID).Scan(&free); err != nil || !free {
		t.Fatalf("oyun satırı oda kilidinden önce kilitlendi (%v)", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if r := <-done; r.code != 200 {
		t.Fatalf("kilit sonrası: %d %s", r.code, r.raw)
	}
}
