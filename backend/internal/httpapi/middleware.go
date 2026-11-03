package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
	"time"
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
func Recover(logger *slog.Logger) Middleware {
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
				logger.Error("panic",
					"request_id", RequestIDFrom(r.Context()),
					"method", r.Method,
					"path", r.URL.Path,
					"panic", fmt.Sprint(rec),
					"stack", string(debug.Stack()),
				)
				WriteError(w, r, http.StatusInternalServerError, CodeInternal, "beklenmeyen hata", nil)
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
func RequireAppVersion(min Version) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isAPIPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			raw := r.Header.Get(AppVersionHeader)
			v, err := ParseVersion(raw)
			if err != nil {
				WriteError(w, r, http.StatusBadRequest, CodeValidationFailed,
					"X-App-Version eksik veya geçersiz",
					map[string]any{"fields": map[string]string{"X-App-Version": "invalid"}})
				return
			}
			if v.Less(min) {
				WriteError(w, r, http.StatusUpgradeRequired, CodeUpgradeRequired,
					"uygulama sürümü desteklenmiyor",
					map[string]any{"min_version": min.String()})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
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
				WriteError(w, r, http.StatusBadRequest, CodeValidationFailed,
					"istek gövdesi çok büyük",
					map[string]any{"max_bytes": maxBytes})
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
