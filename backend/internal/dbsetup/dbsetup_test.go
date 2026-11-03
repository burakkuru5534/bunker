package dbsetup

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestScramVerifierMatchesReference(t *testing.T) {
	// Referans: Python hashlib ile bağımsız hesaplandı (pbkdf2_hmac + hmac).
	const want = "SCRAM-SHA-256$4096:AAECAwQFBgcICQoLDA0ODw==$zHCdol2044/ZyWzPLi7oxApCkamKw9Z+E4U/QApd/5Y=:dd5peBOitVnLNFu7VmwP+HiDaaw4OUCv396eVCWhYiE="
	salt := make([]byte, 16)
	for i := range salt {
		salt[i] = byte(i)
	}
	got, err := ScramVerifier("pencil", salt, 4096)
	if err != nil || got != want {
		t.Fatalf("doğrulayıcı = %q (%v)", got, err)
	}
	if _, err := ScramVerifier("şifre", salt, 4096); err == nil {
		t.Fatal("ASCII dışı şifre reddedilmeli")
	}
	if _, err := ScramVerifier("pencil", salt[:8], 4096); err == nil {
		t.Fatal("kısa tuz reddedilmeli")
	}
	a, _ := NewScramVerifier("pencil")
	b, _ := NewScramVerifier("pencil")
	if a == b || !verifierRe.MatchString(a) {
		t.Fatal("rastgele tuz her seferinde farklı olmalı")
	}
}

func TestNewPassword(t *testing.T) {
	seen := map[string]bool{}
	for range 50 {
		p, err := NewPassword(32)
		if err != nil || len(p) != 32 {
			t.Fatalf("şifre: %q %v", p, err)
		}
		for _, c := range p {
			if !strings.ContainsRune(passwordAlphabet, c) {
				t.Fatalf("alfabe dışı karakter %q", c)
			}
		}
		if url.QueryEscape(p) != p {
			t.Fatal("şifre URL kodlaması gerektirmemeli")
		}
		if seen[p] {
			t.Fatal("tekrarlanan şifre")
		}
		seen[p] = true
	}
	if _, err := NewPassword(16); err == nil {
		t.Fatal("kısa şifre reddedilmeli")
	}
}

func TestSetupSQL(t *testing.T) {
	v, _ := NewScramVerifier("pencil")
	sql, err := SetupSQL(EnvTest, KindApp, v)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"create role bunker_test_app nologin", "password '" + v + "'",
		"set search_path = bunker_test", "statement_timeout = '5s'", "lock_timeout = '3s'",
		"idle_in_transaction_session_timeout = '10s'", "connection limit 10",
		"create schema if not exists bunker_test", "grant usage on schema bunker_test to bunker_test_app",
		"revoke all on schema bunker_test from public",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("SQL'de %q yok", want)
		}
	}
	// Rol nitelikleri yalnızca login, bağlantı sınırı ve şifre; CREATE verilmez.
	start := strings.Index(sql, "alter role bunker_test_app with")
	stmt := sql[start : start+strings.Index(sql[start:], ";")]
	stmt = strings.Join(strings.Fields(strings.ReplaceAll(stmt, v, "<v>")), " ")
	if stmt != "alter role bunker_test_app with login connection limit 10 password '<v>'" {
		t.Fatalf("rol nitelikleri: %s", stmt)
	}
	for _, bad := range []string{"{{", "pencil", "grant create", "grant all"} {
		if strings.Contains(strings.ToLower(sql), bad) {
			t.Errorf("SQL'de %q olmamalı", bad)
		}
	}
	prod, _ := SetupSQL(EnvProd, KindApp, v)
	if !strings.Contains(prod, "bunker_prod_app") || strings.Contains(prod, "bunker_test") {
		t.Fatal("prod SQL yalnızca prod adlarını içermeli")
	}
	for _, bad := range []string{"pencil", "SCRAM-SHA-256$1:x'; drop role x; --$a:b"} {
		if _, err := SetupSQL(EnvTest, KindApp, bad); err == nil {
			t.Errorf("geçersiz doğrulayıcı reddedilmeli: %q", bad)
		}
	}
	if _, err := setupSQL("Bad;", EnvTest.Spec(KindApp), v); err == nil {
		t.Fatal("geçersiz ad reddedilmeli")
	}
	if _, err := ParseEnv("staging"); err == nil {
		t.Fatal("bilinmeyen ortam reddedilmeli")
	}
	if e, _ := ParseEnv("prod"); e.Schema() != "bunker_prod" || e.Role() != "bunker_prod_app" {
		t.Fatal("prod adları yanlış")
	}

	// Migration rolü: CREATE yetkisi, daha uzun sorgu süresi, ayrı .env anahtarı.
	mig, err := SetupSQL(EnvTest, KindMigrator, v)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"create role bunker_test_migrator nologin", "grant usage, create on schema bunker_test to bunker_test_migrator",
		"statement_timeout = '1min'", "lock_timeout = '5s'", "connection limit 2",
	} {
		if !strings.Contains(mig, want) {
			t.Errorf("migrator SQL'de %q yok", want)
		}
	}
	if strings.Contains(mig, "bunker_test_app") {
		t.Error("migrator SQL uygulama rolüne dokunmamalı")
	}
	ms := EnvProd.Spec(KindMigrator)
	if ms.Name != "bunker_prod_migrator" || ms.EnvKey != "MIGRATION_DATABASE_URL" || !ms.CanCreate() || EnvProd.Spec(KindApp).CanCreate() {
		t.Fatalf("migrator spec yanlış: %+v", ms)
	}
	if _, err := ParseKind("admin"); err == nil {
		t.Fatal("bilinmeyen rol türü reddedilmeli")
	}
	if k, _ := ParseKind("migrator"); k != KindMigrator {
		t.Fatal("migrator ayrıştırılmalı")
	}
}

func TestUpsertEnvVar(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".env")

	// Dosya yoksa oluşturulur.
	if err := UpsertEnvVar(p, "DATABASE_URL", "postgres://a", false); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != "DATABASE_URL=postgres://a\n" {
		t.Fatalf("yeni dosya: %q", b)
	}
	if runtime.GOOS != "windows" {
		if st, _ := os.Stat(p); st.Mode().Perm() != 0o600 {
			t.Fatalf("izin 0600 olmalı: %v", st.Mode().Perm())
		}
	}

	// Dolu değer force olmadan korunur, hata değeri içermez.
	err := UpsertEnvVar(p, "DATABASE_URL", "postgres://gizli", false)
	if !errors.Is(err, ErrAlreadySet) || strings.Contains(err.Error(), "gizli") {
		t.Fatalf("ErrAlreadySet bekleniyordu: %v", err)
	}

	// Boş satır yerinde doldurulur; diğer satırlar ve sıra korunur (CRLF dahil).
	if err := os.WriteFile(p, []byte("APP_ENV=local\r\n# yorum\r\nexport DATABASE_URL=\r\nLOG_LEVEL=debug"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := UpsertEnvVar(p, "DATABASE_URL", "postgres://b", false); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != "APP_ENV=local\n# yorum\nDATABASE_URL=postgres://b\nLOG_LEVEL=debug\n" {
		t.Fatalf("yerinde güncelleme: %q", b)
	}

	// force ile yenilenir; anahtar yoksa sona eklenir.
	if err := UpsertEnvVar(p, "DATABASE_URL", "postgres://c", true); err != nil {
		t.Fatal(err)
	}
	if err := UpsertEnvVar(p, "DB_SCHEMA", "bunker_test", false); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != "APP_ENV=local\n# yorum\nDATABASE_URL=postgres://c\nLOG_LEVEL=debug\nDB_SCHEMA=bunker_test\n" {
		t.Fatalf("force/ekleme: %q", b)
	}
	if err := UpsertEnvVar(dir, "X", "y", false); err == nil {
		t.Fatal("klasör yazılamamalı")
	}
	if _, err := os.Stat(p + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("geçici dosya kalmamalı")
	}
}

func TestConnParams(t *testing.T) {
	ok := ConnParams{Role: "bunker_test_app", Password: "Abc23", ProjectRef: "abcdefghijklmnopqrst",
		Host: "aws-0-eu-central-1.pooler.supabase.com", Port: 5432, SSLMode: "verify-full", CertFile: `certs\supabase-ca.crt`}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(ok.URL())
	if err != nil {
		t.Fatal(err)
	}
	pw, _ := u.User.Password()
	if u.User.Username() != "bunker_test_app.abcdefghijklmnopqrst" || pw != "Abc23" ||
		u.Host != "aws-0-eu-central-1.pooler.supabase.com:5432" || u.Path != "/postgres" ||
		u.Query().Get("sslmode") != "verify-full" || u.Query().Get("sslrootcert") != "certs/supabase-ca.crt" {
		t.Fatalf("adres yanlış: %s", u.Redacted())
	}

	local := ConnParams{Role: "r", Password: "p", Host: "127.0.0.1", Port: 55432, SSLMode: "disable"}
	if err := local.Validate(); err != nil || strings.Contains(local.URL(), "sslrootcert") || !strings.Contains(local.URL(), "r:p@127.0.0.1:55432") {
		t.Fatalf("yerel: %v %s", err, local.URL())
	}

	bad := map[string]ConnParams{
		"ref yok":           {Role: "r", Password: "p", Host: "x.pooler.supabase.com", Port: 5432, SSLMode: "verify-full", CertFile: "c"},
		"supabase require":  {Role: "r", Password: "p", ProjectRef: "abcdefghijklmnopqrst", Host: "x.pooler.supabase.com", Port: 5432, SSLMode: "require"},
		"bozuk ref":         {Role: "r", Password: "p", ProjectRef: "ABC", Host: "h", Port: 5432, SSLMode: "disable"},
		"bozuk host":        {Role: "r", Password: "p", Host: "h/../x", Port: 5432, SSLMode: "disable"},
		"port":              {Role: "r", Password: "p", Host: "h", Port: 0, SSLMode: "disable"},
		"sertifika yok":     {Role: "r", Password: "p", Host: "h", Port: 1, SSLMode: "verify-full"},
		"bilinmeyen ssl":    {Role: "r", Password: "p", Host: "h", Port: 1, SSLMode: "prefer"},
		"şifre yok":         {Role: "r", Host: "h", Port: 1, SSLMode: "disable"},
		"supabase.co da da": {Role: "r", Password: "p", Host: "db.x.supabase.co", Port: 5432, SSLMode: "disable"},
	}
	for name, p := range bad {
		if err := p.Validate(); err == nil {
			t.Errorf("%s: reddedilmeli", name)
		}
	}
}

func writeCert(t *testing.T, isCA bool) string {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Test Root 2021 CA"},
		NotBefore: time.Now(), NotAfter: time.Date(2031, 4, 26, 0, 0, 0, 0, time.UTC),
		IsCA: isCA, BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "ca.crt")
	if err := os.WriteFile(p, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestInspectCert(t *testing.T) {
	info, err := InspectCert(writeCert(t, true))
	if err != nil || info.Subject != "CN=Test Root 2021 CA" || !info.IsCA || info.NotAfter != "2031-04-26" || len(info.SHA256) != 64 {
		t.Fatalf("sertifika bilgisi: %+v %v", info, err)
	}
	if info, _ := InspectCert(writeCert(t, false)); info.IsCA {
		t.Fatal("CA olmayan sertifika")
	}
	dir := t.TempDir()
	notPEM := filepath.Join(dir, "x.crt")
	_ = os.WriteFile(notPEM, []byte("merhaba"), 0o644)
	badDER := filepath.Join(dir, "y.crt")
	_ = os.WriteFile(badDER, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte{1, 2, 3}}), 0o644)
	for _, p := range []string{filepath.Join(dir, "yok.crt"), notPEM, badDER} {
		if _, err := InspectCert(p); err == nil {
			t.Errorf("%s reddedilmeli", p)
		}
	}
}

func TestResultHelpers(t *testing.T) {
	if Failed([]Result{{Status: OK}, {Status: Warn}}) || !Failed([]Result{{Status: Fail}}) {
		t.Fatal("Failed yanlış")
	}
	if OK.Symbol() != "✓" || Warn.Symbol() != "!" || Fail.Symbol() != "✗" {
		t.Fatal("simgeler yanlış")
	}
}
