// Package apperr, uygulama genelindeki hata türünü ve makine kodlarını tanımlar.
//
// Domain modülleri (auth, room, game, chat …) HTTP bilmeden *Error döndürür;
// HTTP durum kodu ve JSON gövdesi httpapi katmanında bu koddan türetilir.
// Kod listesi docs/api.md §2 ile birebir aynıdır (test ile doğrulanır).
package apperr

import (
	"errors"
	"fmt"
	"maps"
	"time"
)

// Code, istemcinin l10n metnini seçtiği sabit makine kodudur.
type Code string

// docs/api.md §2 "Hata kodları" tablosu.
const (
	ValidationFailed      Code = "VALIDATION_FAILED"
	NicknameInvalid       Code = "NICKNAME_INVALID"
	MessageInvalid        Code = "MESSAGE_INVALID"
	Unauthorized          Code = "UNAUTHORIZED"
	Forbidden             Code = "FORBIDDEN"
	NotAMember            Code = "NOT_A_MEMBER"
	Kicked                Code = "KICKED"
	NotFound              Code = "NOT_FOUND"
	InviteCodeNotFound    Code = "INVITE_CODE_NOT_FOUND"
	RoomFull              Code = "ROOM_FULL"
	GameInProgress        Code = "GAME_IN_PROGRESS"
	NicknameTakenInRoom   Code = "NICKNAME_TAKEN_IN_ROOM"
	NicknameTaken         Code = "NICKNAME_TAKEN"
	AlreadyFriends        Code = "ALREADY_FRIENDS"
	FriendLimit           Code = "FRIEND_LIMIT"
	AlreadyInRoom         Code = "ALREADY_IN_ROOM"
	InvalidPhase          Code = "INVALID_PHASE"
	NotYourTurn           Code = "NOT_YOUR_TURN"
	CardAlreadyRevealed   Code = "CARD_ALREADY_REVEALED"
	CategoryNotAllowed    Code = "CATEGORY_NOT_ALLOWED"
	AlreadyVoted          Code = "ALREADY_VOTED"
	SelfVote              Code = "SELF_VOTE"
	InvalidTarget         Code = "INVALID_TARGET"
	NotAlive              Code = "NOT_ALIVE"
	NotSpectator          Code = "NOT_SPECTATOR"
	StartConditionsNotMet Code = "START_CONDITIONS_NOT_MET"
	StaleState            Code = "STALE_STATE"
	UpgradeRequired       Code = "UPGRADE_REQUIRED"
	RateLimited           Code = "RATE_LIMITED"
	Internal              Code = "INTERNAL"
)

// All, tanımlı tüm kodlardır (tablo sırasıyla).
var All = []Code{
	ValidationFailed, NicknameInvalid, MessageInvalid, Unauthorized, Forbidden,
	NotAMember, Kicked, NotFound, InviteCodeNotFound, RoomFull, GameInProgress,
	NicknameTakenInRoom, AlreadyInRoom, InvalidPhase, NotYourTurn,
	CardAlreadyRevealed, CategoryNotAllowed, AlreadyVoted, SelfVote,
	InvalidTarget, NotAlive, NotSpectator, StartConditionsNotMet, StaleState,
	UpgradeRequired, RateLimited, Internal, NicknameTaken, AlreadyFriends,
	FriendLimit,
}

// Alan hatası kodları (details.fields değerleri). İstemci bunları alan
// altındaki yardım metnine çevirir.
const (
	FieldRequired     = "required"      // eksik
	FieldInvalid      = "invalid"       // biçim/değer geçersiz
	FieldInvalidType  = "invalid_type"  // JSON türü yanlış
	FieldTooShort     = "too_short"     // en az uzunluğun altında
	FieldTooLong      = "too_long"      // en çok uzunluğun üstünde
	FieldTooLarge     = "too_large"     // gövde boyut sınırı
	FieldUnknown      = "unknown_field" // beklenmeyen alan
	FieldNotAllowed   = "not_allowed"   // izin verilen değerlerden değil
	FieldProfanity    = "profanity"     // küfür filtresi
	FieldInvalidChars = "invalid_chars" // izin verilmeyen karakter
)

// Error, domain katmanının döndürdüğü hatadır.
type Error struct {
	Code Code
	// Message yalnızca hata ayıklama içindir; istemciye gider ama
	// gösterilmez. Gizli veya kişisel bilgi içermemelidir.
	Message string
	// Fields, VALIDATION_FAILED için alan → alan hatası kodu.
	Fields map[string]string
	// Details, koda özgü ek bilgiler (ör. min_version).
	Details map[string]any
	// RetryAfter, RATE_LIMITED için istemcinin beklemesi gereken süre.
	RetryAfter time.Duration
	// Err, loglanacak alttaki neden; istemciye asla gönderilmez.
	Err error
}

func (e *Error) Error() string {
	msg := string(e.Code)
	if e.Message != "" {
		msg += ": " + e.Message
	}
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	return msg
}

func (e *Error) Unwrap() error { return e.Err }

// Is, aynı koda sahip *Error'ları eşit sayar: errors.Is(err, apperr.New(apperr.RoomFull, "")).
func (e *Error) Is(target error) bool {
	var t *Error
	return errors.As(target, &t) && t.Code == e.Code
}

// New, yeni bir hata oluşturur.
func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Newf, biçimli mesajla hata oluşturur.
func Newf(code Code, format string, args ...any) *Error {
	return New(code, fmt.Sprintf(format, args...))
}

// Wrap, alttaki nedeni koruyarak hata oluşturur. Neden yalnızca loglanır.
func Wrap(code Code, message string, err error) *Error {
	return &Error{Code: code, Message: message, Err: err}
}

// Validation, alan hatalarıyla VALIDATION_FAILED oluşturur.
func Validation(fields map[string]string) *Error {
	return &Error{Code: ValidationFailed, Message: "doğrulama başarısız", Fields: maps.Clone(fields)}
}

// Field, tek alanlı VALIDATION_FAILED kısayoludur.
func Field(name, reason string) *Error {
	return Validation(map[string]string{name: reason})
}

// WithDetail, ayrıntı ekler ve aynı hatayı döndürür.
func (e *Error) WithDetail(key string, value any) *Error {
	if e.Details == nil {
		e.Details = map[string]any{}
	}
	e.Details[key] = value
	return e
}

// RateLimit, RATE_LIMITED hatası oluşturur.
func RateLimit(retryAfter time.Duration) *Error {
	return &Error{Code: RateLimited, Message: "sıklık sınırı aşıldı", RetryAfter: retryAfter}
}

// CodeOf, hatanın kodunu döndürür; *Error değilse INTERNAL.
func CodeOf(err error) Code {
	if e, ok := errors.AsType[*Error](err); ok {
		return e.Code
	}
	return Internal
}

// FieldErrors, toplu doğrulama için yardımcıdır:
//
//	var fe apperr.FieldErrors
//	fe.Add("nickname", apperr.FieldTooShort)
//	if err := fe.Err(); err != nil { return err }
type FieldErrors map[string]string

// Add, alan için ilk hatayı kaydeder (sonrakiler yok sayılır).
func (f *FieldErrors) Add(field, reason string) {
	if *f == nil {
		*f = FieldErrors{}
	}
	if _, ok := (*f)[field]; !ok {
		(*f)[field] = reason
	}
}

// Err, hata yoksa nil, varsa VALIDATION_FAILED döndürür.
func (f FieldErrors) Err() error {
	if len(f) == 0 {
		return nil
	}
	return Validation(f)
}
