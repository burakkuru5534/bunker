package engine

import (
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
)

func TestRevealByPlayer(t *testing.T) {
	e, s := newGame(t, 4, 1)
	s, _ = tick(t, e, s, fixedRand(0)) // tur 1, koltuk 0
	now := s.PhaseStartedAt.Add(3 * time.Second)

	cases := []struct {
		name string
		user string
		cat  Category
		want apperr.Code
	}{
		{"sıra başkasında", uid(1), Profession, apperr.NotYourTurn},
		{"tur 1 meslek dışı", uid(0), Health, apperr.CategoryNotAllowed},
		{"bilinmeyen kategori", uid(0), "weapon", apperr.ValidationFailed},
		{"katılımcı değil", "zz", Profession, apperr.NotFound},
	}
	for _, c := range cases {
		got, _, err := e.Reveal(s, c.user, c.cat, now)
		if apperr.CodeOf(err) != c.want || got.Version != s.Version {
			t.Errorf("%s: %v (v%d)", c.name, err, got.Version)
		}
	}

	ns, ev, err := e.Reveal(s, uid(0), Profession, now)
	if err != nil {
		t.Fatal(err)
	}
	card := ns.Players[0].Cards[Profession]
	if card.RevealedRound != 1 || card.Method != RevealByPlayer || ns.Version != s.Version+1 {
		t.Fatalf("kart %+v v%d", card, ns.Version)
	}
	if s.Players[0].Cards[Profession].Revealed() {
		t.Fatal("girdi durumu değişti")
	}
	if ns.TurnSeat != 1 || !ns.Deadline.Equal(now.Add(DefaultDurations.RevealPerPlayer)) || !ns.PhaseStartedAt.Equal(now) {
		t.Fatalf("sıra/süre: koltuk %d, %v", ns.TurnSeat, ns.Deadline)
	}
	if types(ev)[0] != EventCardRevealed || ev[0].Payload["auto"] != false || ev[1].Type != EventPhaseChanged ||
		ev[1].Payload["turn_user_id"] != uid(1) {
		t.Fatalf("olaylar %+v", ev)
	}
	// Aynı oyuncu tekrar: artık sırası değil.
	if _, _, err := e.Reveal(ns, uid(0), Profession, now); apperr.CodeOf(err) != apperr.NotYourTurn {
		t.Fatalf("tekrar: %v", err)
	}
	// Son oyuncu açınca tartışma başlar.
	for _, u := range []string{uid(1), uid(2), uid(3)} {
		if ns, _, err = e.Reveal(ns, u, Profession, now); err != nil {
			t.Fatal(err)
		}
	}
	if ns.Phase != PhaseDiscussion || ns.TurnSeat != -1 {
		t.Fatalf("tartışma: %s", ns.Phase)
	}
	if _, _, err := e.Reveal(ns, uid(0), Health, now); apperr.CodeOf(err) != apperr.InvalidPhase {
		t.Fatalf("aşama dışı: %v", err)
	}
}

func TestRevealLaterRoundFreeChoice(t *testing.T) {
	e, s := newGame(t, 5, 1)
	s = until(t, e, s, PhaseVoting, fixedRand(0))
	s.Votes = []Vote{{uid(0), uid(4)}}
	s = until(t, e, s, PhaseReveal, fixedRand(0)) // tur 2, koltuk 1
	if s.Round != 2 || s.TurnSeat != 1 {
		t.Fatalf("tur 2: r%d koltuk %d", s.Round, s.TurnSeat)
	}
	now := s.PhaseStartedAt
	if _, _, err := e.Reveal(s, uid(1), Profession, now); apperr.CodeOf(err) != apperr.CardAlreadyRevealed {
		t.Fatalf("açık kart: %v", err)
	}
	ns, _, err := e.Reveal(s, uid(1), Secret, now)
	if err != nil || ns.Players[1].Cards[Secret].RevealedRound != 2 {
		t.Fatalf("gizli sır: %v", err)
	}
	// Elenmiş oyuncu (u04): NOT_ALIVE.
	ns.TurnSeat = 4
	if _, _, err := e.Reveal(ns, uid(4), Health, now); apperr.CodeOf(err) != apperr.NotAlive {
		t.Fatalf("elenmiş: %v", err)
	}
}
