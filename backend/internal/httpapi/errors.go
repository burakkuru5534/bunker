package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/observability"
)

// statusByCode, apperr kodlarının HTTP durum kodlarıdır (docs/api.md §2).
var statusByCode = map[apperr.Code]int{
	apperr.ValidationFailed:      http.StatusBadRequest,
	apperr.NicknameInvalid:       http.StatusBadRequest,
	apperr.MessageInvalid:        http.StatusBadRequest,
	apperr.Unauthorized:          http.StatusUnauthorized,
	apperr.Forbidden:             http.StatusForbidden,
	apperr.NotAMember:            http.StatusForbidden,
	apperr.Kicked:                http.StatusForbidden,
	apperr.NotFound:              http.StatusNotFound,
	apperr.InviteCodeNotFound:    http.StatusNotFound,
	apperr.RoomFull:              http.StatusConflict,
	apperr.GameInProgress:        http.StatusConflict,
	apperr.NicknameTakenInRoom:   http.StatusConflict,
	apperr.NicknameTaken:         http.StatusConflict,
	apperr.AlreadyFriends:        http.StatusConflict,
	apperr.FriendLimit:           http.StatusConflict,
	apperr.AlreadyInRoom:         http.StatusConflict,
	apperr.InvalidPhase:          http.StatusConflict,
	apperr.NotYourTurn:           http.StatusConflict,
	apperr.CardAlreadyRevealed:   http.StatusConflict,
	apperr.CategoryNotAllowed:    http.StatusConflict,
	apperr.AlreadyVoted:          http.StatusConflict,
	apperr.SelfVote:              http.StatusConflict,
	apperr.InvalidTarget:         http.StatusConflict,
	apperr.NotAlive:              http.StatusConflict,
	apperr.NotSpectator:          http.StatusConflict,
	apperr.StartConditionsNotMet: http.StatusConflict,
	apperr.StaleState:            http.StatusConflict,
	apperr.UpgradeRequired:       http.StatusUpgradeRequired,
	apperr.RateLimited:           http.StatusTooManyRequests,
	apperr.Internal:              http.StatusInternalServerError,
}

// StatusOf, kodun HTTP durumunu döndürür; bilinmeyen kod 500'dür.
func StatusOf(code apperr.Code) int {
	if s, ok := statusByCode[code]; ok {
		return s
	}
	return http.StatusInternalServerError
}

// WriteAppError, herhangi bir hatayı standart hata gövdesine çevirir.
//
//   - *apperr.Error → kodu, alan hataları (details.fields) ve ayrıntılarıyla yazılır.
//     INTERNAL dışındaki kodlarda Message istemciye gider (hata ayıklama için).
//   - Diğer hatalar ve INTERNAL → 500, genel mesaj; asıl hata yalnızca loglanır.
//   - İstemci bağlantıyı kapattıysa (context.Canceled) yanıt yazılmaz.
//   - Süre aşımı (context.DeadlineExceeded) → 500 INTERNAL, WARNING olarak loglanır.
func WriteAppError(w http.ResponseWriter, r *http.Request, err error) {
	logger := observability.LoggerFrom(r.Context())

	if errors.Is(err, context.Canceled) && r.Context().Err() != nil {
		logger.Debug("istemci isteği iptal etti", "error", err.Error())
		return
	}

	ae, ok := errors.AsType[*apperr.Error](err)
	if !ok || ae.Code == apperr.Internal || StatusOf(ae.Code) == http.StatusInternalServerError {
		if errors.Is(err, context.DeadlineExceeded) {
			logger.Warn("istek süre sınırını aştı", "error", err.Error())
		} else {
			logger.Error("beklenmeyen hata", "error", err.Error())
		}
		WriteError(w, r, http.StatusInternalServerError, string(apperr.Internal), "beklenmeyen hata", nil)
		return
	}

	details := make(map[string]any, len(ae.Details)+1)
	for k, v := range ae.Details {
		details[k] = v
	}
	if len(ae.Fields) > 0 {
		details["fields"] = ae.Fields
	}
	if ae.Code == apperr.RateLimited {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(ae)))
	}
	if ae.Err != nil {
		logger.Debug("istek hatası", "code", string(ae.Code), "error", ae.Err.Error())
	}
	logDenied(r, ae.Code)
	WriteError(w, r, StatusOf(ae.Code), string(ae.Code), ae.Message, details)
}

// deniedEvents, reddedilen isteğin güvenlik olayıdır (SAFE-011). Oyun kuralı
// hataları (409) ve doğrulama (400) güvenlik olayı değildir.
var deniedEvents = map[apperr.Code]observability.SecurityEvent{
	apperr.Unauthorized: observability.EventAuthFailed,
	apperr.Forbidden:    observability.EventAccessDenied,
	apperr.NotAMember:   observability.EventAccessDenied,
	apperr.Kicked:       observability.EventAccessDenied,
	apperr.RateLimited:  observability.EventRateLimited,
}

// logDenied, reddedilen isteği güvenlik olayı olarak loglar: kod, yöntem ve
// rota kalıbı (kimlikler kalıpta yer tutucudur; yol yalnızca kalıp yoksa).
func logDenied(r *http.Request, code apperr.Code) {
	event, ok := deniedEvents[code]
	if !ok {
		return
	}
	route := r.Pattern
	if route == "" {
		route = r.URL.Path
	}
	observability.Security(r.Context(), nil, slog.LevelWarn, event, "istek reddedildi",
		slog.String("code", string(code)),
		slog.String("method", r.Method),
		slog.String("route", route),
	)
}

func retryAfterSeconds(e *apperr.Error) int {
	s := int(math.Ceil(e.RetryAfter.Seconds()))
	return max(s, 1)
}

// HandlerFunc, hata döndürebilen handler'dır. Hata WriteAppError ile yazılır.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// Handle, HandlerFunc'ı http.Handler'a çevirir.
func Handle(fn HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			WriteAppError(w, r, err)
		}
	})
}

// DecodeJSON, istek gövdesini dst'ye katı biçimde çözer:
// Content-Type JSON olmalı, bilinmeyen alan ve ikinci JSON değeri reddedilir.
// Tüm hatalar alan bilgisiyle VALIDATION_FAILED'dir.
func DecodeJSON(r *http.Request, dst any) error {
	if ct := r.Header.Get("Content-Type"); ct != "" {
		mt, _, err := mime.ParseMediaType(ct)
		if err != nil || mt != "application/json" {
			return apperr.Field("Content-Type", apperr.FieldInvalid)
		}
	}
	if r.Body == nil {
		return apperr.Field("body", apperr.FieldRequired)
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return jsonDecodeError(err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return jsonDecodeError(err)
		}
		return apperr.Field("body", apperr.FieldInvalid)
	}
	return nil
}

func jsonDecodeError(err error) error {
	if errors.Is(err, io.EOF) {
		return apperr.Field("body", apperr.FieldRequired)
	}
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return apperr.Field("body", apperr.FieldTooLarge)
	}
	if te, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
		field := te.Field
		if field == "" {
			field = "body"
		}
		return apperr.Field(field, apperr.FieldInvalidType)
	}
	// encoding/json bilinmeyen alanı yalnızca metinle bildirir: `json: unknown field "x"`.
	if name, ok := strings.CutPrefix(err.Error(), "json: unknown field "); ok {
		if unq, uerr := strconv.Unquote(name); uerr == nil {
			name = unq
		}
		return apperr.Field(name, apperr.FieldUnknown)
	}
	// Söz dizimi hatası, beklenmeyen EOF vb.
	e := apperr.Field("body", apperr.FieldInvalid)
	e.Err = err
	return e
}
