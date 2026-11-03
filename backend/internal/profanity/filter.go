// Package profanity, Türkçe ağırlıklı küfür filtresidir (SAFE-013): takma
// adlarda reddeder, sohbette kelimeyi *** ile maskeler.
package profanity

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Filter, normalize edilmiş listelerle eşleştiricidir. Durumsuzdur, eşzamanlı
// kullanılabilir.
type Filter struct {
	prefixes, exact, allow [][]rune
}

// Default, words.go listeleriyle kurulmuş filtredir.
var Default = New(prefixes, exact, allow)

// New, listeleri normalize ederek filtre kurar. Küfür kalıplarının tekrarları
// teke iner (eşleşme tekrar dizilerine esnektir); istisnalar olduğu gibi kalır.
func New(prefixList, exactList, allowList []string) *Filter {
	conv := func(list []string, squash bool) [][]rune {
		var out [][]rune
		for _, w := range list {
			r := []rune(Normalize(w))
			if squash {
				r = collapse(r)
			}
			out = append(out, r)
		}
		return out
	}
	return &Filter{prefixes: conv(prefixList, true), exact: conv(exactList, true), allow: conv(allowList, false)}
}

// badWord: normalize edilmiş tek kelime küfür mü?
func (f *Filter) badWord(word string) bool {
	w := []rune(word)
	if len(w) == 0 {
		return false
	}
	for _, e := range f.exact {
		if matchRuns(w, e, true) {
			return true
		}
	}
	for _, a := range f.allow {
		if hasPrefixExact(w, a) {
			return false
		}
	}
	for _, p := range f.prefixes {
		if matchRuns(w, p, false) {
			return true
		}
	}
	return false
}

// span, metindeki bir kelimenin bayt aralığıdır.
type span struct{ start, end int }

// words, metni boşluk (ve takma adlarda alt çizgi) ile ayrılmış kelimelere böler.
func words(text string, sep func(rune) bool) []span {
	var out []span
	start := -1
	for i, r := range text {
		if sep(r) {
			if start >= 0 {
				out = append(out, span{start, i})
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		out = append(out, span{start, len(text)})
	}
	return out
}

// badSpans, küfür içeren aralıklardır: tek tek kelimeler ve harf aralanarak
// yazılmış diziler ("s i k t i r": tek harfli ardışık kelimeler birleştirilir).
func (f *Filter) badSpans(text string, sep func(rune) bool) []span {
	ws := words(text, sep)
	var out []span
	for i := 0; i < len(ws); {
		// Tek harfli kelime dizisi.
		j := i
		var joined strings.Builder
		for j < len(ws) && utf8.RuneCountInString(Normalize(text[ws[j].start:ws[j].end])) == 1 {
			joined.WriteString(Normalize(text[ws[j].start:ws[j].end]))
			j++
		}
		if j-i >= 2 {
			if f.badWord(Normalize(joined.String())) {
				out = append(out, span{ws[i].start, ws[j-1].end})
			}
			i = j
			continue
		}
		if f.badWord(Normalize(text[ws[i].start:ws[i].end])) {
			out = append(out, ws[i])
		}
		i++
	}
	return out
}

func isSpace(r rune) bool { return unicode.IsSpace(r) }

// Contains, metinde küfür var mı (takma adlar: boşluk ve alt çizgi ayırıcı).
func (f *Filter) Contains(text string) bool {
	return len(f.badSpans(text, func(r rune) bool { return unicode.IsSpace(r) || r == '_' })) > 0
}

// Mask, sohbet metnindeki küfürlü kelimeleri (veya aralanmış dizileri) *** ile
// değiştirir; geri kalan metin aynen kalır.
func (f *Filter) Mask(text string) string {
	spans := f.badSpans(text, isSpace)
	if len(spans) == 0 {
		return text
	}
	var b strings.Builder
	prev := 0
	for _, s := range spans {
		b.WriteString(text[prev:s.start])
		b.WriteString("***")
		prev = s.end
	}
	b.WriteString(text[prev:])
	return b.String()
}
