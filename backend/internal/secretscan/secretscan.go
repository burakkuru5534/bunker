// Package secretscan, depoya girmemesi gereken gizli değerleri arar
// (SAFE-010). Bulguda değerin kendisi asla yazılmaz: yalnızca dosya, satır
// ve tür. Kalıplar bilinen biçimlerdir (Supabase anahtarı, JWT, parolalı
// PostgreSQL adresi, özel anahtar, servis hesabı JSON'u); genel "entropi"
// taraması yapılmaz (yanlış alarm çok olur).
package secretscan

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

// Finding, bir bulgudur. Değer içermez.
type Finding struct {
	Path string
	Line int
	Kind string
}

// IgnoreMarker, satırı bilinçli olarak taramadan çıkarır (ör. belgelerdeki
// biçim örneği). Gerçek bir değerin yanına yazılmamalıdır.
const IgnoreMarker = "secretscan:ignore"

type rule struct {
	kind string
	re   *regexp.Regexp
	// ok, eşleşmenin zararsız olup olmadığını söyler (yer tutucu vb.).
	ok func(m []string) bool
}

// placeholderPasswords, testlerde ve belgelerde kullanılan sahte parolalar.
var placeholderPasswords = map[string]bool{
	"postgres": true, "p": true, "password": true, "pass": true, "parola": true,
	"sifre": true, "şifre": true, "cokgizli": true, "secret": true, "x": true,
}

var rules = []rule{
	{kind: "supabase-key", re: regexp.MustCompile(`sb_secret_[A-Za-z0-9_\-]{16,}`)},
	{kind: "jwt", re: regexp.MustCompile(`eyJ[A-Za-z0-9_\-]{10,}\.eyJ[A-Za-z0-9_\-]{10,}\.[A-Za-z0-9_\-]{10,}`)},
	{
		kind: "postgres-password",
		re:   regexp.MustCompile(`postgres(?:ql)?://([^:/\s"'` + "`" + `@]+):([^@\s"'` + "`" + `]+)@`),
		ok: func(m []string) bool {
			pw := m[2]
			return placeholderPasswords[strings.ToLower(pw)] ||
				strings.HasPrefix(pw, "<") || strings.HasPrefix(pw, "$") || strings.HasPrefix(pw, "{") ||
				strings.Trim(pw, "*.Xx") == ""
		},
	},
	{kind: "private-key", re: regexp.MustCompile(`-----BEGIN (?:[A-Z]+ )?PRIVATE KEY-----`)},
	{kind: "gcp-service-account", re: regexp.MustCompile(`"private_key_id"\s*:\s*"[0-9a-f]{20,}"`)},
	{
		kind: "secret-assignment",
		re:   regexp.MustCompile(`(?i)\b(SUPABASE_SECRET_KEY|SUPABASE_SERVICE_ROLE_KEY|SERVICE_ROLE_KEY|PGPASSWORD|DB_PASSWORD)\s*[=:]\s*["']?([^\s"'#<]{8,})`),
		ok: func(m []string) bool {
			v := m[2]
			return strings.HasPrefix(v, "$") || strings.HasPrefix(v, "{")
		},
	},
}

// Scan, bir dosyanın içeriğini tarar.
func Scan(path string, r io.Reader) ([]Finding, error) {
	var out []Finding
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		if strings.Contains(line, IgnoreMarker) {
			continue
		}
		for _, rl := range rules {
			for _, m := range rl.re.FindAllStringSubmatch(line, -1) {
				if rl.ok != nil && rl.ok(m) {
					continue
				}
				out = append(out, Finding{Path: path, Line: n, Kind: rl.kind})
			}
		}
	}
	return out, sc.Err()
}

// SkipPath, taranmayacak dosyalardır (ikili ve üretilmiş kilit dosyaları).
func SkipPath(path string) bool {
	lower := strings.ToLower(path)
	for _, ext := range []string{
		".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico", ".ttf", ".otf", ".woff", ".woff2",
		".jar", ".zip", ".pdf", ".so", ".dylib", ".exe",
	} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return strings.HasSuffix(lower, "go.sum") || strings.HasSuffix(lower, "pubspec.lock")
}
