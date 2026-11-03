package engine

import (
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
