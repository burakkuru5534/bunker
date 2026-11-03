package room

import (
	"strings"
	"testing"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
)

func TestCleanMessage(t *testing.T) {
	ok := map[string]string{
		"  selam  ":              "selam",
		"iki\nsatır\r\nvar":      "iki satır var",
		"çok    boşluk\t\tvar":   "çok boşluk var",
		"Işık ğüşöç İ":           "Işık ğüşöç İ",
		"emoji 👍🏽 kalır":         "emoji 👍🏽 kalır",
		"ters‮çevir":             "tersçevir",   // yön değiştirme (RLO) atılır
		"gizli​boşluk":           "gizliboşluk", // sıfır genişlikli boşluk atılır
		"zil\x07sesi":            "zilsesi",     // kontrol karakteri atılır
		"é":                     "é",           // NFC: tek karakter
		" başta bölünmez boşluk": "başta bölünmez boşluk",
	}
	for in, want := range ok {
		got, err := CleanMessage(in)
		if err != nil || got != want {
			t.Errorf("%q → %q, %v (beklenen %q)", in, got, err, want)
		}
	}
	if got, err := CleanMessage(strings.Repeat("ş", 200)); err != nil || len([]rune(got)) != 200 {
		t.Fatalf("200 karakter: %v", err)
	}
	for _, in := range []string{"", "   ", "\n\t", "‮​", strings.Repeat("a", 201), "bozuk\xff"} {
		if _, err := CleanMessage(in); apperr.CodeOf(err) != apperr.MessageInvalid {
			t.Errorf("%q: %v", in, err)
		}
	}
	// 201 karakter ama birleşik → NFC sonrası 200: kabul.
	if _, err := CleanMessage(strings.Repeat("a", 199) + "é"); err != nil {
		t.Fatalf("NFC sonrası sayım: %v", err)
	}
}
