package room

import (
	"crypto/rand"
	"strings"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
)

// Davet kodu (api.md §3): 6 karakter, 0/O ve 1/I/L olmayan 31 karakterlik
// alfabe (~887 milyon olasılık). Kodlar büyük harfle saklanır; giriş büyük/
// küçük harf duyarsızdır.
const (
	InviteAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"
	InviteLength   = 6
)

// NewInviteCode, kriptografik rastgele bir davet kodu üretir. Açık odalarla
// çakışma veritabanındaki kısmi tekil indeksle yakalanır ve yeniden denenir
// (ROOM-003).
func NewInviteCode() (string, error) {
	const n = len(InviteAlphabet)
	limit := 256 - 256%n // modulo yanlılığı olmasın: 248'den büyükler atılır
	out := make([]byte, 0, InviteLength)
	buf := make([]byte, 16)
	for len(out) < InviteLength {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		for _, b := range buf {
			if int(b) < limit && len(out) < InviteLength {
				out = append(out, InviteAlphabet[int(b)%n])
			}
		}
	}
	return string(out), nil
}

// NormalizeInviteCode, kullanıcının yazdığı kodu saklama biçimine getirir:
// boşluk ve tireleri atar, büyük harfe çevirir ve doğrular. Geçersizse
// VALIDATION_FAILED (details.fields.invite_code = invalid).
func NormalizeInviteCode(raw string) (string, error) {
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r == ' ' || r == '-' || r == '\t':
			continue
		case r >= 'a' && r <= 'z':
			r -= 'a' - 'A'
		}
		b.WriteRune(r)
	}
	code := b.String()
	if len(code) != InviteLength {
		return "", apperr.Field("invite_code", apperr.FieldInvalid)
	}
	for i := 0; i < len(code); i++ {
		if strings.IndexByte(InviteAlphabet, code[i]) < 0 {
			return "", apperr.Field("invite_code", apperr.FieldInvalid)
		}
	}
	return code, nil
}
