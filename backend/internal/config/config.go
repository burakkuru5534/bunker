// Package config, servisin ortam değişkenlerinden okunan yapılandırmasıdır.
//
// Gizli değerler (ör. DATABASE_URL) yalnızca ortam değişkeninden gelir;
// yerelde .env, Cloud Run'da Secret Manager üzerinden verilir. Gizli değerler
// hiçbir zaman loglanmaz (bkz. docs/security.md).
package config

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

// Env, servisin çalıştığı ortamdır.
type Env string

const (
	EnvLocal Env = "local"
	EnvTest  Env = "test"
	EnvProd  Env = "prod"
)

// Config, servisin yapılandırmasıdır.
type Config struct {
	// Env: local | test | prod (APP_ENV, varsayılan local).
	Env Env
	// Port: dinlenecek TCP portu (PORT; Cloud Run bunu verir, varsayılan 8080).
	Port int
	// ShutdownTimeout: SIGTERM sonrası açık isteklerin bitmesi için beklenecek
	// süre (SHUTDOWN_TIMEOUT, varsayılan 8s). Cloud Run SIGTERM'den ~10 sn sonra
	// süreci öldürdüğü için bundan kısa tutulur.
	ShutdownTimeout time.Duration
}

// Addr, net.Listen için adres döndürür.
func (c Config) Addr() string { return fmt.Sprintf(":%d", c.Port) }

// IsProduction, prod ortamı mı?
func (c Config) IsProduction() bool { return c.Env == EnvProd }

// Load, yapılandırmayı verilen getenv fonksiyonuyla okur ve doğrular.
// Testlerde os.Getenv yerine sahte bir fonksiyon verilir.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Env:             EnvLocal,
		Port:            8080,
		ShutdownTimeout: 8 * time.Second,
	}
	var errs []error

	if v := getenv("APP_ENV"); v != "" {
		switch Env(v) {
		case EnvLocal, EnvTest, EnvProd:
			cfg.Env = Env(v)
		default:
			errs = append(errs, fmt.Errorf("APP_ENV geçersiz: %q (local|test|prod)", v))
		}
	}

	if v := getenv("PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 || p > 65535 {
			errs = append(errs, fmt.Errorf("PORT geçersiz: %q (1-65535)", v))
		} else {
			cfg.Port = p
		}
	}

	if v := getenv("SHUTDOWN_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 || d > time.Minute {
			errs = append(errs, fmt.Errorf("SHUTDOWN_TIMEOUT geçersiz: %q (ör. 8s; 0 < süre ≤ 1m)", v))
		} else {
			cfg.ShutdownTimeout = d
		}
	}

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}
	return cfg, nil
}
