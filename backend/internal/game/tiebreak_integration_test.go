package game

import (
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
)

// voteAll, hayattaki herkesin oyunu HTTP üzerinden verir (votes: oy veren → hedef).
func (a *api) voteAll(l lobby, gameID string, votes map[int]int) {
	a.t.Helper()
	for voter, target := range votes {
		r := a.act(gameID, l.tokens[voter], newActionID(), "vote", fmt.Sprintf(`{"target_user_id":%q}`, l.users[target]))
		if r.code != 200 {
			a.t.Fatalf("oy %d→%d: %d %s", voter, target, r.code, r.raw)
		}
	}
}

// tieLobby: n kişi; out listesindekiler sırayla oyla elenir (izleyici olur);
// sonra hayattakiler tie oylarıyla eşitliğe düşer. TIE_BREAK durumunu döndürür.
func (a *api) tieLobby(n int, out []int, tie map[int]int) (lobby, string, engine.State) {
	a.t.Helper()
	l, gameID := a.startGame(n, 1)
	eliminated := map[int]bool{}
	for _, victim := range out {
		a.toPhase(gameID, engine.PhaseVoting)
		votes := map[int]int{}
		for i := range n {
			if eliminated[i] {
				continue
			}
			if i == victim {
				votes[i] = firstAliveExcept(n, eliminated, victim)
			} else {
				votes[i] = victim
			}
		}
		a.voteAll(l, gameID, votes)
		eliminated[victim] = true
	}
	a.toPhase(gameID, engine.PhaseVoting)
	a.voteAll(l, gameID, tie)
	s := a.load(gameID)
	if s.Phase != engine.PhaseTieBreak {
		a.t.Fatalf("eşitlik bekleniyordu: %s", s.Phase)
	}
	return l, gameID, s
}

func firstAliveExcept(n int, eliminated map[int]bool, except int) int {
	for i := range n {
		if !eliminated[i] && i != except {
			return i
		}
	}
	return -1
}

func TestIntegrationTiebreakVote(t *testing.T) {
	a := newAPI(t, true)
	before := len(a.events.all())
	// 6 kişi; u5 elenir; tur 2'de u0–u1 2'şer oyla eşit.
	l, gameID, s := a.tieLobby(6, []int{5}, map[int]int{0: 1, 1: 0, 2: 0, 3: 1, 4: 2})
	started := a.eventsSince(before, engine.EventTiebreakStarted)
	pair := slices.Sorted(slices.Values([]string{l.users[0], l.users[1]})) // sıra koltuğa göre (rastgele)
	sorted := func(v []string) []string { return slices.Sorted(slices.Values(v)) }
	if len(started) != 1 || !slices.Equal(sorted(started[0].Payload.(map[string]any)["candidates"].([]string)), pair) {
		t.Fatalf("tiebreak_started %+v", started)
	}
	if !slices.Equal(sorted(s.TieCandidates), pair) {
		t.Fatalf("adaylar %v", s.TieCandidates)
	}
	tb := func(voter, target int) resp {
		return a.act(gameID, l.tokens[voter], newActionID(), "tiebreak_vote", fmt.Sprintf(`{"target_user_id":%q}`, l.users[target]))
	}
	if r := tb(2, 0); errCode(r) != "NOT_SPECTATOR" {
		t.Fatalf("hayattaki: %d %s", r.code, r.raw)
	}
	if r := tb(5, 2); errCode(r) != "INVALID_TARGET" {
		t.Fatalf("aday dışı: %d %s", r.code, r.raw)
	}
	if r := a.act(gameID, l.tokens[2], newActionID(), "vote", fmt.Sprintf(`{"target_user_id":%q}`, l.users[0])); errCode(r) != "INVALID_PHASE" {
		t.Fatalf("eşitlikte normal oy: %d %s", r.code, r.raw)
	}
	mark := len(a.events.all())
	// Tek izleyici: kararı o verir, sonuç hemen.
	if r := tb(5, 1); r.code != 200 {
		t.Fatalf("izleyici oyu: %d %s", r.code, r.raw)
	}
	s = a.load(gameID)
	if s.Phase != engine.PhaseElimination || s.Results[1].EliminatedID != l.users[1] || s.Results[1].Method != engine.BySpectators {
		t.Fatalf("sonuç %s %+v", s.Phase, s.Results)
	}
	if got := a.scalar(`select method from round_results where game_id = $1 and round = 2`, gameID); got != "spectators" {
		t.Fatalf("DB yöntemi %v", got)
	}
	if n := a.scalar(`select count(*) from votes where game_id = $1 and round = 2 and kind = 'tiebreak'`, gameID).(int64); n != 1 {
		t.Fatalf("%d eşitlik oyu satırı", n)
	}
	prog := a.eventsSince(mark, engine.EventTiebreakProgress)
	if len(prog) != 1 || prog[0].Payload.(map[string]any)["cast"] != 1 || prog[0].Payload.(map[string]any)["total"] != 1 {
		t.Fatalf("ilerleme %+v", prog)
	}
	el := a.eventsSince(mark, engine.EventPlayerEliminated)
	if len(el) != 1 || el[0].Payload.(map[string]any)["method"] != "spectators" {
		t.Fatalf("eleme olayı %+v", el)
	}
	checkEliminatedEvent(t, a, gameID, el[0], l.users[1])
	if n := a.scalar(`select count(*) from game_actions where game_id = $1 and type = 'tiebreak_vote' and payload <> '{}'::jsonb`, gameID).(int64); n != 0 {
		t.Fatal("eylem kaydında eşitlik oyu hedefi var")
	}
}

// İki izleyici aynı anda farklı adaylara oy verir: ikisi de kabul edilir,
// sonuç tek kez ve kura ile belirlenir.
func TestIntegrationTiebreakConcurrentSplit(t *testing.T) {
	a := newAPI(t, true)
	// 7 kişi; u6, u5 elenir; tur 3'te u0–u1 eşit.
	l, gameID, _ := a.tieLobby(7, []int{6, 5}, map[int]int{0: 1, 1: 0, 2: 0, 3: 1, 4: 2})
	codes := make([]int, 2)
	var wg sync.WaitGroup
	for i, c := range []struct{ voter, target int }{{6, 0}, {5, 1}} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i] = a.act(gameID, l.tokens[c.voter], newActionID(), "tiebreak_vote",
				fmt.Sprintf(`{"target_user_id":%q}`, l.users[c.target])).code
		}()
	}
	wg.Wait()
	if codes[0] != 200 || codes[1] != 200 {
		t.Fatalf("kodlar %v", codes)
	}
	s := a.load(gameID)
	r := s.Results[2]
	if s.Phase != engine.PhaseElimination || r.Method != engine.ByLottery ||
		!slices.Equal(slices.Sorted(slices.Values(r.LotteryCandidates)), slices.Sorted(slices.Values([]string{l.users[0], l.users[1]}))) ||
		!slices.Contains(r.LotteryCandidates, r.EliminatedID) {
		t.Fatalf("kura: %s %+v", s.Phase, r)
	}
	if n := a.scalar(`select count(*) from round_results where game_id = $1 and round = 3`, gameID).(int64); n != 1 {
		t.Fatalf("%d tur sonucu", n)
	}
}
