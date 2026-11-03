package ratelimit

import (
	"sync"
	"testing"
	"time"
)

func fixed(t *time.Time) func() time.Time { return func() time.Time { return *t } }

func TestFivePerMinute(t *testing.T) {
	now := time.Now()
	l := New(5, time.Minute, 100)
	l.now = fixed(&now)

	for i := range 5 {
		if ok, _ := l.Allow("1.2.3.4"); !ok {
			t.Fatalf("%d. istek izinli olmalı", i+1)
		}
	}
	ok, wait := l.Allow("1.2.3.4")
	if ok || wait < 11*time.Second || wait > 12*time.Second {
		t.Fatalf("6. istek reddedilmeli, ~12 sn beklemeli: %v %v", ok, wait)
	}
	// Başka anahtar etkilenmez.
	if ok, _ := l.Allow("5.6.7.8"); !ok {
		t.Fatal("başka IP izinli olmalı")
	}
	// 12 sn sonra bir jeton.
	now = now.Add(12 * time.Second)
	if ok, _ := l.Allow("1.2.3.4"); !ok {
		t.Fatal("jeton dolmalı")
	}
	if ok, _ := l.Allow("1.2.3.4"); ok {
		t.Fatal("yalnızca bir jeton dolmuş olmalı")
	}
	// Uzun bekleme kovayı limitin üstüne taşımaz.
	now = now.Add(time.Hour)
	for range 5 {
		l.Allow("1.2.3.4")
	}
	if ok, _ := l.Allow("1.2.3.4"); ok {
		t.Fatal("ani patlama limit kadar olmalı")
	}
}

func TestMaxKeysBounded(t *testing.T) {
	now := time.Now()
	l := New(5, time.Minute, 3)
	l.now = fixed(&now)
	for _, k := range []string{"a", "b", "c"} {
		l.Allow(k)
	}
	now = now.Add(2 * time.Minute) // hepsi doldu → boşta
	l.Allow("d")
	if l.Len() != 1 {
		t.Fatalf("boştaki kovalar silinmeli: %d", l.Len())
	}
	// Hepsi aktifken sınır aşılırsa sıfırlanır.
	for _, k := range []string{"e", "f"} {
		l.Allow(k)
	}
	l.Allow("g")
	if l.Len() > 3 {
		t.Fatalf("üst sınır aşıldı: %d", l.Len())
	}
}

func TestConcurrentAllow(t *testing.T) {
	l := New(100, time.Hour, 10)
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for range 500 {
		wg.Go(func() {
			if ok, _ := l.Allow("k"); ok {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if allowed != 100 {
		t.Fatalf("tam 100 izin verilmeli: %d", allowed)
	}
}
