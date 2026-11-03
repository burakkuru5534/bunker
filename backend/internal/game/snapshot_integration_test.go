package game

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
)

type fakePresence map[string]bool

func (f fakePresence) Connected(id string) bool { return f[id] }

func (a *api) view(gameID, viewer string) (GameView, string) {
	a.t.Helper()
	v, err := a.svc.GameSnapshot(context.Background(), viewer, gameID)
	if err != nil {
		a.t.Fatal(err)
	}
	raw, _ := json.Marshal(v)
	return v.(GameView), string(raw)
}

// cardRow, DB'deki gerçek kart durumu (testin bağımsız doğruluk kaynağı).
type cardRow struct {
	user, text string
	revealed   bool
	alive      bool
}

func (a *api) cards(gameID string) []cardRow {
	a.t.Helper()
	rows, err := a.env.DB.Pool().Query(context.Background(), `select pc.user_id::text, d.text, pc.revealed_round is not null,
		p.status = 'alive' from player_cards pc join card_definitions d on d.id = pc.card_id
		join game_participants p using (game_id, user_id) where pc.game_id = $1`, gameID)
	if err != nil {
		a.t.Fatal(err)
	}
	defer rows.Close()
	var out []cardRow
	for rows.Next() {
		var c cardRow
		if err := rows.Scan(&c.user, &c.text, &c.revealed, &c.alive); err != nil {
			a.t.Fatal(err)
		}
		out = append(out, c)
	}
	return out
}

// checkPrivacy, ENG-015 değişmezleri: her izleyici için.
func (a *api) checkPrivacy(gameID string, viewers []string) {
	a.t.Helper()
	cards := a.cards(gameID)
	finished := a.scalar(`select phase from games where id = $1`, gameID) == "finished"
	for _, viewer := range viewers {
		v, raw := a.view(gameID, viewer)
		for _, c := range cards {
			visible := c.revealed || finished
			in := strings.Contains(raw, c.text)
			switch {
			case c.user == viewer && !in:
				a.t.Fatalf("%s kendi kartını görmüyor: %q", viewer, c.text)
			case c.user != viewer && !visible && in:
				a.t.Fatalf("%s, %s oyuncusunun kapalı kartını görüyor: %q (aşama %s)", viewer, c.user, c.text, v.Phase)
			case visible && !in:
				a.t.Fatalf("%s açık kartı görmüyor: %q", viewer, c.text)
			}
		}
		if strings.Contains(raw, "target") || strings.Contains(raw, "card_id") || strings.Contains(raw, "balance") {
			a.t.Fatalf("görünümde gizli alan: %s", raw)
		}
		isParticipant := slices.ContainsFunc(v.Participants, func(p ViewParticipant) bool { return p.UserID == viewer })
		if isParticipant && len(v.MyCards) != 5 || !isParticipant && len(v.MyCards) != 0 {
			a.t.Fatalf("%s my_cards %d", viewer, len(v.MyCards))
		}
	}
}

// Rastgele oyunlarda her adımda, her katılımcı ve bir yabancı için
// gizlilik değişmezleri (ENG-015).
func TestIntegrationSnapshotPrivacyProperty(t *testing.T) {
	a := newAPI(t, true)
	for game := range 4 {
		rng := rand.New(rand.NewPCG(uint64(game), 99))
		n := 5 + game
		l, gameID := a.startGame(n, 1+game%2)
		_, outsider := a.guest(fmt.Sprintf("Bakan%d", game))
		viewers := append(slices.Clone(l.users), outsider)
		for steps := 0; ; steps++ {
			a.checkPrivacy(gameID, viewers)
			s := a.load(gameID)
			if s.Phase == engine.PhaseFinished {
				break
			}
			if steps > 400 {
				t.Fatal("oyun bitmedi")
			}
			// Ara sıra biri ayrılır (kapasitenin üstünde kalacak şekilde).
			if s.AliveCount() > s.Capacity+1 && rng.IntN(15) == 0 {
				var alive []string
				for _, p := range s.Players {
					if p.Alive() {
						alive = append(alive, p.UserID)
					}
				}
				a.leaveRoom(l, alive[rng.IntN(len(alive))])
				continue
			}
			switch s.Phase {
			case engine.PhaseReveal:
				if rng.IntN(2) == 0 {
					p := s.Players[s.TurnSeat]
					cat, _ := s.AutoRevealCategory(p)
					a.act(gameID, l.tokenOf(p.UserID), newActionID(), "reveal_card", fmt.Sprintf(`{"category":%q}`, cat))
					continue
				}
			case engine.PhaseVoting:
				for _, p := range s.Players {
					if p.Alive() && rng.IntN(3) > 0 {
						target := s.Players[rng.IntN(n)]
						if target.Alive() && target.UserID != p.UserID {
							a.act(gameID, l.tokenOf(p.UserID), newActionID(), "vote", fmt.Sprintf(`{"target_user_id":%q}`, target.UserID))
						}
					}
				}
			}
			if s = a.load(gameID); s.Phase != engine.PhaseFinished {
				a.advance(gameID, rng)
			}
		}
	}
}

func TestIntegrationSnapshotFields(t *testing.T) {
	a := newAPI(t, true)
	l, gameID := a.startGame(5, 1)
	a.svc.presence = fakePresence{l.users[0]: true}
	v, _ := a.view(gameID, l.users[0])
	if v.Phase != "SCENARIO_INTRO" || v.StateVersion != 1 || v.TotalRounds != 4 || v.Capacity != 1 ||
		v.PhaseDeadlineAt == nil || v.TurnUserID != nil || v.Scenario.Title == "" || len(v.Scenario.Bunker) != 3 ||
		v.LastRoundResult != nil || v.Tiebreak != nil || len(v.Participants) != 5 {
		t.Fatalf("tanıtım görünümü %+v", v)
	}
	for _, p := range v.Participants {
		if p.Connected != (p.UserID == l.users[0]) || len(p.Nickname) < 3 || len(p.RevealedCards) != 0 {
			t.Fatalf("katılımcı %+v", p)
		}
	}
	s := a.advance(gameID, nil)
	if v, _ = a.view(gameID, l.users[1]); v.TurnUserID == nil || *v.TurnUserID != s.Players[s.TurnSeat].UserID || v.Phase != "REVEAL" {
		t.Fatalf("sıra %+v", v.TurnUserID)
	}
	s = a.toPhase(gameID, engine.PhaseDiscussion)
	a.act(gameID, l.tokens[2], newActionID(), "discussion_ready", `{}`)
	if v, _ = a.view(gameID, l.users[2]); !v.MyActionState.DiscussionReady || v.DiscussionReadyCount != 1 {
		t.Fatalf("hazır %+v", v.MyActionState)
	}
	if v, _ = a.view(gameID, l.users[3]); v.MyActionState.DiscussionReady {
		t.Fatal("başkasının hazırı bana yazıldı")
	}
	a.toPhase(gameID, engine.PhaseVoting)
	a.voteAll(l, gameID, map[int]int{0: 3, 1: 3})
	if v, _ = a.view(gameID, l.users[0]); !v.MyActionState.Voted || v.VoteProgress != (Progress{Cast: 2, Total: 5}) {
		t.Fatalf("oy %+v %+v", v.MyActionState, v.VoteProgress)
	}
	if v, _ = a.view(gameID, l.users[4]); v.MyActionState.Voted {
		t.Fatal("başkasının oyu bana yazıldı")
	}
	// Tartışmadaki "Hazırım" oylamada (ve sonraki turlarda) görünmez.
	if v, _ = a.view(gameID, l.users[2]); v.MyActionState.DiscussionReady {
		t.Fatal("tartışma bitti ama hazır hâlâ işaretli")
	}
	a.voteAll(l, gameID, map[int]int{2: 3, 3: 0, 4: 0}) // u3 elenir (3 oy)
	v, raw := a.view(gameID, l.users[0])
	r := v.LastRoundResult
	if v.Phase != "ELIMINATION" || r == nil || r["user_id"] != l.users[3] || r["method"] != "vote" || len(r["cards"].([]map[string]any)) != 5 {
		t.Fatalf("son tur %+v", r)
	}
	if !strings.Contains(raw, a.cardText(gameID, l.users[3], engine.Secret)) {
		t.Fatal("elenenin gizli sırrı görünmüyor")
	}
	// Elenen (izleyici) kendi kartlarını görür; eşitlik yokken tiebreak null.
	if v, _ = a.view(gameID, l.users[3]); len(v.MyCards) != 5 || v.Tiebreak != nil {
		t.Fatalf("izleyici %+v", v.MyCards)
	}
}

func TestIntegrationSnapshotTiebreakAndFinished(t *testing.T) {
	a := newAPI(t, true)
	l, gameID, s := a.tieLobby(6, []int{5}, map[int]int{0: 1, 1: 0, 2: 0, 3: 1, 4: 2})
	v, _ := a.view(gameID, l.users[5])
	if v.Tiebreak == nil || len(v.Tiebreak.Candidates) != 2 || v.Tiebreak.Progress != (Progress{Cast: 0, Total: 1}) {
		t.Fatalf("eşitlik %+v", v.Tiebreak)
	}
	_ = s
	a.act(gameID, l.tokens[5], newActionID(), "tiebreak_vote", fmt.Sprintf(`{"target_user_id":%q}`, v.Tiebreak.Candidates[0]))
	// Oyunu bitir: kalan her şey süre dolumuyla.
	for st := a.load(gameID); st.Phase != engine.PhaseFinished; st = a.load(gameID) {
		a.advance(gameID, rand.New(rand.NewPCG(1, 2)))
	}
	v, raw := a.view(gameID, l.users[0])
	if v.Phase != "FINISHED" || v.PhaseDeadlineAt != nil || v.TurnUserID != nil {
		t.Fatalf("bitmiş %+v", v)
	}
	for _, c := range a.cards(gameID) {
		if !strings.Contains(raw, c.text) {
			t.Fatalf("bitmiş oyunda kart görünmüyor: %q", c.text)
		}
	}
	if _, err := a.svc.GameSnapshot(context.Background(), l.users[0], newUUID()); !isGameNotFound(err) {
		t.Fatalf("olmayan oyun: %v", err)
	}
}
