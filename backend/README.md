# Sığınak — Backend (Go)

Modüler monolith: REST + WebSocket, tek kalıcı veri kaynağı PostgreSQL.
Mimari: [../docs/architecture.md](../docs/architecture.md), API: [../docs/api.md](../docs/api.md).

| | |
|---|---|
| Modül | `github.com/burakkuru5534/bunker/backend` |
| Go | 1.27.0 |
| Giriş noktası | `cmd/server` |
| Dış bağımlılık | Yok (şimdilik yalnızca standart kütüphane) |

## Yapı

```
backend/
  cmd/server/          main: yapılandırma, HTTP sunucusu, graceful shutdown
  internal/
    config/            ortam değişkenleri ve doğrulama
    httpapi/           yönlendirici, middleware, standart JSON yanıt/hata biçimi, erişim logu
    observability/     slog tabanlı yapılandırılmış log, maskeleme, Cloud Trace kimliği
    (auth, room, game, cards, chat, moderation, realtime, persistence — sonraki görevlerde)
```

## Yapılandırma

| Değişken | Varsayılan | Açıklama |
|---|---|---|
| `APP_ENV` | `local` | `local` \| `test` \| `prod` |
| `PORT` | `8080` | Dinlenecek port (Cloud Run verir) |
| `SHUTDOWN_TIMEOUT` | `8s` | SIGTERM sonrası açık istekler için bekleme (0 < süre ≤ 1m) |
| `MIN_APP_VERSION` | `0.1.0` | Desteklenen en eski mobil sürüm; eskisi `426 UPGRADE_REQUIRED` alır |
| `REQUEST_TIMEOUT` | `10s` | REST isteklerinin bağlam süresi (WebSocket hariç) |
| `LOG_LEVEL` | `info` | `debug` \| `info` \| `warn` \| `error` |
| `GCP_PROJECT_ID` | boş | Doluysa loglara Cloud Trace bağlantısı (`logging.googleapis.com/trace`) eklenir; gizli değildir |

Geçersiz değerde servis başlamaz ve tüm hataları birlikte yazar. Örnek: [.env.example](.env.example).

## HTTP katmanı (`internal/httpapi`)

Middleware sırası (dıştan içe):

| # | Middleware | Kapsam | Davranış |
|---|---|---|---|
| 1 | `RequestID` | tümü | Geçerli `X-Request-ID` korunur, yoksa 32 hex üretilir; yanıta ve bağlama eklenir |
| 2 | `AccessLog` | tümü | İstek logger'ını bağlama koyar (`request_id`, varsa trace); bitişte tek `httpRequest` satırı yazar |
| 3 | `Recover` | tümü | panic → `500 INTERNAL` (ayrıntı istemciye gitmez, `request_id` ile loglanır) |
| 4 | `RequireAppVersion` | `/v1/*` | Eksik/biçimsiz → `400 VALIDATION_FAILED`; `MIN_APP_VERSION`'dan eski → `426 UPGRADE_REQUIRED` (`details.min_version`) |
| 5 | `LimitBody` | `/v1/*` (WS hariç) | Gövde ≤ 4 KB; aşılırsa `400 VALIDATION_FAILED` |
| 6 | `Timeout` | `/v1/*` (WS hariç) | İstek bağlamına `REQUEST_TIMEOUT` süresi |

- Eşleşmeyen yol **veya yöntem** → `404 NOT_FOUND` (JSON). Düz metin yanıt üretilmez.
- Hata gövdesi her zaman `{"error":{"code","message","request_id","details":{}}}` (docs/api.md §2).
- Modüller rotalarını `httpapi.Options.Register` ile ekler (ör. `mux.HandleFunc("POST /v1/rooms", ...)`).

## Loglama (`internal/observability`)

- `APP_ENV=local` → okunabilir metin; `test`/`prod` → tek satır JSON (Cloud Logging otomatik ayrıştırır).
- Alan adları Cloud Logging'e uygun: `severity` (DEBUG/INFO/WARNING/ERROR), `message`, `httpRequest`, `logging.googleapis.com/trace`.
- Erişim logu: yöntem, **sorgu dizesi olmadan** yol, durum, yanıt boyutu, süre. 5xx → `ERROR`. İstek gövdesi ve başlıklar loglanmaz.
- Maskeleme: anahtarı `authorization`, `token`, `password`, `secret`, `database_url`, `dsn`, `cookie`, `api_key`, `credential`, `private_key` içeren alanlar ve `Bearer ` içeren değerler `[REDACTED]` olarak yazılır (gruplar dahil). Bu bir emniyet ağıdır; gizli değerleri hiç loglamamak esastır.
- Handler içinde: `observability.LoggerFrom(r.Context()).Info(...)` → satır `request_id` taşır.

## Çalıştırma

```bash
cd backend
go run ./cmd/server
```

Windows PowerShell'de farklı bir port:

```powershell
$env:PORT="9090"; go run ./cmd/server
```

Ctrl+C → açık istekler `SHUTDOWN_TIMEOUT` boyunca beklenir, sonra servis durur.

## Kontroller (CI ile aynı)

```bash
cd backend
gofmt -l .          # çıktı boş olmalı
go vet ./...
go test -race ./...
```

> `-race` Windows'ta cgo (gcc) gerektirir; yoksa `go test ./...` çalıştırın. CI (Linux) her zaman `-race` ile çalışır.
