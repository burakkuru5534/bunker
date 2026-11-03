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
	return loadRoom(ctx, tx, roomID, true)
}

// loadRoom, odayı tüm üyeleriyle okur; lock ise satır kilidi alır.
func loadRoom(ctx context.Context, q persistence.Querier, roomID string, lock bool) (Room, error) {
	r := Room{ID: roomID}
	var status string
	sql := `select invite_code, coalesce(host_user_id::text, ''), capacity, status, current_game_id::text, version
		  from rooms where id = $1`
	if lock {
		sql += " for update"
	}
	err := q.QueryRow(ctx, sql, roomID).
		Scan(&r.InviteCode, &r.HostUserID, &r.Capacity, &status, &r.CurrentGameID, &r.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return Room{}, errRoomNotFound
	}
	if err != nil {
		return Room{}, err
	}
	r.Status = Status(status)
	members, err := loadMembers(ctx, q, roomID)
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

// openRoomIDByCode, açık odayı davet koduyla bulur ve kilitler.
func openRoomIDByCode(ctx context.Context, tx pgx.Tx, code string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `select id::text from rooms where invite_code = $1 and status <> 'closed' for update`, code).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errRoomNotFound
	}
	return id, err
}

// wasParticipant, kullanıcı odanın şu anki oyununun katılımcısı mıydı?
func wasParticipant(ctx context.Context, tx pgx.Tx, gameID *string, userID string) (bool, error) {
	if gameID == nil {
		return false, nil
	}
	var ok bool
	err := tx.QueryRow(ctx, `select exists(select 1 from game_participants where game_id = $1 and user_id = $2)`,
		*gameID, userID).Scan(&ok)
	return ok, err
}

// upsertMember, üyeliği yazar. Daha önce ayrılmış (atılmamış) kullanıcının
// satırı yeniden açılır.
func upsertMember(ctx context.Context, tx pgx.Tx, roomID, userID, nickname string, role Role) (time.Time, error) {
	var joined time.Time
	err := tx.QueryRow(ctx, `insert into room_members (room_id, user_id, nickname, role)
		values ($1, $2, $3, $4)
		on conflict (room_id, user_id) do update
		   set nickname = excluded.nickname, role = excluded.role, is_ready = false,
		       joined_at = now(), left_at = null
		 where room_members.kicked_at is null
		returning joined_at`, roomID, userID, nickname, string(role)).Scan(&joined)
	return joined, err
}

// bumpVersion, oda sürümünü artırır ve yeni sürümü döndürür.
func bumpVersion(ctx context.Context, tx pgx.Tx, roomID string) (int64, error) {
	var v int64
	err := tx.QueryRow(ctx, `update rooms set version = version + 1, updated_at = now() where id = $1 returning version`, roomID).Scan(&v)
	return v, err
}
