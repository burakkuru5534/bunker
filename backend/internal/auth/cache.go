package auth

import (
	"sync"
	"time"
)

// cache, doğrulanmış oturumları kısa süre bellekte tutar (security.md §1:
// ≤ 60 sn). Tek instance olduğu için (ADR-0003) yerel iptal yeterlidir;
// yenilenen/iptal edilen oturum anında çıkarılır. Yalnızca geçerli oturumlar
// önbelleğe girer (geçersiz token'larla doldurulamaz).
type cache struct {
	mu      sync.Mutex
	ttl     time.Duration
	max     int
	entries map[string]cacheEntry // anahtar: token hash'i (string)
}

type cacheEntry struct {
	p       Principal
	expires time.Time
}

func newCache(ttl time.Duration, max int) *cache {
	return &cache{ttl: ttl, max: max, entries: make(map[string]cacheEntry)}
}

func (c *cache) get(key string, now time.Time) (Principal, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok {
		return Principal{}, false
	}
	if !now.Before(e.expires) {
		delete(c.entries, key)
		return Principal{}, false
	}
	return e.p, true
}

// put, girdiyi ttl süresince ama oturumun kendi süresini aşmadan tutar.
func (c *cache) put(key string, p Principal, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= c.max {
		for k, e := range c.entries {
			if !now.Before(e.expires) {
				delete(c.entries, k)
			}
		}
		if len(c.entries) >= c.max {
			// Hepsi tazeyse önbellek temizlenir (nadir; doğruluk etkilenmez).
			clear(c.entries)
		}
	}
	exp := now.Add(c.ttl)
	if p.ExpiresAt.Before(exp) {
		exp = p.ExpiresAt
	}
	c.entries[key] = cacheEntry{p: p, expires: exp}
}

func (c *cache) delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// deleteUser, kullanıcının tüm önbellek girdilerini çıkarır (hesap silme).
func (c *cache) deleteUser(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, e := range c.entries {
		if e.p.UserID == userID {
			delete(c.entries, k)
		}
	}
}

func (c *cache) markTouched(key string, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.entries[key]; ok {
		e.p.lastUsed = &at
		c.entries[key] = e
	}
}
