package engine

import (
	"errors"
	"slices"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
)

// Oyuncu eylemleri (api.md "Oyun" eylem tablosu). Her başarılı eylem
// sürümü bir artırır; hata durumunda girdi durumu aynen döner.

// Reveal, sırası gelen hayattaki oyuncunun bir kartını açmasıdır
// (game-design §6 "Kart açma kuralı"): tur 1'de yalnızca meslek, sonra
// açılmamış herhangi bir kart. Ardından sıra sonraki oyuncuya geçer;
// kimse kalmadıysa tartışma başlar.
func (e Engine) Reveal(s State, userID string, c Category, now time.Time) (State, []Event, error) {
	if !slices.Contains(Categories, c) {
		return s, nil, apperr.Field("payload.category", apperr.FieldInvalid)
	}
	if s.Phase != PhaseReveal {
		return s, nil, invalidPhase()
	}
	p := s.Player(userID)
	if p == nil {
		return s, nil, apperr.New(apperr.NotFound, "oyuncu bulunamadı")
	}
	if !p.Alive() {
		return s, nil, apperr.New(apperr.NotAlive, "elenmiş veya ayrılmış oyuncu kart açamaz")
	}
	if s.TurnSeat != p.Seat {
		return s, nil, apperr.New(apperr.NotYourTurn, "kart açma sırası başkasında")
	}
	if p.Cards[c].Revealed() {
		return s, nil, apperr.New(apperr.CardAlreadyRevealed, "kart zaten açık")
	}
	if s.Round == 1 && c != Profession {
		return s, nil, apperr.New(apperr.CategoryNotAllowed, "1. turda yalnızca meslek açılabilir")
	}
	st := e.begin(s, now, nil)
	st.reveal(st.s.Player(userID), c, RevealByPlayer)
	st.advanceTurn()
	return st.done()
}

// Olay türleri (oyuncu eylemleri).
const (
	EventDiscussionReady = "game.discussion_ready"
	EventVoteProgress    = "game.vote_progress"
)

// ErrNoChange: eylem geçerli ama durumu değiştirmiyor (ör. ikinci "Hazırım").
// Servis bunu başarı sayar; sürüm artmaz, olay yayınlanmaz.
var ErrNoChange = errors.New("engine: değişiklik yok")

// requireAlive, eylemi yapanın hayattaki bir katılımcı olduğunu denetler.
func requireAlive(s State, userID string) (*Player, error) {
	p := s.Player(userID)
	if p == nil {
		return nil, apperr.New(apperr.NotFound, "oyuncu bulunamadı")
	}
	if !p.Alive() {
		return nil, apperr.New(apperr.NotAlive, "elenmiş veya ayrılmış oyuncu bu eylemi yapamaz")
	}
	return p, nil
}

// ReadyCount, bu turda "Hazırım" diyen hayattaki oyuncu sayısıdır.
func (s State) ReadyCount() int {
	n := 0
	for _, p := range s.Players {
		if p.Alive() && p.ReadyRound == s.Round {
			n++
		}
	}
	return n
}

// DiscussionReady, hayattaki oyuncunun tartışmayı bitirmeye hazır olduğunu
// bildirir (game-design §2). Herkes hazırsa oylama hemen başlar. İkinci kez
// gelirse ErrNoChange.
func (e Engine) DiscussionReady(s State, userID string, now time.Time) (State, []Event, error) {
	if s.Phase != PhaseDiscussion {
		return s, nil, invalidPhase()
	}
	p, err := requireAlive(s, userID)
	if err != nil {
		return s, nil, err
	}
	if p.ReadyRound == s.Round {
		return s, nil, ErrNoChange
	}
	st := e.begin(s, now, nil)
	st.s.Player(userID).ReadyRound = st.s.Round
	count, total := st.s.ReadyCount(), st.s.AliveCount()
	st.emit(EventDiscussionReady, map[string]any{"count": count, "total": total})
	if count == total {
		st.setPhase(PhaseVoting, e.D.Voting)
	}
	return st.done()
}

// VotesCast, bu turda oy vermiş hayattaki oyuncu sayısıdır.
func (s State) VotesCast() int {
	n := 0
	for _, v := range s.Votes {
		if p := s.player(v.Voter); p != nil && p.Alive() {
			n++
		}
	}
	return n
}

// Vote, hayattaki oyuncunun gizli oyudur (game-design §8): tur başına bir oy,
// değiştirilemez, kendine verilemez, hedef hayatta olmalı. Hayattaki herkes
// oy verince sayım hemen yapılır (rng eşitlikte kura için).
func (e Engine) Vote(s State, voterID, targetID string, now time.Time, rng Rand) (State, []Event, error) {
	if s.Phase != PhaseVoting {
		return s, nil, invalidPhase()
	}
	if _, err := requireAlive(s, voterID); err != nil {
		return s, nil, err
	}
	if slices.ContainsFunc(s.Votes, func(v Vote) bool { return v.Voter == voterID }) {
		return s, nil, apperr.New(apperr.AlreadyVoted, "bu turda oy verildi")
	}
	if voterID == targetID {
		return s, nil, apperr.New(apperr.SelfVote, "kendine oy verilemez")
	}
	if t := s.player(targetID); t == nil || !t.Alive() {
		return s, nil, apperr.New(apperr.InvalidTarget, "hedef hayatta değil")
	}
	st := e.begin(s, now, rng)
	st.s.Votes = append(st.s.Votes, Vote{Voter: voterID, Target: targetID})
	cast, total := st.s.VotesCast(), st.s.AliveCount()
	st.emit(EventVoteProgress, map[string]any{"cast": cast, "total": total})
	if cast == total {
		st.resolveVoting()
	}
	return st.done()
}
