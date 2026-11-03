package room

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
)

// DefaultCapacity, oda oluşturulurken kapasite verilmezse (game-design §1).
const DefaultCapacity = 2

// inviteAttempts, açık odalarla davet kodu çakışmasında deneme sayısı
// (887 milyon olasılıkla pratikte hiç tekrar gerekmez).
const inviteAttempts = 5

// Service, oda işlemleridir (ROOM-003…).
type Service struct {
	db       *persistence.DB
	presence Presence
	logger   *slog.Logger
	newCode  func() (string, error)
}

// NewService, servisi kurar. presence nil olabilir (RT öncesi).
func NewService(db *persistence.DB, presence Presence, logger *slog.Logger) *Service {
	if presence == nil {
		presence = NoPresence{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{db: db, presence: presence, logger: logger, newCode: NewInviteCode}
}

// Create, yeni oda kurar; oluşturan oda sahibi ve oyuncu olur. Kullanıcı
// başka bir aktif odadaysa ALREADY_IN_ROOM (details.room_id).
func (s *Service) Create(ctx context.Context, p auth.Principal, capacity int) (View, error) {
	if capacity != 1 && capacity != 2 {
		return View{}, apperr.Field("capacity", apperr.FieldNotAllowed)
	}
	for attempt := 1; ; attempt++ {
		code, err := s.newCode()
		if err != nil {
			return View{}, err
		}
		var r Room
		err = s.db.Tx(ctx, func(tx pgx.Tx) error {
			var err error
			r, err = insertRoom(ctx, tx, code, p.UserID, p.Nickname, capacity)
			return err
		})
		switch {
		case err == nil:
			s.logger.Info("oda oluşturuldu", "room_id", r.ID, "capacity", capacity)
			return NewView(r, s.presence), nil
		case persistence.UniqueViolation(err, constraintOneActiveRoom):
			return View{}, s.alreadyInRoom(ctx, p.UserID)
		case persistence.UniqueViolation(err, constraintInviteCode) && attempt < inviteAttempts:
			s.logger.Warn("davet kodu çakıştı, yeniden deneniyor", "attempt", attempt)
			continue
		default:
			return View{}, err
		}
	}
}

func (s *Service) alreadyInRoom(ctx context.Context, userID string) error {
	e := apperr.New(apperr.AlreadyInRoom, "kullanıcı başka bir aktif odada")
	if id, err := activeRoomID(ctx, s.db, userID); err == nil && id != "" {
		e.WithDetail("room_id", id)
	}
	return e
}
