package config

import "log/slog"

// Secret, yanlışlıkla yazdırılmaması gereken bir değerdir (ör. DATABASE_URL).
// fmt, slog ve JSON çıktılarında maskelenir; gerçek değer yalnızca Reveal ile
// alınır.
type Secret string

const redacted = "[REDACTED]"

// Reveal, gerçek değeri döndürür. Yalnızca değeri kullanacak koda (ör.
// veritabanı sürücüsü) verilmelidir.
func (s Secret) Reveal() string { return string(s) }

// IsSet, değer dolu mu?
func (s Secret) IsSet() bool { return s != "" }

func (s Secret) String() string {
	if s == "" {
		return ""
	}
	return redacted
}

// GoString, %#v çıktısını da maskeler.
func (s Secret) GoString() string { return `config.Secret("` + s.String() + `")` }

// LogValue, slog çıktısını maskeler.
func (s Secret) LogValue() slog.Value { return slog.StringValue(s.String()) }

// MarshalText, JSON/metin kodlamayı maskeler.
func (s Secret) MarshalText() ([]byte, error) { return []byte(s.String()), nil }
