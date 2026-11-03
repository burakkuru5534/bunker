package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
	"github.com/burakkuru5534/bunker/backend/internal/observability"
	"github.com/burakkuru5534/bunker/backend/internal/ratelimit"
)

// Misafir oturumu sınırı: IP başına dakikada 5 (security.md §5).
const (
	GuestLimit    = 5
	GuestPer      = time.Minute
	guestLimitMax = 100_000 // izlenen IP üst sınırı (bellek)
)

// HTTP, auth uç noktalarıdır (docs/api.md "Oturum ve kullanıcı").
type HTTP struct {
	svc         *Service
	guestLimit  *ratelimit.Limiter
	trustedHops int
}

// NewHTTP, uç noktaları kurar. trustedHops: config.TrustedProxyHops.
func NewHTTP(svc *Service, trustedHops int) *HTTP {
	return &HTTP{svc: svc, guestLimit: ratelimit.New(GuestLimit, GuestPer, guestLimitMax), trustedHops: trustedHops}
}

// Register, rotaları ekler.
func (h *HTTP) Register(mux *http.ServeMux) {
	mux.Handle("POST /v1/sessions/guest", httpapi.Handle(h.createGuest))
	mux.Handle("POST /v1/sessions/refresh", h.Require(httpapi.Handle(h.refresh)))
	mux.Handle("GET /v1/me", h.Require(httpapi.Handle(h.me)))
}

type principalKey struct{}

// PrincipalFrom, Require'ın bağlama koyduğu oturum sahibini döndürür.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}

// Require, Authorization: Bearer token'ını doğrular; geçersizse 401
// UNAUTHORIZED + WWW-Authenticate. Geçerliyse oturum sahibini ve user_id'li
// logger'ı bağlama koyar.
func (h *HTTP) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := BearerToken(r.Header.Get("Authorization"))
		p, err := h.svc.Authenticate(r.Context(), raw)
		if err != nil {
			if apperr.CodeOf(err) == apperr.Unauthorized {
				w.Header().Set("WWW-Authenticate", `Bearer realm="siginak"`)
			}
			httpapi.WriteAppError(w, r, err)
			return
		}
		ctx := context.WithValue(r.Context(), principalKey{}, p)
		ctx = observability.WithLogger(ctx, observability.LoggerFrom(ctx).With("user_id", p.UserID))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type userJSON struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
}

type tokenJSON struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      *userJSON `json:"user,omitempty"`
}

func writeToken(w http.ResponseWriter, status int, iss Issued, withUser bool) {
	// Token içeren yanıtlar hiçbir ara katmanda saklanmamalı.
	w.Header().Set("Cache-Control", "no-store")
	body := tokenJSON{Token: iss.Token, ExpiresAt: iss.ExpiresAt.UTC()}
	if withUser {
		body.User = &userJSON{ID: iss.User.ID, Nickname: iss.User.Nickname}
	}
	httpapi.WriteJSON(w, status, body)
}

// POST /v1/sessions/guest {"nickname": "…"} → 201
func (h *HTTP) createGuest(w http.ResponseWriter, r *http.Request) error {
	if ok, wait := h.guestLimit.Allow(httpapi.RateKey(httpapi.ClientIP(r, h.trustedHops))); !ok {
		return apperr.RateLimit(wait)
	}
	var req struct {
		Nickname string `json:"nickname"`
	}
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		return err
	}
	iss, err := h.svc.CreateGuest(r.Context(), req.Nickname)
	if err != nil {
		return err
	}
	observability.LoggerFrom(r.Context()).Info("misafir oturumu oluşturuldu", "user_id", iss.User.ID)
	writeToken(w, http.StatusCreated, iss, true)
	return nil
}

// POST /v1/sessions/refresh → 200 (gövde boş veya {}).
func (h *HTTP) refresh(w http.ResponseWriter, r *http.Request) error {
	if r.ContentLength != 0 {
		var empty struct{}
		if err := httpapi.DecodeJSON(r, &empty); err != nil {
			return err
		}
	}
	p, _ := PrincipalFrom(r.Context())
	iss, err := h.svc.Refresh(r.Context(), p)
	if err != nil {
		return err
	}
	writeToken(w, http.StatusOK, iss, false)
	return nil
}

// GET /v1/me → 200 {"id", "nickname", "active_room_id": uuid|null}
func (h *HTTP) me(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFrom(r.Context())
	me, err := h.svc.Me(r.Context(), p)
	if err != nil {
		return err
	}
	w.Header().Set("Cache-Control", "no-store")
	httpapi.WriteJSON(w, http.StatusOK, struct {
		ID           string  `json:"id"`
		Nickname     string  `json:"nickname"`
		ActiveRoomID *string `json:"active_room_id"`
	}{me.User.ID, me.User.Nickname, me.ActiveRoomID})
	return nil
}
