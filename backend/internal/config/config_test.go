package config

import (
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(env(nil))
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if cfg.Env != EnvLocal || cfg.Port != 8080 || cfg.ShutdownTimeout != 8*time.Second ||
		cfg.MinAppVersion != "0.1.0" || cfg.RequestTimeout != 10*time.Second ||
		cfg.LogLevel != "info" || cfg.GCPProjectID != "" {
		t.Fatalf("varsayılanlar yanlış: %+v", cfg)
	}
	if cfg.Addr() != ":8080" {
		t.Fatalf("Addr = %q", cfg.Addr())
	}
	if cfg.IsProduction() {
		t.Fatal("local, prod sayılmamalı")
	}
	if cfg.LogJSON() {
		t.Fatal("yerelde metin log bekleniyordu")
	}
}

func TestLoadValues(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"APP_ENV":          "prod",
		"PORT":             "9090",
		"SHUTDOWN_TIMEOUT": "5s",
		"MIN_APP_VERSION":  "1.4.0",
		"REQUEST_TIMEOUT":  "3s",
		"LOG_LEVEL":        "DEBUG",
		"GCP_PROJECT_ID":   "siginak-prod-123",
		"DATABASE_URL":     " postgres://u:p@h/db ",
	}))
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if cfg.Env != EnvProd || cfg.Port != 9090 || cfg.ShutdownTimeout != 5*time.Second ||
		cfg.MinAppVersion != "1.4.0" || cfg.RequestTimeout != 3*time.Second ||
		cfg.LogLevel != "debug" || cfg.GCPProjectID != "siginak-prod-123" || !cfg.LogJSON() {
		t.Fatalf("değerler okunmadı: %+v", cfg)
	}
	if !cfg.IsProduction() {
		t.Fatal("prod bekleniyordu")
	}
	if cfg.DatabaseURL.Reveal() != "postgres://u:p@h/db" || cfg.DBMaxConns != 5 || cfg.DBSchema != "bunker_prod" || cfg.TrustedProxyHops != 1 {
		t.Fatalf("prod veritabanı varsayılanları yanlış: %d %s", cfg.DBMaxConns, cfg.DBSchema)
	}
}

func TestDatabaseDefaultsAndOverrides(t *testing.T) {
	cfg, err := Load(env(nil))
	if err != nil || cfg.DatabaseURL.IsSet() || cfg.DBMaxConns != 3 || cfg.DBSchema != "bunker_test" || cfg.DBAcquireTimeout != 3*time.Second || cfg.TrustedProxyHops != 0 {
		t.Fatalf("yerel varsayılanlar: %v %+v", err, cfg)
	}
	cfg, err = Load(env(map[string]string{"APP_ENV": "test", "DATABASE_URL": "x", "DB_MAX_CONNS": "7", "DB_SCHEMA": "bunker_dev_1", "DB_ACQUIRE_TIMEOUT": "500ms", "TRUSTED_PROXY_HOPS": "2"}))
	if err != nil || cfg.DBMaxConns != 7 || cfg.DBSchema != "bunker_dev_1" || cfg.DBAcquireTimeout != 500*time.Millisecond || cfg.TrustedProxyHops != 2 {
		t.Fatalf("geçersiz kılma: %v %+v", err, cfg)
	}
}

func TestDatabaseURLRequiredOutsideLocal(t *testing.T) {
	for _, e := range []string{"test", "prod"} {
		_, err := Load(env(map[string]string{"APP_ENV": e, "DATABASE_URL": "   "}))
		if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
			t.Errorf("%s: DATABASE_URL zorunlu olmalı: %v", e, err)
		}
	}
}

func TestDatabaseURLNeverInErrors(t *testing.T) {
	_, err := Load(env(map[string]string{"DATABASE_URL": "postgres://u:cokgizli@h/db", "PORT": "x", "DB_MAX_CONNS": "0"}))
	if err == nil || strings.Contains(err.Error(), "cokgizli") {
		t.Fatalf("gizli değer hata mesajına girmemeli: %v", err)
	}
}

func TestLoadInvalid(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"bilinmeyen ortam", map[string]string{"APP_ENV": "staging"}, "APP_ENV"},
		{"sayı olmayan port", map[string]string{"PORT": "abc"}, "PORT"},
		{"aralık dışı port", map[string]string{"PORT": "70000"}, "PORT"},
		{"sıfır port", map[string]string{"PORT": "0"}, "PORT"},
		{"bozuk süre", map[string]string{"SHUTDOWN_TIMEOUT": "on saniye"}, "SHUTDOWN_TIMEOUT"},
		{"çok uzun süre", map[string]string{"SHUTDOWN_TIMEOUT": "5m"}, "SHUTDOWN_TIMEOUT"},
		{"negatif süre", map[string]string{"SHUTDOWN_TIMEOUT": "-1s"}, "SHUTDOWN_TIMEOUT"},
		{"bozuk sürüm", map[string]string{"MIN_APP_VERSION": "1.2"}, "MIN_APP_VERSION"},
		{"önde sıfırlı sürüm", map[string]string{"MIN_APP_VERSION": "1.02.0"}, "MIN_APP_VERSION"},
		{"bozuk istek süresi", map[string]string{"REQUEST_TIMEOUT": "x"}, "REQUEST_TIMEOUT"},
		{"bilinmeyen log seviyesi", map[string]string{"LOG_LEVEL": "trace"}, "LOG_LEVEL"},
		{"bozuk proje kimliği", map[string]string{"GCP_PROJECT_ID": "Proje Adı"}, "GCP_PROJECT_ID"},
		{"sıfır bağlantı", map[string]string{"DB_MAX_CONNS": "0"}, "DB_MAX_CONNS"},
		{"çok bağlantı", map[string]string{"DB_MAX_CONNS": "21"}, "DB_MAX_CONNS"},
		{"bozuk şema", map[string]string{"DB_SCHEMA": "public; drop"}, "DB_SCHEMA"},
		{"çok vekil", map[string]string{"TRUSTED_PROXY_HOPS": "4"}, "TRUSTED_PROXY_HOPS"},
		{"negatif vekil", map[string]string{"TRUSTED_PROXY_HOPS": "-1"}, "TRUSTED_PROXY_HOPS"},
		{"kısa acquire", map[string]string{"DB_ACQUIRE_TIMEOUT": "10ms"}, "DB_ACQUIRE_TIMEOUT"},
		{"uzun acquire", map[string]string{"DB_ACQUIRE_TIMEOUT": "30s"}, "DB_ACQUIRE_TIMEOUT"},
		{"bozuk acquire", map[string]string{"DB_ACQUIRE_TIMEOUT": "üç"}, "DB_ACQUIRE_TIMEOUT"},
		{"büyük harf şema", map[string]string{"DB_SCHEMA": "Bunker"}, "DB_SCHEMA"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(env(tc.env))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("%s hatası bekleniyordu, alınan: %v", tc.want, err)
			}
		})
	}
}

func TestLoadReportsAllErrors(t *testing.T) {
	_, err := Load(env(map[string]string{"APP_ENV": "x", "PORT": "y"}))
	if err == nil || !strings.Contains(err.Error(), "APP_ENV") || !strings.Contains(err.Error(), "PORT") {
		t.Fatalf("tüm hatalar birlikte raporlanmalı: %v", err)
	}
}
