package game

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
	"github.com/burakkuru5534/bunker/backend/internal/room"
)

// Oyundan ayrılmanın eylem kayıtları (game_actions.type).
const (
	ActionLeave             = "leave"              // oyuncu "Ayrıl" dedi
	ActionDisconnectTimeout = "disconnect_timeout" // bağlantı 60 sn dönmedi
)

type leaveReasonKey struct{}

// withLeaveReason, oda üzerinden yapılan ayrılmanın kaydedilecek sebebidir
// (yalnızca bu paket kullanır; varsayılan ActionLeave).
func withLeaveReason(ctx context.Context, reason string) context.Context {
	return context.WithValue(ctx, leaveReasonKey{}, reason)
}

func leaveReason(ctx context.Context) string {
	if r, ok := ctx.Value(leaveReasonKey{}).(string); ok {
		return r
	}
	return ActionLeave
}

// leaveGame, katılımcıyı oyunda "ayrıldı" yapar (ENG-010). Çağıran odanın
// transaction'ıdır ve oda satırı kilitlidir; oyun burada kilitlenir (sıra
// oda → oyun). Katılımcı değilse, zaten hayatta değilse veya oyun bittiyse
// hiçbir şey yazılmaz.
func (s *Service) leaveGame(ctx context.Context, tx pgx.Tx, gameID, userID string) ([]room.Event, error) {
	old, g, err := loadState(ctx, tx, gameID, true)
	if errors.Is(err, errGameNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	now := s.clock()
	next, evs, err := s.engine.Leave(old, userID, now, s.rng)
	if errors.Is(err, engine.ErrNoChange) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := saveState(ctx, tx, gameID, old, next, now); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `insert into game_actions (game_id, actor_id, action_id, type, payload, state_version)
		values ($1, $2, $3, $4, '{}', $5)`, gameID, userID, newUUID(), leaveReason(ctx), next.Version); err != nil {
		return nil, err
	}
	s.logger.Info("oyuncu oyundan ayrıldı", "game_id", gameID, "user_id", userID, "reason", leaveReason(ctx), "phase", string(old.Phase),
		"finished", next.Phase == engine.PhaseFinished)
	return s.wireEvents(ctx, tx, g, evs, next.Version, now)
}

// RoomHooks, odanın oyun modülüne bağlantısıdır (room.GameStarter):
// başlatma Starter'da, ayrılma Service'te.
type RoomHooks struct {
	Starter *Starter
	Service *Service
}

var _ room.GameStarter = RoomHooks{}

// StartGame, room.GameStarter.
func (h RoomHooks) StartGame(ctx context.Context, tx pgx.Tx, in room.GameStart) (room.GameStarted, error) {
	return h.Starter.StartGame(ctx, tx, in)
}

// LeaveGame, room.GameStarter.
func (h RoomHooks) LeaveGame(ctx context.Context, tx pgx.Tx, gameID, userID string) ([]room.Event, error) {
	return h.Service.leaveGame(ctx, tx, gameID, userID)
}
