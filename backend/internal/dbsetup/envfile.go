package dbsetup

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// ErrAlreadySet, .env'de dolu bir DATABASE_URL olduğunda döner.
var ErrAlreadySet = errors.New("DATABASE_URL zaten dolu; şifreyi yenilemek için -force kullanın")

// UpsertEnvVar, .env dosyasında key satırını değerle yazar: boş satır varsa
// değiştirir, yoksa ekler; dolu satır varsa force olmadan ErrAlreadySet döner.
// Diğer satırlar korunur. Dosya 0600 izniyle yazılır. Değer hiçbir hataya yazılmaz.
func UpsertEnvVar(path, key, value string, force bool) error {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s okunamadı: %w", path, err)
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if text == "" {
		lines = nil
	}

	newLine := key + "=" + value
	found := false
	for i, line := range lines {
		trimmed := strings.TrimPrefix(strings.TrimSpace(line), "export ")
		k, v, ok := strings.Cut(trimmed, "=")
		if !ok || strings.TrimSpace(k) != key {
			continue
		}
		if strings.TrimSpace(v) != "" && !force {
			return ErrAlreadySet
		}
		lines[i] = newLine
		found = true
	}
	if !found {
		if n := len(lines); n > 0 && lines[n-1] == "" {
			lines = lines[:n-1]
		}
		lines = append(lines, newLine)
	}
	out := strings.Join(lines, "\n")
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(out), 0o600); err != nil {
		return fmt.Errorf("%s yazılamadı: %w", path, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("%s yazılamadı: %w", path, err)
	}
	return nil
}
