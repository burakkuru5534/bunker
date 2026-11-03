package httpapi

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/burakkuru5534/bunker/backend/internal/observability"
)

func logLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log satırı çözülemedi: %v (%s)", err, line)
		}
		out = append(out, m)
	}
	return out
}

func accessLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	for _, m := range logLines(t, buf) {
		if m["message"] == "http istek" {
			return m
		}
	}
	t.Fatalf("erişim logu yok: %s", buf.String())
	return nil
}

func newLoggedRouter(t *testing.T, buf *bytes.Buffer, register func(*http.ServeMux)) http.Handler {
	t.Helper()
	return NewRouter(Options{
		Logger:         observability.NewLogger(buf, slog.LevelDebug, observability.FormatJSON),
		ProjectID:      "siginak-test",
		MinAppVersion:  testMin,
		RequestTimeout: 0,
		Register:       register,
	})
}

func TestAccessLogFields(t *testing.T) {
	var buf bytes.Buffer
	h := newLoggedRouter(t, &buf, func(mux *http.ServeMux) {
		mux.HandleFunc("POST /v1/x", func(w http.ResponseWriter, r *http.Request) {
			observability.LoggerFrom(r.Context()).Info("handler içi")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("12345"))
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/x?gizli=abc", strings.NewReader("{}"))
	req.Header.Set(AppVersionHeader, "0.2.0")
	req.Header.Set("Authorization", "Bearer çok-gizli")
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	m := accessLine(t, &buf)
	hr := m["httpRequest"].(map[string]any)
	if hr["requestMethod"] != "POST" || hr["requestUrl"] != "/v1/x" ||
		hr["status"] != float64(201) || hr["responseSize"] != float64(5) {
		t.Fatalf("httpRequest alanları yanlış: %v", hr)
	}
	if !strings.HasSuffix(hr["latency"].(string), "s") {
		t.Fatalf("latency Cloud Logging biçiminde olmalı: %v", hr["latency"])
	}
	if m["severity"] != "INFO" || m["request_id"] != rec.Header().Get(RequestIDHeader) {
		t.Fatalf("seviye / request_id yanlış: %v", m)
	}
	if m[observability.TraceKey] != "projects/siginak-test/traces/4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("izleme alanı yanlış: %v", m[observability.TraceKey])
	}

	out := buf.String()
	if strings.Contains(out, "gizli=abc") || strings.Contains(out, "çok-gizli") {
		t.Fatalf("sorgu dizesi veya Authorization loglanmamalı: %s", out)
	}

	// Handler içindeki log da aynı request_id'yi taşır.
	for _, line := range logLines(t, &buf) {
		if line["message"] == "handler içi" && line["request_id"] != m["request_id"] {
			t.Fatal("handler logu istek kimliğini taşımalı")
		}
	}
}

func TestAccessLogPanicIsError(t *testing.T) {
	var buf bytes.Buffer
	h := newLoggedRouter(t, &buf, func(mux *http.ServeMux) {
		mux.HandleFunc("GET /v1/p", func(http.ResponseWriter, *http.Request) { panic("x") })
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/p", nil)
	req.Header.Set(AppVersionHeader, "0.2.0")
	h.ServeHTTP(httptest.NewRecorder(), req)

	m := accessLine(t, &buf)
	if m["severity"] != "ERROR" || m["httpRequest"].(map[string]any)["status"] != float64(500) {
		t.Fatalf("panic sonrası 500 ERROR olarak loglanmalı: %v", m)
	}
	var panicLogged bool
	for _, line := range logLines(t, &buf) {
		if line["message"] == "panic" && line["request_id"] == m["request_id"] {
			panicLogged = true
		}
	}
	if !panicLogged {
		t.Fatal("panic logu istek kimliğiyle yazılmalı")
	}
}

func TestAccessLogDefaultStatusAndNoTraceWithoutProject(t *testing.T) {
	var buf bytes.Buffer
	h := NewRouter(Options{
		Logger:        observability.NewLogger(&buf, slog.LevelDebug, observability.FormatJSON),
		MinAppVersion: testMin,
		Register: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /health", func(http.ResponseWriter, *http.Request) {})
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	h.ServeHTTP(httptest.NewRecorder(), req)

	m := accessLine(t, &buf)
	if m["httpRequest"].(map[string]any)["status"] != float64(200) {
		t.Fatal("hiçbir şey yazılmazsa 200 sayılmalı")
	}
	if _, ok := m[observability.TraceKey]; ok {
		t.Fatal("proje kimliği yoksa izleme alanı yazılmamalı")
	}
}

// hijackable, Hijack destekleyen sahte yazıcı.
type hijackable struct {
	*httptest.ResponseRecorder
	hijacked bool
}

func (h *hijackable) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h.hijacked = true
	return nil, nil, nil
}

func TestStatusRecorderPassesThroughHijackAndFlush(t *testing.T) {
	under := &hijackable{ResponseRecorder: httptest.NewRecorder()}
	rec := &statusRecorder{ResponseWriter: under}

	if _, _, err := http.NewResponseController(rec).Hijack(); err != nil {
		t.Fatalf("Hijack iletilmeli: %v", err)
	}
	if !under.hijacked || rec.status != http.StatusSwitchingProtocols {
		t.Fatal("Hijack alttaki yazıcıya gitmeli ve 101 kaydedilmeli")
	}

	plain := &statusRecorder{ResponseWriter: httptest.NewRecorder()}
	if _, _, err := plain.Hijack(); err == nil {
		t.Fatal("desteklenmiyorsa hata dönmeli")
	}
	plain.Flush() // panik olmamalı
	if plain.Unwrap() == nil {
		t.Fatal("Unwrap alttaki yazıcıyı döndürmeli")
	}
}
