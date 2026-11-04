package game

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"unicode"

	unorm "golang.org/x/text/unicode/norm"

	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
	"github.com/burakkuru5534/bunker/backend/internal/profanity"
	"github.com/burakkuru5534/bunker/backend/internal/testdb"
)

// CARD-011: migration'larla gelen gerçek katalog içeriğinin doğrulaması.
// İçerik değişince (yeni kart, emekliye ayırma) bu test dağıtımın hâlâ her
// oyuncu sayısında çalıştığını ve metinlerin kurallara uyduğunu güvenceler.

// Asgari içerik: kategori başına kart ve olumsuz olmayan kart sayısı.
const (
	minCardsPerCategory   = 30
	minNonNegativePerCat  = 2 * engine.MaxPlayers // dağıtımın değiş tokuş payı
	maxNegativeShareOfCat = 0.4                   // kategoride olumsuz oranı üst sınırı
)

func TestIntegrationCatalogContent(t *testing.T) {
	env := testdb.NewWithContent(t)
	ctx := context.Background()
	q := env.DB.Pool()

	cat, err := loadCatalog(ctx, q)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("senaryolar", func(t *testing.T) {
		want := []string{"meteor", "nuclear_winter", "red_cough", "silent_machines", "solar_storm"}
		if !slices.Equal(cat.Scenarios, want) {
			t.Fatalf("aktif senaryolar %v, beklenen %v", cat.Scenarios, want)
		}
		titles := map[string]bool{}
		for _, id := range cat.Scenarios {
			s, err := loadScenario(ctx, q, id)
			if err != nil {
				t.Fatalf("%s: %v", id, err)
			}
			if titles[s.Title] {
				t.Errorf("%s: başlık tekrar ediyor: %q", id, s.Title)
			}
			titles[s.Title] = true
			if n := sentences(s.Disaster); n < 2 || n > 3 {
				t.Errorf("%s: felaket metni %d cümle (2–3 olmalı)", id, n)
			}
			for _, txt := range append([]string{s.Title, s.Disaster, s.Duration, s.Outside}, s.Bunker...) {
				checkText(t, id, txt)
			}
		}
	})

	t.Run("kartlar", func(t *testing.T) {
		texts := allCardTexts(t, ctx, env)
		for ci, c := range engine.Categories {
			cards := cat.Cards[c]
			neg, pos := 0, 0
			for _, card := range cards {
				lo, hi := (ci+1)*1000, (ci+2)*1000
				if card.ID < lo || card.ID >= hi {
					t.Errorf("%s: kimlik %d aralık dışında [%d, %d)", c, card.ID, lo, hi)
				}
				switch {
				case card.Balance < 0:
					neg++
				case card.Balance > 0:
					pos++
				}
				checkText(t, fmt.Sprintf("%s/%d", c, card.ID), texts[card.ID])
			}
			if len(cards) < minCardsPerCategory {
				t.Errorf("%s: %d aktif kart (en az %d)", c, len(cards), minCardsPerCategory)
			}
			if len(cards)-neg < minNonNegativePerCat {
				t.Errorf("%s: %d olumsuz olmayan kart (en az %d)", c, len(cards)-neg, minNonNegativePerCat)
			}
			if float64(neg) > maxNegativeShareOfCat*float64(len(cards)) {
				t.Errorf("%s: %d/%d olumsuz kart (oran çok yüksek)", c, neg, len(cards))
			}
			if pos == 0 || neg == 0 {
				t.Errorf("%s: her denge etiketinden kart olmalı (olumlu %d, olumsuz %d)", c, pos, neg)
			}
		}
	})

	// Gerçek katalogla, sunucunun kriptografik kaynağıyla her oyuncu
	// sayısında dağıtım: hata yok, kart tekrarı yok, olumsuz sınırı korunur.
	t.Run("dağıtım", func(t *testing.T) {
		bal := map[int]int{}
		for _, cards := range cat.Cards {
			for _, c := range cards {
				bal[c.ID] = c.Balance
			}
		}
		for n := engine.MinPlayers; n <= engine.MaxPlayers; n++ {
			ids := make([]string, n)
			for i := range ids {
				ids[i] = fmt.Sprintf("u%d", i)
			}
			for range 300 {
				d, err := engine.DealGame(ids, cat, engine.CryptoRand{})
				if err != nil {
					t.Fatalf("%d oyuncu: %v", n, err)
				}
				seen := map[int]bool{}
				for _, p := range d.Players {
					neg := 0
					for _, id := range p.Cards {
						if seen[id] {
							t.Fatalf("%d oyuncu: kart %d iki kez dağıtıldı", n, id)
						}
						seen[id] = true
						if bal[id] < 0 {
							neg++
						}
					}
					if neg > engine.MaxNegativePerPlayer {
						t.Fatalf("%d oyuncu: bir oyuncuya %d olumsuz kart", n, neg)
					}
				}
			}
		}
	})
}

func allCardTexts(t *testing.T, ctx context.Context, env *testdb.Env) map[int]string {
	t.Helper()
	rows, err := env.DB.Pool().Query(ctx, `select id, text from card_definitions where active`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[int]string{}
	for rows.Next() {
		var (
			id   int
			text string
		)
		if err := rows.Scan(&id, &text); err != nil {
			t.Fatal(err)
		}
		out[id] = text
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// checkText: boş değil, NFC, kırpılmış, kontrol/biçim karakteri ve çift
// boşluk yok, küfür filtresinden temiz geçer.
func checkText(t *testing.T, where, s string) {
	t.Helper()
	switch {
	case strings.TrimSpace(s) == "":
		t.Errorf("%s: boş metin", where)
	case s != strings.TrimSpace(s):
		t.Errorf("%s: baş/son boşluk: %q", where, s)
	case !unorm.NFC.IsNormalString(s):
		t.Errorf("%s: NFC değil: %q", where, s)
	case strings.Contains(s, "  "):
		t.Errorf("%s: çift boşluk: %q", where, s)
	case strings.ContainsFunc(s, func(r rune) bool { return unicode.IsControl(r) || unicode.Is(unicode.Cf, r) }):
		t.Errorf("%s: kontrol/biçim karakteri: %q", where, s)
	case profanity.Default.Contains(s):
		t.Errorf("%s: küfür filtresine takılıyor: %q", where, s)
	}
}

// sentences, metindeki cümle sayısıdır (. ! ? ile biten parçalar).
func sentences(s string) int {
	return len(strings.FieldsFunc(s, func(r rune) bool { return r == '.' || r == '!' || r == '?' }))
}
