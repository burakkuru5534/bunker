package room

import (
	"net/http"

	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
)

// HTTP, oda uç noktalarıdır (api.md "Oda").
type HTTP struct {
	svc     *Service
	require func(http.Handler) http.Handler
}

// NewHTTP: require, oturum doğrulayan middleware'dir (auth.HTTP.Require).
func NewHTTP(svc *Service, require func(http.Handler) http.Handler) *HTTP {
	return &HTTP{svc: svc, require: require}
}

// Register, rotaları ekler.
func (h *HTTP) Register(mux *http.ServeMux) {
	mux.Handle("POST /v1/rooms", h.require(httpapi.Handle(h.create)))
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
