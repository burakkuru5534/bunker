package game

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
	"github.com/burakkuru5534/bunker/backend/internal/room"
)

func (a *api) eventsSince(n int, typ string) []room.Event {
	var out []room.Event
	for _, e := range a.events.all()[n:] {
		if typ == "" || e.Type == typ {
			out = append(out, e)
		}
	}
	return out
}

// toPhase, zamanlayıcı yerine elle ilerletir (varsayılan süreler uzun).
func (a *api) toPhase(gameID string, p engine.Phase) engine.State {
	a.t.Helper()
	s := a.load(gameID)
	for s.Phase != p {
		if s.Phase == engine.PhaseFinished {
			a.t.Fatalf("%s aşamasına ulaşılamadı", p)
		}
		s = a.advance(gameID, nil)
	}
	return s
}

func TestIntegrationDiscussionReady(t *testing.T) {
	a := newAPI(t, true)
	l, gameID := a.startGame(4, 1)
	if r := a.act(gameID, l.tokens[0], newActionID(), "discussion_ready", `{}`); errCode(r) != "INVALID_PHASE" {
		t.Fatalf("aşama dışı: %d %s", r.code, r.raw)
	}
	s := a.toPhase(gameID, engine.PhaseDiscussion)
	before := len(a.events.all())
	for i, uid := range l.users {
		r := a.act(gameID, l.tokens[i], newActionID(), "discussion_ready", `{}`)
		if r.code != 200 || r.body["state_version"] != float64(s.Version+int64(i)+1) {
			t.Fatalf("hazır %d: %d %s", i, r.code, r.raw)
		}
		if i == 0 {
			// İkinci "Hazırım": başarı, sürüm aynı, olay yok; aynı id tekrar duplicate.
			id := newActionID()
			r2 := a.act(gameID, l.tokens[0], id, "discussion_ready", `{}`)
			r3 := a.act(gameID, l.tokens[0], id, "discussion_ready", `{}`)
			if r2.code != 200 || r2.body["state_version"] != r.body["state_version"] || r2.body["duplicate"] != false ||
				r3.body["duplicate"] != true || len(a.eventsSince(before, "")) != 1 {
				t.Fatalf("tekrar hazır: %s / %s", r2.raw, r3.raw)
			}
		}
		if got := a.scalar(`select discussion_ready_round from game_participants where game_id = $1 and user_id = $2`, gameID, uid); got != int16(1) {
			t.Fatalf("hazır turu %v", got)
		}
	}
	ready := a.eventsSince(before, engine.EventDiscussionReady)
	for i, e := range ready {
		p := e.Payload.(map[string]any)
		if p["count"] != i+1 || p["total"] != 4 || e.GameID != gameID {
			t.Fatalf("discussion_ready[%d] %+v", i, p)
		}
	}
	if s = a.load(gameID); s.Phase != engine.PhaseVoting {
		t.Fatalf("herkes hazır: %s", s.Phase)
	}
	if r := a.act(gameID, l.tokens[1], newActionID(), "discussion_ready", `{"x":1}`); r.code != 400 {
		t.Fatalf("bilinmeyen alan: %d", r.code)
	}
}

func TestIntegrationVote(t *testing.T) {
	a := newAPI(t, true)
	l, gameID := a.startGame(5, 1)
	s := a.toPhase(gameID, engine.PhaseVoting)
	u := l.users
	vote := func(i int, target string) resp {
		return a.act(gameID, l.tokens[i], newActionID(), "vote", fmt.Sprintf(`{"target_user_id":%q}`, target))
	}
	outsider, outsiderID := a.guest("Yabanci")
	_ = outsider
	checks := []struct {
		r    resp
		code string
	}{
		{vote(0, u[0]), "SELF_VOTE"},
		{vote(0, outsiderID), "INVALID_TARGET"},
		{vote(0, "bozuk"), "VALIDATION_FAILED"},
		{a.act(gameID, l.tokens[0], newActionID(), "vote", `{}`), "VALIDATION_FAILED"},
	}
	for i, c := range checks {
		if errCode(c.r) != c.code {
			t.Errorf("%d: %d %s", i, c.r.code, c.r.raw)
		}
	}
	before := len(a.events.all())
	// u0,u1,u2 → u3; u3 → u0 (büyük harfli UUID de kabul).
	for _, v := range []struct {
		i      int
		target string
	}{{0, u[3]}, {1, strings.ToUpper(u[3])}, {2, u[3]}, {3, u[0]}} {
		if r := vote(v.i, v.target); r.code != 200 {
			t.Fatalf("oy %d: %d %s", v.i, r.code, r.raw)
		}
	}
	if r := vote(0, u[1]); errCode(r) != "ALREADY_VOTED" {
		t.Fatalf("ikinci oy: %d %s", r.code, r.raw)
	}
	if n := a.scalar(`select count(*) from votes where game_id = $1 and round = 1 and kind = 'regular'`, gameID).(int64); n != 4 {
		t.Fatalf("%d oy satırı", n)
	}
	// Gizlilik: hedefler eylem kaydında ve olaylarda yok.
	if n := a.scalar(`select count(*) from game_actions where game_id = $1 and type = 'vote' and payload <> '{}'::jsonb`, gameID).(int64); n != 0 {
		t.Fatal("eylem kaydında oy hedefi var")
	}
	for i, e := range a.eventsSince(before, engine.EventVoteProgress) {
		p := e.Payload.(map[string]any)
		raw, _ := json.Marshal(p)
		if p["cast"] != i+1 || p["total"] != 5 || strings.Contains(string(raw), u[3]) {
			t.Fatalf("vote_progress[%d] %s", i, raw)
		}
	}
	if s = a.load(gameID); s.Phase != engine.PhaseVoting {
		t.Fatal("herkes oy vermeden sayım yapıldı")
	}
	// Son oy: sayım hemen; u3 3 oyla elenir.
	if r := vote(4, u[2]); r.code != 200 {
		t.Fatalf("son oy: %d %s", r.code, r.raw)
	}
	s = a.load(gameID)
	if s.Phase != engine.PhaseElimination || s.Results[0].EliminatedID != u[3] || s.Results[0].Method != engine.ByVote {
		t.Fatalf("eleme: %s %+v", s.Phase, s.Results)
	}
	el := a.eventsSince(before, engine.EventPlayerEliminated)
	if len(el) != 1 {
		t.Fatalf("%d eleme olayı", len(el))
	}
	checkEliminatedEvent(t, a, gameID, el[0], u[3])
	tally := el[0].Payload.(map[string]any)["tally"].([]engine.TallyEntry)
	if tally[0] != (engine.TallyEntry{UserID: u[3], Votes: 3}) {
		t.Fatalf("sayım %+v", tally)
	}
	// Elenmiş oyuncu sonraki turda oy veremez.
	a.toPhase(gameID, engine.PhaseVoting)
	if r := vote(3, u[0]); errCode(r) != "NOT_ALIVE" {
		t.Fatalf("elenmiş oy: %d %s", r.code, r.raw)
	}
}

// checkEliminatedEvent: elenenin 5 kartı metinle, kart kimliği yok.
func checkEliminatedEvent(t *testing.T, a *api, gameID string, e room.Event, user string) {
	t.Helper()
	p := e.Payload.(map[string]any)
	cards := p["cards"].([]map[string]any)
	if p["user_id"] != user || len(cards) != 5 || e.GameID != gameID {
		t.Fatalf("eleme olayı %+v", p)
	}
	for i, c := range cards {
		if c["category"] != string(engine.Categories[i]) || c["text"] != a.cardText(gameID, user, engine.Categories[i]) {
			t.Fatalf("kart %+v", c)
		}
		if _, ok := c["card_id"]; ok {
			t.Fatal("eleme olayında card_id")
		}
	}
}

// Zamanlayıcı yolundaki eleme olayı da metinle gider (card_id sızmaz).
func TestIntegrationTimeoutEliminationEventHasTexts(t *testing.T) {
	a := newAPI(t, true)
	_, gameID := a.startGame(4, 1)
	s := a.toPhase(gameID, engine.PhaseVoting)
	a.expire(gameID)
	before := len(a.events.all())
	if res, err := a.svc.Timeout(context.Background(), gameID, s.Version); err != nil || !res.Advanced {
		t.Fatalf("%+v %v", res, err)
	}
	el := a.eventsSince(before, engine.EventPlayerEliminated)
	if len(el) != 1 {
		t.Fatalf("%d eleme olayı", len(el))
	}
	checkEliminatedEvent(t, a, gameID, el[0], el[0].Payload.(map[string]any)["user_id"].(string))
}

func TestIntegrationVoteConcurrent(t *testing.T) {
	a := newAPI(t, true)
	l, gameID := a.startGame(8, 2)
	a.toPhase(gameID, engine.PhaseVoting)
	before := len(a.events.all())
	codes := make([]int, 8)
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			target := l.users[(i+1)%8]
			if i >= 4 {
				target = l.users[0] // u0'a 4 oy
			}
			codes[i] = a.act(gameID, l.tokens[i], newActionID(), "vote", fmt.Sprintf(`{"target_user_id":%q}`, target)).code
		}()
	}
	wg.Wait()
	for i, c := range codes {
		if c != 200 {
			t.Fatalf("oy %d: %d", i, c)
		}
	}
	var casts []int
	for _, e := range a.eventsSince(before, engine.EventVoteProgress) {
		casts = append(casts, e.Payload.(map[string]any)["cast"].(int))
	}
	slices.Sort(casts)
	if !slices.Equal(casts, []int{1, 2, 3, 4, 5, 6, 7, 8}) {
		t.Fatalf("ilerleme %v", casts)
	}
	if n := a.scalar(`select count(*) from round_results where game_id = $1`, gameID).(int64); n != 1 {
		t.Fatalf("%d tur sonucu", n)
	}
	if s := a.load(gameID); s.Results[0].EliminatedID != l.users[0] {
		t.Fatalf("elenen %s", s.Results[0].EliminatedID)
	}
}

// Oyun, zamanlayıcıyı beklemeden yalnızca oyuncu eylemleriyle sonuna kadar
// oynanabilir: kart açma → hazırım → oy.
func TestIntegrationPlayGameWithActions(t *testing.T) {
	a := newAPI(t, true)
	l, gameID := a.startGame(5, 2)
	s := a.advance(gameID, nil) // senaryo tanıtımı (eylemi yok)
	tok := func(user string) string { return l.tokenOf(user) }
	for steps := 0; s.Phase != engine.PhaseFinished; steps++ {
		if steps > 200 {
			t.Fatal("oyun bitmedi")
		}
		switch s.Phase {
		case engine.PhaseReveal:
			p := s.Players[s.TurnSeat]
			cat, _ := s.AutoRevealCategory(p)
			if r := a.act(gameID, tok(p.UserID), newActionID(), "reveal_card", fmt.Sprintf(`{"category":%q}`, cat)); r.code != 200 {
				t.Fatalf("açma: %s", r.raw)
			}
		case engine.PhaseDiscussion:
			for _, p := range s.Players {
				if p.Alive() {
					a.act(gameID, tok(p.UserID), newActionID(), "discussion_ready", `{}`)
				}
			}
		case engine.PhaseVoting:
			var alive []string
			for _, p := range s.Players {
				if p.Alive() {
					alive = append(alive, p.UserID)
				}
			}
			for i, v := range alive { // herkes bir sonrakine; ilk iki oy aynı kişiye
				target := alive[(i+1)%len(alive)]
				if i == 0 {
					target = alive[2]
				}
				if r := a.act(gameID, tok(v), newActionID(), "vote", fmt.Sprintf(`{"target_user_id":%q}`, target)); r.code != 200 {
					t.Fatalf("oy: %s", r.raw)
				}
			}
		case engine.PhaseElimination:
			s = a.advance(gameID, nil) // elemenin eylemi yok: süre dolumu
			continue
		default:
			t.Fatalf("beklenmeyen aşama %s", s.Phase)
		}
		s = a.load(gameID)
	}
	if len(s.Winners()) != 2 || len(s.Results) != 3 {
		t.Fatalf("bitiş: kazanan %d, tur %d", len(s.Winners()), len(s.Results))
	}
	for _, r := range s.Results {
		if r.Method != engine.ByVote {
			t.Fatalf("tur %d %s ile bitti (oylar tek en yüksek vermeliydi)", r.Round, r.Method)
		}
	}
	// Hiçbir kart süre dolumuyla açılmadı; hazırım/oy eylemleri kayıtlı.
	if n := a.scalar(`select count(*) from player_cards where game_id = $1 and reveal_method = 'timeout'`, gameID).(int64); n != 0 {
		t.Fatalf("%d kart süre dolumuyla açıldı", n)
	}
	if n := a.scalar(`select count(distinct type) from game_actions where game_id = $1 and actor_id is not null`, gameID).(int64); n != 3 {
		t.Fatalf("%d eylem türü", n)
	}
}
