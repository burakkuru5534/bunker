package room

import (
	"net/http"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
	"github.com/burakkuru5534/bunker/backend/internal/ratelimit"
)

// Davet kodu denemesi: kullanıcı başına dakikada 10 (security.md §5).
const (
	JoinLimit    = 10
	JoinPer      = time.Minute
	joinLimitMax = 100_000
)

// HTTP, oda uç noktalarıdır (api.md "Oda").
type HTTP struct {
	svc       *Service
	require   func(http.Handler) http.Handler
	joinLimit *ratelimit.Limiter
}

// NewHTTP: require, oturum doğrulayan middleware'dir (auth.HTTP.Require).
func NewHTTP(svc *Service, require func(http.Handler) http.Handler) *HTTP {
	return &HTTP{svc: svc, require: require, joinLimit: ratelimit.New(JoinLimit, JoinPer, joinLimitMax)}
}

// roomRoute, {roomId} içeren (odaya özgü) bir rotadır. Hepsi
// roomScopedRoutes'tadır: yalıtım testi (SAFE-003) bu tabloyu dolaşır, bu
// yüzden yeni oda rotası buraya eklenmelidir.
type roomRoute struct {
	pattern string
	handler func(*HTTP) httpapi.HandlerFunc
}

var roomScopedRoutes = []roomRoute{
	{"GET /v1/rooms/{roomId}", func(h *HTTP) httpapi.HandlerFunc { return h.get }},
	{"POST /v1/rooms/{roomId}/leave", func(h *HTTP) httpapi.HandlerFunc { return h.leave }},
	{"POST /v1/rooms/{roomId}/ready", func(h *HTTP) httpapi.HandlerFunc { return h.ready }},
	{"PATCH /v1/rooms/{roomId}/settings", func(h *HTTP) httpapi.HandlerFunc { return h.settings }},
	{"POST /v1/rooms/{roomId}/kick", func(h *HTTP) httpapi.HandlerFunc { return h.kick }},
	{"POST /v1/rooms/{roomId}/start", func(h *HTTP) httpapi.HandlerFunc { return h.start }},
}

// Register, rotaları ekler. Tüm oda rotaları oturum ister.
func (h *HTTP) Register(mux *http.ServeMux) {
	mux.Handle("POST /v1/rooms", h.require(httpapi.Handle(h.create)))
	mux.Handle("POST /v1/rooms/join", h.require(httpapi.Handle(h.join)))
	for _, rt := range roomScopedRoutes {
		mux.Handle(rt.pattern, h.require(httpapi.Handle(rt.handler(h))))
	}
}

// POST /v1/rooms/{roomId}/ready {"ready": true} → 204
func (h *HTTP) ready(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Ready *bool `json:"ready"`
	}
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		return err
	}
	if req.Ready == nil {
		return apperr.Field("ready", apperr.FieldRequired)
	}
	p, _ := auth.PrincipalFrom(r.Context())
	if err := h.svc.SetReady(r.Context(), p, r.PathValue("roomId"), *req.Ready); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// PATCH /v1/rooms/{roomId}/settings {"capacity": 1} → 200 RoomView
func (h *HTTP) settings(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Capacity *int `json:"capacity"`
	}
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		return err
	}
	if req.Capacity == nil {
		return apperr.Field("capacity", apperr.FieldRequired)
	}
	p, _ := auth.PrincipalFrom(r.Context())
	view, err := h.svc.ChangeCapacity(r.Context(), p, r.PathValue("roomId"), *req.Capacity)
	if err != nil {
		return err
	}
	httpapi.WriteJSON(w, http.StatusOK, view)
	return nil
}

// POST /v1/rooms/{roomId}/kick {"user_id": "uuid"} → 204
func (h *HTTP) kick(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		return err
	}
	p, _ := auth.PrincipalFrom(r.Context())
	if err := h.svc.Kick(r.Context(), p, r.PathValue("roomId"), req.UserID); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// POST /v1/rooms/{roomId}/leave → 204 (idempotent)
func (h *HTTP) leave(w http.ResponseWriter, r *http.Request) error {
	p, _ := auth.PrincipalFrom(r.Context())
	if err := h.svc.Leave(r.Context(), p, r.PathValue("roomId")); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// POST /v1/rooms/join {"invite_code": "K7M2QX"} → 200 RoomView
func (h *HTTP) join(w http.ResponseWriter, r *http.Request) error {
	p, _ := auth.PrincipalFrom(r.Context())
	// Sınır, kod denetiminden önce: kod tahmini yavaşlatılır.
	if ok, wait := h.joinLimit.Allow(p.UserID); !ok {
		return apperr.RateLimit(wait)
	}
	var req struct {
		InviteCode string `json:"invite_code"`
	}
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		return err
	}
	view, err := h.svc.Join(r.Context(), p, req.InviteCode)
	if err != nil {
		return err
	}
	httpapi.WriteJSON(w, http.StatusOK, view)
	return nil
}

// GET /v1/rooms/{roomId} → 200 RoomView (yalnızca aktif üyeler)
func (h *HTTP) get(w http.ResponseWriter, r *http.Request) error {
	p, _ := auth.PrincipalFrom(r.Context())
	view, err := h.svc.Get(r.Context(), p, r.PathValue("roomId"))
	if err != nil {
		return err
	}
	httpapi.WriteJSON(w, http.StatusOK, view)
	return nil
}

// POST /v1/rooms {"capacity": 2} (ops.) → 201 RoomView
func (h *HTTP) create(w http.ResponseWriter, r *http.Request) error {
	req := struct {
		Capacity *int `json:"capacity"`
	}{}
	if r.ContentLength != 0 {
		if err := httpapi.DecodeJSON(r, &req); err != nil {
			return err
		}
	}
	capacity := DefaultCapacity
	if req.Capacity != nil {
		capacity = *req.Capacity
	}
	p, _ := auth.PrincipalFrom(r.Context())
	view, err := h.svc.Create(r.Context(), p, capacity)
	if err != nil {
		return err
	}
	httpapi.WriteJSON(w, http.StatusCreated, view)
	return nil
}
