package room

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
)

// Rematch, oyun bitince oda sahibinin odayı lobiye döndürmesidir (api.md
// "Tekrar oyna"): aynı oda ve davet kodu; tüm aktif üyeler (izleyiciler
// dahil) oyuncu olur, hazırlar sıfırlanır, oda sürümü artar ve odaya
// room.returned_to_lobby {room: RoomView} yayınlanır. Biten oyun kalıcıdır
// (sonuç GET /v1/games/{id}/result ile okunmaya devam eder).
func (s *Service) Rematch(ctx context.Context, p auth.Principal, roomID string) (View, error) {
	r, err := s.mutate(ctx, p, roomID, func(tx pgx.Tx, r *Room) (change, error) {
		if err := Rematch(*r, p.UserID); err != nil {
			return change{}, err
		}
		if _, err := tx.Exec(ctx, `update rooms set status = 'lobby', current_game_id = null, updated_at = now() where id = $1`, r.ID); err != nil {
			return change{}, err
		}
		if _, err := tx.Exec(ctx, `update room_members set role = 'player', is_ready = false
			where room_id = $1 and left_at is null`, r.ID); err != nil {
			return change{}, err
		}
		r.Status, r.CurrentGameID = StatusLobby, nil
		for i := range r.Members {
			if r.Members[i].Active() {
				r.Members[i].Role, r.Members[i].Ready = RolePlayer, false
			}
		}
		// mutate commit öncesi sürümü tam bir artırır; görünüm o sürümü taşır.
		view := NewView(*r, s.presence)
		view.Version = r.Version + 1
		return change{events: []Event{{Type: EventReturnedToLobby, Payload: map[string]any{"room": view}}}}, nil
	})
	if err != nil {
		return View{}, err
	}
	return NewView(r, s.presence), nil
}

// POST /v1/rooms/{roomId}/rematch → 200 RoomView
func (h *HTTP) rematch(w http.ResponseWriter, r *http.Request) error {
	p, _ := auth.PrincipalFrom(r.Context())
	v, err := h.svc.Rematch(r.Context(), p, r.PathValue("roomId"))
	if err != nil {
		return err
	}
	httpapi.WriteJSON(w, http.StatusOK, v)
	return nil
}
