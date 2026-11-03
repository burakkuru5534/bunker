// Package engine, Sığınak'ın SAF oyun motorudur (ADR-0002): veritabanı, ağ,
// sistem saati ve global rastgelelik yoktur. Her işlem (durum, eylem, zaman,
// rng) → (yeni durum, olaylar, hata) biçimindedir; böylece tüm kurallar
// deterministik birim testleriyle doğrulanır.
//
// Kaynak kurallar: docs/game-design.md §1–§10. Aşama ve alan adları
// veritabanıyla (migrations/postgres/0003) aynıdır.
package engine

import (
	"slices"
	"time"
)

// Phase, oyun aşamasıdır.
type Phase string

const (
	PhaseDealing       Phase = "dealing"
	PhaseScenarioIntro Phase = "scenario_intro"
	PhaseReveal        Phase = "reveal"
	PhaseDiscussion    Phase = "discussion"
	PhaseVoting        Phase = "voting"
	PhaseTieBreak      Phase = "tie_break"
	PhaseElimination   Phase = "elimination"
	PhaseFinished      Phase = "finished"
)

// Category, kart kategorisidir. Sıra, süre dolunca otomatik açma sırasıdır
// (game-design §6: gizli sır en son).
type Category string

const (
	Profession Category = "profession"
	Health     Category = "health"
	Hobby      Category = "hobby"
	Item       Category = "item"
	Secret     Category = "secret"
)

// Categories, açma sırasıyla tüm kategoriler.
var Categories = []Category{Profession, Health, Hobby, Item, Secret}

// RevealRounds, kart açılan son turdur (tur 1–5).
const RevealRounds = 5

// PlayerStatus, katılımcının durumudur.
type PlayerStatus string

const (
	Alive      PlayerStatus = "alive"
	Eliminated PlayerStatus = "eliminated"
	Left       PlayerStatus = "left"
)

// RevealMethod, kartın nasıl açıldığıdır.
type RevealMethod string

const (
	RevealByPlayer      RevealMethod = "player"
	RevealByTimeout     RevealMethod = "timeout"
	RevealByElimination RevealMethod = "elimination"
)

// Card, oyuncuya dağıtılmış karttır.
type Card struct {
	ID            int
	RevealedRound int // 0: kapalı
	Method        RevealMethod
}

// Revealed, kart açık mı?
func (c Card) Revealed() bool { return c.RevealedRound > 0 }

// Player, katılımcıdır. Seat, oyun başında rastgele belirlenen açıklama
// sırasıdır (0'dan başlar, oyunda tekil).
type Player struct {
	UserID          string
	Seat            int
	Status          PlayerStatus
	EliminatedRound int // elenen/ayrılanın turu (oyun başlamadan ayrılma 0)
	ReadyRound      int // tartışmada "Hazırım" dediği tur
	Cards           map[Category]Card
}

// Alive, oyuncu hayatta mı?
func (p Player) Alive() bool { return p.Status == Alive }

// Vote, bir oydur.
type Vote struct {
	Voter  string
	Target string
}

// EliminationMethod, elemenin nasıl belirlendiğidir.
type EliminationMethod string

const (
	ByVote       EliminationMethod = "vote"
	BySpectators EliminationMethod = "spectators"
	ByLottery    EliminationMethod = "lottery"
)

// TallyEntry, adaya gelen oy sayısıdır (kimin kime verdiği yok).
type TallyEntry struct {
	UserID string `json:"user_id"`
	Votes  int    `json:"votes"`
}

// RoundResult, turun sonucudur (round_results).
type RoundResult struct {
	Round             int
	EliminatedID      string
	Method            EliminationMethod
	Tally             []TallyEntry
	LotteryCandidates []string
}

// EndReason, oyunun bitiş nedenidir.
type EndReason string

const (
	EndCapacityReached EndReason = "capacity_reached"
	EndAbandoned       EndReason = "abandoned"
)

// State, oyunun tam durumudur. Motor bu değeri kopyalayarak değiştirir;
// çağıranın elindeki eski durum bozulmaz.
type State struct {
	Phase       Phase
	Round       int
	TotalRounds int
	Capacity    int
	// TurnSeat, REVEAL'da sırası gelenin koltuğu; diğer aşamalarda -1.
	TurnSeat       int
	PhaseStartedAt time.Time
	Deadline       time.Time // FINISHED'da sıfır
	Version        int64
	Players        []Player // koltuk sırasıyla

	Votes         []Vote   // bu turun normal oyları
	TieCandidates []string // TIE_BREAK'te eşit oy alanlar
	TieVotes      []Vote   // izleyici oyları
	PendingTally  []TallyEntry
	Results       []RoundResult
	EndReason     EndReason
	FinishedAt    time.Time
}

// clone, durumun derin kopyasıdır.
func (s State) clone() State {
	c := s
	c.Players = make([]Player, len(s.Players))
	for i, p := range s.Players {
		cp := p
		cp.Cards = make(map[Category]Card, len(p.Cards))
		for k, v := range p.Cards {
			cp.Cards[k] = v
		}
		c.Players[i] = cp
	}
	c.Votes = slices.Clone(s.Votes)
	c.TieCandidates = slices.Clone(s.TieCandidates)
	c.TieVotes = slices.Clone(s.TieVotes)
	c.PendingTally = slices.Clone(s.PendingTally)
	c.Results = slices.Clone(s.Results)
	return c
}

// Player, kullanıcının katılımcı kaydına işaretçi döndürür (yoksa nil).
func (s *State) Player(userID string) *Player {
	for i := range s.Players {
		if s.Players[i].UserID == userID {
			return &s.Players[i]
		}
	}
	return nil
}

// AliveCount, hayattaki oyuncu sayısıdır.
func (s State) AliveCount() int {
	n := 0
	for _, p := range s.Players {
		if p.Alive() {
			n++
		}
	}
	return n
}

// Spectators, eşitlik bozmada oy kullanabilecek izleyicilerdir: elenen ve
// ayrılan katılımcılar (game-design §5). Bağlı olup olmadıkları servis
// katmanının konusudur; motor için izleyici varlığı tie_break'i açar.
func (s State) Spectators() []string {
	var out []string
	for _, p := range s.Players {
		if !p.Alive() {
			out = append(out, p.UserID)
		}
	}
	return out
}

// Winners, oyun bittiyse kazananlardır (hayatta kalanlar).
func (s State) Winners() []string {
	if s.Phase != PhaseFinished {
		return nil
	}
	var out []string
	for _, p := range s.Players {
		if p.Alive() {
			out = append(out, p.UserID)
		}
	}
	return out
}

// Rand, motorun rastgelelik kaynağıdır (kura). Üretimde kriptografik
// kaynak, testlerde deterministik sahte verilir.
type Rand interface {
	IntN(n int) int
}
