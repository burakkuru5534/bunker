package game

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	mrand "math/rand/v2"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
)

func newActionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6], b[8] = b[6]&0x0f|0x40, b[8]&0x3f|0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func (a *api) act(gameID, tok, id, typ, payload string) resp {
	a.t.Helper()
	body := fmt.Sprintf(`{"action_id":%q,"type":%q,"payload":%s}`, id, typ, payload)
	return a.do("POST", "/v1/games/"+gameID+"/actions", tok, body)
}

// tokenOf, kullanıcının token'ıdır.
func (l lobby) tokenOf(userID string) string {
	return l.tokens[slices.Index(l.users, userID)]
}

func (a *api) cardText(gameID, userID string, c engine.Category) string {
	return a.scalar(`select d.text from player_cards pc join card_definitions d on d.id = pc.card_id
		where pc.game_id = $1 and pc.user_id = $2 and pc.category = $3`, gameID, userID, string(c)).(string)
}

func TestIntegrationRevealCard(t *testing.T) {
	a := newAPI(t, true)
	l, gameID := a.startGame(5, 1)
	s := a.advance(gameID, nil) // SCENARIO_INTRO → REVEAL tur 1
	turn := s.Players[s.TurnSeat].UserID
	other := s.Players[(s.TurnSeat+1)%5].UserID
	tok := l.tokenOf(turn)

	if r := a.act(gameID, l.tokenOf(other), newActionID(), "reveal_card", `{"category":"profession"}`); r.code != 409 || errCode(r) != "NOT_YOUR_TURN" {
		t.Fatalf("sırası değil: %d %s", r.code, r.raw)
	}
	if r := a.act(gameID, tok, newActionID(), "reveal_card", `{"category":"health"}`); r.code != 409 || errCode(r) != "CATEGORY_NOT_ALLOWED" {
		t.Fatalf("tur 1 sağlık: %d %s", r.code, r.raw)
	}
	if n := a.scalar(`select count(*) from game_actions`).(int64); n != 0 {
		t.Fatalf("reddedilen eylemler kaydedildi: %d", n)
	}

	before := len(a.events.all())
	id := newActionID()
	r := a.act(gameID, tok, id, "reveal_card", `{"category":"profession"}`)
	if r.code != 200 || r.body["state_version"] != float64(s.Version+1) || r.body["duplicate"] != false {
		t.Fatalf("aç: %d %s", r.code, r.raw)
	}
	var (
		round   *int16
		method  *string
		hasTime bool
	)
	if err := a.env.DB.Pool().QueryRow(context.Background(), `select revealed_round, reveal_method, revealed_at is not null
		from player_cards where game_id = $1 and user_id = $2 and category = 'profession'`, gameID, turn).
		Scan(&round, &method, &hasTime); err != nil {
		t.Fatal(err)
	}
	if round == nil || *round != 1 || *method != "player" || !hasTime {
		t.Fatalf("kart satırı: %v %v %v", round, method, hasTime)
	}
	next := a.load(gameID)
	if next.TurnSeat != (s.TurnSeat+1)%5 || next.Version != s.Version+1 {
		t.Fatalf("sıra geçmedi: koltuk %d", next.TurnSeat)
	}
	evs := a.events.all()[before:]
	if len(evs) != 2 {
		t.Fatalf("%d olay", len(evs))
	}
	cr := evs[0].Payload.(map[string]any)
	if evs[0].Type != engine.EventCardRevealed || evs[0].To != "" || evs[0].Version != s.Version+1 || evs[0].RoomID != l.roomID ||
		cr["user_id"] != turn || cr["category"] != "profession" || cr["auto"] != false ||
		cr["text"] != a.cardText(gameID, turn, engine.Profession) {
		t.Fatalf("card_revealed: %+v", evs[0])
	}
	if _, ok := cr["card_id"]; ok {
		t.Fatal("card_revealed içinde card_id")
	}
	pc := evs[1].Payload.(map[string]any)
	if evs[1].Type != engine.EventPhaseChanged || pc["phase"] != "REVEAL" || pc["turn_user_id"] != other {
		t.Fatalf("phase_changed: %+v", evs[1])
	}

	// Aynı action_id: aynı sürüm, duplicate, yeni olay ve kayıt yok.
	r2 := a.act(gameID, tok, id, "reveal_card", `{"category":"profession"}`)
	if r2.code != 200 || r2.body["state_version"] != r.body["state_version"] || r2.body["duplicate"] != true {
		t.Fatalf("tekrar: %d %s", r2.code, r2.raw)
	}
	if len(a.events.all()) != before+2 || a.scalar(`select count(*) from game_actions`).(int64) != 1 {
		t.Fatal("tekrar yeni olay/kayıt üretti")
	}
	var payload map[string]any
	var typ string
	var actor string
	if err := a.env.DB.Pool().QueryRow(context.Background(), `select type, payload, actor_id::text from game_actions where action_id = $1`, id).
		Scan(&typ, &payload, &actor); err != nil {
		t.Fatal(err)
	}
	if typ != "reveal_card" || payload["category"] != "profession" || actor != turn {
		t.Fatalf("eylem kaydı: %s %v %s", typ, payload, actor)
	}
	// Aynı action_id başka bir oyuncudan: ayrı anahtar (oyuncu başına).
	if r := a.act(gameID, l.tokenOf(other), id, "reveal_card", `{"category":"profession"}`); r.code != 200 || r.body["duplicate"] != false {
		t.Fatalf("diğer oyuncu aynı action_id: %d %s", r.code, r.raw)
	}
	// Yeni action_id ile açık kartı tekrar açmak: artık sırası değil.
	if r := a.act(gameID, tok, newActionID(), "reveal_card", `{"category":"profession"}`); r.code != 409 || errCode(r) != "NOT_YOUR_TURN" {
		t.Fatalf("ikinci açma: %d %s", r.code, r.raw)
	}

	// Gizlilik: odaya giden olaylarda yalnızca açılmış kartların metni var.
	opened := map[string]bool{a.cardText(gameID, turn, engine.Profession): true, a.cardText(gameID, other, engine.Profession): true}
	for _, e := range a.events.all() {
		if e.To != "" {
			continue
		}
		raw, _ := json.Marshal(e.Payload)
		for _, p := range s.Players {
			for _, c := range engine.Categories {
				if text := a.cardText(gameID, p.UserID, c); !opened[text] && strings.Contains(string(raw), text) {
					t.Fatalf("%s olayında açılmamış kart: %q", e.Type, text)
				}
			}
		}
	}
}

func TestIntegrationRevealLaterRoundsAndSpectator(t *testing.T) {
	a := newAPI(t, true)
	l, gameID := a.startGame(5, 1)
	s := a.advance(gameID, nil)
	for s.Phase != engine.PhaseVoting {
		s = a.advance(gameID, nil)
	}
	// Oy yok → izleyici yok → kura; sabit rng 0 → koltuk sırasında ilk aday.
	s = a.advance(gameID, mrand.New(mrand.NewPCG(1, 1)))
	eliminated := s.Results[0].EliminatedID
	s = a.advance(gameID, nil) // ELIMINATION → tur 2 REVEAL
	if s.Round != 2 || s.Phase != engine.PhaseReveal {
		t.Fatalf("tur 2: %s", s.Phase)
	}
	turn := s.Players[s.TurnSeat].UserID
	if r := a.act(gameID, l.tokenOf(turn), newActionID(), "reveal_card", `{"category":"profession"}`); errCode(r) != "CARD_ALREADY_REVEALED" {
		t.Fatalf("açık kart: %d %s", r.code, r.raw)
	}
	if r := a.act(gameID, l.tokenOf(turn), newActionID(), "reveal_card", `{"category":"secret"}`); r.code != 200 {
		t.Fatalf("gizli sır: %d %s", r.code, r.raw)
	}
	if r := a.act(gameID, l.tokenOf(eliminated), newActionID(), "reveal_card", `{"category":"health"}`); r.code != 409 || errCode(r) != "NOT_ALIVE" {
		t.Fatalf("elenmiş: %d %s", r.code, r.raw)
	}
	// Tartışmada kart açma: INVALID_PHASE.
	for s = a.load(gameID); s.Phase == engine.PhaseReveal; {
		s = a.advance(gameID, nil)
	}
	if r := a.act(gameID, l.tokenOf(turn), newActionID(), "reveal_card", `{"category":"health"}`); errCode(r) != "INVALID_PHASE" {
		t.Fatalf("tartışmada: %d %s", r.code, r.raw)
	}
}

func TestIntegrationActionValidationAndAccess(t *testing.T) {
	a := newAPI(t, true)
	l, gameID := a.startGame(4, 1)
	tok := l.tokens[0]
	field := func(r resp) string {
		f, _ := r.body["error"].(map[string]any)["details"].(map[string]any)["fields"].(map[string]any)
		b, _ := json.Marshal(f)
		return string(b)
	}
	cases := map[string]struct{ body, field string }{
		"action_id yok":        {`{"type":"reveal_card","payload":{"category":"profession"}}`, `"action_id":"required"`},
		"action_id bozuk":      {`{"action_id":"abc","type":"reveal_card","payload":{"category":"profession"}}`, `"action_id":"invalid"`},
		"tür yok":              {`{"action_id":"` + newActionID() + `","payload":{}}`, `"type":"required"`},
		"bilinmeyen tür":       {`{"action_id":"` + newActionID() + `","type":"fly","payload":{}}`, `"type":"not_allowed"`},
		"eşitlik oyu hedefsiz": {`{"action_id":"` + newActionID() + `","type":"tiebreak_vote","payload":{}}`, `"payload.target_user_id":"required"`},
		"kategori yok":         {`{"action_id":"` + newActionID() + `","type":"reveal_card","payload":{}}`, `"payload.category":"required"`},
		"yük yok":              {`{"action_id":"` + newActionID() + `","type":"reveal_card"}`, `"payload.category":"required"`},
		"bilinmeyen alan":      {`{"action_id":"` + newActionID() + `","type":"reveal_card","payload":{"category":"profession","x":1}}`, `"payload":"unknown_field"`},
		"yük dizi":             {`{"action_id":"` + newActionID() + `","type":"reveal_card","payload":[1]}`, `"payload":"invalid_type"`},
		"kategori sayı":        {`{"action_id":"` + newActionID() + `","type":"reveal_card","payload":{"category":5}}`, `"payload.category":"invalid_type"`},
		"bilinmeyen kategori":  {`{"action_id":"` + newActionID() + `","type":"reveal_card","payload":{"category":"weapon"}}`, `"payload.category":"invalid"`},
	}
	a.advance(gameID, nil)
	for name, c := range cases {
		r := a.do("POST", "/v1/games/"+gameID+"/actions", tok, c.body)
		if r.code != 400 || errCode(r) != "VALIDATION_FAILED" || !strings.Contains(field(r), c.field) {
			t.Errorf("%s: %d %s", name, r.code, r.raw)
		}
	}
	// Erişim: katılımcı değil / olmayan / bozuk kimlik → aynı 404; oturumsuz 401.
	outsider, _ := a.guest("Yabanci")
	body := `{"action_id":"` + newActionID() + `","type":"reveal_card","payload":{"category":"profession"}}`
	var notFound []string
	for _, c := range []struct{ id, tok string }{{gameID, outsider}, {newActionID(), tok}, {"bozuk", tok}} {
		r := a.do("POST", "/v1/games/"+c.id+"/actions", c.tok, body)
		if r.code != 404 {
			t.Fatalf("erişim %s: %d %s", c.id, r.code, r.raw)
		}
		e := r.body["error"].(map[string]any)
		delete(e, "request_id") // her istekte farklı
		raw, _ := json.Marshal(r.body)
		notFound = append(notFound, string(raw))
	}
	if notFound[0] != notFound[1] || notFound[1] != notFound[2] {
		t.Fatalf("katılımcı olmayan yanıtı farklı: %s / %s", notFound[0], notFound[1])
	}
	if r := a.do("POST", "/v1/games/"+gameID+"/actions", "", body); r.code != 401 {
		t.Fatalf("oturumsuz: %d", r.code)
	}
	if n := a.scalar(`select count(*) from game_actions`).(int64); n != 0 {
		t.Fatalf("%d eylem kaydedildi", n)
	}
}

func TestIntegrationRevealConcurrent(t *testing.T) {
	a := newAPI(t, true)
	l, gameID := a.startGame(6, 2)
	s := a.advance(gameID, nil)
	tok := l.tokenOf(s.Players[s.TurnSeat].UserID)
	run := func(ids []string) []resp {
		out := make([]resp, len(ids))
		var wg sync.WaitGroup
		for i, id := range ids {
			wg.Add(1)
			go func() {
				defer wg.Done()
				out[i] = a.act(gameID, tok, id, "reveal_card", `{"category":"profession"}`)
			}()
		}
		wg.Wait()
		return out
	}
	// Aynı action_id 6 kez: 1 işlenir, 5 duplicate, hepsi aynı sürüm.
	id := newActionID()
	same := run(slices.Repeat([]string{id}, 6))
	dups := 0
	for _, r := range same {
		if r.code != 200 || r.body["state_version"] != float64(s.Version+1) {
			t.Fatalf("aynı id: %d %s", r.code, r.raw)
		}
		if r.body["duplicate"] == true {
			dups++
		}
	}
	if dups != 5 {
		t.Fatalf("%d duplicate", dups)
	}
	// Bir sonraki oyuncu 6 farklı action_id ile aynı anda: 1 başarı.
	s = a.load(gameID)
	tok = l.tokenOf(s.Players[s.TurnSeat].UserID)
	var ids []string
	for range 6 {
		ids = append(ids, newActionID())
	}
	ok := 0
	for _, r := range run(ids) {
		switch {
		case r.code == 200:
			ok++
		case r.code == 409 && (errCode(r) == "NOT_YOUR_TURN" || errCode(r) == "CARD_ALREADY_REVEALED"):
		default:
			t.Fatalf("farklı id: %d %s", r.code, r.raw)
		}
	}
	if ok != 1 || a.scalar(`select count(*) from game_actions`).(int64) != 2 ||
		a.scalar(`select state_version from games where id = $1`, gameID).(int64) != s.Version+1 {
		t.Fatalf("başarı %d", ok)
	}
	revealed := 0
	for _, e := range a.events.all() {
		if e.Type == engine.EventCardRevealed {
			revealed++
		}
	}
	if revealed != 2 {
		t.Fatalf("%d card_revealed", revealed)
	}
}
