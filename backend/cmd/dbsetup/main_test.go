package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/config"
)

func noenv(string) string { return "" }

func caFile(t *testing.T, dir string) string {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Test CA"},
		NotBefore: time.Now(), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true}
	der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	p := filepath.Join(dir, "ca.crt")
	_ = os.WriteFile(p, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644)
	return p
}

func TestInitNeverPrintsPassword(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	cert := caFile(t, dir)
	var out, errOut bytes.Buffer
	err := run([]string{"init", "-env", "test", "-ref", "abcdefghijklmnopqrst",
		"-host", "aws-0-eu-central-1.pooler.supabase.com", "-cert", cert, "-env-file", envFile}, noenv, &out, &errOut)
	if err != nil {
		t.Fatalf("init: %v\n%s", err, errOut.String())
	}

	values, err := config.ReadDotEnv(envFile)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(values["DATABASE_URL"])
	if err != nil {
		t.Fatal(err)
	}
	pw, _ := u.User.Password()
	if len(pw) != 32 || u.User.Username() != "bunker_test_app.abcdefghijklmnopqrst" || u.Query().Get("sslmode") != "verify-full" {
		t.Fatalf(".env adresi beklenen biçimde değil: %s", u.Redacted())
	}
	if strings.Contains(out.String(), pw) || strings.Contains(errOut.String(), pw) {
		t.Fatal("şifre çıktıya yazılmamalı")
	}
	if !strings.HasPrefix(out.String(), "-- Sığınak: bunker_test") || !strings.Contains(out.String(), "SCRAM-SHA-256$4096:") {
		t.Fatalf("stdout yalnızca SQL olmalı: %.80s", out.String())
	}
	if strings.Contains(out.String(), "CA sertifikası") || !strings.Contains(errOut.String(), "CN=Test CA") {
		t.Fatal("bilgi mesajları stderr'e gitmeli")
	}

	// İkinci init dolu adresi ezmez; -force ile yeni şifre üretir.
	if err := run([]string{"init", "-ref", "abcdefghijklmnopqrst", "-host", "aws-0-eu-central-1.pooler.supabase.com",
		"-cert", cert, "-env-file", envFile}, noenv, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("dolu DATABASE_URL korunmalı")
	}
	if err := run([]string{"init", "-ref", "abcdefghijklmnopqrst", "-host", "aws-0-eu-central-1.pooler.supabase.com",
		"-cert", cert, "-env-file", envFile, "-force"}, noenv, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	again, _ := config.ReadDotEnv(envFile)
	if again["DATABASE_URL"] == values["DATABASE_URL"] {
		t.Fatal("-force yeni şifre üretmeli")
	}
}

func TestInitValidation(t *testing.T) {
	dir := t.TempDir()
	cases := map[string][]string{
		"komut yok":        {},
		"bilinmeyen komut": {"sil"},
		"bilinmeyen ortam": {"init", "-env", "staging"},
		"ref yok":          {"init", "-host", "x.pooler.supabase.com", "-cert", caFile(t, dir)},
		"sertifika yok":    {"init", "-ref", "abcdefghijklmnopqrst", "-host", "x.pooler.supabase.com", "-cert", filepath.Join(dir, "yok.crt")},
		"bozuk bayrak":     {"init", "-bilinmeyen"},
	}
	for name, args := range cases {
		if err := run(args, noenv, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
			t.Errorf("%s: hata bekleniyordu", name)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".env")); err == nil {
		t.Fatal("başarısız init .env yazmamalı")
	}
}

func TestCheckRequiresURL(t *testing.T) {
	dir := t.TempDir()
	err := run([]string{"check", "-env-file", filepath.Join(dir, ".env")}, noenv, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("DATABASE_URL hatası bekleniyordu: %v", err)
	}
	// Katı modda şifresiz bağlantı hiç denenmeden reddedilir.
	var out bytes.Buffer
	err = run([]string{"check"}, func(k string) string {
		if k == "DATABASE_URL" {
			return "postgres://u:cokgizli@db.example.com/db?sslmode=disable"
		}
		return ""
	}, &out, &bytes.Buffer{})
	if err == nil || !strings.Contains(out.String(), "verify-full") || strings.Contains(out.String(), "cokgizli") {
		t.Fatalf("verify-full hatası bekleniyordu: %v %s", err, out.String())
	}
}
