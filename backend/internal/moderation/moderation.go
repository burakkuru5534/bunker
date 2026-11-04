// Package moderation, oyuncu engelleme (SAFE-006) ve raporlamadır (SAFE-005);
// Google Play kullanıcı içeriği politikası gereği zorunludur.
package moderation

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
	"github.com/burakkuru5534/bunker/backend/internal/ratelimit"
	"github.com/burakkuru5534/bunker/backend/internal/room"
)

// Rapor nedenleri (reports.reason).
var Reasons = []string{"harassment", "hate", "spam", "cheating", "other"}

// Sınırlar.
const (
	// EvidenceMessages, rapora kopyalanan son mesaj sayısı.
	EvidenceMessages = 50
	// MaxNoteRunes, rapor notunun uzunluğu.
	MaxNoteRunes = 200
)

// Service, engelleme ve raporlamadır.
type Service struct {
	db      *persistence.DB
	logger  *slog.Logger
	reports *ratelimit.Limiter
}

// NewService, servisi kurar.
func NewService(db *persistence.DB, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{db: db, logger: logger, reports: ratelimit.New(10, time.Hour, 100_000)}
}

// Block, kullanıcıyı engeller (tekrar gelirse değişiklik yok). Engellenenin
// mesajları engelleyene gösterilmez; oyun kurallarını etkilemez.
func (s *Service) Block(ctx context.Context, p auth.Principal, target string) error {
	if err := validTarget(p, target, "user_id"); err != nil {
		return err
	}
	return s.db.Tx(ctx, func(tx pgx.Tx) error {
		if err := userExists(ctx, tx, target); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `insert into blocks (blocker_id, blocked_id) values ($1, $2) on conflict do nothing`, p.UserID, target)
		return err
	})
}

// Unblock, engeli kaldırır (yoksa değişiklik yok).
func (s *Service) Unblock(ctx context.Context, p auth.Principal, target string) error {
	if !validUUID(target) {
		return nil
	}
	return s.db.Tx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `delete from blocks where blocker_id = $1 and blocked_id = $2`, p.UserID, target)
		return err
	})
}

// BlockedUser, engellenenler listesindeki satırdır.
type BlockedUser struct {
	UserID    string    `json:"user_id"`
	Nickname  string    `json:"nickname"`
	CreatedAt time.Time `json:"created_at"`
}

// Blocks, kullanıcının engelledikleri (en yeni önce).
func (s *Service) Blocks(ctx context.Context, p auth.Principal) ([]BlockedUser, error) {
	out := []BlockedUser{}
	err := s.db.Read(ctx, func(q persistence.Querier) error {
		rows, err := q.Query(ctx, `select b.blocked_id::text, u.nickname, b.created_at from blocks b
			join users u on u.id = b.blocked_id where b.blocker_id = $1 order by b.created_at desc, b.blocked_id`, p.UserID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var b BlockedUser
			if err := rows.Scan(&b.UserID, &b.Nickname, &b.CreatedAt); err != nil {
				return err
			}
			b.CreatedAt = b.CreatedAt.UTC()
			out = append(out, b)
		}
		return rows.Err()
	})
	return out, err
}

// ReportRequest, POST /v1/reports gövdesidir.
type ReportRequest struct {
	ReportedUserID string  `json:"reported_user_id"`
	RoomID         string  `json:"room_id"`
	Reason         string  `json:"reason"`
	Note           *string `json:"note"`
}

// Report, raporu kaydeder. İki kullanıcı o odayı paylaşmış olmalı (üyelik,
// oyun katılımı veya mesaj); değilse — oda yoksa da — NOT_FOUND. Rapora
// odanın son 50 mesajı kopyalanır (sohbet 30 günde silinse de kanıt kalır).
// Aynı kişi aynı odada açık raporla tekrar raporlanırsa yeni kayıt açılmaz,
// mevcut raporun kimliği döner. Kullanıcı başına saatte 10 rapor.
func (s *Service) Report(ctx context.Context, p auth.Principal, req ReportRequest) (string, error) {
	var fe apperr.FieldErrors
	switch {
	case req.ReportedUserID == "":
		fe.Add("reported_user_id", apperr.FieldRequired)
	case !validUUID(req.ReportedUserID):
		fe.Add("reported_user_id", apperr.FieldInvalid)
	case strings.EqualFold(req.ReportedUserID, p.UserID):
		fe.Add("reported_user_id", apperr.FieldNotAllowed)
	}
	switch {
	case req.RoomID == "":
		fe.Add("room_id", apperr.FieldRequired)
	case !validUUID(req.RoomID):
		fe.Add("room_id", apperr.FieldInvalid)
	}
	if req.Reason == "" {
		fe.Add("reason", apperr.FieldRequired)
	} else if !contains(Reasons, req.Reason) {
		fe.Add("reason", apperr.FieldNotAllowed)
	}
	var note any
	if req.Note != nil {
		n, err := room.CleanMessage(*req.Note) // sohbetle aynı temizlik, 1–200
		switch {
		case err != nil && len(strings.Fields(*req.Note)) == 0:
			// Boş not: yok sayılır.
		case err != nil && utf8.RuneCountInString(strings.Join(strings.Fields(*req.Note), " ")) > MaxNoteRunes:
			fe.Add("note", apperr.FieldTooLong)
		case err != nil:
			fe.Add("note", apperr.FieldInvalid)
		default:
			note = n
		}
	}
	if err := fe.Err(); err != nil {
		return "", err
	}
	if ok, wait := s.reports.Allow(p.UserID); !ok {
		return "", apperr.RateLimit(wait)
	}

	reported := strings.ToLower(req.ReportedUserID)
	var id string
	err := s.db.Tx(ctx, func(tx pgx.Tx) error {
		var shared bool
		if err := tx.QueryRow(ctx, `select bool_and(present) from (
			select exists (select 1 from room_members where room_id = $1 and user_id = u)
			    or exists (select 1 from game_participants gp join games g on g.id = gp.game_id
			               where g.room_id = $1 and gp.user_id = u)
			    or exists (select 1 from chat_messages where room_id = $1 and user_id = u) as present
			from unnest(array[$2::uuid, $3::uuid]) as u) t`, req.RoomID, p.UserID, reported).Scan(&shared); err != nil {
			return err
		}
		if !shared {
			return apperr.New(apperr.NotFound, "ortak oda bulunamadı")
		}
		err := tx.QueryRow(ctx, `select id::text from reports where reporter_id = $1 and reported_id = $2
			and room_id = $3 and status = 'open'`, p.UserID, reported, req.RoomID).Scan(&id)
		if err == nil {
			return nil // açık rapor zaten var
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		return tx.QueryRow(ctx, `insert into reports (reporter_id, reported_id, room_id, reason, note, messages_snapshot)
			select $1, $2, $3, $4, $5, coalesce(jsonb_agg(m order by m.created_at, m.id), '[]'::jsonb)
			from (select c.id, c.user_id, u.nickname, c.body as text, c.created_at
			      from chat_messages c join users u on u.id = c.user_id
			      where c.room_id = $3 order by c.created_at desc, c.id desc limit $6) m
			returning id::text`, p.UserID, reported, req.RoomID, req.Reason, note, EvidenceMessages).Scan(&id)
	})
	if err != nil {
		return "", err
	}
	s.logger.Info("oyuncu raporlandı", "report_id", id, "reporter_id", p.UserID, "reported_id", reported,
		"room_id", req.RoomID, "reason", req.Reason)
	return id, nil
}

func validTarget(p auth.Principal, target, field string) error {
	switch {
	case target == "":
		return apperr.Field(field, apperr.FieldRequired)
	case !validUUID(target):
		return apperr.Field(field, apperr.FieldInvalid)
	case strings.EqualFold(target, p.UserID):
		return apperr.Field(field, apperr.FieldNotAllowed)
	}
	return nil
}

// userExists: silinmemiş kullanıcı (değilse NOT_FOUND).
func userExists(ctx context.Context, q persistence.Querier, id string) error {
	var ok bool
	if err := q.QueryRow(ctx, `select exists (select 1 from users where id = $1 and deleted_at is null)`, id).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return apperr.New(apperr.NotFound, "kullanıcı bulunamadı")
	}
	return nil
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func validUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < 36; i++ {
		c := s[i]
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
				return false
			}
		}
	}
	return true
}
