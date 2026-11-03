package room

import (
	"strings"
	"testing"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
)

func TestNewInviteCode(t *testing.T) {
	counts := map[byte]int{}
	seen := map[string]bool{}
	const n = 20000
	for range n {
		c, err := NewInviteCode()
		if err != nil || len(c) != InviteLength {
			t.Fatalf("kod: %q %v", c, err)
		}
		if norm, err := NormalizeInviteCode(c); err != nil || norm != c {
			t.Fatalf("üretilen kod geçerli olmalı: %q", c)
		}
		seen[c] = true
		for i := 0; i < len(c); i++ {
			counts[c[i]]++
		}
	}
	if len(seen) < n-5 {
		t.Fatalf("çakışma beklenmedik ölçüde çok: %d/%d tekil", len(seen), n)
	}
	// Dağılım yaklaşık tekdüze: her harf beklenenin ±%15'inde.
	want := float64(n*InviteLength) / float64(len(InviteAlphabet))
	for i := 0; i < len(InviteAlphabet); i++ {
		got := float64(counts[InviteAlphabet[i]])
		if got < want*0.85 || got > want*1.15 {
			t.Errorf("%c: %v (beklenen ~%v)", InviteAlphabet[i], got, want)
		}
	}
	for _, bad := range "01ILO" {
		if counts[byte(bad)] != 0 {
			t.Errorf("%c alfabede olmamalı", bad)
		}
	}
}

func TestNormalizeInviteCode(t *testing.T) {
	ok := map[string]string{
		"K7M2QX":    "K7M2QX",
		"k7m2qx":    "K7M2QX",
		" k7m-2qx ": "K7M2QX",
		"K7M 2QX":   "K7M2QX",
	}
	for in, want := range ok {
		if got, err := NormalizeInviteCode(in); err != nil || got != want {
			t.Errorf("%q → %q %v", in, got, err)
		}
	}
	for _, bad := range []string{"", "K7M2Q", "K7M2QXA", "K7M2Q0", "K7M2QO", "K7M2Q1", "K7M2QI", "K7M2QL", "K7M2Qİ", "K7M2Q!"} {
		_, err := NormalizeInviteCode(bad)
		var e *apperr.Error
		if err == nil || !strings.Contains(err.Error(), "VALIDATION_FAILED") {
			t.Errorf("%q reddedilmeli: %v", bad, err)
		}
		_ = e
	}
}
