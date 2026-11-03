package engine

import (
	"errors"
	"math/rand/v2"
	"slices"
	"testing"
)

// catalog: kategori başına size kart; her kategorinin ilk neg kartı olumsuz.
func catalog(size, neg int) Catalog {
	cat := Catalog{Scenarios: []string{"nuclear_winter", "red_cough", "meteor"}, Cards: map[Category][]CatalogCard{}}
	for ci, c := range Categories {
		for i := 0; i < size; i++ {
			b := 1 - i%2
			if i < neg {
				b = -1
			}
			cat.Cards[c] = append(cat.Cards[c], CatalogCard{ID: (ci+1)*1000 + i, Balance: b})
		}
	}
	return cat
}

func users(n int) []string {
	var out []string
	for i := range n {
		out = append(out, uid(i))
	}
	return out
}

func balanceOf(cat Catalog) map[int]int {
	m := map[int]int{}
	for _, cards := range cat.Cards {
		for _, c := range cards {
			m[c.ID] = c.Balance
		}
	}
	return m
}

// checkDeal, dağıtımın tüm değişmezlerini doğrular ve motora girebildiğini gösterir.
func checkDeal(t *testing.T, d Deal, ids []string, cat Catalog) {
	t.Helper()
	bal := balanceOf(cat)
	seen := map[int]bool{}
	var got []string
	for i, p := range d.Players {
		if p.Seat != i || len(p.Cards) != len(Categories) {
			t.Fatalf("oyuncu %d: %+v", i, p)
		}
		got = append(got, p.UserID)
		neg := 0
		for c, id := range p.Cards {
			if seen[id] {
				t.Fatalf("kart %d iki kez", id)
			}
			seen[id] = true
			if id/1000 != slices.Index(Categories, c)+1 {
				t.Fatalf("kart %d yanlış kategoride (%s)", id, c)
			}
			if bal[id] < 0 {
				neg++
			}
		}
		if neg > MaxNegativePerPlayer {
			t.Fatalf("%s: %d olumsuz kart", p.UserID, neg)
		}
	}
	slices.Sort(got)
	want := slices.Sorted(slices.Values(ids))
	if !slices.Equal(got, want) {
		t.Fatalf("oyuncular %v, beklenen %v", got, want)
	}
	if !slices.Contains(cat.Scenarios, d.ScenarioID) {
		t.Fatalf("senaryo %q", d.ScenarioID)
	}
	if _, err := New(DefaultDurations).NewGame(d.Players, 1); err != nil {
		t.Fatalf("motor dağıtımı reddetti: %v", err)
	}
}

func TestDealInvariants(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	// Olumsuz yoğunluğu: yok, normal, sınırda (her kategoride n olumsuz → 5n > 2n, takas gerekir).
	for _, neg := range []int{0, 8, 10, 25} {
		cat := catalog(30, neg)
		for n := MinPlayers; n <= MaxPlayers; n++ {
			for range 200 {
				ids := users(n)
				d, err := DealGame(ids, cat, rng)
				// Mümkünlük: olumsuz olmayan kartlar en az 5n − 2n yuvayı doldurabilmeli.
				feasible := len(Categories)*min(n, 30-neg) >= (len(Categories)-MaxNegativePerPlayer)*n
				if !feasible {
					if !errors.Is(err, ErrCatalog) {
						t.Fatalf("neg=%d n=%d: ErrCatalog beklenirdi, %v", neg, n, err)
					}
					continue
				}
				if err != nil {
					t.Fatalf("neg=%d n=%d: %v", neg, n, err)
				}
				checkDeal(t, d, ids, cat)
			}
		}
	}
}

func TestDealExactCatalogAndTightNegatives(t *testing.T) {
	// Kategori başına tam n kart; olumsuzlar tam 2n (takassız mümkün olmalı).
	n := 10
	cat := Catalog{Scenarios: []string{"s1"}, Cards: map[Category][]CatalogCard{}}
	negLeft := 2 * n
	for ci, c := range Categories {
		for i := range n {
			b := 0
			if negLeft > 0 && ci < 2 {
				b, negLeft = -1, negLeft-1
			}
			cat.Cards[c] = append(cat.Cards[c], CatalogCard{ID: (ci+1)*1000 + i, Balance: b})
		}
	}
	rng := rand.New(rand.NewPCG(3, 4))
	for range 500 {
		d, err := DealGame(users(n), cat, rng)
		if err != nil {
			t.Fatal(err)
		}
		checkDeal(t, d, users(n), cat)
	}
}

func TestDealCatalogErrors(t *testing.T) {
	rng := rand.New(rand.NewPCG(5, 6))
	small := catalog(30, 0)
	small.Cards[Hobby] = small.Cards[Hobby][:5]
	dup := catalog(30, 0)
	dup.Cards[Item] = slices.Repeat(dup.Cards[Item][:3], 10) // 30 satır, 3 farklı kart
	noScenario := catalog(30, 0)
	noScenario.Scenarios = nil
	allNeg := catalog(10, 10) // her kategori tamamen olumsuz: 5 olumsuz/oyuncu kaçınılmaz
	cases := map[string]struct {
		cat Catalog
		n   int
	}{
		"az kart":       {small, 6},
		"tekrar kimlik": {dup, 4},
		"senaryo yok":   {noScenario, 4},
		"hepsi olumsuz": {allNeg, 4},
	}
	for name, c := range cases {
		if _, err := DealGame(users(c.n), c.cat, rng); !errors.Is(err, ErrCatalog) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, n := range []int{3, 11} {
		if _, err := DealGame(users(n), catalog(30, 0), rng); err == nil || errors.Is(err, ErrCatalog) {
			t.Errorf("%d oyuncu: %v", n, err)
		}
	}
	// 5 kartlık hobi kataloğu 5 oyuncuya yeter.
	if _, err := DealGame(users(5), small, rng); err != nil {
		t.Fatal(err)
	}
}

func TestDealDeterministicAndInputOrderIndependent(t *testing.T) {
	cat := catalog(30, 8)
	a, _ := DealGame(users(7), cat, rand.New(rand.NewPCG(9, 9)))
	// Katalog satır sırası farklı, aynı rng → aynı sonuç.
	rev := Catalog{Scenarios: cat.Scenarios, Cards: map[Category][]CatalogCard{}}
	for c, cards := range cat.Cards {
		r := slices.Clone(cards)
		slices.Reverse(r)
		rev.Cards[c] = r
	}
	b, _ := DealGame(users(7), rev, rand.New(rand.NewPCG(9, 9)))
	if a.ScenarioID != b.ScenarioID || len(a.Players) != len(b.Players) {
		t.Fatal("farklı sonuç")
	}
	for i := range a.Players {
		if a.Players[i].UserID != b.Players[i].UserID || !mapsEqual(a.Players[i].Cards, b.Players[i].Cards) {
			t.Fatalf("koltuk %d farklı", i)
		}
	}
}

func mapsEqual(a, b map[Category]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// Koltuk, senaryo ve kart seçimi yaklaşık düzgün dağılmalı (kaba sınırlar;
// yanlış karıştırma — ör. ilk oyuncu hep 0. koltuk — bunu bozar).
func TestDealDistribution(t *testing.T) {
	const trials = 20000
	cat := catalog(30, 0)
	rng := rand.New(rand.NewPCG(7, 8))
	seatOfU0 := make([]int, 6)
	scenario := map[string]int{}
	cardUse := map[int]int{}
	for range trials {
		d, _ := DealGame(users(6), cat, rng)
		for _, p := range d.Players {
			if p.UserID == uid(0) {
				seatOfU0[p.Seat]++
			}
			cardUse[p.Cards[Profession]]++
		}
		scenario[d.ScenarioID]++
	}
	within := func(name string, got, want int) {
		if diff := float64(got-want) / float64(want); diff > 0.08 || diff < -0.08 {
			t.Errorf("%s: %d, beklenen ~%d", name, got, want)
		}
	}
	for s, c := range seatOfU0 {
		within("koltuk "+string(rune('0'+s)), c, trials/6)
	}
	for s, c := range scenario {
		within(s, c, trials/3)
	}
	for _, c := range cardUse {
		within("kart", c, trials*6/30)
	}
	if len(cardUse) != 30 {
		t.Fatalf("%d meslek kartı kullanıldı", len(cardUse))
	}
}

func TestCryptoRand(t *testing.T) {
	var r CryptoRand
	counts := make([]int, 7)
	for range 7000 {
		v := r.IntN(7)
		if v < 0 || v >= 7 {
			t.Fatalf("aralık dışı %d", v)
		}
		counts[v]++
	}
	for i, c := range counts {
		if c < 800 || c > 1200 {
			t.Errorf("%d: %d", i, c)
		}
	}
	if r.IntN(1) != 0 {
		t.Fatal("IntN(1)")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("IntN(0) paniklemedi")
		}
	}()
	r.IntN(0)
}

func TestDealWithCryptoRand(t *testing.T) {
	cat := catalog(30, 10)
	for range 50 {
		d, err := DealGame(users(10), cat, CryptoRand{})
		if err != nil {
			t.Fatal(err)
		}
		checkDeal(t, d, users(10), cat)
	}
}
