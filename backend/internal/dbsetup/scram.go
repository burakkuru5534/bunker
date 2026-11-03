// Package dbsetup, veritabanı rol kurulumu için yardımcılardır (cmd/dbsetup).
//
// Şifre veritabanına hiçbir zaman düz metin gitmez: SCRAM-SHA-256 doğrulayıcısı
// yerelde hesaplanır ve ALTER ROLE ... PASSWORD '<doğrulayıcı>' ile verilir.
// PostgreSQL ön-hash'lenmiş SCRAM doğrulayıcısını olduğu gibi saklar; böylece
// şifre SQL editörü geçmişine veya sunucu loglarına düşmez.
package dbsetup

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// scramIterations, PostgreSQL'in varsayılanıdır (scram_iterations).
const scramIterations = 4096

// passwordAlphabet, URL kodlaması gerektirmeyen karakterlerdir.
const passwordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"

// NewPassword, n karakterlik rastgele şifre üretir (57 karakterli alfabe;
// 32 karakter ≈ 186 bit).
func NewPassword(n int) (string, error) {
	if n < 24 {
		return "", fmt.Errorf("şifre en az 24 karakter olmalı: %d", n)
	}
	out := make([]byte, n)
	buf := make([]byte, 1)
	for i := 0; i < n; {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		// Reddetme örneklemesi: modulo yanlılığı olmasın.
		if int(buf[0]) >= 256-256%len(passwordAlphabet) {
			continue
		}
		out[i] = passwordAlphabet[int(buf[0])%len(passwordAlphabet)]
		i++
	}
	return string(out), nil
}

// ScramVerifier, PostgreSQL'in pg_authid.rolpassword biçiminde SCRAM-SHA-256
// doğrulayıcısını üretir:
//
//	SCRAM-SHA-256$<iter>:<salt>$<StoredKey>:<ServerKey>
//
// password yalnızca ASCII olmalıdır (SASLprep gerektirmez); NewPassword öyle üretir.
func ScramVerifier(password string, salt []byte, iterations int) (string, error) {
	for i := 0; i < len(password); i++ {
		if password[i] < 0x21 || password[i] > 0x7e {
			return "", fmt.Errorf("şifre yalnızca yazdırılabilir ASCII içermeli")
		}
	}
	if len(salt) < 16 {
		return "", fmt.Errorf("tuz en az 16 bayt olmalı")
	}
	salted, err := pbkdf2.Key(sha256.New, password, salt, iterations, sha256.Size)
	if err != nil {
		return "", err
	}
	clientKey := hmacSHA256(salted, "Client Key")
	storedKey := sha256.Sum256(clientKey)
	serverKey := hmacSHA256(salted, "Server Key")
	b64 := base64.StdEncoding.EncodeToString
	return fmt.Sprintf("SCRAM-SHA-256$%d:%s$%s:%s", iterations, b64(salt), b64(storedKey[:]), b64(serverKey)), nil
}

// NewScramVerifier, rastgele tuzla doğrulayıcı üretir.
func NewScramVerifier(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	return ScramVerifier(password, salt, scramIterations)
}

func hmacSHA256(key []byte, msg string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(msg))
	return m.Sum(nil)
}
