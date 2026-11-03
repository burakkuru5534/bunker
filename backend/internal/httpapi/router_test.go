package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
)

var testMin = Version{0, 2, 0}

// newTestRouter, test rotalarıyla bir yönlendirici kurar.
func newTestRouter(t *testing.T, logs *bytes.Buffer) http.Handler {
	t.Helper()
	if logs == nil {
		logs = &bytes.Buffer{}
	}
	return NewRouter(Options{
		Logger:         slog.New(slog.NewJSONHandler(logs, nil)),
		MinAppVersion:  testMin,
		RequestTimeout: 2 * time.Second,
		Register: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /v1/echo", func(w http.ResponseWriter, r *http.Request) {
				_, hasDeadline := r.Context().Deadline()
				WriteJSON(w, http.StatusOK, map[string]any{
					"request_id":   RequestIDFrom(r.Context()),
					"has_deadline": hasDeadline,
				})
			})
			mux.HandleFunc("POST /v1/echo", func(w http.ResponseWriter, r *http.Request) {
				b, err := io.ReadAll(r.Body)
				if err != nil {
					var mbe *http.MaxBytesError
					if errors.As(err, &mbe) {
						WriteError(w, r, http.StatusBadRequest, string(apperr.ValidationFailed), "büyük", nil)
						return
					}
					t.Errorf("okuma hatası: %v", err)
				}
				WriteJSON(w, http.StatusOK, map[string]any{"len": len(b)})
			})
			mux.HandleFunc("GET /v1/ws", func(w http.ResponseWriter, r *http.Request) {
				_, hasDeadline := r.Context().Deadline()
				WriteJSON(w, http.StatusOK, map[string]any{"has_deadline": hasDeadline})
			})
			mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
				_, hasDeadline := r.Context().Deadline()
				WriteJSON(w, http.StatusOK, map[string]any{"has_deadline": hasDeadline})
			})
			mux.HandleFunc("GET /v1/panic", func(http.ResponseWriter, *http.Request) {
				panic("patladı")
			})
			mux.HandleFunc("GET /v1/abort", func(http.ResponseWriter, *http.Request) {
				panic(http.ErrAbortHandler)
			})
		},
	})
}

func do(h http.Handler, method, path string, body io.Reader, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set(AppVersionHeader, "0.2.0")
	for k, v := range headers {
		if v == "" {
			req.Header.Del(k)
		} else {
			req.Header.Set(k, v)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) ErrorDetail {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q, JSON bekleniyordu", ct)
	}
	var body ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("hata gövdesi çözülemedi: %v (%s)", err, rec.Body.String())
	}
	if body.Error.Details == nil {
		t.Fatal("details her zaman nesne olmalı")
	}
	return body.Error
}

func decodeMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("JSON çözülemedi: %v", err)
	}
	return m
}

func TestUnknownPathReturnsStandardNotFound(t *testing.T) {
	h := newTestRouter(t, nil)
	rec := do(h, http.MethodGet, "/v1/yok", nil, nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("durum = %d", rec.Code)
	}
	e := decodeError(t, rec)
	if e.Code != string(apperr.NotFound) {
		t.Fatalf("code = %q", e.Code)
	}
	if e.RequestID == "" || e.RequestID != rec.Header().Get(RequestIDHeader) {
		t.Fatalf("request_id gövde ve başlıkta aynı olmalı: %q / %q", e.RequestID, rec.Header().Get(RequestIDHeader))
	}
}

func TestWrongMethodReturnsJSONNotPlainText(t *testing.T) {
	h := newTestRouter(t, nil)
	rec := do(h, http.MethodDelete, "/v1/echo", nil, nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("durum = %d, 404 bekleniyordu", rec.Code)
	}
	if decodeError(t, rec).Code != string(apperr.NotFound) {
		t.Fatal("NOT_FOUND bekleniyordu")
	}
}

func TestRequestID(t *testing.T) {
	h := newTestRouter(t, nil)
	hex32 := regexp.MustCompile(`^[0-9a-f]{32}$`)

	rec := do(h, http.MethodGet, "/v1/echo", nil, nil)
	id := rec.Header().Get(RequestIDHeader)
	if !hex32.MatchString(id) {
		t.Fatalf("üretilen kimlik biçimi yanlış: %q", id)
	}
	if decodeMap(t, rec)["request_id"] != id {
		t.Fatal("handler bağlamdaki kimliği görmeli")
	}

	rec = do(h, http.MethodGet, "/v1/echo", nil, map[string]string{RequestIDHeader: "client-abc-123"})
	if rec.Header().Get(RequestIDHeader) != "client-abc-123" {
		t.Fatal("geçerli istemci kimliği korunmalı")
	}

	rec = do(h, http.MethodGet, "/v1/echo", nil, map[string]string{RequestIDHeader: "kötü kimlik\n<script>"})
	if !hex32.MatchString(rec.Header().Get(RequestIDHeader)) {
		t.Fatal("geçersiz istemci kimliği yerine yenisi üretilmeli")
	}
}

func TestRecoverPanic(t *testing.T) {
	var logs bytes.Buffer
	h := newTestRouter(t, &logs)

	rec := do(h, http.MethodGet, "/v1/panic", nil, nil)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("durum = %d", rec.Code)
	}
	e := decodeError(t, rec)
	if e.Code != string(apperr.Internal) || strings.Contains(e.Message, "patladı") {
		t.Fatalf("istemciye ayrıntı sızmamalı: %+v", e)
	}
	if !strings.Contains(logs.String(), "patladı") || !strings.Contains(logs.String(), e.RequestID) {
		t.Fatalf("panic request_id ile loglanmalı: %s", logs.String())
	}
}

func TestRecoverRepanicsAbortHandler(t *testing.T) {
	h := newTestRouter(t, nil)
	defer func() {
		if rec := recover(); rec != http.ErrAbortHandler {
			t.Fatalf("ErrAbortHandler yeniden fırlatılmalı, alınan: %v", rec)
		}
	}()
	do(h, http.MethodGet, "/v1/abort", nil, nil)
}

func TestAppVersion(t *testing.T) {
	h := newTestRouter(t, nil)

	tests := []struct {
		name    string
		version string
		status  int
		code    string
	}{
		{"eksik", "", http.StatusBadRequest, string(apperr.ValidationFailed)},
		{"biçimsiz", "v1", http.StatusBadRequest, string(apperr.ValidationFailed)},
		{"eski", "0.1.9", http.StatusUpgradeRequired, string(apperr.UpgradeRequired)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(h, http.MethodGet, "/v1/echo", nil, map[string]string{AppVersionHeader: tc.version})
			if rec.Code != tc.status || decodeError(t, rec).Code != tc.code {
				t.Fatalf("durum %d / kod beklenen %d %s: %s", rec.Code, tc.status, tc.code, rec.Body.String())
			}
		})
	}

	for _, ok := range []string{"0.2.0", "0.10.0", "1.0.0"} {
		if rec := do(h, http.MethodGet, "/v1/echo", nil, map[string]string{AppVersionHeader: ok}); rec.Code != http.StatusOK {
			t.Fatalf("%s kabul edilmeli, durum %d", ok, rec.Code)
		}
	}

	if rec := do(h, http.MethodGet, "/health", nil, map[string]string{AppVersionHeader: ""}); rec.Code != http.StatusOK {
		t.Fatalf("/health sürüm denetimi dışında olmalı, durum %d", rec.Code)
	}
}

func TestUpgradeRequiredIncludesMinVersion(t *testing.T) {
	h := newTestRouter(t, nil)
	rec := do(h, http.MethodGet, "/v1/echo", nil, map[string]string{AppVersionHeader: "0.0.1"})

	if decodeError(t, rec).Details["min_version"] != "0.2.0" {
		t.Fatalf("min_version detayı bekleniyordu: %s", rec.Body.String())
	}
}

func TestBodyLimit(t *testing.T) {
	h := newTestRouter(t, nil)

	ok := do(h, http.MethodPost, "/v1/echo", strings.NewReader(strings.Repeat("a", MaxBodyBytes)), nil)
	if ok.Code != http.StatusOK {
		t.Fatalf("4 KB kabul edilmeli: %d", ok.Code)
	}

	big := do(h, http.MethodPost, "/v1/echo", strings.NewReader(strings.Repeat("a", MaxBodyBytes+1)), nil)
	if big.Code != http.StatusBadRequest || decodeError(t, big).Code != string(apperr.ValidationFailed) {
		t.Fatalf("Content-Length sınırı aşınca reddedilmeli: %d", big.Code)
	}

	// Uzunluğu bildirilmeyen (chunked) gövde okuma sırasında kesilir.
	req := httptest.NewRequest(http.MethodPost, "/v1/echo", io.NopCloser(strings.NewReader(strings.Repeat("a", MaxBodyBytes*2))))
	req.ContentLength = -1
	req.Header.Set(AppVersionHeader, "0.2.0")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bildirilmemiş uzunlukta da sınır uygulanmalı: %d", rec.Code)
	}
}

func TestTimeoutAppliesToRESTOnly(t *testing.T) {
	h := newTestRouter(t, nil)

	if decodeMap(t, do(h, http.MethodGet, "/v1/echo", nil, nil))["has_deadline"] != true {
		t.Fatal("REST isteğinde süre sınırı olmalı")
	}
	if decodeMap(t, do(h, http.MethodGet, "/v1/ws", nil, nil))["has_deadline"] != false {
		t.Fatal("WebSocket yolunda süre sınırı olmamalı")
	}
	if decodeMap(t, do(h, http.MethodGet, "/health", nil, nil))["has_deadline"] != false {
		t.Fatal("/v1 dışında süre sınırı olmamalı")
	}
}

func TestJSONResponseHeaders(t *testing.T) {
	rec := do(newTestRouter(t, nil), http.MethodGet, "/v1/echo", nil, nil)
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("nosniff başlığı bekleniyordu")
	}
}

func TestChainOrder(t *testing.T) {
	var order []string
	mw := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	h := Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { order = append(order, "h") }), mw("a"), mw("b"))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if strings.Join(order, ",") != "a,b,h" {
		t.Fatalf("sıra = %v", order)
	}
}
