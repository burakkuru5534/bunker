# Sığınak — Backend (Go)

Modüler monolith: REST + WebSocket, tek kalıcı veri kaynağı PostgreSQL.
Mimari: [../docs/architecture.md](../docs/architecture.md), API: [../docs/api.md](../docs/api.md).

| | |
|---|---|
| Modül | `github.com/burakkuru5534/bunker/backend` |
| Go | 1.27.3 (`go.mod`; daha eski yerel Go, `GOTOOLCHAIN=auto` ile bu sürümü kendisi indirir) |
| Giriş noktası | `cmd/server` |
| Dış bağımlılık | `github.com/jackc/pgx/v5` v5.11.0 (PostgreSQL sürücüsü + havuz) ve onun dolaylı bağımlılıkları |

## Yapı

```
backend/
  cmd/server/          main: yapılandırma, HTTP sunucusu, graceful shutdown
  cmd/dbsetup/         ortam rolü kurulumu ve denetimi (init, check)
  cmd/migrate/         migration'lar (status, up, new, lint)
  internal/
    apperr/            uygulama hata türü ve makine kodları (HTTP bilmez)
    config/            ortam değişkenleri ve doğrulama
    httpapi/           yönlendirici, middleware, standart JSON yanıt/hata biçimi, erişim logu
    observability/     slog tabanlı yapılandırılmış log, maskeleme, Cloud Trace kimliği
    persistence/       pgx bağlantı havuzu ve açılış güvenlik denetimleri
    dbsetup/           Supabase rol kurulumu (SCRAM şifre, SQL, denetim)
    migrate/           migration yükleme, kurallar, uygulama, yetki eşitleme
    auth/              misafir oturumu: takma ad, token, oturum servisi ve önbellek
    testdb/            entegrasyon testleri için yalıtılmış şema (yalnızca testler)
    (auth, room, game, cards, chat, moderation, realtime — sonraki görevlerde)
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
| `DATABASE_URL` | boş | **Gizli.** PostgreSQL adresi. test/prod'da zorunlu; yerelde boşsa servis veritabanısız başlar |
| `DB_MAX_CONNS` | prod `5`, diğer `3` | Havuz üst sınırı (1–20, ADR-0010) |
| `TRUSTED_PROXY_HOPS` | local `0`, test/prod `1` | Önümüzdeki güvenilir vekil sayısı; istemci IP'si `X-Forwarded-For`'un sağdan bu kadarıncı girdisi (sıklık sınırı). Cloud Run'da ilk dağıtımda doğrulanır |
| `DB_ACQUIRE_TIMEOUT` | `3s` | Havuzdan bağlantı için en uzun bekleme (100ms–10s); dolarsa istek hızlıca `500` alır ve `veritabanı havuzu dolu` uyarısı loglanır |
| `DB_SCHEMA` | prod `bunker_prod`, diğer `bunker_test` | Rolün `search_path`'inde beklenen şema; açılışta doğrulanır |

Geçersiz değerde servis başlamaz ve tüm hataları birlikte yazar. Örnek: [.env.example](.env.example).

**Yerel `.env`:** `APP_ENV` boş veya `local` ise çalışma klasöründeki `.env` okunur (`cd backend; go run ./cmd/server`). Gerçek ortam değişkenleri `.env`'den önceliklidir. Bulutta (`test`/`prod`) dosya okunmaz. `.env` git'e girmez.

**Gizli değerler:** `DATABASE_URL`, `config.Secret` türündedir; `fmt`, `slog` ve JSON çıktılarında `[REDACTED]` görünür. Gerçek değer yalnızca `Reveal()` ile sürücüye verilir. Yapılandırma ve bağlantı hataları adresi içermez.

## HTTP katmanı (`internal/httpapi`)

Middleware sırası (dıştan içe):

| # | Middleware | Kapsam | Davranış |
|---|---|---|---|
| 1 | `RequestID` | tümü | Geçerli `X-Request-ID` korunur, yoksa 32 hex üretilir; yanıta ve bağlama eklenir |
| 2 | `AccessLog` | tümü | İstek logger'ını bağlama koyar (`request_id`, varsa trace); bitişte tek `httpRequest` satırı yazar (başarılı `/health*` → DEBUG) |
| 3 | `Recover` | tümü | panic → `500 INTERNAL` (ayrıntı istemciye gitmez, `request_id` ile loglanır) |
| 4 | `RequireAppVersion` | `/v1/*` | Eksik/biçimsiz → `400 VALIDATION_FAILED`; `MIN_APP_VERSION`'dan eski → `426 UPGRADE_REQUIRED` (`details.min_version`) |
| 5 | `LimitBody` | `/v1/*` (WS hariç) | Gövde ≤ 4 KB; aşılırsa `400 VALIDATION_FAILED` |
| 6 | `Timeout` | `/v1/*` (WS hariç) | İstek bağlamına `REQUEST_TIMEOUT` süresi |

- Eşleşmeyen yol **veya yöntem** → `404 NOT_FOUND` (JSON). Düz metin yanıt üretilmez.
- Hata gövdesi her zaman `{"error":{"code","message","request_id","details":{}}}` (docs/api.md §2).
- Modüller rotalarını `httpapi.Options.Register` ile ekler.

## Sağlık uç noktaları

| Yol | Yanıt | Kullanım |
|---|---|---|
| `GET /health` | `200 {"status":"ok"}` | Süreç ayakta mı; bağımlılıklara bakmaz. Cloud Run başlangıç/canlılık denetimi |
| `GET /health/ready` | `200 {"status":"ready"}` / `503 {"status":"not_ready"}` | `httpapi.Health.Checks` paralel, her biri ≤ 2 sn. Başarısız denetimin adı ve hatası yalnızca loglanır |

`X-App-Version` istemez, `HEAD` destekler, `Cache-Control: no-store`. Başarılı sağlık istekleri erişim logunda `DEBUG` seviyesindedir (prod'da `info` olduğu için yazılmaz → log hacmi düşük); başarısızlar `WARNING`. Veritabanı denetimi DB-001'de eklenecek:

```go
health := &httpapi.Health{Checks: []httpapi.ReadinessCheck{{Name: "database", Check: pool.Ping}}}
```

## Hatalar (`internal/apperr`)

Domain kodu HTTP bilmez; `*apperr.Error` döndürür. HTTP durumu ve JSON gövdesi `httpapi` katmanında koddan türetilir.

```go
mux.Handle("POST /v1/rooms/join", httpapi.Handle(func(w http.ResponseWriter, r *http.Request) error {
	var req joinRequest
	if err := httpapi.DecodeJSON(r, &req); err != nil { // katı: bilinmeyen alan, tür, boyut
		return err
	}
	var fe apperr.FieldErrors
	if len(req.InviteCode) != 6 {
		fe.Add("invite_code", apperr.FieldInvalid)
	}
	if err := fe.Err(); err != nil {
		return err // 400 VALIDATION_FAILED, details.fields
	}
	room, err := rooms.Join(r.Context(), req.InviteCode) // ör. apperr.New(apperr.RoomFull, "") → 409
	if err != nil {
		return err
	}
	httpapi.WriteJSON(w, http.StatusOK, room)
	return nil
}))
```

| Durum | Sonuç |
|---|---|
| `*apperr.Error` (sarılı olsa da) | Kodun HTTP durumu; `Message`, `Details`, `Fields` → `details.fields` |
| `apperr.RateLimit(d)` | `429` + `Retry-After` (saniye, yukarı yuvarlanır, ≥ 1) |
| `apperr.Wrap(code, msg, cause)` | `cause` yalnızca loglanır, istemciye gitmez |
| Düz `error`, `INTERNAL` | `500`, genel mesaj, boş `details`; asıl hata `ERROR` olarak loglanır |
| `context.DeadlineExceeded` | `500`, `WARNING` |
| İstemci bağlantıyı kapattı | Yanıt yazılmaz |

Kod listesi `docs/api.md` §2 ile test edilerek eşlenir (`TestErrorCodesMatchDocs`); yeni kod önce belgeye eklenir.

## Loglama (`internal/observability`)

- `APP_ENV=local` → okunabilir metin; `test`/`prod` → tek satır JSON (Cloud Logging otomatik ayrıştırır).
- Alan adları Cloud Logging'e uygun: `severity` (DEBUG/INFO/WARNING/ERROR), `message`, `httpRequest`, `logging.googleapis.com/trace`.
- Erişim logu: yöntem, **sorgu dizesi olmadan** yol, durum, yanıt boyutu, süre. 5xx → `ERROR`. İstek gövdesi ve başlıklar loglanmaz.
- Maskeleme: anahtarı `authorization`, `token`, `password`, `secret`, `database_url`, `dsn`, `cookie`, `api_key`, `credential`, `private_key` içeren alanlar ve `Bearer ` içeren değerler `[REDACTED]` olarak yazılır (gruplar dahil). Bu bir emniyet ağıdır; gizli değerleri hiç loglamamak esastır.
- Handler içinde: `observability.LoggerFrom(r.Context()).Info(...)` → satır `request_id` taşır.

## Veritabanı (`internal/persistence`)

Supabase'e Supavisor session mode (5432, IPv4) ile `pgxpool` üzerinden bağlanılır (ADR-0009, ADR-0010). Havuz: `MinConns 0`, boşta 5 dk, ömür 30 dk, sağlık 1 dk, bağlantı zaman aşımı 5 sn.

Açılışta (en çok 15 sn) doğrulananlar; biri başarısızsa servis **başlamaz**:

| Denetim | test / prod | local |
|---|---|---|
| TLS | `sslmode=verify-full` zorunlu | Yerel makine (localhost / 127.0.0.1 / ::1 / unix soketi): serbest. Uzak sunucu: en az `require` (şifresiz reddedilir), `verify-full` değilse uyarı |
| Rol | Süper kullanıcı, `BYPASSRLS`, `postgres`, `service_role`, `supabase_*` vb. yönetim rolleri reddedilir | Denetlenmez |
| Şema | `current_schema()` = `DB_SCHEMA` | Aynı |

`/health/ready`, `database` denetimiyle havuzdan ping atar. Kapanışta HTTP sunucusu durduktan sonra havuz kapatılır.

**Erişim (`persistence.DB`, DB-005):** Kod havuza doğrudan değil, iki yardımcıyla erişir; bağlantı fonksiyon dışına sızmaz:

```go
err := db.Tx(ctx, func(tx pgx.Tx) error {        // READ COMMITTED; hata/panic → rollback
	// select ... for update; insert ...; (yayın commit'ten SONRA)
	return nil
})
err := db.Read(ctx, func(q persistence.Querier) error { ... })  // transaction'sız
```

Bağlantı `DB_ACQUIRE_TIMEOUT` içinde alınamazsa `persistence.ErrPoolExhausted` döner (`context.DeadlineExceeded`'ı sarar → 500, WARNING). 500 ms'den uzun beklemeler "geç alındı" uyarısıyla loglanır.

### Supabase kurulumu (`cmd/dbsetup`, DB-001b)

Her ortam (test, prod) kendi şeması ve uygulama rolüyle çalışır (ADR-0009). `dbsetup` bu kurulumu yapar:

- Güçlü bir şifre üretir ve `DATABASE_URL`'i doğrudan `.env`'e yazar (prod için `.env.prod`). **Şifre ekrana basılmaz.**
- Supabase SQL editöründe çalıştırılacak SQL'i üretir. Şifre bu SQL'de yalnızca **SCRAM-SHA-256 doğrulayıcısı** (tek yönlü hash) olarak bulunur; düz şifre Supabase'e, SQL geçmişine veya sunucu loglarına hiç gitmez.
- Rol: yalnızca `LOGIN`, bağlantı sınırı 10; süper kullanıcı / `BYPASSRLS` / `CREATEROLE` yok (yeni rol varsayılanı). `search_path`, `statement_timeout 5s`, `lock_timeout 3s`, `idle_in_transaction_session_timeout 10s`.
- Şema: `PUBLIC`, `anon`, `authenticated`, `service_role` erişemez; uygulama rolü yalnızca `USAGE` alır. Tabloları migration rolü kurar (DB-002).
- SQL tekrar çalıştırılabilir; `-force` ile şifre yenilenir.

```powershell
cd backend
# 1) Sertifikayı certs/supabase-ca.crt olarak kaydedin (certs/README.md)
# 2) Kurulum SQL'ini üretip panoya alın (.env'e adres yazılır)
go run ./cmd/dbsetup init -env test -ref <proje-ref> -host <session-pooler-host> | Set-Clipboard
# 3) Supabase > SQL Editor: yapıştırın, çalıştırın (snippet olarak kaydetmeyin)
# 4) Doğrulayın: verify-full TLS, rol, şema, zaman sınırları, yetki yalıtımı, bağlantılar
#    (-pool-size: Database Settings'teki Supavisor "Pool Size"; verilirse ihtiyaçla karşılaştırılır)
go run ./cmd/dbsetup check -pool-size <panel değeri>
```

`check` yalnızca durum yazdırır (✓ / ! / ✗); adres veya şifre yazdırmaz. Herhangi bir ✗ varsa çıkış kodu 1'dir.

### Migration'lar (`cmd/migrate`, DB-002)

Dosyalar ve kurallar: [../migrations/postgres/README.md](../migrations/postgres/README.md). Tabloları ayrı **migration rolü** (`bunker_<ortam>_migrator`) kurar ve sahibi olur; uygulama rolü yalnızca veri okur/yazar ([ADR-0012](../docs/decisions/0012-migration-role-and-rls-policy.md)).

```powershell
# Bir kez: migration rolünü kur (MIGRATION_DATABASE_URL .env'e yazılır)
go run ./cmd/dbsetup init -env test -role migrator -ref <proje-ref> -host <session-pooler-host> | Set-Clipboard
# Supabase > SQL Editor'de çalıştırın, sonra:
go run ./cmd/dbsetup check -role migrator

go run ./cmd/migrate new create_users     # sıradaki dosya
go run ./cmd/migrate lint                 # kurallar (DB gerekmez)
go run ./cmd/migrate up -dry-run          # ne uygulanacak
go run ./cmd/migrate up                   # uygula + yetki/RLS eşitle
go run ./cmd/migrate status
```

Sunucu migration çalıştırmaz; migration her zaman ayrı ve bilinçli bir adımdır.

### Entegrasyon testleri

Modül testleri gerçek şemayla çalışmak için `internal/testdb` kullanır: rastgele adlı şema + uygulama/migration rolleri kurar, depodaki **tüm migration'ları** uygular ve **uygulama rolüyle** (gerçek yetkiler ve RLS ile) bir `persistence.DB` verir; test bitince her şeyi siler.

```go
env := testdb.New(t)          // TEST_DATABASE_ADMIN_URL yoksa test atlanır
svc := auth.NewService(env.DB, nil, logger)
```

`TEST_DATABASE_ADMIN_URL` tanımlıysa gerçek PostgreSQL'e karşı çalışır: her test rastgele adlı rol ve şema kurar, sonunda siler. Tanımlı değilse atlanır. CI'da geçici bir `postgres:17` servisiyle çalışır. **Supabase adresini buraya vermeyin**; yerel veya tek kullanımlık bir PostgreSQL kullanın.

```bash
# Docker ile örnek
docker run --rm -d -p 55432:5432 -e POSTGRES_PASSWORD=postgres --name pg-test postgres:17-alpine
TEST_DATABASE_ADMIN_URL="postgres://postgres:postgres@localhost:55432/postgres?sslmode=disable" go test ./...
```

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
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...   # bilinen açık taraması
```

> `-race` Windows'ta cgo (gcc) gerektirir; yoksa `go test ./...` çalıştırın. CI (Linux) her zaman `-race` ile çalışır.
