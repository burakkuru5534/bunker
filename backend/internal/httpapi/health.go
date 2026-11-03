package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/observability"
)

// DefaultReadinessTimeout, her hazırlık denetimi için üst süredir.
const DefaultReadinessTimeout = 2 * time.Second

// ReadinessCheck, /health/ready sırasında çalışan bir bağımlılık denetimidir
// (ör. DB-001'de veritabanı ping'i).
type ReadinessCheck struct {
	Name  string
	Check func(ctx context.Context) error
}

// Health, sağlık uç noktalarını sunar (docs/api.md §4 "Sağlık"):
//
//   - GET /health        → 200 {"status":"ok"}; yalnızca süreç ayakta mı.
//     Bağımlılıklara bakmaz; Cloud Run başlangıç/canlılık denetimi budur
//     (DB kesintisi konteyneri yeniden başlatmamalı).
//   - GET /health/ready  → 200 {"status":"ready"} / 503 {"status":"not_ready"};
//     denetimler paralel, her biri Timeout ile sınırlı. Hangi denetimin
//     başarısız olduğu yalnızca loglanır, yanıtta yer almaz.
//
// HEAD de desteklenir (ServeMux GET kalıbı HEAD'i kapsar).
type Health struct {
	Checks  []ReadinessCheck
	Timeout time.Duration
}

// Register, uç noktaları mux'a ekler.
func (h *Health) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.live)
	mux.HandleFunc("GET /health/ready", h.ready)
}

func (h *Health) live(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Health) ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if failed := h.run(r.Context()); len(failed) > 0 {
		observability.LoggerFrom(r.Context()).Warn("hazır değil", "failed_checks", strings.Join(failed, ","))
		WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// run, denetimleri paralel çalıştırır ve başarısız olanların adlarını
// (tanım sırasıyla) döndürür.
func (h *Health) run(ctx context.Context) []string {
	timeout := h.Timeout
	if timeout <= 0 {
		timeout = DefaultReadinessTimeout
	}
	logger := observability.LoggerFrom(ctx)
	errs := make([]error, len(h.Checks))

	var wg sync.WaitGroup
	for i, c := range h.Checks {
		wg.Go(func() {
			cctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			errs[i] = safeCheck(cctx, c.Check)
		})
	}
	wg.Wait()

	var failed []string
	for i, err := range errs {
		if err != nil {
			failed = append(failed, h.Checks[i].Name)
			logger.Warn("hazırlık denetimi başarısız", "check", h.Checks[i].Name, "error", err.Error())
		}
	}
	return failed
}

// safeCheck, denetimi çalıştırır; panic'i ve süre aşımını hataya çevirir.
// Denetim bağlamı yok saysa bile en geç bağlam bitince döner.
func safeCheck(ctx context.Context, check func(context.Context) error) (err error) {
	done := make(chan error, 1)
	go func() {
		defer func() {
			if p := recover(); p != nil {
				done <- panicError{p}
			}
		}()
		done <- check(ctx)
	}()
	select {
	case err = <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

type panicError struct{ v any }

func (p panicError) Error() string { return fmt.Sprint("denetim panic: ", p.v) }

// isHealthPath, sağlık uç noktası mı (erişim logunda gürültüyü azaltmak için).
func isHealthPath(p string) bool {
	return p == "/health" || strings.HasPrefix(p, "/health/")
}
