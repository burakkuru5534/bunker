// Package testdb, entegrasyon testleri için gerçek PostgreSQL'de yalıtılmış
// bir şema kurar: rastgele adlı şema, uygulama ve migration rolleri, depodaki
// tüm migration'lar (migrations/postgres) ve yetki eşitleme. Testler
// uygulama rolüyle (gerçek yetkilerle, RLS dahil) çalışır.
//
// TEST_DATABASE_ADMIN_URL tanımlı değilse test atlanır. Yalnızca testlerden
// kullanılır; Supabase adresi VERİLMEMELİDİR (yerel / CI PostgreSQL).
package testdb

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/migrate"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Env, kurulmuş test şemasıdır.
type Env struct {
	Schema  string
	DB      *persistence.DB // uygulama rolü
	Migrate *pgxpool.Pool   // migration rolü (katalog verisi yazmak için)
	Admin   *pgx.Conn       // süper kullanıcı (yalnızca kurulum/doğrulama)
}

// New, şemayı kurar ve test bitince siler.
func New(t testing.TB) *Env {
	t.Helper()
	adminURL := os.Getenv("TEST_DATABASE_ADMIN_URL")
	if adminURL == "" {
		t.Skip("TEST_DATABASE_ADMIN_URL tanımlı değil; entegrasyon testi atlandı")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("testdb: yönetici bağlantısı: %v", err)
	}
	t.Cleanup(func() { _ = admin.Close(context.Background()) })

	b := make([]byte, 4)
	_, _ = rand.Read(b)
	sfx := hex.EncodeToString(b)
	schema, app, mig := "tdb_"+sfx, "tdb_"+sfx+"_app", "tdb_"+sfx+"_mig"
	t.Cleanup(func() {
		c := context.Background()
		for _, q := range []string{
			"drop schema if exists " + schema + " cascade",
			"drop owned by " + app, "drop owned by " + mig,
			"drop role if exists " + app, "drop role if exists " + mig,
		} {
			_, _ = admin.Exec(c, q)
		}
	})
	for _, q := range []string{
		"create schema " + schema,
		"revoke all on schema " + schema + " from public",
		fmt.Sprintf("create role %s login password 'app'", app),
		fmt.Sprintf("alter role %s set search_path = %s", app, schema),
		fmt.Sprintf("grant usage on schema %s to %s", schema, app),
		fmt.Sprintf("create role %s login password 'mig'", mig),
		fmt.Sprintf("alter role %s set search_path = %s", mig, schema),
		fmt.Sprintf("grant usage, create on schema %s to %s", schema, mig),
	} {
		if _, err := admin.Exec(ctx, q); err != nil {
			t.Fatalf("testdb: %s: %v", q, err)
		}
	}

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	open := func(user, pass string) *pgxpool.Pool {
		u, _ := url.Parse(adminURL)
		u.User = url.UserPassword(user, pass)
		p, err := persistence.Open(ctx, persistence.Options{URL: u.String(), MaxConns: 4, Schema: schema}, logger)
		if err != nil {
			t.Fatalf("testdb: %s bağlantısı: %v", user, err)
		}
		t.Cleanup(p.Close)
		return p
	}
	migPool := open(mig, "mig")
	migs, err := migrate.Load(migrationsDir())
	if err != nil {
		t.Fatalf("testdb: migration'lar: %v", err)
	}
	if _, _, err := migrate.Up(ctx, migPool, migrate.Options{Schema: schema, AppRole: app, Logger: logger}, migs, false); err != nil {
		t.Fatalf("testdb: migrate up: %v", err)
	}
	return &Env{
		Schema:  schema,
		DB:      persistence.NewDB(open(app, "app"), 2*time.Second, logger),
		Migrate: migPool,
		Admin:   admin,
	}
}

// migrationsDir, depo kökündeki migrations/postgres'tir (bu dosyaya göre).
func migrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "migrations", "postgres")
}
