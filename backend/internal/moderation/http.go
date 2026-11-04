package moderation

import (
	"net/http"

	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
)

// HTTP, moderasyon uç noktalarıdır (api.md "Moderasyon").
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
	mux.Handle("POST /v1/reports", h.require(httpapi.Handle(h.report)))
	mux.Handle("GET /v1/blocks", h.require(httpapi.Handle(h.blocks)))
	mux.Handle("POST /v1/blocks", h.require(httpapi.Handle(h.block)))
	mux.Handle("DELETE /v1/blocks/{userId}", h.require(httpapi.Handle(h.unblock)))
}

// POST /v1/reports → 201 {"id"}
func (h *HTTP) report(w http.ResponseWriter, r *http.Request) error {
	var req ReportRequest
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		return err
	}
	p, _ := auth.PrincipalFrom(r.Context())
	id, err := h.svc.Report(r.Context(), p, req)
	if err != nil {
		return err
	}
	httpapi.WriteJSON(w, http.StatusCreated, map[string]string{"id": id})
	return nil
}

// GET /v1/blocks → 200 [ {user_id, nickname, created_at} ]
func (h *HTTP) blocks(w http.ResponseWriter, r *http.Request) error {
	p, _ := auth.PrincipalFrom(r.Context())
	list, err := h.svc.Blocks(r.Context(), p)
	if err != nil {
		return err
	}
	httpapi.WriteJSON(w, http.StatusOK, list)
	return nil
}

// POST /v1/blocks {"user_id"} → 204
func (h *HTTP) block(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		return err
	}
	p, _ := auth.PrincipalFrom(r.Context())
	if err := h.svc.Block(r.Context(), p, req.UserID); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// DELETE /v1/blocks/{userId} → 204 (idempotent)
func (h *HTTP) unblock(w http.ResponseWriter, r *http.Request) error {
	p, _ := auth.PrincipalFrom(r.Context())
	if err := h.svc.Unblock(r.Context(), p, r.PathValue("userId")); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
