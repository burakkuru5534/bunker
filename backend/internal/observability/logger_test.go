package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

func decodeLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	line := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("JSON log çözülemedi: %v (%s)", err, line)
	}
	return m
}

func TestJSONUsesCloudLoggingFields(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger(&buf, slog.LevelDebug, FormatJSON)

	l.Warn("dikkat", "oda", "r1")

	m := decodeLine(t, &buf)
	if m["severity"] != "WARNING" || m["message"] != "dikkat" || m["oda"] != "r1" {
		t.Fatalf("alanlar yanlış: %v", m)
	}
	if _, ok := m["level"]; ok {
		t.Fatal("level yerine severity olmalı")
	}
	if _, ok := m["msg"]; ok {
		t.Fatal("msg yerine message olmalı")
	}
	if _, ok := m["time"]; !ok {
		t.Fatal("time alanı olmalı")
	}
}

func TestSeverityMapping(t *testing.T) {
	cases := map[slog.Level]string{
		slog.LevelDebug: "DEBUG",
		slog.LevelInfo:  "INFO",
		slog.LevelWarn:  "WARNING",
		slog.LevelError: "ERROR",
		slog.Level(12):  "ERROR",
	}
	for lvl, want := range cases {
		if got := severity(lvl); got != want {
			t.Errorf("%v → %q, %q bekleniyordu", lvl, got, want)
		}
	}
}

func TestSensitiveValuesAreRedacted(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger(&buf, slog.LevelInfo, FormatJSON)

	l.Info("gizli",
		"Authorization", "Bearer abc.def",
		"session_token", "s3cr3t",
		"DATABASE_URL", "postgres://u:p@h/db",
		"password", "123",
		"not", "header: Bearer xyz",
		slog.Group("req", slog.String("api_key", "k")),
		"room_id", "r1",
	)

	out := buf.String()
	for _, leak := range []string{"abc.def", "s3cr3t", "u:p@h", "\"123\"", "xyz", "\"k\""} {
		if strings.Contains(out, leak) {
			t.Fatalf("gizli değer sızdı (%s): %s", leak, out)
		}
	}
	m := decodeLine(t, &buf)
	if m["room_id"] != "r1" {
		t.Fatal("hassas olmayan alan korunmalı")
	}
	if m["session_token"] != Redacted {
		t.Fatalf("maskeleme bekleniyordu: %v", m["session_token"])
	}
}

func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger(&buf, slog.LevelWarn, FormatJSON)
	l.Info("görünmemeli")
	if buf.Len() != 0 {
		t.Fatal("seviye altı log yazılmamalı")
	}
}

func TestTextFormat(t *testing.T) {
	var buf bytes.Buffer
	NewLogger(&buf, slog.LevelInfo, FormatText).Info("merhaba", "token", "x")
	out := buf.String()
	if !strings.Contains(out, "severity=INFO") || !strings.Contains(out, `message=merhaba`) || !strings.Contains(out, "token="+Redacted) {
		t.Fatalf("metin biçimi yanlış: %s", out)
	}
}

func TestParseLevel(t *testing.T) {
	for in, want := range map[string]slog.Level{"debug": slog.LevelDebug, "": slog.LevelInfo, "INFO": slog.LevelInfo, "warning": slog.LevelWarn, "error": slog.LevelError} {
		got, ok := ParseLevel(in)
		if !ok || got != want {
			t.Errorf("ParseLevel(%q) = %v,%v", in, got, ok)
		}
	}
	if _, ok := ParseLevel("trace"); ok {
		t.Error("trace reddedilmeli")
	}
}

func TestLoggerContext(t *testing.T) {
	if LoggerFrom(context.Background()) != slog.Default() {
		t.Fatal("bağlamda yoksa varsayılan logger dönmeli")
	}
	l := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	if LoggerFrom(WithLogger(context.Background(), l)) != l {
		t.Fatal("bağlamdaki logger dönmeli")
	}
}

func TestTraceID(t *testing.T) {
	const id = "4bf92f3577b34da6a3ce929d0e0e4736"
	r := httptest.NewRequest("GET", "/", nil)
	if TraceID(r) != "" {
		t.Fatal("başlık yoksa boş olmalı")
	}

	r.Header.Set("traceparent", "00-"+id+"-00f067aa0ba902b7-01")
	if TraceID(r) != id {
		t.Fatal("traceparent okunmalı")
	}

	r = httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Cloud-Trace-Context", strings.ToUpper(id)+"/1;o=1")
	if TraceID(r) != id {
		t.Fatal("X-Cloud-Trace-Context okunmalı (küçük harfe çevrilmiş)")
	}

	r = httptest.NewRequest("GET", "/", nil)
	r.Header.Set("traceparent", "00-"+strings.Repeat("0", 32)+"-00f067aa0ba902b7-01")
	r.Header.Set("X-Cloud-Trace-Context", "bozuk")
	if TraceID(r) != "" {
		t.Fatal("geçersiz izleme kimlikleri yok sayılmalı")
	}

	if TraceResource("", id) != "" || TraceResource("p", "") != "" {
		t.Fatal("eksik bilgide alan yazılmamalı")
	}
	if TraceResource("proj", id) != "projects/proj/traces/"+id {
		t.Fatal("izleme kaynağı biçimi yanlış")
	}
}
