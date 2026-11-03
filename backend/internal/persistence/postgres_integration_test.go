package persistence

// Gerçek PostgreSQL ile entegrasyon testleri. TEST_DATABASE_ADMIN_URL
// tanımlı değilse atlanır. CI'da postgres servis konteyneri ile çalışır
// (.github/workflows/backend.yml). Her test kendi rastgele adlı rolünü ve
// şemasını oluşturur, sonunda siler.

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type fixture struct {
	admin  *pgx.Conn
	base   *url.URL
	suffix string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_ADMIN_URL")
	if raw == "" {
		t.Skip("TEST_DATABASE_ADMIN_URL tanımlı değil; entegrasyon testi atlandı")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, raw)
	if err != nil {
		t.Fatalf("yönetici bağlantısı: %v", err)
	}
	t.Cleanup(func() { _ = admin.Close(context.Background()) })
	base, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return &fixture{admin: admin, base: base, suffix: hex.EncodeToString(b)}
}

func (f *fixture) exec(t *testing.T, sql string) {
	t.Helper()
	if _, err := f.admin.Exec(context.Background(), sql); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

// role, search_path'i schema olan bir LOGIN rolü ve (varsa) şemayı kurar;
// rolün bağlantı adresini döndürür.
func (f *fixture) role(t *testing.T, name, schema, extra string) string {
	t.Helper()
	name = fmt.Sprintf("it_%s_%s", f.suffix, name)
	pass := "p_" + f.suffix + "_" + name
	f.exec(t, fmt.Sprintf(`create role %s login password '%s' %s`, name, pass, extra))
	t.Cleanup(func() {
		_, _ = f.admin.Exec(context.Background(), "drop owned by "+name)
		_, _ = f.admin.Exec(context.Background(), "drop role if exists "+name)
	})
	if schema != "" {
		f.exec(t, fmt.Sprintf(`alter role %s set search_path = %s`, name, schema))
	}
	u := *f.base
	u.User = url.UserPassword(name, pass)
	return u.String()
}

func (f *fixture) schema(t *testing.T, name string) string {
	t.Helper()
	name = fmt.Sprintf("it_%s_%s", f.suffix, name)
	f.exec(t, "create schema "+name)
	t.Cleanup(func() { _, _ = f.admin.Exec(context.Background(), "drop schema if exists "+name+" cascade") })
	return name
}

func ctx5(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestIntegrationOpenAppRole(t *testing.T) {
	f := newFixture(t)
	schema := f.schema(t, "app")
	appURL := f.role(t, "app", schema, "")
	f.exec(t, fmt.Sprintf("grant usage on schema %s to it_%s_app", schema, f.suffix))

	var logs bytes.Buffer
	pool, err := Open(ctx5(t), Options{URL: appURL, MaxConns: 2, Schema: schema, AppName: "siginak-it"}, quietLogger(&logs))
	if err != nil {
		t.Fatalf("açılmalı: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx5(t)); err != nil {
		t.Fatalf("ping: %v", err)
	}
	if pool.Config().MaxConns != 2 {
		t.Fatalf("MaxConns = %d", pool.Config().MaxConns)
	}
	var app string
	if err := pool.QueryRow(ctx5(t), "select current_setting('application_name')").Scan(&app); err != nil || app != "siginak-it" {
		t.Fatalf("application_name = %q (%v)", app, err)
	}

	// Strict kimlik denetimi gerçek veritabanında da geçer (TLS ayrı denetlenir).
	if err := Verify(ctx5(t), pool, Options{Schema: schema, Strict: true}); err != nil {
		t.Fatalf("strict doğrulama: %v", err)
	}
}

func TestIntegrationSchemaMismatch(t *testing.T) {
	f := newFixture(t)
	// search_path var olmayan şemayı gösteriyor → current_schema() NULL.
	missing := f.role(t, "nosch", fmt.Sprintf("it_%s_yok", f.suffix), "")
	_, err := Open(ctx5(t), Options{URL: missing, MaxConns: 1, Schema: "bunker_test"}, quietLogger(&bytes.Buffer{}))
	if err == nil || !strings.Contains(err.Error(), "(yok)") {
		t.Fatalf("şema yok hatası bekleniyordu: %v", err)
	}

	other := f.schema(t, "diger")
	wrong := f.role(t, "wrong", other, "")
	f.exec(t, fmt.Sprintf("grant usage on schema %s to it_%s_wrong", other, f.suffix))
	_, err = Open(ctx5(t), Options{URL: wrong, MaxConns: 1, Schema: "bunker_test"}, quietLogger(&bytes.Buffer{}))
	if err == nil || !strings.Contains(err.Error(), other) {
		t.Fatalf("yanlış şema hatası bekleniyordu: %v", err)
	}
}

func TestIntegrationStrictRejectsPrivilegedRoles(t *testing.T) {
	f := newFixture(t)
	schema := f.schema(t, "priv")
	bypass := f.role(t, "bypass", schema, "bypassrls")
	f.exec(t, fmt.Sprintf("grant usage on schema %s to it_%s_bypass", schema, f.suffix))

	pool, err := Open(ctx5(t), Options{URL: bypass, MaxConns: 1, Schema: schema}, quietLogger(&bytes.Buffer{}))
	if err != nil {
		t.Fatalf("yerel modda açılmalı: %v", err)
	}
	defer pool.Close()
	if err := Verify(ctx5(t), pool, Options{Schema: schema, Strict: true}); err == nil || !strings.Contains(err.Error(), "BYPASSRLS") {
		t.Fatalf("BYPASSRLS reddedilmeli: %v", err)
	}

	// Yönetici (süper kullanıcı) bağlantısı.
	adminPool, err := Open(ctx5(t), Options{URL: f.base.String(), MaxConns: 1, Schema: "public"}, quietLogger(&bytes.Buffer{}))
	if err != nil {
		t.Fatalf("yönetici yerel modda açılmalı: %v", err)
	}
	defer adminPool.Close()
	if err := Verify(ctx5(t), adminPool, Options{Schema: "public", Strict: true}); err == nil || !strings.Contains(err.Error(), "süper") {
		t.Fatalf("süper kullanıcı reddedilmeli: %v", err)
	}
}

func TestIntegrationWrongPasswordDoesNotLeak(t *testing.T) {
	f := newFixture(t)
	u := *f.base
	u.User = url.UserPassword(fmt.Sprintf("it_%s_yok", f.suffix), "cokgizli-sifre")
	_, err := Open(ctx5(t), Options{URL: u.String(), MaxConns: 1, Schema: "x"}, quietLogger(&bytes.Buffer{}))
	if err == nil {
		t.Fatal("hata bekleniyordu")
	}
	if strings.Contains(err.Error(), "cokgizli") {
		t.Fatalf("hata şifre içermemeli: %v", err)
	}
}

func TestIntegrationUnreachableRespectsContext(t *testing.T) {
	// 127.0.0.1:1 genelde kapalıdır; bağlantı hızla reddedilmeli.
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := Open(ctx, Options{URL: "postgres://u:p@127.0.0.1:1/db?sslmode=disable", MaxConns: 1, Schema: "x"}, quietLogger(&bytes.Buffer{}))
	if err == nil || time.Since(start) > 3*time.Second {
		t.Fatalf("hızlı hata bekleniyordu: %v (%v)", err, time.Since(start))
	}
}
