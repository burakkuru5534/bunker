// Package migrate, sürümlü SQL migration'larını hedef şemaya uygular (DB-002).
//
// Kurallar (docs/architecture.md §11, ADR-0009):
//   - Dosyalar migrations/postgres/NNNN_ad.sql; yalnızca ileri yönlü
//     (geri alma = yeni migration). Uygulanmış dosya değiştirilemez (sağlama).
//   - Dosyalar şemadan ve ortamdan bağımsızdır: şema adı, search_path, rol,
//     GRANT/REVOKE, BEGIN/COMMIT içermez. Bunları araç yönetir.
//   - Her migration kendi transaction'ında çalışır; ilk satırı
//     "-- migrate:no-transaction" olan dosya (ör. CREATE INDEX CONCURRENTLY)
//     transaction dışında ve tek ifade olarak çalışır.
//   - Migration rolüyle (bunker_<ortam>_migrator) uygulanır; sonunda uygulama
//     rolüne yetkiler ve RLS politikası eşitlenir (ADR-0012).
package migrate

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// NoTxDirective, dosyanın transaction dışında çalışacağını belirtir.
const NoTxDirective = "-- migrate:no-transaction"

// Migration, tek bir migration dosyasıdır.
type Migration struct {
	Version  int
	Name     string
	File     string
	SQL      string // satır sonları LF'ye çevrilmiş içerik
	Checksum string // SQL'in SHA-256'sı (hex)
	NoTx     bool
}

var (
	fileRe = regexp.MustCompile(`^(\d{4})_([a-z0-9]+(?:_[a-z0-9]+)*)\.sql$`)
	nameRe = regexp.MustCompile(`^[a-z0-9]+(?:_[a-z0-9]+)*$`)
)

// Load, klasördeki migration'ları okur, denetler ve sürüme göre sıralar.
// .sql dışındaki dosyalar (README.md, .gitkeep) yok sayılır; adı kurala
// uymayan .sql dosyası hatadır (sessizce atlanmasın).
func Load(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("migration klasörü okunamadı: %w", err)
	}
	var (
		out  []Migration
		errs []error
		seen = map[int]string{}
	)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		m := fileRe.FindStringSubmatch(e.Name())
		if m == nil {
			errs = append(errs, fmt.Errorf("%s: ad NNNN_kucuk_harf_ad.sql biçiminde olmalı", e.Name()))
			continue
		}
		version, _ := strconv.Atoi(m[1])
		if version == 0 {
			errs = append(errs, fmt.Errorf("%s: sürüm 0001'den başlar", e.Name()))
			continue
		}
		if prev, ok := seen[version]; ok {
			errs = append(errs, fmt.Errorf("%s: sürüm %04d zaten %s dosyasında", e.Name(), version, prev))
			continue
		}
		seen[version] = e.Name()

		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", e.Name(), err))
			continue
		}
		mig := parse(version, m[2], e.Name(), raw)
		if lintErrs := Lint(mig); len(lintErrs) > 0 {
			errs = append(errs, lintErrs...)
			continue
		}
		out = append(out, mig)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	slices.SortFunc(out, func(a, b Migration) int { return a.Version - b.Version })
	return out, nil
}

func parse(version int, name, file string, raw []byte) Migration {
	text := strings.TrimPrefix(string(raw), "\uFEFF")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	sum := sha256.Sum256([]byte(text))
	return Migration{
		Version: version, Name: name, File: file, SQL: text,
		Checksum: hex.EncodeToString(sum[:]),
		NoTx:     firstLine(text) == NoTxDirective,
	}
}

func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			return t
		}
	}
	return ""
}

var (
	lineComment  = regexp.MustCompile(`--[^\n]*`)
	blockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
)

// stripComments, yorumları çıkarır (kural denetimi için; yaklaşık).
func stripComments(sql string) string {
	return lineComment.ReplaceAllString(blockComment.ReplaceAllString(sql, " "), " ")
}

type rule struct {
	re  *regexp.Regexp
	msg string
}

var rules = []rule{
	{regexp.MustCompile(`(?i)\bbunker_(test|prod)\b`), "ortam/şema adı içeremez; dosyalar şemadan bağımsız yazılır"},
	{regexp.MustCompile(`(?i)\bset\s+(session\s+|local\s+)?(search_path|role|session\s+authorization)\b`), "search_path veya rol değiştirilemez"},
	{regexp.MustCompile(`(?i)\b(grant|revoke)\b`), "GRANT/REVOKE kullanılamaz; uygulama rolü yetkileri `migrate up` sonunda verilir"},
	{regexp.MustCompile(`(?i)\b(create|drop|alter)\s+schema\b`), "şema oluşturulamaz/değiştirilemez; şema dbsetup ile yönetilir"},
	{regexp.MustCompile(`(?im)^\s*(begin|commit|rollback|start\s+transaction)\s*;`), "BEGIN/COMMIT kullanılamaz; transaction'ı araç yönetir"},
	{regexp.MustCompile(`(?i)\bdisable\s+row\s+level\s+security\b`), "RLS kapatılamaz (ADR-0012)"},
	{regexp.MustCompile(`(?i)\bno\s+force\s+row\s+level\s+security\b|\bbypassrls\b`), "RLS atlatılamaz (ADR-0012)"},
	{regexp.MustCompile(`(?i)\bschema_migrations\b`), "schema_migrations tablosuna dokunulamaz"},
	{regexp.MustCompile(`(?i)\bapp_access\b`), "app_access politikası araç tarafından yönetilir"},
}

var concurrently = regexp.MustCompile(`(?i)\bconcurrently\b`)

// Lint, dosya kurallarını denetler.
func Lint(m Migration) []error {
	var errs []error
	fail := func(msg string) { errs = append(errs, fmt.Errorf("%s: %s", m.File, msg)) }

	code := stripComments(m.SQL)
	if strings.TrimSpace(strings.ReplaceAll(code, ";", "")) == "" {
		fail("boş migration (yalnızca yorum)")
		return errs
	}
	for _, r := range rules {
		if r.re.MatchString(code) {
			fail(r.msg)
		}
	}
	if concurrently.MatchString(code) && !m.NoTx {
		fail("CONCURRENTLY için ilk satır " + NoTxDirective + " olmalı")
	}
	if m.NoTx && statementCount(code) > 1 {
		fail(NoTxDirective + " dosyası tek ifade içermeli (PostgreSQL çoklu ifadeyi örtük transaction'da çalıştırır)")
	}
	return errs
}

func statementCount(code string) int {
	n := 0
	for _, part := range strings.Split(code, ";") {
		if strings.TrimSpace(part) != "" {
			n++
		}
	}
	return n
}

// NextFile, klasördeki en büyük sürümden bir sonraki dosya adını ve şablonunu
// döndürür. Yazmaz (cmd/migrate new yazar).
func NextFile(dir, name string) (string, string, error) {
	if !nameRe.MatchString(name) {
		return "", "", fmt.Errorf("ad küçük harf, rakam ve _ içermeli: %q", name)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", "", fmt.Errorf("migration klasörü okunamadı: %w", err)
	}
	maxV := 0
	for _, e := range entries {
		if m := fileRe.FindStringSubmatch(e.Name()); m != nil {
			v, _ := strconv.Atoi(m[1])
			maxV = max(maxV, v)
		}
	}
	if maxV >= 9999 {
		return "", "", errors.New("sürüm üst sınırı (9999) aşıldı")
	}
	file := fmt.Sprintf("%04d_%s.sql", maxV+1, name)
	body := fmt.Sprintf(`-- %04d %s
-- Kurallar (migrations/postgres/README.md): şema adı, search_path, GRANT/REVOKE,
-- BEGIN/COMMIT kullanmayın. Uygulama rolü yetkileri ve RLS politikası
-- 'migrate up' sonunda otomatik verilir. Uygulandıktan sonra bu dosyayı
-- değiştirmeyin; düzeltme için yeni migration ekleyin.

`, maxV+1, strings.ReplaceAll(name, "_", " "))
	return file, body, nil
}
