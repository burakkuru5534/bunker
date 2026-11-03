// Package ratelimit, anahtar başına bellek içi token bucket sınırlayıcıdır
// (security.md §5). Tek instance olduğu için (ADR-0003) yereldir; yeniden
// başlatmada sıfırlanır.
package ratelimit

import (
	"math"
	"sync"
	"time"
)

// Limiter, her anahtar için "per süresinde en fazla limit" izin verir;
// kova sürekli dolar (ani patlama limit kadar).
type Limiter struct {
	mu      sync.Mutex
	limit   float64
	rate    float64 // saniyede eklenen jeton
	maxKeys int
	buckets map[string]*bucket
	now     func() time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// New, örn. New(5, time.Minute, 100_000) → anahtar başına dakikada 5.
// maxKeys aşılırsa dolmuş (boşta) kovalar silinir; hâlâ fazlaysa tümü
// sıfırlanır (bellek sınırı doğruluktan önce gelir).
func New(limit int, per time.Duration, maxKeys int) *Limiter {
	return &Limiter{
		limit: float64(limit), rate: float64(limit) / per.Seconds(),
		maxKeys: maxKeys, buckets: make(map[string]*bucket), now: time.Now,
	}
}

// Allow, bir jeton harcar. İzin yoksa bir sonraki jetona kalan süreyi
// döndürür (Retry-After için).
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	b, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) >= l.maxKeys {
			l.sweep(now)
		}
		b = &bucket{tokens: l.limit, last: now}
		l.buckets[key] = b
	}
	b.tokens = math.Min(l.limit, b.tokens+now.Sub(b.last).Seconds()*l.rate)
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
	return false, wait
}

// sweep, tamamen dolmuş (yani boşta olan) kovaları siler.
func (l *Limiter) sweep(now time.Time) {
	for k, b := range l.buckets {
		if b.tokens+now.Sub(b.last).Seconds()*l.rate >= l.limit {
			delete(l.buckets, k)
		}
	}
	if len(l.buckets) >= l.maxKeys {
		clear(l.buckets)
	}
}

// Len, izlenen anahtar sayısıdır (test ve gözlem için).
func (l *Limiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
}
