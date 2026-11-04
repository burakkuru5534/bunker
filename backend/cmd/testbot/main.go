// Command testbot, eksik oyuncuların yerine odaya otomatik oyuncular sokar
// (yerel geliştirme ve test ortamı için; üretimde kullanılmaz).
//
//	go run ./cmd/testbot -code K7M2QX            # 3 bot koda katılır
//	go run ./cmd/testbot -create                 # bot oda kurar, kodu yazar;
//	                                             # 4 oyuncu hazır olunca başlatır
//	go run ./cmd/testbot -code K7M2QX -bots 5 -delay 500ms
//
// Ctrl+C: botlar odadan ayrılır ve misafir hesaplarını siler.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/testbot"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "hata:", err)
		os.Exit(1)
	}
}

type options struct {
	api         *url.URL
	code        string
	create      bool
	bots        int
	startAt     int
	delay       time.Duration
	appVersion  string
	allowRemote bool
}

func parse(args []string) (options, error) {
	fs := flag.NewFlagSet("testbot", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var o options
	api := fs.String("api", "http://localhost:8080", "sunucu adresi")
	fs.StringVar(&o.code, "code", "", "katılınacak oda kodu")
	fs.BoolVar(&o.create, "create", false, "ilk bot oda kursun ve kodu yazsın")
	fs.IntVar(&o.bots, "bots", 3, "bot sayısı (1–9)")
	fs.IntVar(&o.startAt, "start-at", 4, "-create ile: bu kadar oyuncu hazır olunca başlat")
	fs.DurationVar(&o.delay, "delay", 1500*time.Millisecond, "her eylemden önce bekleme")
	fs.StringVar(&o.appVersion, "app-version", "0.1.0", "X-App-Version")
	fs.BoolVar(&o.allowRemote, "allow-remote", false, "yerel ağ dışındaki sunucuya izin ver (yalnızca test ortamı)")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	u, err := url.Parse(*api)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return o, fmt.Errorf("geçersiz -api: %q", *api)
	}
	o.api = u
	switch {
	case o.bots < 1 || o.bots > 9:
		return o, errors.New("-bots 1–9 olmalı")
	case o.create == (o.code != ""):
		return o, errors.New("ya -code ya da -create verin")
	case o.startAt < 4 || o.startAt > 10:
		return o, errors.New("-start-at 4–10 olmalı")
	case !o.allowRemote && !isLocal(u.Hostname()):
		return o, fmt.Errorf("%s yerel bir adres değil; test ortamı için -allow-remote ekleyin (üretimde kullanmayın)", u.Hostname())
	}
	o.code = strings.ToUpper(strings.TrimSpace(o.code))
	return o, nil
}

// isLocal: localhost, döngü, özel ağ (192.168.x, 10.x, 172.16–31.x) veya emülatör.
func isLocal(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate())
}

func run(ctx context.Context, args []string, stdout io.Writer) error {
	o, err := parse(args)
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewTextHandler(stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := testbot.Config{API: o.api, AppVersion: o.appVersion, Delay: o.delay, Logger: logger}

	var bots []*testbot.Bot
	defer func() {
		// Ctrl+C sonrası da temizlik yapılsın: ayrı, süreli bağlam.
		cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, b := range bots {
			b.Cleanup(cctx)
		}
		if len(bots) > 0 {
			fmt.Fprintf(stdout, "%d bot ayrıldı ve hesapları silindi.\n", len(bots))
		}
	}()

	code := o.code
	for i := range o.bots {
		b := testbot.New(cfg, fmt.Sprintf("Bot %d", i+1))
		if err := b.Register(ctx); err != nil {
			var apiErr *testbot.APIError
			if errors.As(err, &apiErr) && apiErr.Status == 429 {
				return fmt.Errorf("%s hesap açamadı: aynı ağdan dakikada 5 misafir hesap sınırı; 1 dk bekleyip tekrar deneyin", b.Nickname)
			}
			return fmt.Errorf("%s hesap açamadı (sunucu çalışıyor mu?): %w", b.Nickname, err)
		}
		bots = append(bots, b)
		if i == 0 && o.create {
			if code, err = b.Create(ctx, o.startAt); err != nil {
				return fmt.Errorf("oda kurulamadı: %w", err)
			}
			fmt.Fprintf(stdout, "Oda kodu: %s  (%d oyuncu hazır olunca oyun başlar)\n", code, o.startAt)
			continue
		}
		if err := b.Join(ctx, code); err != nil {
			return fmt.Errorf("%s katılamadı: %w", b.Nickname, err)
		}
	}
	fmt.Fprintf(stdout, "%d bot odada. Çıkmak için Ctrl+C.\n", len(bots))

	var wg sync.WaitGroup
	for _, b := range bots {
		wg.Go(func() {
			if err := b.Run(ctx); err != nil {
				logger.Warn("bot durdu", "bot", b.Nickname, "error", err)
			}
		})
	}
	wg.Wait()
	return nil
}
