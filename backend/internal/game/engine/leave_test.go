package engine

import (
	"errors"
	"testing"
)

func leaveOK(t *testing.T, e Engine, s State, user string) (State, []Event) {
	t.Helper()
	ns, ev, err := e.Leave(s, user, s.PhaseStartedAt, fixedRand(0))
	if err != nil {
		t.Fatalf("ayrılma %s: %v", user, err)
	}
	if ns.Version != s.Version+1 {
		t.Fatal("sürüm artmadı")
	}
	return ns, ev
}

func TestLeaveBasics(t *testing.T) {
	e, s := newGame(t, 5, 1) // SCENARIO_INTRO, tur 0
	ns, ev := leaveOK(t, e, s, uid(2))
	p := ns.Players[2]
	if p.Status != Left || p.EliminatedRound != 0 || ns.Phase != PhaseScenarioIntro || len(ns.Results) != 0 {
		t.Fatalf("ayrılan %+v, aşama %s", p, ns.Phase)
	}
	for _, c := range p.Cards {
		if c.RevealedRound != 1 || c.Method != RevealByElimination {
			t.Fatalf("kart %+v (tanıtımda ayrılan kartları tur 1'de açılmalı)", c)
		}
	}
	if ev[0].Type != EventPlayerEliminated || ev[0].Payload["method"] != "left" || len(ev[0].Payload["cards"].([]map[string]any)) != 5 {
		t.Fatalf("olay %+v", ev)
	}
	for _, again := range []string{uid(2), "yok"} {
		if _, _, err := e.Leave(ns, again, t0, fixedRand(0)); !errors.Is(err, ErrNoChange) {
			t.Fatalf("%s: %v", again, err)
		}
	}
	// Ayrılan sıraya girmez: tur 1 sırası 0,1,3,4.
	ns, _ = tick(t, e, ns, fixedRand(0))
	if got := ns.RevealOrder(); len(got) != 4 || got[2] != 3 {
		t.Fatalf("sıra %v", got)
	}
}

func TestLeaveOnTurnAdvances(t *testing.T) {
	e, s := newGame(t, 5, 1)
	s, _ = tick(t, e, s, fixedRand(0)) // REVEAL, koltuk 0
	ns, ev := leaveOK(t, e, s, uid(0))
	if ns.Phase != PhaseReveal || ns.TurnSeat != 1 || !ns.Deadline.Equal(s.PhaseStartedAt.Add(DefaultDurations.RevealPerPlayer)) {
		t.Fatalf("sıra geçmedi: %s koltuk %d", ns.Phase, ns.TurnSeat)
	}
	if types(ev)[1] != EventPhaseChanged {
		t.Fatalf("olaylar %v", types(ev))
	}
	// Sırası gelmeyen biri ayrılırsa sıra değişmez.
	ns2, ev2 := leaveOK(t, e, ns, uid(3))
	if ns2.TurnSeat != 1 || len(ev2) != 1 {
		t.Fatalf("gereksiz geçiş: koltuk %d, %d olay", ns2.TurnSeat, len(ev2))
	}
}

func TestLeaveCompletesDiscussionAndVoting(t *testing.T) {
	e, s := newGame(t, 6, 1)
	s = until(t, e, s, PhaseDiscussion, fixedRand(0))
	for _, u := range []string{uid(0), uid(1), uid(2), uid(3), uid(4)} {
		s, _, _ = e.DiscussionReady(s, u, s.PhaseStartedAt)
	}
	s, _ = leaveOK(t, e, s, uid(5)) // hazır olmayan tek kişi ayrıldı
	if s.Phase != PhaseVoting {
		t.Fatalf("tartışma bitmedi: %s", s.Phase)
	}
	// u4 dışında herkes oy verdi; iki oy u3'e. u4 ayrılınca sayım hemen.
	for _, v := range [][2]string{{uid(0), uid(3)}, {uid(1), uid(3)}, {uid(2), uid(4)}, {uid(3), uid(0)}} {
		var err error
		if s, _, err = e.Vote(s, v[0], v[1], s.PhaseStartedAt, fixedRand(0)); err != nil {
			t.Fatal(err)
		}
	}
	s, _ = leaveOK(t, e, s, uid(4))
	if s.Phase != PhaseElimination || s.Results[0].EliminatedID != uid(3) {
		t.Fatalf("sayım: %s %+v", s.Phase, s.Results)
	}
	// Ayrılana verilen oy geçersiz: sayımda u4 yok.
	for _, te := range s.Results[0].Tally {
		if te.UserID == uid(4) {
			t.Fatal("ayrılan sayımda")
		}
	}
	if s.Players[4].Status != Left {
		t.Fatal("ayrılan elenmiş olarak değil 'left' kalmalı")
	}
}

func TestLeaveEndsGame(t *testing.T) {
	e, s := newGame(t, 4, 2)
	s, _ = tick(t, e, s, fixedRand(0))
	s, _ = leaveOK(t, e, s, uid(1))
	if s.Phase == PhaseFinished {
		t.Fatal("3 hayatta, kapasite 2: bitmemeli")
	}
	s, ev := leaveOK(t, e, s, uid(3))
	if s.Phase != PhaseFinished || s.EndReason != EndCapacityReached || len(s.Winners()) != 2 || types(ev)[1] != EventFinished {
		t.Fatalf("bitiş: %s %s %v", s.Phase, s.EndReason, types(ev))
	}
	if _, _, err := e.Leave(s, uid(0), t0, fixedRand(0)); !errors.Is(err, ErrNoChange) {
		t.Fatalf("bitmiş oyunda: %v", err)
	}
	// Hayatta kalan kapasitenin altına inerse kalanlar kazanır.
	e, s = newGame(t, 4, 2)
	for _, u := range []string{uid(0), uid(1)} {
		s, _ = leaveOK(t, e, s, u)
	}
	if s.Phase != PhaseFinished || len(s.Winners()) != 2 {
		t.Fatalf("kapasiteye inince: %s", s.Phase)
	}
}

func TestLeaveTiebreakCandidate(t *testing.T) {
	e, s := tieGame(t) // u00–u01 eşit; u05, u06 izleyici
	s, _ = leaveOK(t, e, s, uid(0))
	if s.Phase != PhaseElimination || s.Results[2].EliminatedID != uid(1) {
		t.Fatalf("tek aday kaldı: %s %+v", s.Phase, s.Results)
	}
	// Aday olmayan biri ayrılırsa eşitlik sürer (ayrılan da izleyici olur).
	_, s = tieGame(t)
	s, _ = leaveOK(t, e, s, uid(3))
	if s.Phase != PhaseTieBreak || len(s.Spectators()) != 3 {
		t.Fatalf("eşitlik: %s, %d izleyici", s.Phase, len(s.Spectators()))
	}
}

func TestLeaveAllAbandons(t *testing.T) {
	e, s := newGame(t, 4, 1)
	s, _ = tick(t, e, s, fixedRand(0))
	// Ayrılmalar tek tek işlendiği ve hayatta kalan kapasiteye inince oyun
	// bittiği için gerçek oyunda 0 kişiye inilemez; "terk edildi" dalı
	// savunma amaçlıdır. Dalı sınamak için kapasite yapay olarak 0 yapılır.
	s.Capacity = 0
	for _, u := range []string{uid(0), uid(1), uid(2), uid(3)} {
		s, _ = leaveOK(t, e, s, u)
	}
	if s.Phase != PhaseFinished || s.EndReason != EndAbandoned || len(s.Winners()) != 0 {
		t.Fatalf("terk: %s %s", s.Phase, s.EndReason)
	}
}
