package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// ReadDotEnv, yerel geliştirme için .env dosyasını okur. Dosya yoksa boş
// harita döner. Biçim: KEY=VALUE satırları; boş satır ve # yorumları yok
// sayılır; isteğe bağlı "export " öneki ve tek/çift tırnak desteklenir.
// Hata mesajları yalnızca satır numarası içerir, değer içermez.
func ReadDotEnv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf(".env açılamadı: %w", err)
	}
	defer f.Close()
	return parseDotEnv(f)
}

func parseDotEnv(r io.Reader) (map[string]string, error) {
	out := map[string]string{}
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || !envKey(key) {
			return nil, fmt.Errorf(".env satır %d: KEY=VALUE biçiminde olmalı", n)
		}
		val = strings.TrimSpace(val)
		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[len(val)-1] == val[0] {
			val = val[1 : len(val)-1]
		} else if i := strings.Index(val, " #"); i >= 0 {
			val = strings.TrimSpace(val[:i]) // satır sonu yorumu
		}
		out[key] = val
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf(".env okunamadı: %w", err)
	}
	return out, nil
}

func envKey(k string) bool {
	if k == "" || (k[0] >= '0' && k[0] <= '9') {
		return false
	}
	for _, c := range k {
		if !(c == '_' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

// WithFallback, önce getenv'e bakar; boşsa fallback haritasına düşer.
// Gerçek ortam değişkenleri .env'den önceliklidir.
func WithFallback(getenv func(string) string, fallback map[string]string) func(string) string {
	return func(k string) string {
		if v := getenv(k); v != "" {
			return v
		}
		return fallback[k]
	}
}
