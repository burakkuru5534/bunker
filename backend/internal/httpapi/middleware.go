package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/observability"
)

// Middleware, bir handler'ı sarar.
type Middleware func(http.Handler) http.Handler

// Chain, middleware'leri sırayla uygular: Chain(h, a, b) → a(b(h)).
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

type ctxKey int

const requestIDKey ctxKey = iota

// RequestIDHeader, istek kimliği başlığı.
const RequestIDHeader = "X-Request-ID"

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`)

// RequestIDFrom, bağlamdaki istek kimliğini döndürür (yoksa "").
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// RequestID, her isteğe bir kimlik verir. İstemci geçerli bir X-Request-ID
// gönderdiyse o kullanılır; yoksa rastgele üretilir. Yanıta da eklenir.
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(RequestIDHeader)
			if !validRequestID.MatchString(id) {
				id = newRequestID()
			}
			w.Header().Set(RequestIDHeader, id)
			ctx := context.WithValue(r.Context(), requestIDKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Recover, handler'daki panic'i yakalar, loglar ve 500 INTERNAL döner.
// İstemciye ayrıntı gösterilmez. http.ErrAbortHandler yeniden fırlatılır.
// Log, istek kapsamlı logger'a (request_id, trace) yazılır.
func Recover() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(rec)
				}
				observability.LoggerFrom(r.Context()).Error("panic",
					"method", r.Method,
					"path", r.URL.Path,
					"panic", fmt.Sprint(rec),
					"stack", string(debug.Stack()),
				)
				WriteError(w, r, http.StatusInternalServerError, string(apperr.Internal), "beklenmeyen hata", nil)
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// AppVersionHeader, istemci sürüm başlığı.
const AppVersionHeader = "X-App-Version"

// RequireAppVersion, /v1 altındaki isteklerde X-App-Version başlığını denetler.
//   - Eksik veya biçimsiz → 400 VALIDATION_FAILED
//   - min'den eski → 426 UPGRADE_REQUIRED
//
// /v1/ws hariç: WebSocket'te sürüm hatası yükseltmeden sonra 4026 kapanış
// koduyla bildirilir (docs/api.md §5).
func RequireAppVersion(min Version) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// WebSocket: istemci HTTP hata durumunu okuyamaz; sürüm WS
			// işleyicisinde CheckAppVersion ile denetlenir ve 4026 ile kapatılır.
			if !isAPIPath(r.URL.Path) || isWebSocketPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			if err := CheckAppVersion(r, min); err != nil {
				WriteAppError(w, r, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CheckAppVersion, X-App-Version başlığını denetler: eksik/biçimsiz →
// VALIDATION_FAILED, min'den eski → UPGRADE_REQUIRED (details.min_version).
func CheckAppVersion(r *http.Request, min Version) error {
	v, err := ParseVersion(r.Header.Get(AppVersionHeader))
	if err != nil {
		e := apperr.Field(AppVersionHeader, apperr.FieldInvalid)
		e.Message = "X-App-Version eksik veya geçersiz"
		return e
	}
	if v.Less(min) {
		return apperr.New(apperr.UpgradeRequired, "uygulama sürümü desteklenmiyor").
			WithDetail("min_version", min.String())
	}
	return nil
}

// LimitBody, /v1 REST isteklerinin gövdesini sınırlar (docs/api.md §1: 4 KB).
// Content-Length sınırı aşıyorsa istek hemen reddedilir; bildirilmemişse
// okuma sırasında http.MaxBytesReader devreye girer.
func LimitBody(maxBytes int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isAPIPath(r.URL.Path) || isWebSocketPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			if r.ContentLength > maxBytes {
				e := apperr.Field("body", apperr.FieldTooLarge).WithDetail("max_bytes", maxBytes)
				e.Message = "istek gövdesi çok büyük"
				WriteAppError(w, r, e)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// Timeout, /v1 REST isteklerinin bağlamına süre sınırı koyar. Handler'lar ve
// veritabanı sorguları bu bağlama uyar. WebSocket bağlantıları hariçtir
// (uzun yaşarlar).
func Timeout(d time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isAPIPath(r.URL.Path) || isWebSocketPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func isAPIPath(p string) bool       { return p == "/v1" || strings.HasPrefix(p, "/v1/") }
func isWebSocketPath(p string) bool { return p == "/v1/ws" }
