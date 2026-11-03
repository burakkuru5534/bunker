package realtime

import (
	"context"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/room"
)

// Snapshot, session.snapshot payload'ıdır (api.md §5).
type Snapshot struct {
	ServerTime time.Time `json:"server_time"`
	Me         SnapMe    `json:"me"`
	Room       room.View `json:"room"`
	// Game, oyun sürüyorsa izleyiciye göre süzülmüş oyun durumu (ENG-013);
	// yoksa null.
	Game any   `json:"game"`
	Chat []any `json:"chat"` // sohbet modülüyle (CHAT) dolar
}

// SnapMe, bağlanan kullanıcıdır.
type SnapMe struct {
	UserID string    `json:"user_id"`
	Role   room.Role `json:"role"`
}

// GameSnapshotter, oyunun izleyiciye özel görünümünü kurar (ENG-013/015).
type GameSnapshotter interface {
	GameSnapshot(ctx context.Context, userID, gameID string) (any, error)
}

// RoomSnapshotter, Snapshotter'ın oda (ve varsa oyun) görünümüyle uygulamasıdır.
type RoomSnapshotter struct {
	Rooms *room.Service
	Games GameSnapshotter // nil: game alanı null
	Now   func() time.Time
}

// Snapshot, Snapshotter. Üye değilse (oda yok, kapalı, bozuk kimlik dahil)
// NOT_FOUND.
func (s RoomSnapshotter) Snapshot(ctx context.Context, p auth.Principal, roomID string) (any, error) {
	m, err := s.Rooms.Membership(ctx, p.UserID, roomID)
	if err != nil {
		return nil, err
	}
	view, err := s.Rooms.Get(ctx, p, roomID)
	if err != nil {
		return nil, err
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	snap := Snapshot{ServerTime: now().UTC(), Me: SnapMe{UserID: p.UserID, Role: m.Role}, Room: view, Chat: []any{}}
	if s.Games != nil && m.CurrentGameID != nil {
		if snap.Game, err = s.Games.GameSnapshot(ctx, p.UserID, *m.CurrentGameID); err != nil {
			return nil, err
		}
	}
	return snap, nil
}
