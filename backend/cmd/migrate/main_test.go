package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func noenv(string) string { return "" }

func TestNewAndLint(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	if err := run([]string{"new", "create_users", "-dir", dir}, noenv, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	path := strings.TrimSpace(out.String())
	if filepath.Base(path) != "0001_create_users.sql" {
		t.Fatalf("dosya: %s", path)
	}
	// Doldurulmamış şablon lint'ten geçmez.
	if err := run([]string{"lint", "-dir", dir}, noenv, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("boş şablon reddedilmeli")
	}
	if err := os.WriteFile(path, []byte("create table users (id uuid primary key);\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := run([]string{"lint", "-dir", dir}, noenv, &out, &bytes.Buffer{}); err != nil || !strings.Contains(out.String(), "1 migration") {
		t.Fatalf("lint: %v %s", err, out.String())
	}
	// Aynı ad ikinci kez → 0002; var olan dosyanın üzerine yazılmaz.
	out.Reset()
	if err := run([]string{"new", "create_users", "-dir", dir}, noenv, &out, &bytes.Buffer{}); err != nil || !strings.Contains(out.String(), "0002_create_users.sql") {
		t.Fatalf("ikinci new: %v %s", err, out.String())
	}
}

func TestUsageAndGuards(t *testing.T) {
	dir := t.TempDir()
	cases := map[string][]string{
		"komut yok":           {},
		"bilinmeyen komut":    {"down"},
		"new adsız":           {"new"},
		"new bayrakla başlar": {"new", "-dir", dir},
		"new kötü ad":         {"new", "Kötü", "-dir", dir},
		"bilinmeyen ortam":    {"status", "-env", "staging", "-dir", dir},
		"prod onaysız":        {"up", "-env", "prod", "-dir", dir},
		"bağlantı adresi yok": {"status", "-dir", dir, "-env-file", filepath.Join(dir, ".env")},
		"klasör yok":          {"lint", "-dir", filepath.Join(dir, "yok")},
		"bozuk bayrak":        {"status", "-bilinmeyen"},
		"geçersiz migration":  {"status", "-dir", writeBad(t)},
	}
	for name, args := range cases {
		if err := run(args, noenv, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
			t.Errorf("%s: hata bekleniyordu", name)
		}
	}
	err := run([]string{"up", "-env", "prod", "-dir", dir}, noenv, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "-yes-prod") {
		t.Fatalf("prod koruması: %v", err)
	}
	err = run([]string{"status", "-dir", dir, "-env-file", filepath.Join(dir, ".env")}, noenv, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "MIGRATION_DATABASE_URL") || !strings.Contains(err.Error(), "-role migrator") {
		t.Fatalf("adres yok mesajı yol göstermeli: %v", err)
	}
}

func TestStrictRejectsPlaintextBeforeConnecting(t *testing.T) {
	dir := t.TempDir()
	err := run([]string{"status", "-dir", dir}, func(k string) string {
		if k == "MIGRATION_DATABASE_URL" {
			return "postgres://u:cokgizli@db.example.com/postgres?sslmode=require"
		}
		return ""
	}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "verify-full") || strings.Contains(err.Error(), "cokgizli") {
		t.Fatalf("verify-full zorunlu: %v", err)
	}
}

func writeBad(t *testing.T) string {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "0001_a.sql"), []byte("grant all on x to y;"), 0o644)
	return dir
}
