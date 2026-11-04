// Command server, Sığınak backend servisidir (REST + WebSocket).
//
// Yaşam döngüsü:
//  1. Yapılandırma ortam değişkenlerinden okunur (internal/config). Yerelde
//     (APP_ENV boş veya local) çalışma klasöründeki .env de okunur; gerçek
//     ortam değişkenleri önceliklidir.
//  2. DATABASE_URL varsa bağlantı havuzu kurulur ve doğrulanır; başarısızsa
//     servis başlamaz (yerelde DATABASE_URL yoksa veritabanısız çalışır).
//  3. HTTP sunucusu PORT üzerinde dinlemeye başlar.
//  4. SIGINT / SIGTERM gelince yeni bağlantı kabul edilmez, açık istekler
//     SHUTDOWN_TIMEOUT süresince beklenir, havuz kapatılır, süreç çıkar
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

	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/config"
	"github.com/burakkuru5534/bunker/backend/internal/game"
	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
	"github.com/burakkuru5534/bunker/backend/internal/moderation"
	"github.com/burakkuru5534/bunker/backend/internal/observability"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
	"github.com/burakkuru5534/bunker/backend/internal/profanity"
	"github.com/burakkuru5534/bunker/backend/internal/realtime"
	"github.com/burakkuru5534/bunker/backend/internal/room"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	getenv, err := localEnv(os.Getenv, ".env")
	if err != nil {
		fmt.Fprintf(os.Stderr, "yapılandırma: %v\n", err)
		os.Exit(1)
	}
	if err := run(ctx, getenv, os.Stdout, nil); err != nil {
		fmt.Fprintf(os.Stderr, "sunucu hatası: %v\n", err)
		os.Exit(1)
	}
}

// localEnv, yerelde (APP_ENV boş veya local) .env dosyasını ortam
// değişkenlerinin arkasına ekler. Bulutta dosya okunmaz.
func localEnv(getenv func(string) string, path string) (func(string) string, error) {
	if e := getenv("APP_ENV"); e != "" && e != string(config.EnvLocal) {
		return getenv, nil
	}
	values, err := config.ReadDotEnv(path)
	if err != nil {
		return nil, err
	}
	return config.WithFallback(getenv, values), nil
}

// dbStartupTimeout, açılışta veritabanı bağlantısı ve doğrulaması için üst süre.
const dbStartupTimeout = 15 * time.Second

// deps, handler'ların bağımlılıklarıdır.
type deps struct {
	DB     *persistence.DB // DATABASE_URL yoksa (yalnızca yerel) nil
	Timers *game.Timers    // DB varsa; aşama süre dolumları (ADR-0005)
	Hub    *realtime.Hub   // DB varsa; WebSocket bağlantıları (ADR-0003)
	// Life, handler kurulurken eklenen açılış/kapanış işleridir (deps değer
	// olarak geçtiği için işaretçi).
	Life *lifecycle
}

// lifecycle, arka plan bileşenlerinin açılış ve kapanış işleridir.
type lifecycle struct {
	start []func()
	stop  []func(ctx context.Context) error
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

	var d deps
	if cfg.DatabaseURL.IsSet() {
		db, err := openDB(ctx, cfg, logger)
		if err != nil {
			return err
		}
		defer db.Close()
		d.DB = db
		d.Timers = game.NewTimers(logger)
		d.Hub = realtime.NewHub(logger)
		d.Life = &lifecycle{}
	} else {
		logger.Warn("DATABASE_URL tanımlı değil; veritabanısız çalışılıyor (yalnızca yerel)")
	}

	srv := &http.Server{
		Handler:  handlerFactory(cfg, logger, d),
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
	// Açılış kurtarması: bitmemiş oyunların süreleri yüklenir; geçmişler hemen işlenir.
	d.Timers.Start()
	if d.Life != nil {
		for _, start := range d.Life.start {
			start()
		}
	}

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
	// WebSocket bağlantıları Shutdown'ın beklediği istekler değildir (hijack);
	// 1012 ile kapatılır, istemciler yeniden bağlanır (ADR-0006).
	if d.Hub != nil {
		d.Hub.Shutdown()
	}
	// Zamanlayıcılar HTTP'den sonra durur (son istekler süre dolumu tetikleyebilir).
	if d.Life != nil {
		for _, stop := range d.Life.stop {
			if err := stop(shutdownCtx); err != nil {
				logger.Warn("arka plan işi zamanında durmadı", "error", err)
			}
		}
	}
	if err := d.Timers.Stop(shutdownCtx); err != nil {
		logger.Warn("zamanlayıcılar zamanında durmadı", "error", err)
	}
	logger.Info("sunucu durdu")
	return nil
}

// openDB, havuzu kurar ve doğrular (persistence.Open). Test/prod'da TLS ve
// rol denetimleri katıdır.
func openDB(ctx context.Context, cfg config.Config, logger *slog.Logger) (*persistence.DB, error) {
	dbCtx, cancel := context.WithTimeout(ctx, dbStartupTimeout)
	defer cancel()
	pool, err := persistence.Open(dbCtx, persistence.Options{
		URL:      cfg.DatabaseURL.Reveal(),
		MaxConns: int32(cfg.DBMaxConns),
		Schema:   cfg.DBSchema,
		Strict:   cfg.Env != config.EnvLocal,
		AppName:  "siginak-api-" + string(cfg.Env),
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("veritabanı: %w", err)
	}
	logger.Info("veritabanı bağlandı", "schema", cfg.DBSchema, "max_conns", cfg.DBMaxConns,
		"acquire_timeout", cfg.DBAcquireTimeout.String())
	return persistence.NewDB(pool, cfg.DBAcquireTimeout, logger), nil
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
// Modül rotaları sonraki görevlerde Register içine eklenecek.
func newHandler(cfg config.Config, logger *slog.Logger, d deps) http.Handler {
	minVersion, err := httpapi.ParseVersion(cfg.MinAppVersion)
	if err != nil {
		// config.Load biçimi zaten doğruladı; buraya gelinmemeli.
		panic(err)
	}
	health := &httpapi.Health{Timeout: httpapi.DefaultReadinessTimeout}
	var modules []func(*http.ServeMux)
	if d.DB != nil {
		health.Checks = append(health.Checks, httpapi.ReadinessCheck{Name: "database", Check: d.DB.Ping})
		// Küfür filtresi (SAFE-013): takma adda red, sohbette *** maskeleme.
		authSvc := auth.NewService(d.DB, profanity.Default.Contains, logger)
		authHTTP := auth.NewHTTP(authSvc, cfg.TrustedProxyHops)
		// Tüm olaylar zamanlayıcıdan geçer: phase_changed bitişi planlar (RT gelince
		// gerçek yayıncı buraya bağlanır).
		// Olay zinciri: zamanlayıcı (bitişleri planlar) → hub (WebSocket).
		publisher := d.Timers.Publisher(d.Hub)
		gameSvc := game.NewService(d.DB, publisher, logger).WithPresence(d.Hub)
		d.Timers.Bind(gameSvc)
		hooks := game.RoomHooks{Starter: game.NewStarter(logger), Service: gameSvc}
		roomSvc := room.NewService(d.DB, d.Hub, publisher, logger).WithGames(hooks).WithTextFilter(profanity.Default)
		roomHTTP := room.NewHTTP(roomSvc, authHTTP.Require)
		gameHTTP := game.NewHTTP(gameSvc, authHTTP.Require)
		modHTTP := moderation.NewHTTP(moderation.NewService(d.DB, logger), authHTTP.Require)
		modules = append(modules, modHTTP.Register)
		// Bağlantısı kopan oyuncular: 60 sn bekleme, sonra ayrılmış sayılır (RT-007).
		disconnects := game.NewDisconnects(gameSvc, roomSvc, d.Hub, logger)
		d.Hub.SetListener(disconnects)
		if d.Life != nil {
			d.Life.start = append(d.Life.start, disconnects.Start)
			d.Life.stop = append(d.Life.stop, disconnects.Stop)
		}
		ws := realtime.NewHandler(d.Hub, authSvc, realtime.RoomSnapshotter{Rooms: roomSvc, Games: gameSvc},
			realtime.DefaultOptions(minVersion), logger)
		modules = append(modules, ws.Register)
		modules = append(modules, authHTTP.Register, roomHTTP.Register, gameHTTP.Register)
	} else {
		logger.Warn("veritabanı yok: /v1 uç noktaları kayıtlı değil (yalnızca yerel)")
	}
	return httpapi.NewRouter(httpapi.Options{
		Logger:         logger,
		ProjectID:      cfg.GCPProjectID,
		MinAppVersion:  minVersion,
		RequestTimeout: cfg.RequestTimeout,
		Register: func(mux *http.ServeMux) {
			health.Register(mux)
			for _, register := range modules {
				register(mux)
			}
		},
	})
}
