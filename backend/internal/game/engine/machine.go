package engine

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

// Olay türleri (api.md §5 "Oyun olayları"). Motor kart metnini bilmez;
// card_id'yi servis katmanı metne çevirir.
const (
	EventPhaseChanged     = "game.phase_changed"
	EventCardRevealed     = "game.card_revealed"
	EventTiebreakStarted  = "game.tiebreak_started"
	EventPlayerEliminated = "game.player_eliminated"
	EventFinished         = "game.finished"
)

// Event, motorun ürettiği olaydır. To boşsa odaya, doluysa yalnızca o
// kullanıcıya gider (gizli bilgi, ENG-015).
type Event struct {
	Type    string
	To      string
	Payload map[string]any
}

// İç hatalar (istemciye gitmez): zamanlayıcı ve eşzamanlılık için.
var (
	// ErrStale: beklenen sürüm güncel değil (aşama zaten ilerlemiş; ADR-0005).
	ErrStale = errors.New("engine: durum sürümü güncel değil")
	// ErrNotDue: aşamanın süresi henüz dolmadı.
	ErrNotDue = errors.New("engine: aşama süresi dolmadı")
	// ErrNoDeadline: biten oyunun süresi yoktur.
	ErrNoDeadline = errors.New("engine: oyun bitti")
)

// Oyuncu sayısı ve kapasite (game-design §1).
const (
	MinPlayers = 4
	MaxPlayers = 10
)

// Engine, kuralları ve süreleri taşır. Durumsuzdur.
type Engine struct {
	D Durations
}

// New, motoru verilen sürelerle kurar.
func New(d Durations) Engine { return Engine{D: d} }

// SeatedPlayer, oyun başında koltuğu ve kartları belirlenmiş oyuncudur
// (koltuk sırası ve dağıtım rastgeleliği ENG-003/004'te).
type SeatedPlayer struct {
	UserID string
	Seat   int
	Cards  map[Category]int // kategori → kart kimliği
}

// NewGame, DEALING aşamasında yeni bir oyun durumu kurar ve girdiyi
// doğrular: 4–10 oyuncu, kapasite 1/2, koltuklar 0..n-1, her oyuncuya her
// kategoriden bir kart, oyunda her kart tekil.
func (e Engine) NewGame(players []SeatedPlayer, capacity int) (State, error) {
	n := len(players)
	if n < MinPlayers || n > MaxPlayers {
		return State{}, fmt.Errorf("engine: oyuncu sayısı %d (4–10)", n)
	}
	if capacity != 1 && capacity != 2 {
		return State{}, fmt.Errorf("engine: kapasite %d (1 veya 2)", capacity)
	}
	sorted := slices.Clone(players)
	slices.SortFunc(sorted, func(a, b SeatedPlayer) int { return a.Seat - b.Seat })
	seenUser, seenCard := map[string]bool{}, map[int]bool{}
	s := State{Phase: PhaseDealing, Capacity: capacity, TotalRounds: n - capacity, TurnSeat: -1}
	for i, sp := range sorted {
		if sp.Seat != i {
			return State{}, fmt.Errorf("engine: koltuklar 0..%d olmalı", n-1)
		}
		if sp.UserID == "" || seenUser[sp.UserID] {
			return State{}, fmt.Errorf("engine: kullanıcı boş veya tekrar: %q", sp.UserID)
		}
		seenUser[sp.UserID] = true
		p := Player{UserID: sp.UserID, Seat: i, Status: Alive, Cards: map[Category]Card{}}
		for _, c := range Categories {
			id, ok := sp.Cards[c]
			if !ok {
				return State{}, fmt.Errorf("engine: %s için %s kartı yok", sp.UserID, c)
			}
			if seenCard[id] {
				return State{}, fmt.Errorf("engine: kart %d iki kez dağıtılmış", id)
			}
			seenCard[id] = true
			p.Cards[c] = Card{ID: id}
		}
		if len(sp.Cards) != len(Categories) {
			return State{}, fmt.Errorf("engine: %s için fazladan kart", sp.UserID)
		}
		s.Players = append(s.Players, p)
	}
	return s, nil
}

// step, tek bir işlemin geçici çalışma alanıdır.
type step struct {
	e      Engine
	s      State
	now    time.Time
	rng    Rand
	events []Event
}

func (e Engine) begin(s State, now time.Time, rng Rand) *step {
	return &step{e: e, s: s.clone(), now: now, rng: rng}
}

// done, sürümü bir artırır ve sonucu döndürür (işlem başına bir sürüm).
func (st *step) done() (State, []Event, error) {
	st.s.Version++
	return st.s, st.events, nil
}

func (st *step) emit(typ string, payload map[string]any) {
	st.events = append(st.events, Event{Type: typ, Payload: payload})
}

// setPhase, aşamayı ve bitiş zamanını ayarlar, phase_changed yayınlar.
func (st *step) setPhase(p Phase, d time.Duration) {
	st.s.Phase = p
	st.s.PhaseStartedAt = st.now
	st.s.Deadline = st.now.Add(d)
	if p != PhaseReveal {
		st.s.TurnSeat = -1
	}
	var turn any
	if p == PhaseReveal && st.s.TurnSeat >= 0 {
		turn = st.s.Players[st.s.TurnSeat].UserID
	}
	st.emit(EventPhaseChanged, map[string]any{
		"phase": string(p), "round": st.s.Round, "phase_deadline_at": st.s.Deadline, "turn_user_id": turn,
	})
}

// Begin, dağıtım bittikten sonra senaryo tanıtımına geçer.
func (e Engine) Begin(s State, now time.Time) (State, []Event, error) {
	if s.Phase != PhaseDealing {
		return s, nil, invalidPhase()
	}
	st := e.begin(s, now, nil)
	st.setPhase(PhaseScenarioIntro, e.D.ScenarioIntro)
	return st.done()
}

// Timeout, aşama süresi dolduğunda sistem eylemidir (ADR-0005).
// expectedVersion, zamanlayıcının planlandığı andaki sürümdür: uyuşmazsa
// aşama zaten ilerlemiştir ve ErrStale döner (durum değişmez).
func (e Engine) Timeout(s State, expectedVersion int64, now time.Time, rng Rand) (State, []Event, error) {
	switch {
	case s.Phase == PhaseFinished:
		return s, nil, ErrNoDeadline
	case s.Version != expectedVersion:
		return s, nil, ErrStale
	case now.Before(s.Deadline):
		return s, nil, ErrNotDue
	}
	st := e.begin(s, now, rng)
	switch s.Phase {
	case PhaseDealing:
		return s, nil, invalidPhase() // dağıtım Begin ile ilerler
	case PhaseScenarioIntro:
		st.beginRound(1)
	case PhaseReveal:
		st.autoReveal()
		st.advanceTurn()
	case PhaseDiscussion:
		st.setPhase(PhaseVoting, e.D.Voting)
	case PhaseVoting:
		st.resolveVoting()
	case PhaseTieBreak:
		st.resolveTieBreak()
	case PhaseElimination:
		st.afterElimination()
	}
	return st.done()
}

// --- Tur ve kart açma --------------------------------------------------------

// beginRound, yeni turu başlatır: tur 1–5'te REVEAL, sonra doğrudan tartışma.
func (st *step) beginRound(r int) {
	st.s.Round = r
	st.s.Votes, st.s.TieVotes, st.s.TieCandidates, st.s.PendingTally = nil, nil, nil, nil
	if r <= RevealRounds {
		if seat, ok := st.nextRevealer(); ok {
			st.s.TurnSeat = seat
			st.setPhase(PhaseReveal, st.e.D.RevealPerPlayer)
			return
		}
	}
	st.setPhase(PhaseDiscussion, st.e.D.discussion(st.s.AliveCount()))
}

// RevealOrder, bu turun açıklama sırasıdır: hayattaki oyuncular, koltuk
// halkasında (tur−1). koltuktan başlayarak; her tur başlangıç bir kayar,
// elenenler atlanır (game-design §2).
func (s State) RevealOrder() []int {
	n := len(s.Players)
	if n == 0 || s.Round < 1 {
		return nil
	}
	var out []int
	start := (s.Round - 1) % n
	for i := 0; i < n; i++ {
		seat := (start + i) % n
		if s.Players[seat].Alive() {
			out = append(out, seat)
		}
	}
	return out
}

// revealedThisRound, oyuncu bu turda kart açtı mı?
func (s State) revealedThisRound(p Player) bool {
	for _, c := range p.Cards {
		if c.RevealedRound == s.Round && c.Method != RevealByElimination {
			return true
		}
	}
	return false
}

// nextRevealer, sırada bu turda henüz açmamış ilk hayattaki oyuncudur.
func (st *step) nextRevealer() (int, bool) {
	for _, seat := range st.s.RevealOrder() {
		if !st.s.revealedThisRound(st.s.Players[seat]) {
			return seat, true
		}
	}
	return -1, false
}

// AutoRevealCategory, süre dolunca açılacak kategoridir: tur 1'de meslek,
// sonra açılmamışların en düşük sıradakisi (gizli sır en son).
func (s State) AutoRevealCategory(p Player) (Category, bool) {
	if s.Round == 1 {
		return Profession, !p.Cards[Profession].Revealed()
	}
	for _, c := range Categories {
		if !p.Cards[c].Revealed() {
			return c, true
		}
	}
	return "", false
}

func (st *step) reveal(p *Player, c Category, m RevealMethod) {
	card := p.Cards[c]
	card.RevealedRound, card.Method = st.s.Round, m
	p.Cards[c] = card
	st.emit(EventCardRevealed, map[string]any{
		"user_id": p.UserID, "category": string(c), "card_id": card.ID, "auto": m == RevealByTimeout,
	})
}

func (st *step) autoReveal() {
	if st.s.TurnSeat < 0 {
		return
	}
	p := &st.s.Players[st.s.TurnSeat]
	if c, ok := st.s.AutoRevealCategory(*p); ok && p.Alive() && !st.s.revealedThisRound(*p) {
		st.reveal(p, c, RevealByTimeout)
	}
}

// advanceTurn, sıradaki açıklayana geçer; kimse kalmadıysa tartışmaya.
func (st *step) advanceTurn() {
	if seat, ok := st.nextRevealer(); ok {
		st.s.TurnSeat = seat
		st.setPhase(PhaseReveal, st.e.D.RevealPerPlayer)
		return
	}
	st.setPhase(PhaseDiscussion, st.e.D.discussion(st.s.AliveCount()))
}

// --- Oylama, eşitlik bozma, eleme ---------------------------------------------

// Tally, bu turun oy sayımıdır: yalnızca hayattaki oyuncuların hayattaki
// adaylara oyları geçerlidir (ayrılan adaya verilen oy geçersiz, §9).
// Hayattaki her oyuncu (0 oyla da) listededir; çoktan aza, eşitlikte koltuk.
func (s State) Tally() []TallyEntry {
	count := map[string]int{}
	for _, v := range s.Votes {
		voter, target := s.player(v.Voter), s.player(v.Target)
		if voter != nil && target != nil && voter.Alive() && target.Alive() && v.Voter != v.Target {
			count[v.Target]++
		}
	}
	var out []TallyEntry
	for _, p := range s.Players {
		if p.Alive() {
			out = append(out, TallyEntry{UserID: p.UserID, Votes: count[p.UserID]})
		}
	}
	slices.SortStableFunc(out, func(a, b TallyEntry) int { return b.Votes - a.Votes })
	return out
}

func (s State) player(userID string) *Player {
	for i := range s.Players {
		if s.Players[i].UserID == userID {
			return &s.Players[i]
		}
	}
	return nil
}

func (st *step) resolveVoting() {
	tally := st.s.Tally()
	if len(tally) == 0 {
		st.afterElimination()
		return
	}
	var top []string
	for _, t := range tally {
		if t.Votes == tally[0].Votes {
			top = append(top, t.UserID)
		}
	}
	switch {
	case len(top) == 1:
		st.eliminate(top[0], ByVote, tally, nil)
	case len(st.s.Spectators()) > 0:
		st.s.TieCandidates, st.s.PendingTally = top, tally
		st.setPhase(PhaseTieBreak, st.e.D.TieBreak)
		st.emit(EventTiebreakStarted, map[string]any{"candidates": slices.Clone(top), "phase_deadline_at": st.s.Deadline})
	default:
		st.lottery(top, tally)
	}
}

// resolveTieBreak: süre içinde oy veren izleyicilerin hepsi aynı (hâlâ
// hayattaki) adayı seçtiyse o elenir; aksi halde (dağınık veya hiç oy) kura
// (game-design §10).
func (st *step) resolveTieBreak() {
	spectators := st.s.Spectators()
	var alive []string
	for _, c := range st.s.TieCandidates {
		if p := st.s.player(c); p != nil && p.Alive() {
			alive = append(alive, c)
		}
	}
	chosen := ""
	unanimous := true
	for _, v := range st.s.TieVotes {
		if !slices.Contains(spectators, v.Voter) || !slices.Contains(alive, v.Target) {
			continue
		}
		if chosen == "" {
			chosen = v.Target
		} else if chosen != v.Target {
			unanimous = false
		}
	}
	tally := st.s.PendingTally
	switch {
	case chosen != "" && unanimous:
		st.eliminate(chosen, BySpectators, tally, nil)
	default:
		st.lottery(alive, tally)
	}
}

// lottery, adaylar arasından kriptografik rastgele seçer. Tek aday kaldıysa
// (diğerleri ayrıldı) o elenir; hiç kalmadıysa eleme olmaz.
func (st *step) lottery(candidates []string, tally []TallyEntry) {
	switch len(candidates) {
	case 0:
		st.afterElimination()
		return
	case 1:
		st.eliminate(candidates[0], ByVote, tally, nil)
		return
	}
	c := slices.Clone(candidates)
	slices.SortFunc(c, func(a, b string) int { return st.s.player(a).Seat - st.s.player(b).Seat })
	st.eliminate(c[st.rng.IntN(len(c))], ByLottery, tally, c)
}

// eliminate, adayı eler: tüm kartları açılır, tur sonucu yazılır, ELIMINATION.
func (st *step) eliminate(userID string, m EliminationMethod, tally []TallyEntry, lottery []string) {
	p := st.s.Player(userID)
	p.Status, p.EliminatedRound = Eliminated, st.s.Round
	var cards []map[string]any
	for _, c := range Categories {
		card := p.Cards[c]
		if !card.Revealed() {
			card.RevealedRound, card.Method = st.s.Round, RevealByElimination
			p.Cards[c] = card
		}
		cards = append(cards, map[string]any{"category": string(c), "card_id": card.ID})
	}
	st.s.Results = append(st.s.Results, RoundResult{Round: st.s.Round, EliminatedID: userID, Method: m,
		Tally: slices.Clone(tally), LotteryCandidates: slices.Clone(lottery)})
	// Oylar (normal ve eşitlik) tur sonuna kadar kalır (beginRound temizler);
	// veritabanından yüklenen durumla aynı olsun diye.
	st.s.TieCandidates, st.s.PendingTally = nil, nil
	st.setPhase(PhaseElimination, st.e.D.Elimination)
	st.emit(EventPlayerEliminated, map[string]any{
		"round": st.s.Round, "user_id": userID, "method": string(m),
		"tally": slices.Clone(tally), "lottery_candidates": slices.Clone(lottery), "cards": cards,
	})
}

// afterElimination: hayatta kalan ≤ kapasite ise oyun biter; aksi halde
// sonraki tur (game-design §3).
func (st *step) afterElimination() {
	alive := st.s.AliveCount()
	switch {
	case alive == 0:
		st.finish(EndAbandoned)
	case alive <= st.s.Capacity, st.s.Round >= st.s.TotalRounds:
		st.finish(EndCapacityReached)
	default:
		st.beginRound(st.s.Round + 1)
	}
}

func (st *step) finish(reason EndReason) {
	st.s.Phase, st.s.EndReason, st.s.FinishedAt = PhaseFinished, reason, st.now
	st.s.Deadline, st.s.TurnSeat = time.Time{}, -1
	st.s.PhaseStartedAt = st.now
	st.emit(EventFinished, map[string]any{"end_reason": string(reason), "winners": st.s.Winners()})
}
