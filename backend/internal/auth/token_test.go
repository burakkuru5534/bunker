package auth

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestNewTokenAndHash(t *testing.T) {
	seen := map[string]bool{}
	for range 100 {
		tok, err := NewToken()
		if err != nil {
			t.Fatal(err)
		}
		if len(tok.Raw) != TokenLength || len(tok.Hash) != sha256.Size || seen[tok.Raw] {
			t.Fatalf("token biçimi: %d %d", len(tok.Raw), len(tok.Hash))
		}
		seen[tok.Raw] = true
		h, err := HashToken(tok.Raw)
		if err != nil || !bytes.Equal(h, tok.Hash) {
			t.Fatalf("hash eşleşmeli: %v", err)
		}
		if strings.ContainsAny(tok.Raw, "+/=") {
			t.Fatal("base64url dolgusuz olmalı (URL/başlık güvenli)")
		}
	}
}

func TestHashTokenRejectsMalformed(t *testing.T) {
	valid, _ := NewToken()
	bad := []string{
		"",
		"kısa",
		valid.Raw + "A",
		valid.Raw[:42],
		strings.Repeat("!", TokenLength),
		base64.StdEncoding.EncodeToString(make([]byte, 32))[:TokenLength-1] + "+",
		// Son karakterin kullanılmayan bitleri dolu: Strict kod çözme reddeder.
		valid.Raw[:42] + string(flip(valid.Raw[42])),
	}
	for _, s := range bad {
		if _, err := HashToken(s); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("%q reddedilmeli: %v", s, err)
		}
	}
}

// flip, son base64 karakterinin kullanılmayan alt bitlerinden birini değiştirir.
func flip(c byte) byte {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	i := strings.IndexByte(alphabet, c)
	return alphabet[i^1]
}

func TestBearerToken(t *testing.T) {
	cases := map[string]string{
		"Bearer abc":     "abc",
		"bearer abc":     "abc",
		"BEARER   abc  ": "abc",
		"  Bearer abc":   "abc",
	}
	for h, want := range cases {
		if got, ok := BearerToken(h); !ok || got != want {
			t.Errorf("%q → %q %v", h, got, ok)
		}
	}
	for _, h := range []string{"", "Bearer", "Bearer ", "Basic abc", "abc", "Token abc"} {
		if _, ok := BearerToken(h); ok {
			t.Errorf("%q reddedilmeli", h)
		}
	}
}
