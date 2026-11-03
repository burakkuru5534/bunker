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
	if cfg.Env != EnvLocal || cfg.Port != 8080 || cfg.ShutdownTimeout != 8*time.Second {
		t.Fatalf("varsayılanlar yanlış: %+v", cfg)
	}
	if cfg.Addr() != ":8080" {
		t.Fatalf("Addr = %q", cfg.Addr())
	}
	if cfg.IsProduction() {
		t.Fatal("local, prod sayılmamalı")
	}
}

func TestLoadValues(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"APP_ENV":          "prod",
		"PORT":             "9090",
		"SHUTDOWN_TIMEOUT": "5s",
	}))
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if cfg.Env != EnvProd || cfg.Port != 9090 || cfg.ShutdownTimeout != 5*time.Second {
		t.Fatalf("değerler okunmadı: %+v", cfg)
	}
	if !cfg.IsProduction() {
		t.Fatal("prod bekleniyordu")
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
