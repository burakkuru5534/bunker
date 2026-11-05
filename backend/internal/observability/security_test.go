package observability

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

func TestSecurityUsesRequestLoggerAndClientIP(t *testing.T) {
	var reqBuf, fbBuf bytes.Buffer
	req := NewLogger(&reqBuf, slog.LevelDebug, FormatJSON).With("request_id", "r1")
	ctx := WithClientIP(WithLogger(context.Background(), req), "203.0.113.7")

	Security(ctx, NewLogger(&fbBuf, slog.LevelDebug, FormatJSON), slog.LevelWarn, EventAuthFailed, "istek reddedildi",
		slog.String("code", "UNAUTHORIZED"), slog.String("token", "gizli"))

	if fbBuf.Len() != 0 {
		t.Fatal("bağlamda logger varken yedeğe yazıldı")
	}
	m := decodeLine(t, &reqBuf)
	for k, want := range map[string]any{
		"security_event": "auth_failed", "client_ip": "203.0.113.7", "request_id": "r1",
		"severity": "WARNING", "code": "UNAUTHORIZED", "token": Redacted,
	} {
		if m[k] != want {
			t.Errorf("%s = %v, beklenen %v", k, m[k], want)
		}
	}
}

func TestSecurityFallbackWithoutIP(t *testing.T) {
	var buf bytes.Buffer
	Security(context.Background(), NewLogger(&buf, slog.LevelDebug, FormatJSON), slog.LevelInfo,
		EventAccountDeleted, "hesap silindi", slog.String("user_id", "u1"))
	m := decodeLine(t, &buf)
	if m["security_event"] != "account_deleted" || m["severity"] != "INFO" || m["user_id"] != "u1" {
		t.Fatalf("beklenmeyen satır: %v", m)
	}
	if _, ok := m["client_ip"]; ok {
		t.Fatal("bağlamda IP yokken client_ip yazıldı")
	}
}
