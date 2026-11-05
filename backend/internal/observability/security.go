package observability

import (
	"context"
	"log/slog"
)

// SecurityEvent, güvenlik olayının adıdır (docs/security.md §7, SAFE-011).
// Cloud Logging'de `jsonPayload.security_event` alanıyla süzülür; adlar
// sabittir, uyarı/panel tanımları bunlara dayanır.
type SecurityEvent string

const (
	// Reddedilen istekler (WARNING).
	EventAuthFailed   SecurityEvent = "auth_failed"   // geçersiz/süresi dolmuş/iptal token (401, WS 4001)
	EventAccessDenied SecurityEvent = "access_denied" // yetki/üyelik/atılma (403, WS 4003)
	EventRateLimited  SecurityEvent = "rate_limited"  // sıklık sınırı (429, WS 4008)

	// Hesap ve moderasyon olayları (INFO).
	EventSessionCreated SecurityEvent = "session_created" // misafir hesap açıldı
	EventAccountDeleted SecurityEvent = "account_deleted" // DELETE /v1/me
	EventReportCreated  SecurityEvent = "report_created"
	EventUserBlocked    SecurityEvent = "user_blocked"
	EventMemberKicked   SecurityEvent = "member_kicked"
)

// SecurityEventKey ve ClientIPKey, güvenlik loglarının alan adlarıdır.
const (
	SecurityEventKey = "security_event"
	ClientIPKey      = "client_ip"
)

type clientIPKey struct{}

// WithClientIP, istemci IP'sini bağlama ekler. IP yalnızca güvenlik
// olaylarında loglanır; erişim logu ve diğer loglar onu yazmaz
// (docs/security.md §4).
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey{}, ip)
}

// ClientIPFrom, bağlamdaki istemci IP'sidir (yoksa "").
func ClientIPFrom(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey{}).(string)
	return ip
}

// Security, bir güvenlik olayını tek tip alanlarla loglar: `security_event`,
// bağlamda varsa `client_ip`, ve çağıranın alanları. Logger bağlamdaki istek
// logger'ıdır (request_id, user_id ve izleme alanıyla); bağlamda yoksa
// fallback (o da nil ise slog.Default) kullanılır.
//
// Token, sohbet içeriği, rapor notu gibi değerler alan olarak verilmemelidir;
// gizli anahtar adları zaten maskelenir (replaceAttr).
func Security(ctx context.Context, fallback *slog.Logger, level slog.Level, event SecurityEvent, msg string, attrs ...slog.Attr) {
	logger, ok := ctx.Value(loggerKey{}).(*slog.Logger)
	if !ok {
		logger = fallback
	}
	if logger == nil {
		logger = slog.Default()
	}
	all := make([]slog.Attr, 0, len(attrs)+2)
	all = append(all, slog.String(SecurityEventKey, string(event)))
	if ip := ClientIPFrom(ctx); ip != "" {
		all = append(all, slog.String(ClientIPKey, ip))
	}
	all = append(all, attrs...)
	logger.LogAttrs(ctx, level, msg, all...)
}
