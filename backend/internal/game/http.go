package game

import (
	"net/http"

	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
)

// HTTP, oyun uç noktalarıdır.
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
	mux.Handle("POST /v1/games/{gameId}/actions", h.require(httpapi.Handle(h.act)))
	mux.Handle("GET /v1/games/{gameId}/result", h.require(httpapi.Handle(h.result)))
}

// POST /v1/games/{gameId}/actions {"action_id","type","payload"} → 200 {"state_version","duplicate"}
func (h *HTTP) act(w http.ResponseWriter, r *http.Request) error {
	var req ActionRequest
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		return err
	}
	p, _ := auth.PrincipalFrom(r.Context())
	res, err := h.svc.Act(r.Context(), p, r.PathValue("gameId"), req)
	if err != nil {
		return err
	}
	httpapi.WriteJSON(w, http.StatusOK, res)
	return nil
}

// GET /v1/games/{gameId}/result → 200 Result
func (h *HTTP) result(w http.ResponseWriter, r *http.Request) error {
	p, _ := auth.PrincipalFrom(r.Context())
	res, err := h.svc.Result(r.Context(), p, r.PathValue("gameId"))
	if err != nil {
		return err
	}
	httpapi.WriteJSON(w, http.StatusOK, res)
	return nil
}
