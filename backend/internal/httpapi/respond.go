package httpapi

import (
	"encoding/json"
	"net/http"
)

// Hata kodları (docs/api.md §2). Yalnızca HTTP katmanının ürettikleri burada;
// alan hataları BE-004'te eklenecek.
const (
	CodeValidationFailed = "VALIDATION_FAILED"
	CodeNotFound         = "NOT_FOUND"
	CodeUpgradeRequired  = "UPGRADE_REQUIRED"
	CodeInternal         = "INTERNAL"
)

// ErrorBody, standart hata gövdesidir:
//
//	{"error":{"code":"ROOM_FULL","message":"...","request_id":"...","details":{}}}
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail, hata gövdesinin içeriğidir.
type ErrorDetail struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	RequestID string         `json:"request_id"`
	Details   map[string]any `json:"details"`
}

// WriteJSON, değeri JSON olarak yazar.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError, standart hata gövdesini yazar. message yalnızca hata ayıklama
// içindir; istemci metni code'a göre seçer.
func WriteError(w http.ResponseWriter, r *http.Request, status int, code, message string, details map[string]any) {
	if details == nil {
		details = map[string]any{}
	}
	WriteJSON(w, status, ErrorBody{Error: ErrorDetail{
		Code:      code,
		Message:   message,
		RequestID: RequestIDFrom(r.Context()),
		Details:   details,
	}})
}
