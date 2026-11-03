package httpapi

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/observability"
)

func healthRouter(h *Health, logs *bytes.Buffer) http.Handler {
	return NewRouter(Options{
		Logger:         observability.NewLogger(logs, slog.LevelDebug, observability.FormatJSON),
		MinAppVersion:  testMin,
		RequestTimeout: time.Second,
		Register:       h.Register,
	})
}

func get(h http.Handler, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestHealthLiveIgnoresChecksAndVersion(t *testing.T) {
	var logs bytes.Buffer
	h := healthRouter(&Health{Checks: []ReadinessCheck{{Name: "database", Check: func(context.Context) error {
		t.Fatal("canlılık denetimi bağımlılık çalıştırmamalı")
		return nil
	}}}}, &logs)

	rec := get(h, http.MethodGet, "/health") // X-App-Version yok
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("/health: %d %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("başlıklar: %v", rec.Header())
	}
	if !strings.Contains(logs.String(), `"severity":"DEBUG","message":"http istek"`) {
		t.Fatalf("başarılı sağlık isteği DEBUG loglanmalı: %s", logs.String())
	}
}

func TestHealthHead(t *testing.T) {
	h := healthRouter(&Health{}, &bytes.Buffer{})
	for _, p := range []string{"/health", "/health/ready"} {
		if rec := get(h, http.MethodHead, p); rec.Code != 200 {
			t.Errorf("HEAD %s: %d", p, rec.Code)
		}
	}
	if rec := get(h, http.MethodPost, "/health"); rec.Code != 404 {
		t.Errorf("POST /health 404 olmalı: %d", rec.Code)
	}
}

func TestReadyAllPass(t *testing.T) {
	var calls atomic.Int32
	ok := func(context.Context) error { calls.Add(1); return nil }
	h := healthRouter(&Health{Checks: []ReadinessCheck{{"a", ok}, {"b", ok}}}, &bytes.Buffer{})

	rec := get(h, http.MethodGet, "/health/ready")
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != `{"status":"ready"}` || calls.Load() != 2 {
		t.Fatalf("hazır: %d %s (çağrı %d)", rec.Code, rec.Body.String(), calls.Load())
	}
	if get(healthRouter(&Health{}, &bytes.Buffer{}), http.MethodGet, "/health/ready").Code != 200 {
		t.Fatal("denetim yoksa hazır sayılmalı")
	}
}

func TestReadyFailureHidesDetails(t *testing.T) {
	var logs bytes.Buffer
	h := healthRouter(&Health{Checks: []ReadinessCheck{
		{"cache", func(context.Context) error { return nil }},
		{"database", func(context.Context) error { return errors.New("dial tcp 10.0.0.5:5432: password=gizli") }},
	}}, &logs)

	rec := get(h, http.MethodGet, "/health/ready")
	if rec.Code != http.StatusServiceUnavailable || strings.TrimSpace(rec.Body.String()) != `{"status":"not_ready"}` {
		t.Fatalf("hazır değil: %d %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("no-store olmalı")
	}
	out := logs.String()
	if !strings.Contains(out, `"check":"database"`) || !strings.Contains(out, `"failed_checks":"database"`) {
		t.Fatalf("başarısız denetim loglanmalı: %s", out)
	}
	if !strings.Contains(out, `"severity":"WARNING","message":"http istek"`) || strings.Contains(out, `"severity":"ERROR"`) {
		t.Fatalf("503 hazırlık yanıtı WARNING olmalı: %s", out)
	}
}

func TestReadyTimeoutAndPanic(t *testing.T) {
	var logs bytes.Buffer
	block := make(chan struct{})
	defer close(block)
	h := healthRouter(&Health{Timeout: 50 * time.Millisecond, Checks: []ReadinessCheck{
		// Bağlamı yok sayan, takılan denetim.
		{"takili", func(context.Context) error { <-block; return nil }},
		{"panik", func(context.Context) error { panic("bum") }},
		{"yavas", func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }},
	}}, &logs)

	start := time.Now()
	rec := get(h, http.MethodGet, "/health/ready")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("503 bekleniyordu: %d", rec.Code)
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("denetimler paralel ve süre sınırlı olmalı, %v sürdü", d)
	}
	out := logs.String()
	for _, want := range []string{`"failed_checks":"takili,panik,yavas"`, "denetim panic: bum", "context deadline exceeded"} {
		if !strings.Contains(out, want) {
			t.Errorf("logda %q yok: %s", want, out)
		}
	}
}

func TestReadyDefaultTimeout(t *testing.T) {
	var got time.Duration
	h := &Health{Checks: []ReadinessCheck{{"x", func(ctx context.Context) error {
		dl, _ := ctx.Deadline()
		got = time.Until(dl)
		return nil
	}}}}
	if failed := h.run(context.Background()); len(failed) != 0 {
		t.Fatal(failed)
	}
	if got <= DefaultReadinessTimeout-time.Second || got > DefaultReadinessTimeout {
		t.Fatalf("varsayılan süre %v civarı olmalı: %v", DefaultReadinessTimeout, got)
	}
}

func TestAccessLogLevel(t *testing.T) {
	cases := []struct {
		path   string
		status int
		want   slog.Level
	}{
		{"/health", 200, slog.LevelDebug},
		{"/health/ready", 503, slog.LevelWarn},
		{"/healthz", 200, slog.LevelInfo},
		{"/v1/x", 404, slog.LevelInfo},
		{"/v1/x", 500, slog.LevelError},
	}
	for _, c := range cases {
		if got := accessLogLevel(c.path, c.status); got != c.want {
			t.Errorf("%s %d → %v, %v bekleniyordu", c.path, c.status, got, c.want)
		}
	}
}
