package engine

import (
	"errors"
	"testing"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
)

func TestDiscussionReady(t *testing.T) {
	e, s := newGame(t, 4, 1)
	if _, _, err := e.DiscussionReady(s, uid(0), t0); apperr.CodeOf(err) != apperr.InvalidPhase {
		t.Fatalf("aşama dışı: %v", err)
	}
	s = until(t, e, s, PhaseDiscussion, fixedRand(0))
	now := s.PhaseStartedAt
	ns, ev, err := e.DiscussionReady(s, uid(2), now)
	if err != nil || ns.Version != s.Version+1 || ns.Phase != PhaseDiscussion {
		t.Fatalf("hazır: %v", err)
	}
	if ev[0].Type != EventDiscussionReady || ev[0].Payload["count"] != 1 || ev[0].Payload["total"] != 4 {
		t.Fatalf("olay %+v", ev)
	}
	if _, _, err := e.DiscussionReady(ns, uid(2), now); !errors.Is(err, ErrNoChange) {
		t.Fatalf("ikinci hazır: %v", err)
	}
	if _, _, err := e.DiscussionReady(ns, "yok", now); apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("yabancı: %v", err)
	}
	for _, u := range []string{uid(0), uid(1)} {
		ns, _, _ = e.DiscussionReady(ns, u, now)
	}
	if ns.Phase != PhaseDiscussion {
		t.Fatal("herkes hazır olmadan oylama başladı")
	}
	ns, ev, err = e.DiscussionReady(ns, uid(3), now)
	if err != nil || ns.Phase != PhaseVoting || !ns.Deadline.Equal(now.Add(DefaultDurations.Voting)) || types(ev)[1] != EventPhaseChanged {
		t.Fatalf("erken oylama: %s %v", ns.Phase, types(ev))
	}
}

func TestDiscussionReadyIgnoresEliminatedAndResetsPerRound(t *testing.T) {
	e, s := newGame(t, 5, 1)
	s = until(t, e, s, PhaseDiscussion, fixedRand(0))
	for i := range 5 {
		s.Players[i].ReadyRound = 1 // tur 1'de herkes hazırdı (önceki tur gibi)
	}
	s = until(t, e, s, PhaseVoting, fixedRand(0))
	s.Votes = []Vote{{uid(0), uid(4)}}
	s = until(t, e, s, PhaseDiscussion, fixedRand(0)) // tur 2
	if s.Round != 2 || s.ReadyCount() != 0 {
		t.Fatalf("tur 2 hazır sayısı %d", s.ReadyCount())
	}
	if _, _, err := e.DiscussionReady(s, uid(4), s.PhaseStartedAt); apperr.CodeOf(err) != apperr.NotAlive {
		t.Fatalf("elenmiş: %v", err)
	}
	var err error
	for _, u := range []string{uid(0), uid(1), uid(2), uid(3)} {
		if s, _, err = e.DiscussionReady(s, u, s.PhaseStartedAt); err != nil {
			t.Fatal(err)
		}
	}
	if s.Phase != PhaseVoting {
		t.Fatalf("4 hayattaki hazır: %s", s.Phase)
	}
}

func TestVoteRules(t *testing.T) {
	e, s := newGame(t, 5, 1)
	if _, _, err := e.Vote(s, uid(0), uid(1), t0, fixedRand(0)); apperr.CodeOf(err) != apperr.InvalidPhase {
		t.Fatalf("aşama dışı: %v", err)
	}
	s = until(t, e, s, PhaseVoting, fixedRand(0))
	now := s.PhaseStartedAt
	cases := []struct {
		voter, target string
		want          apperr.Code
	}{
		{uid(0), uid(0), apperr.SelfVote},
		{uid(0), "yok", apperr.InvalidTarget},
		{"yok", uid(1), apperr.NotFound},
	}
	for _, c := range cases {
		if _, _, err := e.Vote(s, c.voter, c.target, now, fixedRand(0)); apperr.CodeOf(err) != c.want {
			t.Errorf("%s→%s: %v", c.voter, c.target, err)
		}
	}
	ns, ev, err := e.Vote(s, uid(0), uid(3), now, fixedRand(0))
	if err != nil || len(ns.Votes) != 1 || ev[0].Type != EventVoteProgress || ev[0].Payload["cast"] != 1 || ev[0].Payload["total"] != 5 {
		t.Fatalf("oy: %v %+v", err, ev)
	}
	if len(s.Votes) != 0 {
		t.Fatal("girdi durumu değişti")
	}
	for _, p := range ev {
		if _, ok := p.Payload["target"]; ok {
			t.Fatal("ilerleme olayı hedefi sızdırıyor")
		}
	}
	if _, _, err := e.Vote(ns, uid(0), uid(2), now, fixedRand(0)); apperr.CodeOf(err) != apperr.AlreadyVoted {
		t.Fatalf("ikinci oy: %v", err)
	}
	// Hepsi oy verince sayım hemen: u03 3 oyla elenir.
	for _, v := range [][2]string{{uid(1), uid(3)}, {uid(2), uid(3)}, {uid(3), uid(0)}} {
		if ns, _, err = e.Vote(ns, v[0], v[1], now, fixedRand(0)); err != nil || ns.Phase != PhaseVoting {
			t.Fatalf("ara oy: %v %s", err, ns.Phase)
		}
	}
	ns, ev, err = e.Vote(ns, uid(4), uid(0), now, fixedRand(0))
	if err != nil || ns.Phase != PhaseElimination || ns.Results[0].EliminatedID != uid(3) || ns.Results[0].Method != ByVote {
		t.Fatalf("erken sayım: %v %s %+v", err, ns.Phase, ns.Results)
	}
	if got := types(ev); got[0] != EventVoteProgress || got[len(got)-1] != EventPlayerEliminated {
		t.Fatalf("olaylar %v", got)
	}
	// Elenmiş oyuncuya / elenmiş oyuncudan oy (sonraki tur).
	ns = until(t, e, ns, PhaseVoting, fixedRand(0))
	if _, _, err := e.Vote(ns, uid(0), uid(3), ns.PhaseStartedAt, fixedRand(0)); apperr.CodeOf(err) != apperr.InvalidTarget {
		t.Fatalf("elenmiş hedef: %v", err)
	}
	if _, _, err := e.Vote(ns, uid(3), uid(0), ns.PhaseStartedAt, fixedRand(0)); apperr.CodeOf(err) != apperr.NotAlive {
		t.Fatalf("elenmiş oy veren: %v", err)
	}
}

func TestVoteEarlyTieGoesToTieBreak(t *testing.T) {
	e, s := newGame(t, 6, 1)
	s = until(t, e, s, PhaseVoting, fixedRand(0))
	s.Votes = []Vote{{uid(0), uid(5)}}
	s, _ = tick(t, e, s, fixedRand(0))            // u05 elendi
	s = until(t, e, s, PhaseVoting, fixedRand(0)) // tur 2: u05 izleyici
	if s.Round != 2 {
		t.Fatalf("tur %d", s.Round)
	}
	var err error
	for _, v := range [][2]string{{uid(0), uid(1)}, {uid(1), uid(0)}, {uid(2), uid(1)}, {uid(3), uid(0)}, {uid(4), uid(2)}} {
		if s, _, err = e.Vote(s, v[0], v[1], s.PhaseStartedAt, fixedRand(0)); err != nil {
			t.Fatal(err)
		}
	}
	if s.Phase != PhaseTieBreak || len(s.TieCandidates) != 2 {
		t.Fatalf("eşitlik: %s %v", s.Phase, s.TieCandidates)
	}
}
