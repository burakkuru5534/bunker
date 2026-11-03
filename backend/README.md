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
    (httpapi, auth, room, game, cards, chat, moderation, realtime, persistence, observability — sonraki görevlerde)
```

## Yapılandırma

| Değişken | Varsayılan | Açıklama |
|---|---|---|
| `APP_ENV` | `local` | `local` \| `test` \| `prod` |
| `PORT` | `8080` | Dinlenecek port (Cloud Run verir) |
| `SHUTDOWN_TIMEOUT` | `8s` | SIGTERM sonrası açık istekler için bekleme (0 < süre ≤ 1m) |

Geçersiz değerde servis başlamaz ve tüm hataları birlikte yazar. Örnek: [.env.example](.env.example).

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
