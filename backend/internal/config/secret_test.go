package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestSecretIsMaskedEverywhere(t *testing.T) {
	const raw = "postgres://u:cokgizli@h/db"
	s := Secret(raw)
	cfg := Config{DatabaseURL: s}

	var logs bytes.Buffer
	slog.New(slog.NewJSONHandler(&logs, nil)).Info("x", "db", s, "cfg", cfg)
	js, _ := json.Marshal(cfg)

	for name, out := range map[string]string{
		"%v":   fmt.Sprintf("%v", cfg),
		"%+v":  fmt.Sprintf("%+v", cfg),
		"%#v":  fmt.Sprintf("%#v", cfg),
		"%s":   fmt.Sprintf("%s", s),
		"slog": logs.String(),
		"json": string(js),
	} {
		if strings.Contains(out, "cokgizli") {
			t.Errorf("%s çıktısında gizli değer var: %s", name, out)
		}
		if !strings.Contains(out, redacted) {
			t.Errorf("%s çıktısında maske yok: %s", name, out)
		}
	}
	if s.Reveal() != raw || !s.IsSet() {
		t.Fatal("Reveal gerçek değeri döndürmeli")
	}
	if Secret("").String() != "" || Secret("").IsSet() {
		t.Fatal("boş gizli değer boş görünmeli")
	}
}
