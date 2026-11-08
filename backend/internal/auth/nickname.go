// Package auth, misafir oturumunun yapı taşlarıdır (AUTH-001):
// takma ad kuralları, opak token üretimi ve doğrulaması.
// Tasarım: docs/security.md §1, ADR-0008.
package auth

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
)

// Takma ad uzunluk sınırları (karakter = Unicode kod noktası, NFC sonrası;
// veritabanındaki char_length ile aynı).
const (
	NicknameMin = 3
	NicknameMax = 16
)

// DeletedNickname, hesabı silinen kullanıcının yeni takma adıdır (SAFE-009).
const DeletedNickname = "Silinmiş oyuncu"

// reserved, kullanıcıların alamayacağı adlardır (Türkçe küçük harfle).
var reserved = []string{DeletedNickname, "sistem", "system", "admin", "yönetici", "moderatör", "moderator", "sığınak"}

var lowerTR = cases.Lower(language.Turkish)

// ProfanityChecker, metinde küfür olup olmadığını söyler (SAFE-013'te
// Türkçe kelime listesiyle sağlanır). nil ise denetim yapılmaz.
type ProfanityChecker func(text string) bool

// NormalizeNickname, takma adı karşılaştırma ve saklama biçimine getirir:
// Unicode NFC (aynı görünen harfin tek kodlaması) ve baş/son boşluk kırpma.
func NormalizeNickname(raw string) string {
	return strings.TrimFunc(norm.NFC.String(raw), unicode.IsSpace)
}

// ValidateNickname, takma adı normalleştirip kurallara göre denetler
// (docs/api.md §3):
//   - 3–16 karakter
//   - yalnızca Latin alfabesi harfleri (Türkçe dahil), 0–9, boşluk ve _
//     (Latin'e benzeyen başka alfabe harfleriyle taklit ve birleşik işaret
//     yığınları engellenir)
//   - en az bir harf, ardışık boşluk yok
//   - ayrılmış adlar (ör. "Silinmiş oyuncu") ve küfür yok
//
// Geçerliyse normalleştirilmiş adı, değilse NICKNAME_INVALID döndürür;
// details.fields.nickname alan hatası kodunu taşır.
func ValidateNickname(raw string, profane ProfanityChecker) (string, error) {
	nick := NormalizeNickname(raw)
	fail := func(reason string) (string, error) {
		return "", &apperr.Error{Code: apperr.NicknameInvalid, Message: "takma ad geçersiz",
			Fields: map[string]string{"nickname": reason}}
	}

	if !utf8.ValidString(raw) {
		return fail(apperr.FieldInvalidChars)
	}
	n := utf8.RuneCountInString(nick)
	switch {
	case n == 0:
		return fail(apperr.FieldRequired)
	case n < NicknameMin:
		return fail(apperr.FieldTooShort)
	case n > NicknameMax:
		return fail(apperr.FieldTooLong)
	}

	hasLetter, prevSpace := false, false
	for _, r := range nick {
		switch {
		case unicode.IsLetter(r) && unicode.Is(unicode.Latin, r):
			hasLetter = true
		case r >= '0' && r <= '9', r == '_':
		case r == ' ':
			if prevSpace {
				return fail(apperr.FieldInvalid) // ardışık boşluk
			}
		default:
			return fail(apperr.FieldInvalidChars)
		}
		prevSpace = r == ' '
	}
	if !hasLetter {
		return fail(apperr.FieldInvalid)
	}

	lower := lowerTR.String(nick)
	for _, r := range reserved {
		if lower == lowerTR.String(r) {
			return fail(apperr.FieldNotAllowed)
		}
	}
	if profane != nil && profane(nick) {
		return fail(apperr.FieldProfanity)
	}
	return nick, nil
}

// NicknameKey, takma adın küresel benzersizlik anahtarıdır (users.nickname_key):
// NFC, kırpılmış, Türkçe küçük harf. "IŞIK" ve "ışık" aynı anahtardır.
func NicknameKey(nick string) string {
	return lowerTR.String(NormalizeNickname(nick))
}

// SameNickname, iki takma adın oda içinde aynı sayılıp sayılmadığını
// söyler (Türkçe büyük/küçük harf duyarsız: "IŞIK" = "ışık").
// Veritabanındaki lower() karşılaştırması bunun emniyet ağıdır.
func SameNickname(a, b string) bool {
	return lowerTR.String(NormalizeNickname(a)) == lowerTR.String(NormalizeNickname(b))
}
