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
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/config"
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

	srv := &http.Server{
		Handler: handlerFactory(),
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
	fmt.Fprintf(stdout, "sunucu başladı env=%s addr=%s\n", cfg.Env, ln.Addr())
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

	fmt.Fprintln(stdout, "kapanış başladı")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = srv.Close()
		return fmt.Errorf("kapanış zaman aşımı: %w", err)
	}
	<-serveErr
	fmt.Fprintln(stdout, "sunucu durdu")
	return nil
}

// handlerFactory, testlerde yavaş bir handler ile değiştirilebilsin diye
// değişkendir.
var handlerFactory = newHandler

// newHandler, HTTP yönlendiricisini döndürür. Rotalar, middleware ve standart
// hata biçimi BE-002 / BE-004'te; /health BE-005'te eklenecek.
func newHandler() http.Handler {
	return http.NewServeMux()
}
