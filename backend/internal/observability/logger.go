// Package observability, yapılandırılmış loglama ve istek bağlamı yardımcılarıdır.
//
// Loglar Google Cloud Logging'in beklediği JSON alanlarıyla yazılır:
// "severity", "message", "time", "httpRequest", "logging.googleapis.com/trace".
// Gizli değerler (token, parola, bağlantı dizesi…) her zaman maskelenir.
package observability

import (
	"context"
	"io"
	"log/slog"
	"strings"
)

// Format, log çıktı biçimi.
type Format string

const (
	FormatJSON Format = "json" // Cloud Run (test / prod)
	FormatText Format = "text" // yerel geliştirme
)

// Redacted, maskelenmiş değerlerin yerine yazılan metin.
const Redacted = "[REDACTED]"

// sensitiveKeys, değeri asla loglanmayacak anahtarlar (küçük harf, alt dize).
var sensitiveKeys = []string{
	"authorization", "token", "password", "secret", "database_url", "dsn",
	"cookie", "api_key", "apikey", "credential", "private_key",
}

// NewLogger, verilen seviye ve biçimle bir logger oluşturur.
func NewLogger(w io.Writer, level slog.Level, format Format) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level, ReplaceAttr: replaceAttr}
	var h slog.Handler
	if format == FormatText {
		h = slog.NewTextHandler(w, opts)
	} else {
		h = slog.NewJSONHandler(w, opts)
	}
	return slog.New(h)
}

// replaceAttr, Cloud Logging alan adlarını uygular ve gizli değerleri maskeler.
func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if len(groups) == 0 {
		switch a.Key {
		case slog.LevelKey:
			lvl, _ := a.Value.Any().(slog.Level)
			return slog.String("severity", severity(lvl))
		case slog.MessageKey:
			a.Key = "message"
			return a
		}
	}
	if isSensitiveKey(a.Key) {
		return slog.String(a.Key, Redacted)
	}
	if a.Value.Kind() == slog.KindString && containsBearer(a.Value.String()) {
		return slog.String(a.Key, Redacted)
	}
	return a
}

func isSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	for _, s := range sensitiveKeys {
		if strings.Contains(k, s) {
			return true
		}
	}
	return false
}

func containsBearer(s string) bool {
	return strings.Contains(strings.ToLower(s), "bearer ")
}

// severity, slog seviyesini Cloud Logging seviyesine çevirir.
func severity(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "ERROR"
	case l >= slog.LevelWarn:
		return "WARNING"
	case l >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}

// ParseLevel, "debug|info|warn|error" metnini çözer.
func ParseLevel(s string) (slog.Level, bool) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, true
	case "info", "":
		return slog.LevelInfo, true
	case "warn", "warning":
		return slog.LevelWarn, true
	case "error":
		return slog.LevelError, true
	}
	return 0, false
}

type loggerKey struct{}

// WithLogger, bağlama istek kapsamlı bir logger ekler.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}

// LoggerFrom, bağlamdaki logger'ı döndürür; yoksa slog.Default().
func LoggerFrom(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
