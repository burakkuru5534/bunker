package auth

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
)

func reason(t *testing.T, err error) string {
	t.Helper()
	var e *apperr.Error
	if !errors.As(err, &e) || e.Code != apperr.NicknameInvalid {
		t.Fatalf("NICKNAME_INVALID bekleniyordu: %v", err)
	}
	return e.Fields["nickname"]
}

func TestValidNicknames(t *testing.T) {
	cases := map[string]string{
		"Burak":            "Burak",
		"  Ayşe  ":         "Ayşe",
		"ÇĞİÖŞÜ çğıöşü":    "ÇĞİÖŞÜ çğıöşü",
		"Işık_42":          "Işık_42",
		"Deniz Kaya":       "Deniz Kaya",
		"ab1":              "ab1",
		"a234567890123456": "a234567890123456",
		"Zoë":              "Zoë",
		"şeker":           "şeker", // ayrık işaret NFC ile birleşir
		"\tCan\n":          "Can",
	}
	for in, want := range cases {
		got, err := ValidateNickname(in, nil)
		if err != nil || got != want {
			t.Errorf("%q → %q, %v (beklenen %q)", in, got, err, want)
		}
	}
}

func TestInvalidNicknames(t *testing.T) {
	cases := map[string]string{
		"":                  apperr.FieldRequired,
		"   ":               apperr.FieldRequired,
		"ab":                apperr.FieldTooShort,
		"a2345678901234567": apperr.FieldTooLong,
		"Bu-rak":            apperr.FieldInvalidChars,
		"Burak!":            apperr.FieldInvalidChars,
		"Бурак":             apperr.FieldInvalidChars, // Kiril
		"Вurak":             apperr.FieldInvalidChars, // Kiril В, Latin B taklidi
		"Burak😀":            apperr.FieldInvalidChars,
		"Burá́k":           apperr.FieldInvalidChars, // birleşik işaret yığını
		"Bu​rak":            apperr.FieldInvalidChars, // sıfır genişlikli boşluk
		"Bu rak":            apperr.FieldInvalidChars, // bölünmez boşluk
		"Deniz  Kaya":       apperr.FieldInvalid,      // ardışık boşluk
		"123":               apperr.FieldInvalid,      // harf yok
		"___":               apperr.FieldInvalid,
		"Silinmiş oyuncu":   apperr.FieldNotAllowed,
		"SİLİNMİŞ OYUNCU":   apperr.FieldNotAllowed, // Türkçe büyük harf
		"SİSTEM":            apperr.FieldNotAllowed,
		"Admin":             apperr.FieldNotAllowed,
		"\xff\xfeab":        apperr.FieldInvalidChars,
	}
	for in, want := range cases {
		_, err := ValidateNickname(in, nil)
		if got := reason(t, err); got != want {
			t.Errorf("%q → %s (beklenen %s)", in, got, want)
		}
	}
}

func TestProfanityHook(t *testing.T) {
	bad := func(s string) bool { return strings.Contains(strings.ToLower(s), "kötü") }
	if _, err := ValidateNickname("KötüAdam", bad); reason(t, err) != apperr.FieldProfanity {
		t.Fatal("küfür denetimi uygulanmalı")
	}
	if _, err := ValidateNickname("İyiAdam", bad); err != nil {
		t.Fatal(err)
	}
}

func TestSameNickname(t *testing.T) {
	for _, p := range [][2]string{{"IŞIK", "ışık"}, {"İrem", "irem"}, {" Burak", "burak"}, {"şeker", "Şeker"}} {
		if !SameNickname(p[0], p[1]) {
			t.Errorf("%q = %q olmalı", p[0], p[1])
		}
	}
	// Türkçe kurallarla I → ı, İ → i: "IRMAK" ile "irmak" farklı adlardır.
	if SameNickname("IRMAK", "irmak") {
		t.Error("Türkçe'de I'nın küçüğü ı'dır")
	}
}

// Fuzz: geçerli sayılan her ad veritabanı kısıtını sağlar ve kararlıdır.
func FuzzValidateNickname(f *testing.F) {
	for _, s := range []string{"Burak", " Ayşe ", "Deniz  Kaya", "Бурак", "á", "😀😀😀", "Işık_42"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		got, err := ValidateNickname(in, nil)
		if err != nil {
			return
		}
		n := utf8.RuneCountInString(got)
		if n < NicknameMin || n > NicknameMax || strings.TrimSpace(got) != got || strings.Contains(got, "  ") {
			t.Fatalf("%q → %q kısıtları ihlal ediyor", in, got)
		}
		again, err := ValidateNickname(got, nil)
		if err != nil || again != got {
			t.Fatalf("kararsız: %q → %q → %q, %v", in, got, again, err)
		}
	})
}
