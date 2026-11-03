package migrate

// TEST_DATABASE_ADMIN_URL ile gerçek PostgreSQL. Her test rastgele şema,
// uygulama rolü ve migration rolü kurar; gerçek bunker_* adlarına dokunmaz.

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
	"strings"
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/persistence"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type env struct {
	admin            *pgx.Conn
	adminURL         string
	schema, app, mig string
	appPool, migPool *pgxpool.Pool
}

func setup(t *testing.T) *env {
	t.Helper()
	adminURL := os.Getenv("TEST_DATABASE_ADMIN_URL")
	if adminURL == "" {
		t.Skip("TEST_DATABASE_ADMIN_URL tanımlı değil")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Close(context.Background()) })

	b := make([]byte, 4)
	_, _ = rand.Read(b)
	sfx := hex.EncodeToString(b)
	e := &env{admin: admin, adminURL: adminURL, schema: "it_" + sfx, app: "it_" + sfx + "_app", mig: "it_" + sfx + "_mig"}
	t.Cleanup(func() {
		for _, q := range []string{
			"drop schema if exists " + e.schema + " cascade",
			"drop owned by " + e.app, "drop owned by " + e.mig,
			"drop role if exists " + e.app, "drop role if exists " + e.mig,
		} {
			_, _ = admin.Exec(context.Background(), q)
		}
	})
	for _, q := range []string{
		"create schema " + e.schema,
		"revoke all on schema " + e.schema + " from public",
		fmt.Sprintf("create role %s login password 'app'", e.app),
		fmt.Sprintf("alter role %s set search_path = %s", e.app, e.schema),
		fmt.Sprintf("grant usage on schema %s to %s", e.schema, e.app),
		fmt.Sprintf("create role %s login password 'mig'", e.mig),
		fmt.Sprintf("alter role %s set search_path = %s", e.mig, e.schema),
		fmt.Sprintf("grant usage, create on schema %s to %s", e.schema, e.mig),
		`do $$ begin if not exists (select 1 from pg_roles where rolname = 'anon') then create role anon nologin; end if; end $$`,
		fmt.Sprintf("grant anon to %s", currentUser(t, admin)),
	} {
		if _, err := admin.Exec(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	e.appPool = e.open(t, e.app, "app")
	e.migPool = e.open(t, e.mig, "mig")
	return e
}

func currentUser(t *testing.T, c *pgx.Conn) string {
	var u string
	if err := c.QueryRow(context.Background(), "select current_user").Scan(&u); err != nil {
		t.Fatal(err)
	}
	return u
}

func (e *env) open(t *testing.T, user, pass string) *pgxpool.Pool {
	t.Helper()
	u, _ := url.Parse(e.adminURL)
	u.User = url.UserPassword(user, pass)
	p, err := persistence.Open(context.Background(), persistence.Options{URL: u.String(), MaxConns: 2, Schema: e.schema},
		slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

func (e *env) opts() Options {
	return Options{Schema: e.schema, AppRole: e.app, Logger: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), LockWait: 400 * time.Millisecond}
}

func ctx10(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

const (
	m1 = "create table items (\n  id bigint generated always as identity primary key,\n  name text not null\n);\n"
	m2 = "-- migrate:no-transaction\ncreate index concurrently items_name_idx on items (name);\n"
)

func TestIntegrationUpGrantsAndRLS(t *testing.T) {
	e := setup(t)
	ctx := ctx10(t)
	dir := writeFiles(t, map[string]string{"0001_create_items.sql": m1, "0002_items_name_idx.sql": m2})
	migs, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	// dry-run hiçbir şey yazmaz (sürüm tablosu bile oluşmaz).
	pending, _, err := Up(ctx, e.migPool, e.opts(), migs, true)
	if err != nil || len(pending) != 2 {
		t.Fatalf("dry-run: %v %v", pending, err)
	}
	var exists bool
	_ = e.admin.QueryRow(ctx, "select to_regclass($1) is not null", e.schema+".schema_migrations").Scan(&exists)
	if exists {
		t.Fatal("dry-run sürüm tablosu oluşturmamalı")
	}

	applied, rep, err := Up(ctx, e.migPool, e.opts(), migs, false)
	if err != nil || len(applied) != 2 {
		t.Fatalf("up: %v %v", applied, err)
	}
	if rep.Tables != 2 || len(rep.RLSEnabled) != 2 || len(rep.PoliciesCreated) != 1 || rep.PoliciesCreated[0] != "items" {
		t.Fatalf("eşitleme raporu: %+v", rep)
	}

	// Uygulama rolü: RLS politikasıyla okur/yazar.
	if _, err := e.appPool.Exec(ctx, "insert into items (name) values ('kart')"); err != nil {
		t.Fatalf("uygulama yazabilmeli: %v", err)
	}
	var n int
	if err := e.appPool.QueryRow(ctx, "select count(*) from items").Scan(&n); err != nil || n != 1 {
		t.Fatalf("uygulama okuyabilmeli: %d %v", n, err)
	}
	if _, err := e.appPool.Exec(ctx, "update items set name = 'x'"); err != nil {
		t.Fatalf("güncelleme: %v", err)
	}
	// Yasak olanlar.
	for _, q := range []string{
		"select * from schema_migrations",
		"truncate items",
		"create table x (id int)",
		"alter table items add column y int",
		"drop table items",
	} {
		if _, err := e.appPool.Exec(ctx, q); err == nil {
			t.Errorf("uygulama rolü yapamamalı: %s", q)
		}
	}

	// Supabase anon rolü: tabloya erişemez (yetki yok + politika yok).
	tx, _ := e.admin.Begin(ctx)
	_, _ = tx.Exec(ctx, "set local role anon")
	_, err = tx.Exec(ctx, "select * from "+e.schema+".items")
	_ = tx.Rollback(ctx)
	if err == nil {
		t.Fatal("anon erişememeli")
	}

	// Tekrar up: değişiklik yok, idempotent.
	applied, rep, err = Up(ctx, e.migPool, e.opts(), migs, false)
	if err != nil || len(applied) != 0 || len(rep.PoliciesCreated) != 0 || len(rep.RLSEnabled) != 0 || rep.Tables != 2 {
		t.Fatalf("ikinci up: %v %+v %v", applied, rep, err)
	}
	st, err := GetStatus(ctx, e.migPool, e.opts(), migs)
	if err != nil || len(st.Applied) != 2 || len(st.Pending) != 0 || st.Applied[0].Name != "create_items" {
		t.Fatalf("durum: %+v %v", st, err)
	}

	// Uygulanmış dosya değişirse çalışmaz.
	if err := os.WriteFile(filepath.Join(dir, "0001_create_items.sql"), []byte(m1+"-- değişti\nselect 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, _ := Load(dir)
	if _, _, err := Up(ctx, e.migPool, e.opts(), changed, false); err == nil || !strings.Contains(err.Error(), "değiştirilmiş") {
		t.Fatalf("sağlama hatası bekleniyordu: %v", err)
	}
}

func TestIntegrationFailedMigrationRollsBack(t *testing.T) {
	e := setup(t)
	ctx := ctx10(t)
	dir := writeFiles(t, map[string]string{
		"0001_create_items.sql": m1,
		"0002_broken.sql":       "create table half (id int);\n\nalter table items add column x nope_type;\n",
	})
	migs, _ := Load(dir)
	applied, _, err := Up(ctx, e.migPool, e.opts(), migs, false)
	if err == nil || len(applied) != 1 {
		t.Fatalf("ilk başarılı, ikinci hatalı olmalı: %v %v", applied, err)
	}
	for _, want := range []string{"0002_broken.sql", "nope_type", "satır 3", "geri alındı"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("hata %q içermeli: %v", want, err)
		}
	}
	var half bool
	_ = e.admin.QueryRow(ctx, "select to_regclass($1) is not null", e.schema+".half").Scan(&half)
	if half {
		t.Fatal("hatalı dosyanın ilk ifadesi de geri alınmalı")
	}
	st, _ := GetStatus(ctx, e.migPool, e.opts(), migs)
	if len(st.Applied) != 1 || len(st.Pending) != 1 {
		t.Fatalf("yalnızca 0001 kayıtlı olmalı: %+v", st)
	}
}

func TestIntegrationLockPreventsConcurrentRuns(t *testing.T) {
	e := setup(t)
	ctx := ctx10(t)
	key := "siginak.migrate." + e.schema
	if _, err := e.admin.Exec(ctx, "select pg_advisory_lock(hashtextextended($1, 0))", key); err != nil {
		t.Fatal(err)
	}
	migs, _ := Load(writeFiles(t, map[string]string{"0001_create_items.sql": m1}))
	start := time.Now()
	_, _, err := Up(ctx, e.migPool, e.opts(), migs, false)
	if err == nil || !strings.Contains(err.Error(), "başka bir migration") || time.Since(start) < 300*time.Millisecond {
		t.Fatalf("kilit beklenip hata verilmeli: %v", err)
	}
	_, _ = e.admin.Exec(ctx, "select pg_advisory_unlock(hashtextextended($1, 0))", key)
	if _, _, err := Up(ctx, e.migPool, e.opts(), migs, false); err != nil {
		t.Fatalf("kilit bırakılınca çalışmalı: %v", err)
	}
	// Up kendi kilidini bırakmış olmalı.
	var ok bool
	_ = e.admin.QueryRow(ctx, "select pg_try_advisory_lock(hashtextextended($1, 0))", key).Scan(&ok)
	if !ok {
		t.Fatal("up kilidi bırakmalı")
	}
	_, _ = e.admin.Exec(ctx, "select pg_advisory_unlock(hashtextextended($1, 0))", key)
}

func TestIntegrationSyncFixesDrift(t *testing.T) {
	e := setup(t)
	ctx := ctx10(t)
	migs, _ := Load(writeFiles(t, map[string]string{"0001_create_items.sql": m1}))
	if _, _, err := Up(ctx, e.migPool, e.opts(), migs, false); err != nil {
		t.Fatal(err)
	}
	// Biri elle RLS'i kapatıp politikayı silse ve yetkiyi alsa bile, sonraki up düzeltir.
	for _, q := range []string{
		"alter table " + e.schema + ".items disable row level security",
		"drop policy app_access on " + e.schema + ".items",
		"revoke all on " + e.schema + ".items from " + e.app,
	} {
		if _, err := e.admin.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	_, rep, err := Up(ctx, e.migPool, e.opts(), migs, false)
	if err != nil || len(rep.RLSEnabled) != 1 || len(rep.PoliciesCreated) != 1 {
		t.Fatalf("sapma düzeltilmeli: %+v %v", rep, err)
	}
	if _, err := e.appPool.Exec(ctx, "insert into items (name) values ('a')"); err != nil {
		t.Fatalf("yetki geri gelmeli: %v", err)
	}
}

// Depodaki migration'lar boş bir şemaya baştan sona uygulanabilmeli.
func TestIntegrationRepositoryMigrationsApply(t *testing.T) {
	e := setup(t)
	ctx := ctx10(t)
	migs, err := Load(filepath.Join("..", "..", "..", "migrations", "postgres"))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Up(ctx, e.migPool, e.opts(), migs, false); err != nil {
		t.Fatalf("depo migration'ları uygulanamadı: %v", err)
	}
	if _, _, err := Up(ctx, e.migPool, e.opts(), migs, false); err != nil {
		t.Fatalf("ikinci çalıştırma: %v", err)
	}
}

func (e *env) policyCmd(t *testing.T, table string) string {
	t.Helper()
	var cmd string
	if err := e.admin.QueryRow(context.Background(),
		"select cmd from pg_policies where schemaname = $1 and tablename = $2 and policyname = 'app_access'", e.schema, table).Scan(&cmd); err != nil {
		t.Fatalf("%s politikası: %v", table, err)
	}
	return cmd
}

func TestIntegrationReadOnlyMarker(t *testing.T) {
	e := setup(t)
	ctx := ctx10(t)
	dir := writeFiles(t, map[string]string{
		"0001_catalog.sql": "create table catalog (id int primary key, name text not null);\n" +
			"comment on table catalog is '@app:readonly katalog';\n" +
			"insert into catalog values (1, 'a');\n",
	})
	migs, _ := Load(dir)
	_, rep, err := Up(ctx, e.migPool, e.opts(), migs, false)
	if err != nil || len(rep.ReadOnly) != 1 || rep.ReadOnly[0] != "catalog" {
		t.Fatalf("salt okunur: %+v %v", rep, err)
	}
	if cmd := e.policyCmd(t, "catalog"); cmd != "SELECT" {
		t.Fatalf("politika SELECT olmalı: %s", cmd)
	}
	var n int
	if err := e.appPool.QueryRow(ctx, "select count(*) from catalog").Scan(&n); err != nil || n != 1 {
		t.Fatalf("uygulama okuyabilmeli: %d %v", n, err)
	}
	for _, q := range []string{"insert into catalog values (2, 'b')", "update catalog set name = 'x'", "delete from catalog"} {
		if _, err := e.appPool.Exec(ctx, q); err == nil {
			t.Errorf("salt okunur tabloda yapılamamalı: %s", q)
		}
	}

	// İşaret kaldırılırsa (yeni migration), politika ALL'a döner ve yazma açılır.
	if err := os.WriteFile(filepath.Join(dir, "0002_writable.sql"), []byte("comment on table catalog is 'artık yazılabilir';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	migs, _ = Load(dir)
	_, rep, err = Up(ctx, e.migPool, e.opts(), migs, false)
	if err != nil || len(rep.PoliciesReplaced) != 1 || len(rep.ReadOnly) != 0 {
		t.Fatalf("politika değişmeli: %+v %v", rep, err)
	}
	if cmd := e.policyCmd(t, "catalog"); cmd != "ALL" {
		t.Fatalf("politika ALL olmalı: %s", cmd)
	}
	if _, err := e.appPool.Exec(ctx, "insert into catalog values (2, 'b')"); err != nil {
		t.Fatalf("yazma açılmalı: %v", err)
	}
}

func TestIntegrationPolicyWithWrongRoleIsReplaced(t *testing.T) {
	e := setup(t)
	ctx := ctx10(t)
	migs, _ := Load(writeFiles(t, map[string]string{"0001_create_items.sql": m1}))
	if _, _, err := Up(ctx, e.migPool, e.opts(), migs, false); err != nil {
		t.Fatal(err)
	}
	// Biri politikayı anon'a açacak şekilde değiştirse bile düzeltilir.
	for _, q := range []string{
		"drop policy app_access on " + e.schema + ".items",
		"create policy app_access on " + e.schema + ".items for all to anon using (true)",
	} {
		if _, err := e.admin.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	_, rep, err := Up(ctx, e.migPool, e.opts(), migs, false)
	if err != nil || len(rep.PoliciesReplaced) != 1 {
		t.Fatalf("yanlış rollü politika düzeltilmeli: %+v %v", rep, err)
	}
	var roles string
	_ = e.admin.QueryRow(ctx, "select roles::text from pg_policies where schemaname = $1 and tablename = 'items'", e.schema).Scan(&roles)
	if roles != "{"+e.app+"}" {
		t.Fatalf("politika yalnızca uygulama rolüne olmalı: %s", roles)
	}
}
