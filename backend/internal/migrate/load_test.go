package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestLoadSortsAndParses(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"0002_add_index.sql":    "-- migrate:no-transaction\ncreate index concurrently items_name_idx on items (name);\n",
		"0001_create_items.sql": "create table items (id bigint primary key);\r\n",
		"README.md":             "yok sayılır",
		".gitkeep":              "",
	})
	migs, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(migs) != 2 || migs[0].Version != 1 || migs[1].Version != 2 || migs[0].Name != "create_items" {
		t.Fatalf("sıra/ad yanlış: %+v", migs)
	}
	if migs[0].NoTx || !migs[1].NoTx {
		t.Fatal("no-transaction yönergesi algılanmalı")
	}
	if strings.Contains(migs[0].SQL, "\r") {
		t.Fatal("CRLF normalleştirilmeli")
	}
}

func TestChecksumIgnoresLineEndingsAndBOM(t *testing.T) {
	a := parse(1, "x", "0001_x.sql", []byte("create table a (id int);\nselect 1;\n"))
	b := parse(1, "x", "0001_x.sql", []byte("\uFEFFcreate table a (id int);\r\nselect 1;\r\n"))
	c := parse(1, "x", "0001_x.sql", []byte("create table a (id bigint);\nselect 1;\n"))
	if a.Checksum != b.Checksum || a.Checksum == c.Checksum || len(a.Checksum) != 64 {
		t.Fatalf("sağlama: %s %s %s", a.Checksum, b.Checksum, c.Checksum)
	}
}

func TestLoadRejectsBadFiles(t *testing.T) {
	cases := map[string]map[string]string{
		"kötü ad":        {"0001-create.sql": "select 1;"},
		"büyük harf":     {"0001_Create.sql": "select 1;"},
		"sürüm 0":        {"0000_x.sql": "select 1;"},
		"tekrar sürüm":   {"0001_a.sql": "select 1;", "0001_b.sql": "select 1;"},
		"boş":            {"0001_a.sql": "-- yalnızca yorum\n/* blok */\n"},
		"şema adı":       {"0001_a.sql": "create table bunker_test.users (id int);"},
		"search_path":    {"0001_a.sql": "set search_path = public;"},
		"set local role": {"0001_a.sql": "SET LOCAL ROLE postgres;"},
		"grant":          {"0001_a.sql": "create table a (id int); grant select on a to anon;"},
		"revoke":         {"0001_a.sql": "revoke all on a from public;"},
		"create schema":  {"0001_a.sql": "create schema x;"},
		"begin":          {"0001_a.sql": "begin;\ncreate table a (id int);\ncommit;"},
		"rls kapatma":    {"0001_a.sql": "alter table a disable row level security;"},
		"bypassrls":      {"0001_a.sql": "alter role x bypassrls;"},
		"sürüm tablosu":  {"0001_a.sql": "delete from schema_migrations;"},
		"politika adı":   {"0001_a.sql": "drop policy app_access on a;"},
		"concurrently":   {"0001_a.sql": "create index concurrently i on a (id);"},
		"notx çok ifade": {"0001_a.sql": "-- migrate:no-transaction\ncreate index concurrently i on a (id);\ncreate index concurrently j on a (id);"},
	}
	for name, files := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			for f, body := range files {
				p := filepath.Join(dir, f)
				_ = os.MkdirAll(filepath.Dir(p), 0o755)
				if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := Load(dir); err == nil {
				t.Fatal("hata bekleniyordu")
			}
		})
	}
	if _, err := Load(filepath.Join(t.TempDir(), "yok")); err == nil {
		t.Fatal("olmayan klasör hata vermeli")
	}
}

func TestLintIgnoresComments(t *testing.T) {
	m := parse(1, "a", "0001_a.sql", []byte(`-- grant ve bunker_test yorumda serbest
/* begin; commit; */
create table a (id int); -- revoke burada da serbest
`))
	if errs := Lint(m); len(errs) != 0 {
		t.Fatalf("yorumdaki kelimeler yok sayılmalı: %v", errs)
	}
	nt := parse(2, "b", "0002_b.sql", []byte("-- migrate:no-transaction\ncreate index concurrently i on a (id);\n"))
	if errs := Lint(nt); len(errs) != 0 {
		t.Fatalf("tek ifadeli no-transaction geçerli: %v", errs)
	}
}

func TestLoadReportsAllErrors(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"0001_a.sql": "grant select on a to b;",
		"0002_b.sql": "create schema x;",
	})
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "0001_a.sql") || !strings.Contains(err.Error(), "0002_b.sql") {
		t.Fatalf("tüm hatalar birlikte: %v", err)
	}
}

func TestPlan(t *testing.T) {
	m1 := parse(1, "a", "0001_a.sql", []byte("create table a (id int);"))
	m2 := parse(2, "b", "0002_b.sql", []byte("create table b (id int);"))
	m3 := parse(3, "c", "0003_c.sql", []byte("create table c (id int);"))
	rec := func(m Migration) Applied {
		return Applied{Version: m.Version, Name: m.Name, Checksum: m.Checksum, AppliedAt: time.Now()}
	}

	st, err := plan([]Applied{rec(m1)}, []Migration{m1, m2, m3})
	if err != nil || len(st.Applied) != 1 || len(st.Pending) != 2 || st.Pending[0].Version != 2 {
		t.Fatalf("bekleyenler: %+v %v", st, err)
	}

	bad := map[string]struct {
		applied []Applied
		migs    []Migration
		want    string
	}{
		"dosya silinmiş":   {[]Applied{rec(m1), rec(m2)}, []Migration{m1}, "dosyası yok"},
		"dosya değişmiş":   {[]Applied{{Version: 1, Name: "a", Checksum: "eski"}}, []Migration{m1}, "değiştirilmiş"},
		"yeniden adlandı":  {[]Applied{{Version: 1, Name: "eski_ad", Checksum: m1.Checksum}}, []Migration{m1}, "yeniden adlandırılmamalı"},
		"sıra dışı ekleme": {[]Applied{rec(m1), rec(m3)}, []Migration{m1, m2, m3}, "sıra dışı"},
	}
	for name, tc := range bad {
		if _, err := plan(tc.applied, tc.migs); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %q bekleniyordu: %v", name, tc.want, err)
		}
	}
}

func TestNextFile(t *testing.T) {
	dir := writeFiles(t, map[string]string{"0001_a.sql": "select 1;", "0007_b.sql": "select 1;", "notlar.txt": ""})
	file, body, err := NextFile(dir, "create_users")
	if err != nil || file != "0008_create_users.sql" || !strings.HasPrefix(body, "-- 0008 create users") {
		t.Fatalf("%s %v\n%s", file, err, body)
	}
	// Şablon tek başına boş sayılır (doldurulmadan CI'dan geçemez).
	if errs := Lint(parse(8, "create_users", file, []byte(body))); len(errs) != 1 || !strings.Contains(errs[0].Error(), "boş") {
		t.Fatalf("boş şablon reddedilmeli: %v", errs)
	}
	for _, bad := range []string{"", "Create", "a-b", "a__b", "_a", "ğ"} {
		if _, _, err := NextFile(dir, bad); err == nil {
			t.Errorf("%q reddedilmeli", bad)
		}
	}
	if _, _, err := NextFile(filepath.Join(dir, "yok"), "a"); err == nil {
		t.Fatal("olmayan klasör")
	}
	full := writeFiles(t, map[string]string{"9999_son.sql": "select 1;"})
	if _, _, err := NextFile(full, "a"); err == nil {
		t.Fatal("9999 üst sınır")
	}
}

// Depodaki gerçek migration'lar her zaman kurallara uymalı (CI).
func TestRepositoryMigrationsAreValid(t *testing.T) {
	if _, err := Load(filepath.Join("..", "..", "..", "migrations", "postgres")); err != nil {
		t.Fatalf("migrations/postgres: %v", err)
	}
}
