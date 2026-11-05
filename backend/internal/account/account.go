// Package account, hesap silmedir (SAFE-009; Google Play hesap silme
// gerekliliği). Oturum (auth) ve oda (room) modüllerini birlikte kullandığı
// için ayrı pakettedir (room → auth bağımlılığı tersine dönmesin).
package account

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
	"github.com/burakkuru5534/bunker/backend/internal/observability"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
	"github.com/burakkuru5534/bunker/backend/internal/room"
)

// Service, hesap işlemleridir.
type Service struct {
	db     *persistence.DB
	auth   *auth.Service
	rooms  *room.Service
	logger *slog.Logger
}

// NewService, servisi kurar.
func NewService(db *persistence.DB, a *auth.Service, rooms *room.Service, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{db: db, auth: a, rooms: rooms, logger: logger}
}

// Delete, hesabı siler (docs/security.md §1):
//  1. Aktif odadan "Ayrıl" ile aynı yoldan ayrılır (oyundaysa oyunda da
//     elenmiş sayılır; sahiplik devredilir).
//  2. Tek transaction'da Anonymize.
//  3. Oturum önbelleği düşürülür (açık istekler bile artık doğrulanamaz).
//
// Raporlar moderasyon için kalır (reporter_id/reported_id kullanıcıya bağlı;
// takma ad anonimleşmiştir). Rapor kanıtındaki mesaj kopyaları 180 günlük
// rapor saklamasına tabidir.
func (s *Service) Delete(ctx context.Context, p auth.Principal) error {
	var roomID string
	err := s.db.Read(ctx, func(q persistence.Querier) error {
		return q.QueryRow(ctx, `select room_id::text from room_members where user_id = $1 and left_at is null`, p.UserID).Scan(&roomID)
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if roomID != "" {
		if err := s.rooms.Leave(ctx, p, roomID); err != nil {
			return err
		}
	}
	err = s.db.Tx(ctx, func(tx pgx.Tx) error { return Anonymize(ctx, tx, p.UserID) })
	if err != nil {
		return err
	}
	s.auth.ForgetUser(p.UserID)
	observability.Security(ctx, s.logger, slog.LevelInfo, observability.EventAccountDeleted, "hesap silindi",
		slog.String("user_id", p.UserID), slog.Bool("left_room", roomID != ""))
	return nil
}

// Anonymize, hesap silmenin veritabanı kısmıdır (tek transaction içinde
// çağrılır): takma ad "Silinmiş oyuncu" ve deleted_at (oyun geçmişinde böyle
// görünür), geçmiş oda üyeliklerindeki takma ad da; tüm oturumlar, kendi
// sohbet mesajları ve engelleme kayıtları (iki yönde) silinir. Aktif odadan
// ayrılma ve oturum önbelleği çağıranın işidir. Hem DELETE /v1/me hem
// kullanılmayan hesap temizliği (internal/maintenance, SAFE-014) kullanır.
func Anonymize(ctx context.Context, tx pgx.Tx, userID string) error {
	for _, q := range []string{
		`update users set nickname = $2, deleted_at = coalesce(deleted_at, now()) where id = $1`,
		`update room_members set nickname = $2 where user_id = $1`,
	} {
		if _, err := tx.Exec(ctx, q, userID, auth.DeletedNickname); err != nil {
			return err
		}
	}
	for _, q := range []string{
		`delete from sessions where user_id = $1`,
		`delete from chat_messages where user_id = $1`,
		`delete from blocks where blocker_id = $1 or blocked_id = $1`,
	} {
		if _, err := tx.Exec(ctx, q, userID); err != nil {
			return err
		}
	}
	return nil
}

// HTTP, hesap uç noktalarıdır.
type HTTP struct {
	svc     *Service
	require func(http.Handler) http.Handler
}

// NewHTTP, uç noktaları oturum ara katmanıyla kurar.
func NewHTTP(svc *Service, require func(http.Handler) http.Handler) *HTTP {
	return &HTTP{svc: svc, require: require}
}

// Register, rotaları ekler.
func (h *HTTP) Register(mux *http.ServeMux) {
	mux.Handle("DELETE /v1/me", h.require(httpapi.Handle(h.delete)))
}

// DELETE /v1/me → 204
func (h *HTTP) delete(w http.ResponseWriter, r *http.Request) error {
	p, _ := auth.PrincipalFrom(r.Context())
	if err := h.svc.Delete(r.Context(), p); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
