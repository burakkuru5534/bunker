package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/persistence"
)

// errNotFound, token hash'iyle oturum bulunamadığında (iç kullanım).
var errNotFound = errors.New("oturum bulunamadı")

// errAlreadyRotated, eşzamanlı iki yenilemeden ikincisinde döner.
var errAlreadyRotated = errors.New("oturum zaten yenilenmiş veya iptal edilmiş")

// User, misafir kullanıcıdır.
type User struct {
	ID       string
	Nickname string
}

// sessionRow, doğrulama için okunan oturum + kullanıcı satırıdır.
type sessionRow struct {
	SessionID  string
	UserID     string
	Nickname   string
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	LastUsedAt *time.Time
	DeletedAt  *time.Time
}

// store, auth modülünün veritabanı sorgularıdır (docs/queries.md "Oturum
// doğrulama", "Kullanıcının aktif odası").
type store struct{ db *persistence.DB }

func (s store) createGuest(ctx context.Context, nickname string, hash []byte, expiresAt time.Time) (User, string, error) {
	u := User{Nickname: nickname}
	var sessionID string
	err := s.db.Tx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `insert into users (nickname) values ($1) returning id`, nickname).Scan(&u.ID); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `insert into sessions (user_id, token_hash, expires_at) values ($1, $2, $3) returning id`,
			u.ID, hash, expiresAt).Scan(&sessionID)
	})
	return u, sessionID, err
}

func (s store) lookup(ctx context.Context, hash []byte) (sessionRow, error) {
	var r sessionRow
	err := s.db.Read(ctx, func(q persistence.Querier) error {
		return q.QueryRow(ctx, `
			select s.id, s.user_id, u.nickname, s.expires_at, s.revoked_at, s.last_used_at, u.deleted_at
			  from sessions s join users u on u.id = s.user_id
			 where s.token_hash = $1`, hash).
			Scan(&r.SessionID, &r.UserID, &r.Nickname, &r.ExpiresAt, &r.RevokedAt, &r.LastUsedAt, &r.DeletedAt)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return r, errNotFound
	}
	return r, err
}

// rotate, eski oturumu iptal eder ve aynı kullanıcı için yenisini açar
// (tek transaction). Eski oturum zaten iptal edilmişse errAlreadyRotated.
func (s store) rotate(ctx context.Context, oldSessionID string, hash []byte, expiresAt time.Time) (string, error) {
	var newID string
	err := s.db.Tx(ctx, func(tx pgx.Tx) error {
		var userID string
		err := tx.QueryRow(ctx, `update sessions set revoked_at = now()
			 where id = $1 and revoked_at is null and expires_at > now() returning user_id`, oldSessionID).Scan(&userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return errAlreadyRotated
		}
		if err != nil {
			return err
		}
		return tx.QueryRow(ctx, `insert into sessions (user_id, token_hash, expires_at) values ($1, $2, $3) returning id`,
			userID, hash, expiresAt).Scan(&newID)
	})
	return newID, err
}

// touch, son kullanım zamanlarını yazar (en fazla saatte bir çağrılır).
func (s store) touch(ctx context.Context, sessionID, userID string) error {
	return s.db.Tx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `update sessions set last_used_at = now() where id = $1`, sessionID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `update users set last_seen_at = now() where id = $1`, userID)
		return err
	})
}

// activeRoom, kullanıcının aktif odasını döndürür (yoksa nil).
func (s store) activeRoom(ctx context.Context, userID string) (*string, error) {
	var room *string
	err := s.db.Read(ctx, func(q persistence.Querier) error {
		err := q.QueryRow(ctx, `select room_id from room_members where user_id = $1 and left_at is null`, userID).Scan(&room)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	})
	return room, err
}
