package engine

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
)

var t0 = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

// fixedRand her zaman aynı indeksi döndürür (mod n).
type fixedRand int

func (f fixedRand) IntN(n int) int { return int(f) % n }

func uid(i int) string { return fmt.Sprintf("u%02d", i) }

func seated(n int) []SeatedPlayer {
	var out []SeatedPlayer
	for i := 0; i < n; i++ {
		cards := map[Category]int{}
		for j, c := range Categories {
			cards[c] = i*10 + j + 1
		}
		out = append(out, SeatedPlayer{UserID: uid(i), Seat: i, Cards: cards})
	}
	return out
}

func newGame(t *testing.T, n, capacity int) (Engine, State) {
	t.Helper()
	e := New(DefaultDurations)
	s, err := e.NewGame(seated(n), capacity)
	if err != nil {
		t.Fatal(err)
	}
	s, _, err = e.Begin(s, t0)
	if err != nil {
		t.Fatal(err)
	}
	return e, s
}

// tick, süreyi doldurup Timeout çalıştırır.
func tick(t *testing.T, e Engine, s State, rng Rand) (State, []Event) {
	t.Helper()
	ns, ev, err := e.Timeout(s, s.Version, s.Deadline, rng)
	if err != nil {
		t.Fatalf("timeout (%s): %v", s.Phase, err)
	}
	if ns.Version != s.Version+1 {
		t.Fatalf("sürüm %d → %d", s.Version, ns.Version)
	}
	return ns, ev
}

// until, verilen aşamaya kadar ilerletir.
func until(t *testing.T, e Engine, s State, p Phase, rng Rand) State {
	t.Helper()
	for i := 0; s.Phase != p; i++ {
		if s.Phase == PhaseFinished || i > 500 {
			t.Fatalf("%s aşamasına ulaşılamadı (şu an %s)", p, s.Phase)
		}
		s, _ = tick(t, e, s, rng)
	}
	return s
}

func types(ev []Event) []string {
	var out []string
	for _, e := range ev {
		out = append(out, e.Type)
	}
	return out
}

func TestNewGameValidation(t *testing.T) {
	e := New(DefaultDurations)
	dupCard := seated(4)
	dupCard[1].Cards[Health] = dupCard[0].Cards[Health]
	missing := seated(4)
	delete(missing[2].Cards, Secret)
	badSeat := seated(4)
	badSeat[3].Seat = 7
	dupUser := seated(4)
	dupUser[1].UserID = dupUser[0].UserID
	cases := map[string]struct {
		p   []SeatedPlayer
		cap int
	}{
		"3 oyuncu":         {seated(3), 1},
		"11 oyuncu":        {seated(11), 1},
		"kapasite 3":       {seated(5), 3},
		"kapasite 0":       {seated(5), 0},
		"tekrar kart":      {dupCard, 1},
		"eksik kart":       {missing, 1},
		"koltuk boşluğu":   {badSeat, 1},
		"tekrar kullanıcı": {dupUser, 1},
	}
	for name, c := range cases {
		if _, err := e.NewGame(c.p, c.cap); err == nil {
			t.Errorf("%s: hata bekleniyordu", name)
		}
	}
	// Koltuk sırası girdiden bağımsız.
	p := seated(5)
	slices.Reverse(p)
	s, err := e.NewGame(p, 2)
	if err != nil {
		t.Fatal(err)
	}
	if s.Players[0].UserID != uid(0) || s.TotalRounds != 3 || s.Phase != PhaseDealing || s.TurnSeat != -1 {
		t.Fatalf("durum: %+v", s)
	}
}

func TestBeginAndGuards(t *testing.T) {
	e, s := newGame(t, 5, 1)
	if s.Phase != PhaseScenarioIntro || !s.Deadline.Equal(t0.Add(30*time.Second)) || s.Version != 1 {
		t.Fatalf("başlangıç: %s %v v%d", s.Phase, s.Deadline, s.Version)
	}
	if _, _, err := e.Begin(s, t0); apperr.CodeOf(err) != apperr.InvalidPhase {
		t.Fatalf("ikinci Begin: %v", err)
	}
	if _, _, err := e.Timeout(s, s.Version, s.Deadline.Add(-time.Millisecond), fixedRand(0)); !errors.Is(err, ErrNotDue) {
		t.Fatalf("erken: %v", err)
	}
	if _, _, err := e.Timeout(s, s.Version-1, s.Deadline, fixedRand(0)); !errors.Is(err, ErrStale) {
		t.Fatalf("eski sürüm: %v", err)
	}
	// Girdi durumu değişmez (saf fonksiyon).
	before := s.clone()
	ns, _ := tick(t, e, s, fixedRand(0))
	if s.Phase != before.Phase || s.Version != before.Version || ns.Phase != PhaseReveal {
		t.Fatal("girdi durumu değişti veya geçiş yanlış")
	}
}

func TestRevealTurnsAndAutoReveal(t *testing.T) {
	e, s := newGame(t, 4, 1)
	s, _ = tick(t, e, s, fixedRand(0)) // tur 1
	var order []int
	for s.Phase == PhaseReveal {
		order = append(order, s.TurnSeat)
		seat := s.TurnSeat
		var ev []Event
		s, ev = tick(t, e, s, fixedRand(0))
		c := s.Players[seat].Cards[Profession]
		if c.RevealedRound != 1 || c.Method != RevealByTimeout {
			t.Fatalf("koltuk %d meslek açılmadı: %+v", seat, c)
		}
		if ev[0].Type != EventCardRevealed || ev[0].Payload["auto"] != true || ev[0].Payload["category"] != "profession" {
			t.Fatalf("olay: %+v", ev[0])
		}
	}
	if !slices.Equal(order, []int{0, 1, 2, 3}) {
		t.Fatalf("tur 1 sırası %v", order)
	}
	if s.Phase != PhaseDiscussion || s.Deadline.Sub(s.PhaseStartedAt) != 90*time.Second {
		t.Fatalf("tartışma: %s %v", s.Phase, s.Deadline.Sub(s.PhaseStartedAt))
	}
}

func TestRevealOrderRotatesAndSkipsEliminated(t *testing.T) {
	s := State{Round: 3, Players: make([]Player, 5)}
	for i := range s.Players {
		s.Players[i] = Player{UserID: uid(i), Seat: i, Status: Alive}
	}
	s.Players[3].Status = Eliminated
	if got := s.RevealOrder(); !slices.Equal(got, []int{2, 4, 0, 1}) {
		t.Fatalf("sıra %v", got)
	}
	s.Round = 7 // 6 % 5 = 1
	if got := s.RevealOrder(); !slices.Equal(got, []int{1, 2, 4, 0}) {
		t.Fatalf("sıra %v", got)
	}
}

func TestAutoRevealCategoryOrder(t *testing.T) {
	p := Player{Cards: map[Category]Card{}}
	for i, c := range Categories {
		p.Cards[c] = Card{ID: i + 1}
	}
	s := State{Round: 2}
	p.Cards[Profession] = Card{ID: 1, RevealedRound: 1}
	p.Cards[Hobby] = Card{ID: 3, RevealedRound: 1}
	if c, _ := s.AutoRevealCategory(p); c != Health {
		t.Fatalf("beklenen health, gelen %s", c)
	}
	for _, c := range Categories {
		p.Cards[c] = Card{ID: 9, RevealedRound: 1}
	}
	if _, ok := s.AutoRevealCategory(p); ok {
		t.Fatal("hepsi açıkken kategori döndü")
	}
}

func TestVotingSingleMax(t *testing.T) {
	e, s := newGame(t, 5, 1)
	s = until(t, e, s, PhaseVoting, fixedRand(0))
	s.Votes = []Vote{{uid(0), uid(2)}, {uid(1), uid(2)}, {uid(3), uid(4)}, {uid(2), uid(2)}}
	s, ev := tick(t, e, s, fixedRand(0))
	if s.Phase != PhaseElimination || s.Player(uid(2)).Status != Eliminated {
		t.Fatalf("eleme: %s", s.Phase)
	}
	r := s.Results[0]
	if r.Method != ByVote || r.Tally[0] != (TallyEntry{uid(2), 2}) || r.Tally[1] != (TallyEntry{uid(4), 1}) {
		t.Fatalf("sonuç: %+v", r)
	}
	for _, c := range s.Player(uid(2)).Cards {
		if !c.Revealed() {
			t.Fatal("elenenin kartı kapalı kaldı")
		}
	}
	if !slices.Contains(types(ev), EventPlayerEliminated) {
		t.Fatalf("olaylar %v", types(ev))
	}
	s, _ = tick(t, e, s, fixedRand(0))
	if s.Round != 2 || s.Phase != PhaseReveal || s.Votes != nil {
		t.Fatalf("sonraki tur: r%d %s", s.Round, s.Phase)
	}
	if s.TurnSeat != 1 {
		t.Fatalf("tur 2 koltuk 1'den başlamalı, %d", s.TurnSeat)
	}
}

func TestTallyIgnoresInvalidVotes(t *testing.T) {
	s := State{Players: []Player{
		{UserID: "a", Seat: 0, Status: Alive}, {UserID: "b", Seat: 1, Status: Alive},
		{UserID: "c", Seat: 2, Status: Left}, {UserID: "d", Seat: 3, Status: Eliminated},
	}}
	s.Votes = []Vote{{"a", "c"}, {"d", "b"}, {"c", "b"}, {"x", "b"}, {"a", "a"}, {"b", "a"}}
	got := s.Tally()
	if !slices.Equal(got, []TallyEntry{{"a", 1}, {"b", 0}}) {
		t.Fatalf("sayım %v", got)
	}
}

func TestTieWithoutSpectatorsGoesToLottery(t *testing.T) {
	e, s := newGame(t, 4, 1)
	s = until(t, e, s, PhaseVoting, fixedRand(0))
	s.Votes = []Vote{{uid(0), uid(3)}, {uid(3), uid(0)}}
	s, _ = tick(t, e, s, fixedRand(1))
	r := s.Results[0]
	if r.Method != ByLottery || !slices.Equal(r.LotteryCandidates, []string{uid(0), uid(3)}) || r.EliminatedID != uid(3) {
		t.Fatalf("kura: %+v", r)
	}
}

// spectatorGame: 6 oyuncu, ilk tur u05 elenmiş; ikinci turun oylaması.
func spectatorGame(t *testing.T) (Engine, State) {
	e, s := newGame(t, 6, 1)
	s = until(t, e, s, PhaseVoting, fixedRand(0))
	s.Votes = []Vote{{uid(0), uid(5)}}
	s, _ = tick(t, e, s, fixedRand(0))
	s = until(t, e, s, PhaseVoting, fixedRand(0))
	s.Votes = []Vote{{uid(0), uid(1)}, {uid(1), uid(0)}, {uid(2), uid(3)}, {uid(3), uid(2)}}
	return e, s
}

func TestTieBreakUnanimousSpectators(t *testing.T) {
	e, s := spectatorGame(t)
	s, ev := tick(t, e, s, fixedRand(0))
	if s.Phase != PhaseTieBreak || len(s.TieCandidates) != 4 || !slices.Contains(types(ev), EventTiebreakStarted) {
		t.Fatalf("eşitlik: %s %v", s.Phase, s.TieCandidates)
	}
	s.TieVotes = []Vote{{uid(5), uid(2)}, {uid(0), uid(1)}} // canlı oyuncunun oyu sayılmaz
	s, _ = tick(t, e, s, fixedRand(0))
	r := s.Results[1]
	if r.Method != BySpectators || r.EliminatedID != uid(2) || len(r.Tally) != 5 {
		t.Fatalf("izleyici sonucu: %+v", r)
	}
	if s.TieCandidates != nil || s.PendingTally != nil {
		t.Fatal("eşitlik verisi temizlenmedi")
	}
}

func TestTieBreakNoVotesOrSplitGoesToLottery(t *testing.T) {
	for name, votes := range map[string][]Vote{
		"oy yok":    nil,
		"aday dışı": {{uid(5), uid(4)}},
	} {
		e, s := spectatorGame(t)
		s, _ = tick(t, e, s, fixedRand(0))
		s.TieVotes = votes
		s, _ = tick(t, e, s, fixedRand(2))
		r := s.Results[1]
		if r.Method != ByLottery || r.EliminatedID != uid(2) || len(r.LotteryCandidates) != 4 {
			t.Fatalf("%s: %+v", name, r)
		}
	}
	// İki izleyici farklı adaylar seçerse kura.
	e, s := newGame(t, 7, 1)
	for range 2 {
		s = until(t, e, s, PhaseVoting, fixedRand(0))
		s.Votes = []Vote{{uid(0), s.Players[len(s.Players)-1-len(s.Results)].UserID}}
		s, _ = tick(t, e, s, fixedRand(0))
	}
	s = until(t, e, s, PhaseVoting, fixedRand(0))
	s.Votes = []Vote{{uid(0), uid(1)}, {uid(1), uid(0)}}
	s, _ = tick(t, e, s, fixedRand(0))
	s.TieVotes = []Vote{{uid(6), uid(0)}, {uid(5), uid(1)}}
	s, _ = tick(t, e, s, fixedRand(0))
	if r := s.Results[2]; r.Method != ByLottery || r.EliminatedID != uid(0) {
		t.Fatalf("dağınık: %+v", r)
	}
}

func TestFullGames(t *testing.T) {
	for n := MinPlayers; n <= MaxPlayers; n++ {
		for _, capacity := range []int{1, 2} {
			t.Run(fmt.Sprintf("%d_kişi_kapasite_%d", n, capacity), func(t *testing.T) {
				e, s := newGame(t, n, capacity)
				rng := rand.New(rand.NewPCG(uint64(n), uint64(capacity)))
				for steps := 0; s.Phase != PhaseFinished; steps++ {
					if steps > 1000 {
						t.Fatal("oyun bitmedi")
					}
					if s.Deadline.IsZero() || !s.Deadline.After(s.PhaseStartedAt) {
						t.Fatalf("%s aşamasında süre yok", s.Phase)
					}
					if s.Phase == PhaseVoting { // rastgele oylar
						for _, p := range s.Players {
							if p.Alive() {
								s.Votes = append(s.Votes, Vote{p.UserID, s.Players[rng.IntN(n)].UserID})
							}
						}
					}
					if s.Phase == PhaseTieBreak && rng.IntN(2) == 0 {
						for _, sp := range s.Spectators() {
							s.TieVotes = append(s.TieVotes, Vote{sp, s.TieCandidates[0]})
						}
					}
					s, _ = tick(t, e, s, rng)
				}
				if s.AliveCount() != capacity || len(s.Winners()) != capacity || s.EndReason != EndCapacityReached {
					t.Fatalf("bitiş: hayatta %d, neden %s", s.AliveCount(), s.EndReason)
				}
				if len(s.Results) != n-capacity || s.Round != n-capacity {
					t.Fatalf("tur %d, sonuç %d", s.Round, len(s.Results))
				}
				for _, p := range s.Players {
					opened := 0
					for _, c := range p.Cards {
						if c.Revealed() {
							opened++
						}
					}
					want := min(s.Round, RevealRounds)
					if !p.Alive() {
						want = len(Categories)
					}
					if opened != want {
						t.Fatalf("%s açık kart %d, beklenen %d", p.UserID, opened, want)
					}
				}
				if _, _, err := e.Timeout(s, s.Version, s.FinishedAt.Add(time.Hour), rng); !errors.Is(err, ErrNoDeadline) {
					t.Fatalf("bitmiş oyun: %v", err)
				}
			})
		}
	}
}

func TestRoundsAfterFiveSkipReveal(t *testing.T) {
	e, s := newGame(t, 9, 1)
	for s.Round < 6 {
		s = until(t, e, s, PhaseVoting, fixedRand(0))
		s, _ = tick(t, e, s, fixedRand(0))
		s = until(t, e, s, PhaseElimination, fixedRand(0))
		s, _ = tick(t, e, s, fixedRand(0))
	}
	if s.Round != 6 || s.Phase != PhaseDiscussion {
		t.Fatalf("tur 6: %s", s.Phase)
	}
	if got := s.Deadline.Sub(s.PhaseStartedAt); got != 90*time.Second {
		t.Fatalf("4 kişilik tartışma %v", got) // 9−5=4 hayatta → küçük masa
	}
}
