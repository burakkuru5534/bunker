// Package httpapi, HTTP katmanıdır: yönlendirici, middleware ve standart
// yanıt / hata biçimi (docs/api.md). İş kuralı içermez; modüllerin
// handler'larını bağlar.
package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
)

// MaxBodyBytes, /v1 REST istek gövdesi üst sınırı (docs/api.md §1).
const MaxBodyBytes = 4 << 10

// Options, yönlendirici ayarları.
type Options struct {
	Logger *slog.Logger
	// ProjectID: Cloud Logging izleme alanı için GCP proje kimliği (ops.).
	ProjectID      string
	MinAppVersion  Version
	RequestTimeout time.Duration
	// TrustedProxyHops: istemci IP'si için güvenilir vekil sayısı (ClientIP).
	TrustedProxyHops int
	// Register, modüllerin rotalarını ekler (ör. "POST /v1/rooms").
	Register func(mux *http.ServeMux)
}

// NewRouter, middleware zinciriyle sarılmış yönlendiriciyi döndürür.
//
// Sıra (dıştan içe): RequestID → ClientIPContext → AccessLog → Recover →
// RequireAppVersion → LimitBody → Timeout → ServeMux. AccessLog, Recover'ın dışında olduğundan
// panic sonucu dönen 500 de loglanır.
//
// Eşleşmeyen yol veya yöntem için standart 404 NOT_FOUND döner ("/" kalıbı
// tüm yöntemlerle eşleştiği için ServeMux düz metin 405 üretmez).
func NewRouter(opts Options) http.Handler {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	mux := http.NewServeMux()
	if opts.Register != nil {
		opts.Register(mux)
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		WriteAppError(w, r, apperr.New(apperr.NotFound, "bulunamadı"))
	})

	return Chain(mux,
		RequestID(),
		ClientIPContext(opts.TrustedProxyHops),
		AccessLog(logger, opts.ProjectID),
		Recover(),
		RequireAppVersion(opts.MinAppVersion),
		LimitBody(MaxBodyBytes),
		Timeout(opts.RequestTimeout),
	)
}
