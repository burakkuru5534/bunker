// Command server, Sığınak backend servisidir (REST + WebSocket).
//
// Yaşam döngüsü:
//  1. Yapılandırma ortam değişkenlerinden okunur (internal/config).
//  2. HTTP sunucusu PORT üzerinde dinlemeye başlar.
//  3. SIGINT / SIGTERM gelince yeni bağlantı kabul edilmez, açık istekler
//     SHUTDOWN_TIMEOUT süresince beklenir, sonra süreç çıkar
//     (Cloud Run SIGTERM'den ~10 sn sonra süreci sonlandırır; ADR-0011).
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/config"
	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
	"github.com/burakkuru5534/bunker/backend/internal/observability"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Getenv, os.Stdout, nil); err != nil {
		fmt.Fprintf(os.Stderr, "sunucu hatası: %v\n", err)
		os.Exit(1)
	}
}

// run, servisi ctx iptal edilene kadar çalıştırır. Test edilebilirlik için
// ortam okuyucu, çıktı ve "dinlemeye başladı" geri çağrısı dışarıdan verilir.
func run(
	ctx context.Context,
	getenv func(string) string,
	stdout io.Writer,
	onListening func(addr net.Addr),
) error {
	cfg, err := config.Load(getenv)
	if err != nil {
		return fmt.Errorf("yapılandırma: %w", err)
	}

	logger := newLogger(cfg, stdout)
	slog.SetDefault(logger)

	srv := &http.Server{
		Handler:  handlerFactory(cfg, logger),
		ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
		// Başlıklar için üst sınır (Slowloris'e karşı). WebSocket bağlantıları
		// uzun yaşadığından ReadTimeout / WriteTimeout bilinçli olarak yok;
		// istek gövdesi sınırları ve zaman aşımları BE-002 middleware'inde.
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}

	ln, err := net.Listen("tcp", cfg.Addr())
	if err != nil {
		return fmt.Errorf("dinleme (%s): %w", cfg.Addr(), err)
	}
	logger.Info("sunucu başladı", "env", string(cfg.Env), "addr", ln.Addr().String())
	if onListening != nil {
		onListening(ln.Addr())
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("sunucu: %w", err)
	case <-ctx.Done():
	}

	logger.Info("kapanış başladı", "timeout", cfg.ShutdownTimeout.String())
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = srv.Close()
		return fmt.Errorf("kapanış zaman aşımı: %w", err)
	}
	<-serveErr
	logger.Info("sunucu durdu")
	return nil
}

// newLogger, yapılandırmaya göre logger kurar (yerelde metin, bulutta JSON).
func newLogger(cfg config.Config, w io.Writer) *slog.Logger {
	level, _ := observability.ParseLevel(cfg.LogLevel) // config doğruladı
	format := observability.FormatText
	if cfg.LogJSON() {
		format = observability.FormatJSON
	}
	return observability.NewLogger(w, level, format)
}

// handlerFactory, testlerde yavaş bir handler ile değiştirilebilsin diye
// değişkendir.
var handlerFactory = newHandler

// newHandler, middleware zinciriyle sarılmış yönlendiriciyi döndürür.
// Modül rotaları ve /health sonraki görevlerde Register içine eklenecek.
func newHandler(cfg config.Config, logger *slog.Logger) http.Handler {
	minVersion, err := httpapi.ParseVersion(cfg.MinAppVersion)
	if err != nil {
		// config.Load biçimi zaten doğruladı; buraya gelinmemeli.
		panic(err)
	}
	return httpapi.NewRouter(httpapi.Options{
		Logger:         logger,
		ProjectID:      cfg.GCPProjectID,
		MinAppVersion:  minVersion,
		RequestTimeout: cfg.RequestTimeout,
	})
}
