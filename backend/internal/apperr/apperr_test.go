package apperr

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

func TestCodesAreUniqueAndUpperSnake(t *testing.T) {
	seen := map[Code]bool{}
	for _, c := range All {
		if seen[c] {
			t.Fatalf("tekrarlanan kod: %s", c)
		}
		seen[c] = true
		if strings.ToUpper(string(c)) != string(c) || strings.ContainsAny(string(c), " -") {
			t.Fatalf("kod BÜYÜK_YILAN biçiminde olmalı: %s", c)
		}
	}
	if len(All) != 27 {
		t.Fatalf("docs/api.md 27 kod tanımlar, %d var", len(All))
	}
}

func TestErrorMessageAndUnwrap(t *testing.T) {
	cause := io.ErrUnexpectedEOF
	e := Wrap(Internal, "okuma", cause)
	if e.Error() != "INTERNAL: okuma: unexpected EOF" {
		t.Fatalf("Error() = %q", e.Error())
	}
	if !errors.Is(e, io.ErrUnexpectedEOF) {
		t.Fatal("neden errors.Is ile bulunmalı")
	}
	if New(RoomFull, "").Error() != "ROOM_FULL" {
		t.Fatal("mesajsız hata yalnızca kodu yazmalı")
	}
	if Newf(NotFound, "oda %d", 3).Message != "oda 3" {
		t.Fatal("Newf biçimlemeli")
	}
}

func TestIsMatchesByCode(t *testing.T) {
	err := fmt.Errorf("katman: %w", New(RoomFull, "10/10"))
	if !errors.Is(err, New(RoomFull, "")) {
		t.Fatal("aynı kod eşleşmeli")
	}
	if errors.Is(err, New(GameInProgress, "")) {
		t.Fatal("farklı kod eşleşmemeli")
	}
}

func TestValidationCopiesFields(t *testing.T) {
	in := map[string]string{"nickname": FieldTooShort}
	e := Validation(in)
	in["nickname"] = "değişti"
	if e.Code != ValidationFailed || e.Fields["nickname"] != FieldTooShort {
		t.Fatalf("alanlar kopyalanmalı: %+v", e)
	}
	if f := Field("capacity", FieldNotAllowed); f.Fields["capacity"] != FieldNotAllowed {
		t.Fatal("Field tek alan üretmeli")
	}
}

func TestWithDetailAndRateLimit(t *testing.T) {
	e := New(UpgradeRequired, "").WithDetail("min_version", "1.0.0").WithDetail("x", 1)
	if e.Details["min_version"] != "1.0.0" || e.Details["x"] != 1 {
		t.Fatalf("ayrıntılar: %v", e.Details)
	}
	r := RateLimit(1500 * time.Millisecond)
	if r.Code != RateLimited || r.RetryAfter != 1500*time.Millisecond {
		t.Fatal("RateLimit yanlış")
	}
}

func TestCodeOf(t *testing.T) {
	if CodeOf(fmt.Errorf("x: %w", New(Kicked, ""))) != Kicked {
		t.Fatal("sarılı hata kodu bulunmalı")
	}
	if CodeOf(errors.New("düz")) != Internal || CodeOf(nil) != Internal {
		t.Fatal("bilinmeyen hata INTERNAL olmalı")
	}
}

func TestFieldErrors(t *testing.T) {
	var fe FieldErrors
	if fe.Err() != nil {
		t.Fatal("boşsa nil")
	}
	fe.Add("nickname", FieldTooShort)
	fe.Add("nickname", FieldProfanity) // ilk hata kalır
	fe.Add("capacity", FieldNotAllowed)

	var e *Error
	if !errors.As(fe.Err(), &e) || e.Code != ValidationFailed {
		t.Fatal("VALIDATION_FAILED dönmeli")
	}
	if e.Fields["nickname"] != FieldTooShort || e.Fields["capacity"] != FieldNotAllowed || len(e.Fields) != 2 {
		t.Fatalf("alanlar: %v", e.Fields)
	}
}
