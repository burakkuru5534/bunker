package room

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
)

// GameStarter, oyunu odanın transaction'ı içinde kurar (ENG-003). Oyun
// modülü uygular; room paketi oyun paketini bilmez (bağımlılık yönü
// game → room).
type GameStarter interface {
	StartGame(ctx context.Context, tx pgx.Tx, in GameStart) (GameStarted, error)
}

// GameStart, başlatılacak oyunun oda tarafındaki girdisidir.
type GameStart struct {
	RoomID   string
	Capacity int
	Players  []Member // katılma sırasıyla; koltukları oyun modülü karıştırır
}

// GameStarted, kurulan oyun ve commit sonrası yayınlanacak olaylardır.
// Oyun olaylarının Version'ı oyunun state_version'ıdır (api.md §5: oyun
// sürerken zarfın state_version'ı oyun sürümüdür); mutate bunu ezmez.
type GameStarted struct {
	GameID string
	Events []Event
}

var errNoGames = errors.New("room: oyun modülü bağlı değil")

// WithGames, oyun başlatıcısını bağlar (main ve testler).
func (s *Service) WithGames(g GameStarter) *Service {
	s.games = g
	return s
}

// Start, oda sahibinin oyunu başlatmasıdır (ENG-003): oda kilitliyken
// koşullar denetlenir (CanStart), oyun aynı transaction'da kurulur, oda
// in_game olur ve oda sürümü artar. Olaylar commit sonrası yayınlanır.
// Eşzamanlı ikinci başlatma, kilit sonrası oda lobide olmadığı için
// INVALID_PHASE alır.
func (s *Service) Start(ctx context.Context, p auth.Principal, roomID string) (string, error) {
	var gameID string
	_, err := s.mutate(ctx, p, roomID, func(tx pgx.Tx, r *Room) (change, error) {
		if err := CanStart(*r, p.UserID); err != nil {
			return change{}, err
		}
		if s.games == nil {
			return change{}, errNoGames
		}
		started, err := s.games.StartGame(ctx, tx, GameStart{RoomID: r.ID, Capacity: r.Capacity, Players: r.Players()})
		if err != nil {
			return change{}, err
		}
		if err := markInGame(ctx, tx, r.ID, started.GameID); err != nil {
			return change{}, err
		}
		r.Status, r.CurrentGameID = StatusInGame, &started.GameID
		gameID = started.GameID
		s.logger.Info("oyun başladı", "room_id", r.ID, "game_id", gameID, "players", len(r.Players()))
		return change{events: started.Events}, nil
	})
	return gameID, err
}

func markInGame(ctx context.Context, tx pgx.Tx, roomID, gameID string) error {
	_, err := tx.Exec(ctx, `update rooms set status = 'in_game', current_game_id = $2, updated_at = now() where id = $1`, roomID, gameID)
	return err
}

// POST /v1/rooms/{roomId}/start → 201 {"game_id"}
func (h *HTTP) start(w http.ResponseWriter, r *http.Request) error {
	p, _ := auth.PrincipalFrom(r.Context())
	id, err := h.svc.Start(r.Context(), p, r.PathValue("roomId"))
	if err != nil {
		return err
	}
	httpapi.WriteJSON(w, http.StatusCreated, map[string]string{"game_id": id})
	return nil
}
