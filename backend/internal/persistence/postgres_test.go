package persistence

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func quietLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

const remote = "postgres://bunker_test_app.ref:cokgizli@aws-0-eu-central-1.pooler.supabase.com:5432/postgres"

func TestPoolConfigTLSPolicy(t *testing.T) {
	cases := []struct {
		name     string
		url      string
		strict   bool
		wantErr  string // boşsa hata beklenmez
		wantWarn bool
	}{
		{"strict verify-full", remote + "?sslmode=verify-full", true, "", false},
		{"strict require", remote + "?sslmode=require", true, "verify-full", false},
		{"strict verify-ca", remote + "?sslmode=verify-ca", true, "verify-full", false},
		{"strict varsayılan prefer", remote, true, "verify-full", false},
		{"strict yerel disable", "postgres://u:p@localhost/db?sslmode=disable", true, "verify-full", false},
		{"yerel uzak verify-full", remote + "?sslmode=verify-full", false, "", false},
		{"yerel uzak require uyarır", remote + "?sslmode=require", false, "", true},
		{"yerel uzak disable", remote + "?sslmode=disable", false, "şifresiz", false},
		{"yerel uzak prefer (geri düşüş şifresiz)", remote + "?sslmode=prefer", false, "şifresiz", false},
		{"loopback disable", "postgres://u:p@127.0.0.1:5432/db?sslmode=disable", false, "", false},
		{"localhost prefer", "postgres://u:p@localhost/db", false, "", false},
		{"ipv6 loopback", "postgres://u:p@[::1]:5432/db?sslmode=disable", false, "", false},
		{"unix soketi", "postgres://u:p@/db?host=/var/run/postgresql&sslmode=disable", false, "", false},
		{"çoklu host biri uzak", "postgres://u:p@localhost:5432,db.example.com:5432/db?sslmode=disable", false, "şifresiz", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			_, err := poolConfig(Options{URL: tc.url, MaxConns: 3, Strict: tc.strict}, quietLogger(&logs))
			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("hata beklenmiyordu: %v", err)
			case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
				t.Fatalf("%q içeren hata bekleniyordu: %v", tc.wantErr, err)
			}
			if got := strings.Contains(logs.String(), "doğrulanmıyor"); got != tc.wantWarn {
				t.Fatalf("uyarı = %v, %v bekleniyordu: %s", got, tc.wantWarn, logs.String())
			}
			if err != nil && strings.Contains(err.Error(), "cokgizli") {
				t.Fatal("hata şifre içermemeli")
			}
		})
	}
}

func TestPoolConfigSettings(t *testing.T) {
	cfg, err := poolConfig(Options{URL: remote + "?sslmode=verify-full", MaxConns: 5, AppName: "siginak-api-test"}, quietLogger(&bytes.Buffer{}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxConns != 5 || cfg.MinConns != 0 || cfg.MaxConnIdleTime != maxConnIdleTime ||
		cfg.MaxConnLifetime != maxConnLifetime || cfg.HealthCheckPeriod != healthCheckPeriod {
		t.Fatalf("havuz ayarları yanlış: %+v", cfg)
	}
	if cfg.ConnConfig.ConnectTimeout != connectTimeout || cfg.ConnConfig.RuntimeParams["application_name"] != "siginak-api-test" {
		t.Fatalf("bağlantı ayarları yanlış: %v %v", cfg.ConnConfig.ConnectTimeout, cfg.ConnConfig.RuntimeParams)
	}
	if cfg.ConnConfig.TLSConfig.ServerName != "aws-0-eu-central-1.pooler.supabase.com" {
		t.Fatal("verify-full sunucu adını doğrulamalı")
	}

	// Adreste verilen connect_timeout korunur.
	cfg, _ = poolConfig(Options{URL: remote + "?sslmode=verify-full&connect_timeout=2", MaxConns: 1}, quietLogger(&bytes.Buffer{}))
	if cfg.ConnConfig.ConnectTimeout.Seconds() != 2 {
		t.Fatalf("connect_timeout korunmalı: %v", cfg.ConnConfig.ConnectTimeout)
	}
}

func TestPoolConfigRejectsBadInputWithoutLeaking(t *testing.T) {
	for _, url := range []string{
		"",
		"postgres://u:cokgizli@h:abc/db",
		"postgres://u:cokgizli@h/db?sslmode=bilinmeyen",
		"mysql://u:cokgizli@h/db",
		"bozuk cokgizli",
	} {
		_, err := poolConfig(Options{URL: url, MaxConns: 3, Strict: true}, quietLogger(&bytes.Buffer{}))
		if err == nil {
			t.Errorf("%q reddedilmeliydi", url)
			continue
		}
		if strings.Contains(err.Error(), "cokgizli") {
			t.Errorf("hata şifre içermemeli: %v", err)
		}
	}
	if _, err := poolConfig(Options{URL: remote + "?sslmode=verify-full", MaxConns: 0}, quietLogger(&bytes.Buffer{})); err == nil {
		t.Error("MaxConns 0 reddedilmeli")
	}
}

func TestCheckIdentity(t *testing.T) {
	app := Identity{User: "bunker_test_app", Schema: "bunker_test"}
	strict := Options{Schema: "bunker_test", Strict: true}
	if err := checkIdentity(app, strict); err != nil {
		t.Fatalf("uygulama rolü kabul edilmeli: %v", err)
	}

	cases := map[string]struct {
		id   Identity
		opts Options
		want string
	}{
		"süper":           {Identity{User: "x", Schema: "bunker_test", Super: true}, strict, "süper"},
		"bypassrls":       {Identity{User: "x", Schema: "bunker_test", BypassRLS: true}, strict, "BYPASSRLS"},
		"postgres":        {Identity{User: "postgres", Schema: "bunker_test"}, strict, "yönetim"},
		"service_role":    {Identity{User: "service_role", Schema: "bunker_test"}, strict, "yönetim"},
		"supabase_ önek":  {Identity{User: "supabase_yeni_rol", Schema: "bunker_test"}, strict, "yönetim"},
		"yanlış şema":     {Identity{User: "bunker_test_app", Schema: "public"}, strict, `"bunker_test"`},
		"şema yok":        {Identity{User: "bunker_test_app", Schema: ""}, strict, "(yok)"},
		"yerelde de şema": {Identity{User: "postgres", Schema: "public", Super: true}, Options{Schema: "bunker_test"}, "şema"},
	}
	for name, tc := range cases {
		if err := checkIdentity(tc.id, tc.opts); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %q içeren hata bekleniyordu: %v", name, tc.want, err)
		}
	}

	// Yerelde süper kullanıcı (ör. yerel postgres) şema doğruysa kabul edilir.
	if err := checkIdentity(Identity{User: "postgres", Schema: "public", Super: true}, Options{Schema: "public"}); err != nil {
		t.Fatalf("yerelde kabul edilmeli: %v", err)
	}
}

func TestTLSLevelString(t *testing.T) {
	if tlsNone.String() != "şifresiz" || tlsVerified.String() != "doğrulanmış" {
		t.Fatal("düzey adları yanlış")
	}
}
