package auth

import (
	"testing"
	"time"
)

func TestCacheExpiryAndBounds(t *testing.T) {
	now := time.Now()
	c := newCache(time.Minute, 2)
	p := Principal{UserID: "u1", ExpiresAt: now.Add(time.Hour)}
	c.put("a", p, now)
	if _, ok := c.get("a", now.Add(59*time.Second)); !ok {
		t.Fatal("ttl içinde bulunmalı")
	}
	if _, ok := c.get("a", now.Add(time.Minute)); ok {
		t.Fatal("ttl dolunca düşmeli")
	}

	// Oturumun kendi süresi ttl'den kısaysa o kadar tutulur.
	c.put("b", Principal{UserID: "u2", ExpiresAt: now.Add(10 * time.Second)}, now)
	if _, ok := c.get("b", now.Add(11*time.Second)); ok {
		t.Fatal("oturum süresini aşmamalı")
	}

	// Sınır: süresi geçmişler temizlenir, hepsi tazeyse önbellek boşaltılır.
	c.put("x", p, now)
	c.put("y", p, now)
	c.put("z", p, now) // max 2 → temizlik
	if len(c.entries) > 2 {
		t.Fatalf("üst sınır aşıldı: %d", len(c.entries))
	}
	if _, ok := c.get("z", now); !ok {
		t.Fatal("son eklenen bulunmalı")
	}

	c.put("w", Principal{UserID: "u9", ExpiresAt: now.Add(time.Hour)}, now)
	c.deleteUser("u9")
	if _, ok := c.get("w", now); ok {
		t.Fatal("kullanıcının girdileri silinmeli")
	}
	c.markTouched("yok", now) // panik olmamalı
}
