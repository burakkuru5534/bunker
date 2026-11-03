package config

import "testing"

// .env.example, okunabilir ve geçerli bir yerel yapılandırma olmalı.
func TestEnvExampleIsValidLocalConfig(t *testing.T) {
	m, err := ReadDotEnv("../../.env.example")
	if err != nil {
		t.Fatal(err)
	}
	if len(m) == 0 {
		t.Fatal(".env.example boş okundu")
	}
	if _, err := Load(WithFallback(func(string) string { return "" }, m)); err != nil {
		t.Fatalf(".env.example geçersiz: %v", err)
	}
}
