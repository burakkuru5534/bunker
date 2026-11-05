package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

var fakeKey = "sb_" + "secret_" + strings.Repeat("Ab3", 8)

func TestRun(t *testing.T) {
	clean := map[string][]byte{"a.go": []byte("package a"), "logo.png": []byte(fakeKey)}
	dirty := map[string][]byte{"a.go": []byte("package a\nvar k = \"" + fakeKey + "\"")}
	var gotStaged bool
	lister := func(files map[string][]byte, err error) lister {
		return func(staged bool) (map[string][]byte, error) {
			gotStaged = staged
			return files, err
		}
	}

	var out, errOut bytes.Buffer
	if code := run(nil, lister(clean, nil), &out, &errOut); code != 0 || !strings.Contains(out.String(), "bulunmadı") {
		t.Fatalf("temiz: %d %s", code, out.String())
	}

	out.Reset()
	if code := run([]string{"-staged"}, lister(dirty, nil), &out, &errOut); code != 1 || !gotStaged {
		t.Fatalf("kirli: %d staged=%v", code, gotStaged)
	}
	if !strings.Contains(out.String(), "a.go:2: olası gizli değer (supabase-key)") {
		t.Fatalf("çıktı: %s", out.String())
	}
	if strings.Contains(out.String(), fakeKey) {
		t.Fatal("değer çıktıya yazıldı")
	}

	if code := run(nil, lister(nil, errors.New("git yok")), &out, &errOut); code != 2 {
		t.Fatalf("git hatası: %d", code)
	}
	if code := run([]string{"-bilinmeyen"}, lister(clean, nil), &out, &errOut); code != 2 {
		t.Fatalf("bozuk bayrak: %d", code)
	}
}
