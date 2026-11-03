package httpapi

import "github.com/burakkuru5534/bunker/backend/internal/apperr"

// WebSocket kapanış kodları (docs/api.md §5). Yetki, üyelik ve sürüm
// hataları WebSocket'te HTTP durumu olarak değil, yükseltmeden sonra bu
// kodlarla bildirilir: mobil istemci yükseltme öncesi HTTP durumunu okuyamaz.
const (
	CloseNormal          = 1000
	CloseInternal        = 1011
	CloseServiceRestart  = 1012
	CloseUnauthorized    = 4001
	CloseNotMember       = 4003
	CloseTooMany         = 4008
	CloseSlowClient      = 4009
	CloseUpgradeRequired = 4026
)

// CloseCodeFor, bağlantı kurulurken oluşan hatanın kapanış kodudur.
// Bilinmeyen hatalar 1011'dir (istemci yeniden dener). Sürüm denetimi için
// VersionCloseCode kullanılır (eksik/biçimsiz başlık da 4026'dır).
func VersionCloseCode(err error) int {
	if err == nil {
		return 0
	}
	return CloseUpgradeRequired
}

// CloseCodeFor, yetki ve üyelik hatalarını kapanış koduna çevirir.
func CloseCodeFor(err error) int {
	switch apperr.CodeOf(err) {
	case apperr.Unauthorized:
		return CloseUnauthorized
	case apperr.NotAMember, apperr.Kicked, apperr.NotFound, apperr.Forbidden, apperr.ValidationFailed:
		// Geçersiz room_id dahil: odaya bağlanılamaz → ana ekran.
		return CloseNotMember
	case apperr.UpgradeRequired:
		return CloseUpgradeRequired
	case apperr.RateLimited:
		return CloseTooMany
	default:
		return CloseInternal
	}
}
