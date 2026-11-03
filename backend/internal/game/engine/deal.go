package engine

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"slices"
)

// MaxNegativePerPlayer, bir oyuncuya düşebilecek en fazla olumsuz kart
// sayısıdır (game-design §6 "Denge etiketi").
const MaxNegativePerPlayer = 2

// CatalogCard, dağıtımın gördüğü kart: kimlik ve gizli denge etiketi
// (+1 olumlu, 0 nötr, −1 olumsuz). Metin motora girmez.
type CatalogCard struct {
	ID      int
	Balance int
}

// Catalog, dağıtım anındaki aktif senaryolar ve kartlardır.
type Catalog struct {
	Scenarios []string
	Cards     map[Category][]CatalogCard
}

// Deal, oyun başlangıç dağıtımının sonucudur.
type Deal struct {
	ScenarioID string
	Players    []SeatedPlayer // koltuk sırasıyla
}

// ErrCatalog: katalog bu oyuncu sayısı için yetersiz (içerik hatası;
// istemciye INTERNAL olarak gider, loglanır).
var ErrCatalog = errors.New("engine: kart kataloğu yetersiz")

// DealGame, oyuncuları rastgele koltuklara oturtur, senaryo seçer ve her
// oyuncuya her kategoriden bir kart verir:
//   - aynı oyunda bir kart iki kez verilmez;
//   - hiçbir oyuncuya MaxNegativePerPlayer'dan fazla olumsuz kart düşmez.
//
// rng sunucuda kriptografik olmalıdır (CryptoRand); testler sabit verir.
func DealGame(userIDs []string, cat Catalog, rng Rand) (Deal, error) {
	n := len(userIDs)
	if n < MinPlayers || n > MaxPlayers {
		return Deal{}, fmt.Errorf("engine: oyuncu sayısı %d (4–10)", n)
	}
	if len(cat.Scenarios) == 0 {
		return Deal{}, fmt.Errorf("%w: aktif senaryo yok", ErrCatalog)
	}

	// 1) Her kategoriden n farklı kart seç (karıştırılmış havuzun başı).
	picked := map[Category][]CatalogCard{}
	rest := map[Category][]CatalogCard{}
	for _, c := range Categories {
		pool := uniqueCards(cat.Cards[c])
		if len(pool) < n {
			return Deal{}, fmt.Errorf("%w: %s için %d kart var, %d gerekli", ErrCatalog, c, len(pool), n)
		}
		shuffle(pool, rng)
		picked[c], rest[c] = pool[:n], pool[n:]
	}

	// 2) Toplam olumsuz kart 2n'yi aşıyorsa, rastgele bir seçili olumsuz
	// kartı aynı kategoriden kullanılmamış olumsuz olmayan bir kartla değiştir.
	for negatives(picked) > MaxNegativePerPlayer*n {
		var swappable []Category
		for _, c := range Categories {
			if slices.ContainsFunc(picked[c], isNegative) && slices.ContainsFunc(rest[c], notNegative) {
				swappable = append(swappable, c)
			}
		}
		if len(swappable) == 0 {
			return Deal{}, fmt.Errorf("%w: olumsuz kart sınırı sağlanamıyor", ErrCatalog)
		}
		c := swappable[rng.IntN(len(swappable))]
		out := pickIndex(picked[c], isNegative, rng)
		in := pickIndex(rest[c], notNegative, rng)
		picked[c][out], rest[c][in] = rest[c][in], picked[c][out]
	}

	// 3) Koltuklar rastgele.
	seats := slices.Clone(userIDs)
	shuffle(seats, rng)
	players := make([]SeatedPlayer, n)
	for i, u := range seats {
		players[i] = SeatedPlayer{UserID: u, Seat: i, Cards: map[Category]int{}}
	}

	// 4) Kartları ver: her kategoride olumsuz kartlar o an en az olumsuz
	// kartı olan oyunculara gider (eşitlik rastgele). Yük farkı hiçbir
	// zaman 1'i aşmaz, böylece en fazla tavan(toplam/n) ≤ 2 olur.
	negCount := make([]int, n)
	for _, c := range Categories {
		cards := picked[c]
		order := make([]int, n)
		for i := range order {
			order[i] = i
		}
		shuffle(order, rng)
		slices.SortStableFunc(order, func(a, b int) int { return negCount[a] - negCount[b] })
		var neg, other []CatalogCard
		for _, card := range cards {
			if isNegative(card) {
				neg = append(neg, card)
			} else {
				other = append(other, card)
			}
		}
		for i, seat := range order {
			var card CatalogCard
			if i < len(neg) {
				card = neg[i]
				negCount[seat]++
			} else {
				card = other[i-len(neg)]
			}
			players[seat].Cards[c] = card.ID
		}
	}
	return Deal{ScenarioID: cat.Scenarios[rng.IntN(len(cat.Scenarios))], Players: players}, nil
}

func isNegative(c CatalogCard) bool  { return c.Balance < 0 }
func notNegative(c CatalogCard) bool { return c.Balance >= 0 }

func negatives(m map[Category][]CatalogCard) int {
	total := 0
	for _, cards := range m {
		for _, c := range cards {
			if isNegative(c) {
				total++
			}
		}
	}
	return total
}

// pickIndex, koşulu sağlayan elemanlardan rastgele birinin indeksidir.
func pickIndex(cards []CatalogCard, ok func(CatalogCard) bool, rng Rand) int {
	var idx []int
	for i, c := range cards {
		if ok(c) {
			idx = append(idx, i)
		}
	}
	return idx[rng.IntN(len(idx))]
}

// uniqueCards, kimliğe göre tekilleştirilmiş, sıralı bir kopyadır
// (girdi sırası sonucu etkilemesin; yalnızca rng etkiler).
func uniqueCards(cards []CatalogCard) []CatalogCard {
	out := slices.Clone(cards)
	slices.SortFunc(out, func(a, b CatalogCard) int { return a.ID - b.ID })
	return slices.CompactFunc(out, func(a, b CatalogCard) bool { return a.ID == b.ID })
}

// shuffle, Fisher–Yates karıştırmasıdır.
func shuffle[T any](s []T, rng Rand) {
	for i := len(s) - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		s[i], s[j] = s[j], s[i]
	}
}

// CryptoRand, crypto/rand ile modulo yanlılığı olmadan [0,n) üretir.
type CryptoRand struct{}

// IntN, n ≤ 0 ise panikler (math/rand ile aynı sözleşme).
func (CryptoRand) IntN(n int) int {
	if n <= 0 {
		panic("engine: IntN için n > 0 olmalı")
	}
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		// crypto/rand Go 1.24+ hata döndürmez; döndürürse güvenli rastgelelik yoktur.
		panic(fmt.Sprintf("engine: crypto/rand: %v", err))
	}
	return int(v.Int64())
}
