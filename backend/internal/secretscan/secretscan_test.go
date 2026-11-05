package secretscan

import (
	"strings"
	"testing"
)

// Sahte değerler parçalardan kurulur: bu dosyanın kendisi taramaya takılmasın.
var (
	fakeSupabase = "sb_" + "secret_" + strings.Repeat("Ab3", 8)
	fakeJWT      = "ey" + "J" + strings.Repeat("a", 20) + ".ey" + "J" + strings.Repeat("b", 20) + "." + strings.Repeat("c", 20)
	fakePGURL    = "postgres" + "://bunker_test_app.abcd:" + "Zx9kQ2mP7wLr" + "@aws-0-eu-central-1.pooler.supabase.com:5432/postgres"
	fakePEM      = "-----BEGIN " + "PRIVATE KEY-----"
	fakeSA       = `"private_` + `key_id": "` + strings.Repeat("0f", 20) + `"`
	fakeAssign   = "SUPABASE_" + "SECRET_KEY=" + "q8Zr2LmN0pXv"
)

func scan(t *testing.T, text string) []Finding {
	t.Helper()
	f, err := Scan("dosya.txt", strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestScanFindsKnownSecretFormats(t *testing.T) {
	for kind, line := range map[string]string{
		"supabase-key":        "key := \"" + fakeSupabase + "\"",
		"jwt":                 "Authorization: Bearer " + fakeJWT,
		"postgres-password":   "DATABASE_URL=" + fakePGURL,
		"private-key":         fakePEM,
		"gcp-service-account": fakeSA,
		"secret-assignment":   fakeAssign,
	} {
		f := scan(t, "ilk satır\n"+line+"\n")
		if len(f) != 1 || f[0].Kind != kind || f[0].Line != 2 || f[0].Path != "dosya.txt" {
			t.Errorf("%s: %+v", kind, f)
		}
	}
}

func TestScanAllowsPlaceholders(t *testing.T) {
	for _, line := range []string{
		"postgres" + "://u:cokgizli@h/db",
		"postgres" + "://kullanıcı:şifre@sunucu:5432/postgres?sslmode=verify-full",
		"postgres" + "://postgres:postgres@localhost:5432/postgres?sslmode=disable",
		"postgres" + "://bunker_test_app.<ref>:<parola>@aws-0-eu-central-1.pooler.supabase.com:5432/postgres",
		"postgres" + "://user:${DB_PASSWORD}@host/db",
		"postgres" + "://user:****@host/db",
		"postgres" + "://host/db?sslmode=verify-full", // parolasız
		"SUPABASE_" + "SECRET_KEY=${{ secrets.X }}",
		"SUPABASE_" + "SECRET_KEY=<anahtar>",
		"sb_" + "secret_kisa",
		fakeSupabase + " // " + IgnoreMarker,
	} {
		if f := scan(t, line); len(f) != 0 {
			t.Errorf("yanlış alarm: %q → %+v", line, f)
		}
	}
}

func TestScanCountsEachOccurrence(t *testing.T) {
	f := scan(t, fakeSupabase+" "+fakeSupabase+"\n\n"+fakePEM)
	if len(f) != 3 || f[2].Line != 3 {
		t.Fatalf("%+v", f)
	}
}

func TestSkipPath(t *testing.T) {
	for p, want := range map[string]bool{
		"apps/mobile/assets/fonts/x.ttf": true, "a/b.PNG": true, "backend/go.sum": true,
		"apps/mobile/pubspec.lock": true, "backend/main.go": false, "docs/security.md": false,
	} {
		if SkipPath(p) != want {
			t.Errorf("%s: %v", p, !want)
		}
	}
}
