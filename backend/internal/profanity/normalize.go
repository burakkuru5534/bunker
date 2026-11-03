package profanity

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// leet, harf yerine kullanılan rakam ve simgeler.
var leet = map[rune]rune{
	'@': 'a', '4': 'a', '3': 'e', '€': 'e', '1': 'i', '!': 'i', '|': 'i',
	'0': 'o', '5': 's', '$': 's', '7': 't',
}

// ascii, Türkçe harfin Türkçe klavyesi olmayan birinin yazacağı karşılığı.
var ascii = map[rune]rune{'ş': 's', 'ç': 'c', 'ğ': 'g', 'ö': 'o', 'ü': 'u', 'ı': 'i', 'â': 'a', 'î': 'i', 'û': 'u'}

// Normalize, kelimeyi karşılaştırma biçimine getirir: NFC, Türkçe küçük harf,
// simge/rakam → harf, harf olmayanlar atılır (s.i-k → sik). Tekrar eden
// harfler korunur: "sikke" ile "sike…" ayrışsın diye (tekrarlar eşleşmede
// esnek ele alınır, bkz. matchRuns).
func Normalize(word string) string {
	word = strings.ToLowerSpecial(unicode.TurkishCase, norm.NFC.String(word))
	var b strings.Builder
	for _, r := range word {
		if m, ok := leet[r]; ok {
			r = m
		}
		if unicode.IsLetter(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// collapse, ardışık aynı harfleri teke indirir (küfür kalıpları için:
// "yarrak" → "yarak").
func collapse(w []rune) []rune {
	var out []rune
	for i, r := range w {
		if i == 0 || r != w[i-1] {
			out = append(out, r)
		}
	}
	return out
}

// same, metindeki t harfi listedeki e harfiyle eşleşir mi: aynıysa veya t,
// e'nin düz karşılığıysa ("serefsiz" ~ "şerefsiz"). Tersi eşleşmez: metindeki
// Türkçe harf yalnızca kendisiyle eşleşir — "şikayet" ≁ "sik", "sıkıcı" ≁ "sik".
func same(t, e rune) bool { return t == e || ascii[e] == t }

// matchRuns, küfür kalıbı p (tekrarsız) metin kelimesi w'nin başıyla
// eşleşiyor mu: kalıbın her harfi metinde aynı harften oluşan bir diziyi
// karşılar ("siiiktir" ~ "siktir"). whole ise kelimenin tamamı tüketilmeli.
func matchRuns(w, p []rune, whole bool) bool {
	i := 0
	for _, pr := range p {
		if i >= len(w) || !same(w[i], pr) {
			return false
		}
		c := w[i]
		for i < len(w) && w[i] == c {
			i++
		}
	}
	return !whole || i == len(w)
}

// hasPrefixExact, harf harf (tekrarlar dahil) önek eşleşmesi (istisnalar için:
// "sikke" yalnızca gerçekten çift k ile).
func hasPrefixExact(w, p []rune) bool {
	if len(w) < len(p) {
		return false
	}
	for i := range p {
		if !same(w[i], p[i]) {
			return false
		}
	}
	return true
}
