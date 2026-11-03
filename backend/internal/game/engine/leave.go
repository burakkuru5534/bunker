package engine

import (
	"slices"
	"time"
)

// ByLeaving, oyundan ayrılmanın eleme olayındaki yöntemidir (api.md
// game.player_eliminated method "left"). Tur sonucu (round_results) yazılmaz:
// turun elemesi yine oylamayla yapılır (game-design §4).
const ByLeaving EliminationMethod = "left"

// Leave, hayattaki oyuncunun oyundan ayrılmasıdır ("Ayrıl" veya bağlantı
// bekleme süresinin dolması; game-design §4). Oyuncu "ayrıldı" olur ve
// elenmiş sayılır, tüm kartları açılır. Sonra:
//   - hayatta kalan ≤ kapasite → oyun hemen biter (kimse kalmadıysa terk edildi);
//   - REVEAL'da sıra ondaysa sıra hemen geçer;
//   - kalan herkes hazırsa / oy verdiyse aşama erken biter;
//   - eşitlik bozmada tek aday kaldıysa sonuç hemen belirlenir.
//
// Oyuncu zaten hayatta değilse veya oyun bittiyse ErrNoChange.
func (e Engine) Leave(s State, userID string, now time.Time, rng Rand) (State, []Event, error) {
	p := s.Player(userID)
	if p == nil {
		return s, nil, ErrNoChange
	}
	if !p.Alive() || s.Phase == PhaseFinished {
		return s, nil, ErrNoChange
	}
	st := e.begin(s, now, rng)
	leaver := st.s.Player(userID)
	leaver.Status, leaver.EliminatedRound = Left, st.s.Round
	revealRound := max(st.s.Round, 1) // tanıtımda ayrılırsa tur 1 (kart turu 1–9)
	var cards []map[string]any
	for _, c := range Categories {
		card := leaver.Cards[c]
		if !card.Revealed() {
			card.RevealedRound, card.Method = revealRound, RevealByElimination
			leaver.Cards[c] = card
		}
		cards = append(cards, map[string]any{"category": string(c), "card_id": card.ID})
	}
	st.emit(EventPlayerEliminated, map[string]any{
		"round": st.s.Round, "user_id": userID, "method": string(ByLeaving),
		"tally": nil, "lottery_candidates": nil, "cards": cards,
	})

	alive := st.s.AliveCount()
	switch {
	case alive == 0:
		st.finish(EndAbandoned)
	case alive <= st.s.Capacity:
		st.finish(EndCapacityReached)
	case st.s.Phase == PhaseReveal && st.s.TurnSeat == leaver.Seat:
		st.advanceTurn()
	case st.s.Phase == PhaseDiscussion && st.s.ReadyCount() == alive:
		st.setPhase(PhaseVoting, e.D.Voting)
	case st.s.Phase == PhaseVoting && st.s.VotesCast() == alive:
		st.resolveVoting()
	case st.s.Phase == PhaseTieBreak && aliveCandidates(st.s) <= 1:
		st.resolveTieBreak()
	}
	return st.done()
}

func aliveCandidates(s State) int {
	return len(slices.DeleteFunc(slices.Clone(s.TieCandidates), func(id string) bool {
		p := s.player(id)
		return p == nil || !p.Alive()
	}))
}
