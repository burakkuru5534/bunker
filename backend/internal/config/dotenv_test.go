package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDotEnv(t *testing.T) {
	in := `
# yorum
APP_ENV=local
export PORT = 9090
DATABASE_URL="postgres://u:p@h:5432/db?sslmode=verify-full&x=1"
QUOTED='a # b'
TRAILING=deger # açıklama
EMPTY=
WITH_EQ=a=b
`
	got, err := parseDotEnv(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"APP_ENV": "local", "PORT": "9090",
		"DATABASE_URL": "postgres://u:p@h:5432/db?sslmode=verify-full&x=1",
		"QUOTED":       "a # b", "TRAILING": "deger", "EMPTY": "", "WITH_EQ": "a=b",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, %q bekleniyordu", k, got[k], v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("fazla anahtar: %v", got)
	}
}

func TestParseDotEnvErrorsHideValues(t *testing.T) {
	for _, in := range []string{"SADECE_ANAHTAR", "1KEY=gizli1", "BAD-KEY=gizli2", "=gizli3"} {
		_, err := parseDotEnv(strings.NewReader("OK=1\n" + in))
		if err == nil || !strings.Contains(err.Error(), "satır 2") {
			t.Errorf("%q: satır 2 hatası bekleniyordu: %v", in, err)
		}
		if err != nil && strings.Contains(err.Error(), "gizli") {
			t.Errorf("hata değeri içermemeli: %v", err)
		}
	}
}

func TestReadDotEnvFile(t *testing.T) {
	dir := t.TempDir()
	m, err := ReadDotEnv(filepath.Join(dir, "yok.env"))
	if err != nil || len(m) != 0 {
		t.Fatalf("dosya yoksa boş harita: %v %v", m, err)
	}
	p := filepath.Join(dir, ".env")
	if err := os.WriteFile(p, []byte("PORT=1234\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err = ReadDotEnv(p)
	if err != nil || m["PORT"] != "1234" {
		t.Fatalf("okunmalı: %v %v", m, err)
	}
	if _, err := ReadDotEnv(dir); err == nil {
		t.Fatal("klasör okunamamalı")
	}
}

func TestWithFallbackPrefersRealEnv(t *testing.T) {
	get := WithFallback(env(map[string]string{"PORT": "1"}), map[string]string{"PORT": "2", "LOG_LEVEL": "debug"})
	if get("PORT") != "1" || get("LOG_LEVEL") != "debug" || get("YOK") != "" {
		t.Fatal("öncelik yanlış")
	}
}
