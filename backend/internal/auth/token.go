package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

// Oturum token'ı (ADR-0008): 32 bayt kriptografik rastgele, base64url
// (dolgusuz, 43 karakter). Veritabanında yalnızca SHA-256'sı saklanır.
const (
	TokenBytes  = 32
	TokenLength = 43 // base64.RawURLEncoding.EncodedLen(32)
	// SessionTTL, oturum süresi; refresh token'ı döndürür (yenisini verir).
	SessionTTL = 90 * 24 * time.Hour
)

// ErrInvalidToken, biçimi bozuk token içindir (veritabanına sorulmadan
// reddedilir). İstemciye 401 UNAUTHORIZED olarak döner.
var ErrInvalidToken = errors.New("token biçimi geçersiz")

// Token, ham token ve hash'idir. Ham değer yalnızca istemciye bir kez
// gönderilir; asla loglanmaz veya saklanmaz.
type Token struct {
	Raw  string
	Hash []byte // SHA-256(ham 32 bayt), sessions.token_hash
}

// NewToken, yeni bir oturum token'ı üretir.
func NewToken() (Token, error) {
	b := make([]byte, TokenBytes)
	if _, err := rand.Read(b); err != nil {
		return Token{}, err
	}
	sum := sha256.Sum256(b)
	return Token{Raw: base64.RawURLEncoding.EncodeToString(b), Hash: sum[:]}, nil
}

// HashToken, istemcinin gönderdiği token'ı doğrular ve veritabanında
// aranacak hash'i döndürür. Biçim bozuksa ErrInvalidToken.
func HashToken(raw string) ([]byte, error) {
	if len(raw) != TokenLength {
		return nil, ErrInvalidToken
	}
	b, err := base64.RawURLEncoding.Strict().DecodeString(raw)
	if err != nil || len(b) != TokenBytes {
		return nil, ErrInvalidToken
	}
	sum := sha256.Sum256(b)
	return sum[:], nil
}

// BearerToken, "Authorization: Bearer <token>" başlığından token'ı alır.
// Şema adı büyük/küçük harf duyarsızdır (RFC 6750).
func BearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(strings.TrimSpace(header), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}
