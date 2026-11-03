package engine

import (
	"slices"
	"testing"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
)

// tieGame: 7 kişi; u06 ve u05 elendi (izleyici), tur 3 oylamasında u00–u01 eşit.
func tieGame(t *testing.T) (Engine, State) {
	t.Helper()
	e, s := newGame(t, 7, 1)
	for _, out := range []int{6, 5} {
		s = until(t, e, s, PhaseVoting, fixedRand(0))
		s.Votes = []Vote{{uid(0), uid(out)}}
		s, _ = tick(t, e, s, fixedRand(0))
	}
	s = until(t, e, s, PhaseVoting, fixedRand(0))
	if s.Round != 3 {
		t.Fatalf("tur %d", s.Round)
	}
	var err error
	for _, v := range [][2]string{{uid(0), uid(1)}, {uid(1), uid(0)}, {uid(2), uid(0)}, {uid(3), uid(1)}, {uid(4), uid(2)}} {
		if s, _, err = e.Vote(s, v[0], v[1], s.PhaseStartedAt, fixedRand(0)); err != nil {
			t.Fatal(err)
		}
	}
	if s.Phase != PhaseTieBreak || !slices.Equal(s.TieCandidates, []string{uid(0), uid(1)}) {
		t.Fatalf("eşitlik: %s %v", s.Phase, s.TieCandidates)
	}
	return e, s
}

func TestTiebreakVoteRules(t *testing.T) {
	e, s := tieGame(t)
	now := s.PhaseStartedAt
	cases := []struct {
		voter, target string
		want          apperr.Code
	}{
		{uid(2), uid(0), apperr.NotSpectator},
		{uid(0), uid(1), apperr.NotSpectator},
		{uid(6), uid(2), apperr.InvalidTarget}, // hayatta ama aday değil
		{uid(6), uid(5), apperr.InvalidTarget}, // elenmiş
		{"yok", uid(0), apperr.NotFound},
	}
	for _, c := range cases {
		if _, _, err := e.TiebreakVote(s, c.voter, c.target, now, fixedRand(0)); apperr.CodeOf(err) != c.want {
			t.Errorf("%s→%s: %v", c.voter, c.target, err)
		}
	}
	ns, ev, err := e.TiebreakVote(s, uid(6), uid(1), now, fixedRand(0))
	if err != nil || ns.Phase != PhaseTieBreak || ev[0].Type != EventTiebreakProgress ||
		ev[0].Payload["cast"] != 1 || ev[0].Payload["total"] != 2 {
		t.Fatalf("ilk oy: %v %+v", err, ev)
	}
	if _, _, err := e.TiebreakVote(ns, uid(6), uid(0), now, fixedRand(0)); apperr.CodeOf(err) != apperr.AlreadyVoted {
		t.Fatalf("ikinci oy: %v", err)
	}
	// Son izleyici aynı adayı seçer: sonuç hemen, izleyici kararı.
	ns, ev, err = e.TiebreakVote(ns, uid(5), uid(1), now, fixedRand(0))
	if err != nil || ns.Phase != PhaseElimination {
		t.Fatalf("erken sonuç: %v %s", err, ns.Phase)
	}
	if r := ns.Results[2]; r.EliminatedID != uid(1) || r.Method != BySpectators {
		t.Fatalf("sonuç %+v", r)
	}
	if got := types(ev); got[0] != EventTiebreakProgress || got[len(got)-1] != EventPlayerEliminated {
		t.Fatalf("olaylar %v", got)
	}
	if _, _, err := e.TiebreakVote(ns, uid(5), uid(1), now, fixedRand(0)); apperr.CodeOf(err) != apperr.InvalidPhase {
		t.Fatalf("aşama dışı: %v", err)
	}
}

func TestTiebreakVoteSplitGoesToLottery(t *testing.T) {
	e, s := tieGame(t)
	now := s.PhaseStartedAt
	s, _, _ = e.TiebreakVote(s, uid(6), uid(0), now, fixedRand(1))
	s, _, err := e.TiebreakVote(s, uid(5), uid(1), now, fixedRand(1))
	if err != nil {
		t.Fatal(err)
	}
	r := s.Results[2]
	if r.Method != ByLottery || r.EliminatedID != uid(1) || !slices.Equal(r.LotteryCandidates, []string{uid(0), uid(1)}) {
		t.Fatalf("kura: %+v", r)
	}
}
