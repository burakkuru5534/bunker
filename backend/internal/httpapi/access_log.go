package httpapi

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/observability"
)

// AccessLog, her isteğe istek kapsamlı bir logger verir (request_id ve varsa
// Cloud Trace alanıyla) ve istek bitince tek satır erişim logu yazar.
//
// Loglanan: yöntem, yol (sorgu dizesi HARİÇ), durum, süre, yanıt boyutu.
// Loglanmayan: başlıklar, gövde, token, IP (docs/security.md §4).
func AccessLog(base *slog.Logger, projectID string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			logger := base.With("request_id", RequestIDFrom(r.Context()))
			if trace := observability.TraceResource(projectID, observability.TraceID(r)); trace != "" {
				logger = logger.With(observability.TraceKey, trace)
			}
			r = r.WithContext(observability.WithLogger(r.Context(), logger))

			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)

			status := rec.status
			if status == 0 {
				status = http.StatusOK
			}
			latency := time.Since(start)
			level := accessLogLevel(r.URL.Path, status)
			logger.LogAttrs(r.Context(), level, "http istek",
				slog.Group("httpRequest",
					slog.String("requestMethod", r.Method),
					slog.String("requestUrl", r.URL.Path),
					slog.Int("status", status),
					slog.Int64("responseSize", rec.bytes),
					slog.String("latency", fmt.Sprintf("%.3fs", latency.Seconds())),
				),
			)
		})
	}
}

// statusRecorder, durum kodunu ve yazılan bayt sayısını kaydeder.
// WebSocket yükseltmesi için Hijack ve akış için Flush'ı alttaki yazıcıya
// iletir; Unwrap sayesinde http.ResponseController da çalışır.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += int64(n)
	return n, err
}

func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := s.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("httpapi: alttaki yazıcı Hijack desteklemiyor")
	}
	if s.status == 0 {
		s.status = http.StatusSwitchingProtocols
	}
	return h.Hijack()
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// accessLogLevel: 5xx → ERROR, diğerleri INFO. Sağlık denetimleri sık ve
// sıradandır: başarılıysa DEBUG (log hacmi/maliyeti), başarısızsa WARNING
// (hazırlık denetimindeki ayrıntı ayrıca loglanır).
func accessLogLevel(path string, status int) slog.Level {
	switch {
	case isHealthPath(path) && status < 400:
		return slog.LevelDebug
	case isHealthPath(path):
		return slog.LevelWarn
	case status >= 500:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
