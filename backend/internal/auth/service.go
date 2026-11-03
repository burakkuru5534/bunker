package auth

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
)

// Doğrulama önbelleği ve son kullanım yazımı (security.md §1).
const (
	CacheTTL      = 60 * time.Second
	cacheMax      = 20000
	touchInterval = time.Hour
)

// Principal, doğrulanmış isteğin sahibidir.
type Principal struct {
	SessionID string
	UserID    string
	Nickname  string
	ExpiresAt time.Time
	tokenKey  string     // önbellek anahtarı (hash); dışarı açılmaz
	lastUsed  *time.Time // son kullanım yazımı (saatte bir)
}

// Issued, yeni verilen oturumdur. Token.Raw yalnızca yanıtta bir kez gider.
type Issued struct {
	Token     string
	ExpiresAt time.Time
	User      User
}

// Me, GET /v1/me yanıtının verisidir.
type Me struct {
	User         User
	ActiveRoomID *string
}

// Service, misafir oturumlarını yönetir (AUTH-002).
type Service struct {
	store     store
	cache     *cache
	profanity ProfanityChecker
	now       func() time.Time
	logger    *slog.Logger
}

// NewService, servisi kurar. profanity nil olabilir (SAFE-013 öncesi).
func NewService(db *persistence.DB, profanity ProfanityChecker, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{store: store{db}, cache: newCache(CacheTTL, cacheMax), profanity: profanity, now: time.Now, logger: logger}
}

// errUnauthorized, istemciye nedeni ayrıştırılmadan döner (security.md §1).
func errUnauthorized() error { return apperr.New(apperr.Unauthorized, "oturum geçersiz") }

// CreateGuest, misafir kullanıcı ve oturum oluşturur.
func (s *Service) CreateGuest(ctx context.Context, rawNickname string) (Issued, error) {
	nick, err := ValidateNickname(rawNickname, s.profanity)
	if err != nil {
		return Issued{}, err
	}
	tok, err := NewToken()
	if err != nil {
		return Issued{}, err
	}
	expires := s.now().Add(SessionTTL).UTC().Truncate(time.Second)
	user, _, err := s.store.createGuest(ctx, nick, tok.Hash, expires)
	if err != nil {
		return Issued{}, err
	}
	return Issued{Token: tok.Raw, ExpiresAt: expires, User: user}, nil
}

// Authenticate, ham token'ı doğrular. Biçimi bozuk, bilinmeyen, süresi
// dolmuş, iptal edilmiş veya kullanıcısı silinmiş oturum UNAUTHORIZED'dır.
func (s *Service) Authenticate(ctx context.Context, raw string) (Principal, error) {
	hash, err := HashToken(raw)
	if err != nil {
		return Principal{}, errUnauthorized()
	}
	key := string(hash)
	now := s.now()
	if p, ok := s.cache.get(key, now); ok {
		s.maybeTouch(ctx, p, now)
		return p, nil
	}

	r, err := s.store.lookup(ctx, hash)
	switch {
	case errors.Is(err, errNotFound):
		return Principal{}, errUnauthorized()
	case err != nil:
		return Principal{}, err
	case r.RevokedAt != nil, !now.Before(r.ExpiresAt), r.DeletedAt != nil:
		return Principal{}, errUnauthorized()
	}
	p := Principal{SessionID: r.SessionID, UserID: r.UserID, Nickname: r.Nickname, ExpiresAt: r.ExpiresAt,
		tokenKey: key, lastUsed: r.LastUsedAt}
	s.cache.put(key, p, now)
	s.maybeTouch(ctx, p, now)
	return p, nil
}

// maybeTouch, son kullanımı en fazla saatte bir yazar. Hata isteği
// bozmaz; yalnızca loglanır.
func (s *Service) maybeTouch(ctx context.Context, p Principal, now time.Time) {
	if p.lastUsed != nil && now.Sub(*p.lastUsed) < touchInterval {
		return
	}
	if err := s.store.touch(ctx, p.SessionID, p.UserID); err != nil {
		s.logger.Warn("oturum son kullanımı yazılamadı", "error", err.Error())
		return
	}
	s.cache.markTouched(p.tokenKey, now)
}

// Refresh, oturumu döndürür: eskisi iptal edilir, yenisi verilir.
// Aynı token'la eşzamanlı ikinci yenileme UNAUTHORIZED alır.
func (s *Service) Refresh(ctx context.Context, p Principal) (Issued, error) {
	tok, err := NewToken()
	if err != nil {
		return Issued{}, err
	}
	expires := s.now().Add(SessionTTL).UTC().Truncate(time.Second)
	if _, err := s.store.rotate(ctx, p.SessionID, tok.Hash, expires); err != nil {
		if errors.Is(err, errAlreadyRotated) {
			s.cache.delete(p.tokenKey)
			return Issued{}, errUnauthorized()
		}
		return Issued{}, err
	}
	s.cache.delete(p.tokenKey)
	return Issued{Token: tok.Raw, ExpiresAt: expires, User: User{ID: p.UserID, Nickname: p.Nickname}}, nil
}

// Me, kullanıcının bilgisini ve aktif odasını döndürür.
func (s *Service) Me(ctx context.Context, p Principal) (Me, error) {
	room, err := s.store.activeRoom(ctx, p.UserID)
	if err != nil {
		return Me{}, err
	}
	return Me{User: User{ID: p.UserID, Nickname: p.Nickname}, ActiveRoomID: room}, nil
}

// ForgetUser, kullanıcının önbellekteki oturumlarını düşürür (hesap silme,
// takma ad değişikliği gibi durumlarda; SAFE-009).
func (s *Service) ForgetUser(userID string) { s.cache.deleteUser(userID) }
