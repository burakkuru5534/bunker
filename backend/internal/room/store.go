package room

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/persistence"
)

// Veritabanı kısıt adları (migrations/postgres/0001).
const (
	constraintOneActiveRoom = "room_members_one_active_room_key"
	constraintInviteCode    = "rooms_invite_code_open_key"
	constraintNickname      = "room_members_active_nickname_key"
)

var errRoomNotFound = errors.New("oda bulunamadı")

// loadForUpdate, odayı satır kilidiyle ve tüm üyeleriyle okur (ADR-0004).
// Yalnızca transaction içinde çağrılır.
func loadForUpdate(ctx context.Context, tx pgx.Tx, roomID string) (Room, error) {
	r := Room{ID: roomID}
	var status string
	err := tx.QueryRow(ctx, `select invite_code, coalesce(host_user_id::text, ''), capacity, status,
		       current_game_id::text, version
		  from rooms where id = $1 for update`, roomID).
		Scan(&r.InviteCode, &r.HostUserID, &r.Capacity, &status, &r.CurrentGameID, &r.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return Room{}, errRoomNotFound
	}
	if err != nil {
		return Room{}, err
	}
	r.Status = Status(status)
	members, err := loadMembers(ctx, tx, roomID)
	r.Members = members
	return r, err
}

func loadMembers(ctx context.Context, q persistence.Querier, roomID string) ([]Member, error) {
	rows, err := q.Query(ctx, `select user_id::text, nickname, role, is_ready, joined_at, left_at, kicked_at
		  from room_members where room_id = $1 order by joined_at, user_id`, roomID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Member, error) {
		var m Member
		var role string
		err := row.Scan(&m.UserID, &m.Nickname, &role, &m.Ready, &m.JoinedAt, &m.LeftAt, &m.KickedAt)
		m.Role = Role(role)
		return m, err
	})
}

// insertRoom, odayı ve oda sahibinin oyuncu üyeliğini yazar.
func insertRoom(ctx context.Context, tx pgx.Tx, code, hostID, hostNick string, capacity int) (Room, error) {
	r := Room{InviteCode: code, HostUserID: hostID, Capacity: capacity, Status: StatusLobby}
	if err := tx.QueryRow(ctx, `insert into rooms (invite_code, host_user_id, capacity) values ($1, $2, $3)
		returning id::text, version`, code, hostID, capacity).Scan(&r.ID, &r.Version); err != nil {
		return Room{}, err
	}
	var joined time.Time
	if err := tx.QueryRow(ctx, `insert into room_members (room_id, user_id, nickname, role)
		values ($1, $2, $3, 'player') returning joined_at`, r.ID, hostID, hostNick).Scan(&joined); err != nil {
		return Room{}, err
	}
	r.Members = []Member{{UserID: hostID, Nickname: hostNick, Role: RolePlayer, JoinedAt: joined}}
	return r, nil
}

// activeRoomID, kullanıcının aktif odasıdır (yoksa boş).
func activeRoomID(ctx context.Context, db *persistence.DB, userID string) (string, error) {
	var id string
	err := db.Read(ctx, func(q persistence.Querier) error {
		return q.QueryRow(ctx, `select room_id::text from room_members where user_id = $1 and left_at is null`, userID).Scan(&id)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}
