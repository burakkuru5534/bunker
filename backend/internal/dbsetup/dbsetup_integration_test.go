package dbsetup

// TEST_DATABASE_ADMIN_URL ile gerçek PostgreSQL: kurulum SQL'i, Supabase'in
// postgres rolü gibi süper kullanıcı OLMAYAN bir rolle çalıştırılır. Tüm adlar
// rastgeledir; gerçek bunker_* adlarına dokunulmaz.

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/persistence"
	"github.com/jackc/pgx/v5"
)

func TestIntegrationSetupAndCheck(t *testing.T) {
	adminURL := os.Getenv("TEST_DATABASE_ADMIN_URL")
	if adminURL == "" {
		t.Skip("TEST_DATABASE_ADMIN_URL tanımlı değil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close(context.Background())
	exec := func(c *pgx.Conn, sql string) {
		t.Helper()
		if _, err := c.Exec(ctx, sql); err != nil {
			t.Fatalf("%s: %v", strings.SplitN(sql, "\n", 2)[0], err)
		}
	}

	b := make([]byte, 4)
	_, _ = rand.Read(b)
	sfx := hex.EncodeToString(b)
	supa, schema, role, other := "it_"+sfx+"_supa", "bunker_it"+sfx, "bunker_it"+sfx+"_app", "bunker_it"+sfx+"_diger"
	t.Cleanup(func() {
		c := context.Background()
		for _, q := range []string{
			"drop schema if exists " + schema + " cascade", "drop schema if exists " + other + " cascade",
			"drop owned by " + role, "drop role if exists " + role, "drop role if exists " + supa,
		} {
			_, _ = admin.Exec(c, q)
		}
	})

	// Supabase'deki postgres rolüne benzer: süper kullanıcı değil, CREATEROLE.
	exec(admin, fmt.Sprintf(`create role %s login password 'supa' createrole createdb bypassrls`, supa))
	exec(admin, fmt.Sprintf(`grant create on database %s to %s`, pgx.Identifier{admin.Config().Database}.Sanitize(), supa))
	exec(admin, "create schema "+other)
	for _, r := range []string{"anon", "authenticated"} {
		exec(admin, fmt.Sprintf(`do $$ begin if not exists (select 1 from pg_roles where rolname = '%s') then create role %s nologin; end if; end $$`, r, r))
	}

	supaURL, _ := url.Parse(adminURL)
	supaURL.User = url.UserPassword(supa, "supa")
	supaConn, err := pgx.Connect(ctx, supaURL.String())
	if err != nil {
		t.Fatal(err)
	}
	defer supaConn.Close(context.Background())

	password, _ := NewPassword(32)
	verifier, _ := NewScramVerifier(password)
	appSpec := EnvTest.Spec(KindApp)
	appSpec.Name = role
	sql, err := setupSQL(schema, appSpec, verifier)
	if err != nil {
		t.Fatal(err)
	}
	exec(supaConn, sql)
	exec(supaConn, sql) // tekrar çalıştırmak güvenli

	// Şifre sunucuya yalnızca doğrulayıcı olarak gitti; düz şifreyle giriş çalışır.
	appURL, _ := url.Parse(adminURL)
	appURL.User = url.UserPassword(role, password)
	pool, err := persistence.Open(ctx, persistence.Options{URL: appURL.String(), MaxConns: 1, Schema: schema},
		slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	if err != nil {
		t.Fatalf("uygulama rolüyle bağlanılmalı: %v", err)
	}
	defer pool.Close()
	if err := persistence.Verify(ctx, pool, persistence.Options{Schema: schema, Strict: true}); err != nil {
		t.Fatalf("katı kimlik denetimi: %v", err)
	}

	results := Check(ctx, pool, schema, appSpec)
	if Failed(results) {
		t.Fatalf("denetimler geçmeli: %+v", results)
	}
	want := map[string]bool{"statement_timeout": false, "şemada DDL": false, "ortam yalıtımı": false, "Data API: anon": false}
	for _, r := range results {
		if _, ok := want[r.Name]; ok && r.Status == OK {
			want[r.Name] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("%s denetimi OK olmalı: %+v", name, results)
		}
	}

	// Migration rolü: aynı şemada CREATE yetkisiyle kurulur ve denetimden geçer.
	migSpec := EnvTest.Spec(KindMigrator)
	migSpec.Name = role + "_mig"
	t.Cleanup(func() {
		c := context.Background()
		_, _ = admin.Exec(c, "drop owned by "+migSpec.Name)
		_, _ = admin.Exec(c, "drop role if exists "+migSpec.Name)
	})
	migPass, _ := NewPassword(32)
	migVer, _ := NewScramVerifier(migPass)
	migSQL, err := setupSQL(schema, migSpec, migVer)
	if err != nil {
		t.Fatal(err)
	}
	exec(supaConn, migSQL)
	migURL, _ := url.Parse(adminURL)
	migURL.User = url.UserPassword(migSpec.Name, migPass)
	migPool, err := persistence.Open(ctx, persistence.Options{URL: migURL.String(), MaxConns: 1, Schema: schema},
		slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	if err != nil {
		t.Fatalf("migration rolüyle bağlanılmalı: %v", err)
	}
	defer migPool.Close()
	if rs := Check(ctx, migPool, schema, migSpec); Failed(rs) {
		t.Fatalf("migration rolü denetimleri geçmeli: %+v", rs)
	}
	// Uygulama rolü denetimi hâlâ geçer (migrator kurulumu ona dokunmaz).
	if rs := Check(ctx, pool, schema, appSpec); Failed(rs) {
		t.Fatalf("uygulama rolü denetimleri bozulmamalı: %+v", rs)
	}

	// Bozulmalar yakalanır: diğer ortamın şemasına erişim ve anon erişimi.
	exec(admin, fmt.Sprintf("grant usage on schema %s to %s", other, role))
	exec(admin, fmt.Sprintf("grant usage on schema %s to anon", schema))
	exec(admin, fmt.Sprintf("alter role %s set statement_timeout = '0'", role))
	pool.Reset() // yeni oturum: rol ayarları yeniden okunur
	got := map[string]Status{}
	for _, r := range Check(ctx, pool, schema, appSpec) {
		got[r.Name] = r.Status
	}
	if got["ortam yalıtımı"] != Fail || got["Data API: anon"] != Fail || got["statement_timeout"] != Fail {
		t.Fatalf("bozulmalar yakalanmalı: %v", got)
	}
}
