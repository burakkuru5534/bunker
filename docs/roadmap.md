# Yol Haritası ve Görevler

> Çalışma prensibi: İncele → Planla → Tek küçük görevi uygula → Test et → Raporla → Kullanıcı onayı → Sonraki görev.
> Her görev ayrı onay gerektirir. Bir görev yalnızca kabul kriterleri sağlandığında **Done** olur.

**Durumlar:** Backlog · Ready · In Progress · Blocked · Awaiting User Approval · Done
**Öncelik:** P0 = MVP için zorunlu · P1 = MVP için önemli · P2 = MVP sonrası / ertelendi

Ayrıntılı kapsam, kabul kriterleri ve testler, görev **Ready** durumuna geçtiğinde ilgili bölüme eklenir.

---

## Faz 0 — Keşif ve ürün tanımı

| ID | Başlık | Öncelik | Bağımlılık | Durum |
|---|---|---|---|---|
| INIT-001 | Mevcut projeyi incele | P0 | — | Done |
| INIT-002 | Teknik ortamı doğrula | P0 | INIT-001 | Done |
| INIT-003 | Bağımlılıkları ve konfigürasyonları incele | P0 | INIT-001 | Done |
| GAME-001 | MVP kapsamını kesinleştir | P0 | INIT-001 | Done |
| GAME-002 | Oyuncu sayısı ve sığınak kapasitesi | P0 | GAME-001 | Done |
| GAME-003 | Oyun aşamaları ve kazanma koşulları | P0 | GAME-002 | Done |
| GAME-004 | Kart kategorileri ve gizlilik kuralları | P0 | GAME-003 | Done |
| GAME-005 | İlk felaket senaryoları | P0 | GAME-004 | Done |
| GAME-006 | Süreler, oylama ve beraberlik kuralları | P0 | GAME-003 | Done |
| DOC-001 | Doküman iskeleti ve kararların kaydı | P0 | GAME-006 | Done |
| ARCH-001 | Sistem mimarisi | P0 | DOC-001 | Done |
| ARCH-002 | WebSocket ve PostgreSQL veri akışı | P0 | ARCH-001 | Done |
| ARCH-003 | Güvenlik ve maliyet kararları | P0 | ARCH-001 | Done |
| ARCH-004 | API sözleşmeleri | P0 | ARCH-002 | Done |
| ARCH-005 | Cloud Run ve Supabase bağlantı modeli | P0 | ARCH-001 | Done |

### Tamamlanma notları

- **INIT-001:** Repo boş (yalnızca `.git`, commit yok). Branch `main`, remote `origin → github.com/burakkuru5534/bunker` (SSH). Supabase/GCP yapılandırması yok.
- **INIT-002:** Flutter 3.47.1, Dart 3.13.1, Go 1.27.0, Android SDK 36.0.0 doğrulandı. Android lisans adımı yeni Android CLI ile gereksiz; gerçek derleme MOB-001'de doğrulanacak. Testler bulut çalışma alanında aynı sürümlerle (Go 1.27.0, Flutter 3.47.1, yerel PostgreSQL) çalıştırılacak — kullanıcı onaylı.
- **INIT-003:** İncelenecek bağımlılık/konfigürasyon yok.
- **GAME-001…006:** Kararlar [product-requirements.md](product-requirements.md) ve [game-design.md](game-design.md) içinde.
- **DOC-001:** README, .gitignore, ürün gereksinimleri, oyun tasarımı ve yol haritası oluşturuldu.
- **ARCH-001:** [architecture.md](architecture.md) ve ADR-0001…0006 yazıldı. Kararlar: modül yapısı (4 sapma onaylı), satır kilidi + sürüm + action_id, ayrı test/prod ortamları, yerelde Docker Desktop ile PostgreSQL.
- **ARCH-002:** Komutlar REST, WS yalnızca sunucu → istemci ([ADR-0007](decisions/0007-rest-commands-ws-push.md)). Veri akışı [architecture.md §10](architecture.md), tablo taslağı [data-model.md](data-model.md).
- **ARCH-003:** [security.md](security.md), [cost-model.md](cost-model.md), [ADR-0008](decisions/0008-opaque-session-tokens.md). Maliyet kararları: Supabase Free + tek proje (test/prod şema izolasyonu, [ADR-0009](decisions/0009-single-supabase-schema-isolation.md)); Cloud Run `europe-west1`, Supabase `eu-central-1`.
- **ARCH-004:** [api.md](api.md) — REST uç noktaları, hata kodları, doğrulama kuralları, WS bağlantı/kapanış kodları, olay payload'ları. `/votes` → `/actions` içine birleştirildi; `PATCH/DELETE /v1/me`, `kick`, `rematch`, `settings`, `messages`, `reports`, `blocks` eklendi.
- **ARCH-005:** [architecture.md §11](architecture.md), [ADR-0010](decisions/0010-supavisor-session-mode.md) (Supavisor session mode, IPv4, pgxpool prod 5 / test 3), [ADR-0011](decisions/0011-cloud-run-configuration.md) (instance-based, min 0 / max 1, 1 vCPU / 512 MiB, timeout 3600 sn, europe-west1).

### DOC-001 — Doküman iskeleti

- **Kapsam:** `README.md`, `.gitignore`, `docs/product-requirements.md`, `docs/game-design.md`, `docs/roadmap.md`.
- **Kabul kriterleri:** GAME-001…006 kararlarının tümü dokümanlarda yer alır; secret içermez; `.gitignore` `.env` ve anahtar dosyalarını dışlar.
- **Testler:** Manuel inceleme (kod yok).

### ARCH-001 — Sistem mimarisi

- **Kapsam:** Bileşenler, backend modül sınırları, Flutter feature yapısı, dağıtım modeli; `docs/architecture.md` ve ilk ADR'ler (`docs/decisions/`).
- **Kabul kriterleri:** Kullanıcı mimariyi onaylar; alternatifler ve sınırlamalar yazılıdır.

---

## Faz 1 — UI/UX

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| UI-001 | Marka ve görsel dil ([ui/brand.md](ui/brand.md)) | P1 | Done |
| UI-002 | Ekran haritası ([ui/screen-map.md](ui/screen-map.md)) | P0 | Done |
| UI-003 | Tasarım sistemi ([ui/design-system.md](ui/design-system.md)) | P1 | Done |
| UI-004 | Ana ekran | P0 | Done |
| UI-005 | Oda oluşturma | P0 | Done |
| UI-006 | Odaya katılma | P0 | Done |
| UI-007 | Lobi | P0 | Done |
| UI-008 | Karakter kartları | P0 | Done |
| UI-009 | Tartışma ekranı | P0 | Done |
| UI-010 | Oylama ekranı | P0 | Done |
| UI-011 | Sonuç ekranı | P0 | Done |
| UI-012 | Hata ve bağlantı durumları | P0 | Done |

> UI-004…012 tek bir görsel mockup setinde hazırlandı (13 telefon ekranı, Claude Design kanvası "Sığınak Ekran Mockupları"). Kaynak: [ui/screen-map.md](ui/screen-map.md), [ui/design-system.md](ui/design-system.md).

## Faz 2 — Flutter temeli

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| MOB-001 | Flutter yapılandırması (Android derlemesi doğrulama dahil) | P0 | Done |
| MOB-002 | Ortam konfigürasyonları | P0 | Done |
| MOB-003 | Feature-first mimari ([apps/mobile/ARCHITECTURE.md](../apps/mobile/ARCHITECTURE.md)) | P0 | Done |
| MOB-004 | Riverpod | P0 | Done |
| MOB-005 | go_router | P0 | Done |
| MOB-006 | Dio | P0 | Done |
| MOB-007 | WebSocket istemcisi | P0 | Done |
| MOB-008a | Tasarım token'ları, yazı tipleri, tema, GameTheme | P1 | Done |
| MOB-008b | Ortak `core/ui` bileşenleri | P1 | Done |
| MOB-009 | Hata yönetimi ve l10n altyapısı | P0 | Done |
| MOB-010 | Güvenli oturum depolama | P0 | Done |
| MOB-011 | Widget testleri | P0 | Done |
| MOB-012 | CI doğrulaması (`.github/workflows/mobile.yml`) | P1 | Done |

- **MOB-001:** `apps/mobile` oluşturuldu (Flutter 3.47.1). Dart paketi `siginak`, **applicationId `com.bunker.app`** (kullanıcı kararı, kalıcı), uygulama adı "Sığınak", yalnızca dikey, sürüm 0.1.0+1. `dart format`, `flutter analyze`, `flutter test` bulut çalışma alanında geçti. Android derlemesi kullanıcı cihazında doğrulanacak.
- **MOB-002:** `--dart-define-from-file` ile `local`/`test`/`prod` yapılandırması (`apps/mobile/config/`), `AppConfig` doğrulaması (test/prod'da https/wss zorunlu), düz http yalnızca debug derlemede, release manifest'e `INTERNET` izni. `local.json` git dışı. 10 test geçti. `test.json`/`prod.json` adresleri Cloud Run kurulunca güncellenecek.
- **MOB-003:** `lib/app`, `lib/core`, `lib/features` yapısı; `SiginakApp` → `app/app.dart`, geçici ekran → `features/home/presentation/home_screen.dart`. Bağımlılık kuralları `test/architecture/import_rules_test.dart` ile otomatik denetleniyor. `analysis_options.yaml` sıkılaştırıldı (strict modlar; package import, print, unawaited futures = hata). 14 test geçti.
- **MOB-004:** `flutter_riverpod` 3.4.3 (en güncel, Dart ^3.12.0), kod üretici yok. `ProviderScope` + `appConfigProvider` (override zorunlu). `HomeScreen` → `ConsumerWidget`. Test yardımcısı `test/helpers/pump_app.dart`. Riverpod kuralları ARCHITECTURE.md §4. 16 test geçti.
- **MOB-005:** `go_router` ^18.1.0 (kilit: 18.1.2). 8 rota (ekran haritası §2) geçici ekranlarla; saf `appRedirect` (güncelleme → oturum → aktif oda), `refreshListenable` ile Riverpod bağlantısı, geçersiz oda kimliği/bilinmeyen adres → `NotFoundScreen`. Ekranlar gezinmeyi geri çağrıyla alır. Oturum şimdilik bellek içi iskelet ("Başla (geçici)"); gerçek oturum MOB-010/AUTH-002. 32 test geçti.
- **MOB-006:** `dio` ^5.11.1. `ApiClient` (taban adres, `X-App-Version`, Bearer token, zaman aşımları 10/10/15 sn), `ApiException` + hata eşleme (timeout, network, 401, 426 → `/update`, 429 + Retry-After, standart hata gövdesi, alan hataları). Token/gövde loglanmaz, otomatik yeniden deneme yok. `kAppVersion` pubspec ile testle eşleniyor. 46 test geçti.
- **MOB-007:** `dart:io` WebSocket (yeni çalışma zamanı bağımlılığı yok; test için `fake_async`). `RealtimeClient`: `/v1/ws?room_id=`, token yalnızca başlıkta, v1 zarf çözümleme, `state_version` (snapshot bekleme, kopya atma, boşlukta `sync`), üstel yeniden bağlanma 0,5→10 sn ±%20, 4001/4003/4008/4026'da kalıcı durma, `onAppResumed`. 70 test geçti.
- **MOB-008a:** MOB-008 ikiye bölündü. Tema `core/theme/` altında (bağımlılık kuralları nedeniyle `app/theme` yerine). Renk/boşluk/köşe/hareket token'ları, `buildAppTheme()` (koyu M3 + bileşen temaları), `GameTheme` (kategori renk+ikon+etiket, avatar döngüsü), Türkçe büyük/küçük harf yardımcıları. Oswald (500/600/700) ve Inter (400/500/600) google/fonts deposundan alındı, sabit ağırlık + Latin/Türkçe alt kümesi (≈430 KB), OFL lisansları `LicenseRegistry`'de. WCAG kontrast testleri. 115 test geçti.
- **MOB-008b:** `core/ui`: `AppButton` (primary/secondary/danger, yükleniyor, devre dışı nedeni), `AppTextField`, `InviteCodeInput` (+ formatter), `PlayerAvatar` (3 boyut, 3 durum, Türkçe baş harf), `StatusBadge` (7 tür), `ConnectionBanner` (4 durum), `showConfirmDialog`, `showAppSnackBar`, `LoadingView`/`ErrorView`/`EmptyState`. Yeni paket yok. 134 test geçti.
- **MOB-009:** l10n altyapısı (`flutter_localizations` SDK + `intl` ^0.20.2, `l10n.yaml`, `app_tr.arb`, üretilen kod repoda). Uygulama ve Material metinleri Türkçe. `core/ui` metinleri ARB'ye taşındı. `error_messages.dart`: api.md'deki 27 hata kodunun her biri, `ApiException` türleri, 429 bekleme süresi, WS durma nedenleri → Türkçe metin (doküman testle denetleniyor). `installGlobalErrorHandlers` (FlutterError + PlatformDispatcher; release'de sade hata kutusu). 147 test geçti.
- **MOB-010:** `flutter_secure_storage` ^11.2.0 (Keystore, AES-GCM, minSdk 24). `SecureStore` soyutlaması, `SessionStore` (`session.v1`, önbellek, süresi dolan/bozuk veriyi siler, token loglanmaz). Token okuyucu depoya bağlandı. `SessionController` açılışta depoyu okuyor (unknown → authenticated/unauthenticated), `signIn`/`signOut`; 401 → `signOut` (main override). Android yedekleme/aktarım kapalı (`allowBackup=false`, `data_extraction_rules.xml`). 159 test geçti.
- **MOB-011 + MOB-012:** Mevcut 159 test (birim + widget + mimari + kontrast) temel olarak kabul edildi; ekran testleri gerçek ekranlarla (ROOM-011, PLAY-*) yazılacak. GitHub Actions `mobile` işi: format, l10n güncelliği, `analyze --fatal-infos`, test; `main`'e push'ta Android debug APK. Flutter 3.47.1 sabit, eşzamanlı çalışmalar iptal, `contents: read`. actionlint temiz; adımlar yerelde simüle edildi. GitHub Actions yeşil (Android debug APK derlendi → MOB-001'in cihaz/derleme kriteri karşılandı).

### Faz 3 notları

- **BE-001:** `backend/` Go 1.27.0 modülü (`github.com/burakkuru5534/bunker/backend`), dış bağımlılık yok. `internal/config` (APP_ENV, PORT, SHUTDOWN_TIMEOUT; tüm hatalar birlikte), `cmd/server` (ReadHeaderTimeout 10s, IdleTimeout 120s, 16 KB başlık sınırı; SIGINT/SIGTERM → graceful shutdown, süre aşılırsa zorla kapatma). Testler: yapılandırma %100, sunucu %71 kapsama; açık isteğin kapanışta tamamlanması ve zaman aşımı testli; `-race -count=3` temiz. `.github/workflows/backend.yml` (gofmt, vet, `test -race`).
- **BE-002:** `internal/httpapi` (yalnızca standart kütüphane): Go 1.22+ `ServeMux` yöntem/yol kalıpları; middleware RequestID → Recover → RequireAppVersion (400/426, `MIN_APP_VERSION`) → LimitBody (4 KB) → Timeout (`REQUEST_TIMEOUT`, WS hariç). Eşleşmeyen yol/yöntem → JSON 404. Standart hata gövdesi yazıcısı (docs/api.md §2) bu görevde eklendi; alan hataları ve alan hatası → kod eşlemesi BE-004'te. httpapi kapsama %98,9.
- **BE-003:** `internal/observability`: `log/slog` tabanlı logger; local → metin, test/prod → JSON. Cloud Logging alanları `severity`/`message`/`httpRequest`/`logging.googleapis.com/trace` (W3C `traceparent` ya da `X-Cloud-Trace-Context`; yalnız `GCP_PROJECT_ID` doluysa). Hassas anahtar/`Bearer` değerleri gruplar dahil `[REDACTED]`. `AccessLog` middleware: istek başına tek satır, sorgu dizesi/başlık/gövde yok, 5xx → ERROR, handler logları `request_id` taşır; WebSocket için Hijack/Flush iletimi. Yeni yapılandırma: `LOG_LEVEL`, `GCP_PROJECT_ID`. Kapsama: observability %100, httpapi %98,6, config %100, server %76,4; `-race -count=2` temiz.
- **BE-004:** `internal/apperr`: 27 makine kodu, `*apperr.Error` (Message, Fields, Details, RetryAfter, loglanacak neden), `FieldErrors` toplayıcısı, kod bazlı `errors.Is`. `internal/httpapi`: kod → HTTP durumu tablosu, `WriteAppError` (INTERNAL/düz hata → genel mesaj + ERROR log; neden istemciye gitmez; 429 → `Retry-After`; istemci giderse yanıt yok), `Handle` adaptörü, katı `DecodeJSON` (Content-Type, bilinmeyen alan, tür, tek değer, 4 KB). Alan hata kodları docs/api.md §2'ye eklendi. `TestErrorCodesMatchDocs` tabloyu belgeyle karşılaştırır (mutasyonla doğrulandı); backend CI `docs/api.md` değişince de çalışır. Kapsama: apperr %100, httpapi %99.
- **BE-005:** `httpapi.Health`: `GET /health` (canlılık, bağımlılık yok) ve `GET /health/ready` (denetimler paralel, her biri ≤ 2 sn; panic ve bağlamı yok sayan denetim de güvenle zaman aşımına düşer; 503 yanıtında ayrıntı yok, yalnızca log). HEAD desteği, `no-store`, sürüm başlığı gerekmez. Başarılı sağlık istekleri erişim logunda DEBUG (log hacmi), başarısızlar WARNING. Veritabanı denetimi DB-001'de eklenecek. architecture.md Cloud Run tablosu: `/health` başlangıç/canlılık, `/health/ready` probe değil. httpapi kapsama %99,2.
- **DB-001a:** İlk dış bağımlılık `pgx/v5` v5.11.0 (v5.12.0 3 günlük olduğu için bilinçli olarak bir önceki). `internal/persistence`: pgxpool (ADR-0010 ayarları), TLS ilkesi (test/prod verify-full zorunlu; yerelde uzak sunucuya şifresiz bağlantı reddedilir), açılışta rol (süper/BYPASSRLS/yönetim rolü reddi) ve şema doğrulaması; başarısızsa servis başlamaz. `/health/ready` → `database` ping; kapanışta havuz kapanır. Yapılandırma: `DATABASE_URL` (`config.Secret`: fmt/slog/JSON'da maskeli; test/prod'da zorunlu), `DB_MAX_CONNS`, `DB_SCHEMA`; yerelde `.env` okunur (gerçek ortam öncelikli). govulncheck, Go 1.27.0'da sunucunun çağırdığı HTTP/2, TLS ve x509 kodunda 7 açık ve `x/text` için 2 açık buldu → Go 1.27.3 ve `x/text` v0.43.0'a yükseltildi; tarama temiz. CI: `postgres:17` servisiyle entegrasyon testleri + ayrı govulncheck işi. Ayrıca kapanış testlerindeki aralıklı takılma giderildi (istemcinin istek göndermediği keep-alive bağlantısı Shutdown'ı 5 sn bekletiyordu; testler keep-alive'sız istemci kullanır). Kapsama: persistence %97,3, config %99, server %81,7.
- **DB-001b (kod kısmı):** `cmd/dbsetup init|check` + `internal/dbsetup`. init: 32 karakter rastgele şifre → `.env`'e `DATABASE_URL` (ekrana basılmaz; dolu değer `-force` olmadan ezilmez), Supabase SQL'i stdout'a (şifre yalnızca SCRAM-SHA-256 doğrulayıcısı; Python ile bağımsız hesaplanan referansla test edildi). SQL Supabase'in süper kullanıcı olmayan postgres rolüyle çalışabilecek biçimde rol niteliklerini değiştirmez; tekrar çalıştırılabilir. check: katı modda (verify-full) bağlanır; zaman sınırları, şemada DDL yokluğu, ortam yalıtımı ve Data API rollerinin erişimsizliğini raporlar. Yerelde doğrulandı: süper kullanıcı olmayan CREATEROLE rolüyle kurulum (iki kez), SCRAM ile giriş, TLS'li PostgreSQL'de göreli `sslrootcert` ile verify-full başarılı; yanlış CA ve yanlış sunucu adı reddedildi. Supabase'te uygulandı (2026-11-03): PostgreSQL 17.11, bölge eu-central-1 (aws-1 session pooler); `dbsetup check` 12/12 ✓ — verify-full TLS, rol/şema, zaman sınırları, DDL yokluğu, ortam yalıtımı, Data API rollerinin erişimsizliği. Not: Supabase eu-central-1 (Frankfurt), Cloud Run europe-west1 (Belçika) — gecikme birkaç ms, kabul edilebilir.
- **DB-002:** `internal/migrate` + `cmd/migrate` (status, up, up -dry-run, new, lint); harici araç yok. Dosyalar `migrations/postgres/NNNN_ad.sql`, yalnızca ileri yönlü; SHA-256 sağlaması (CRLF/BOM'dan bağımsız), silinen/yeniden adlandırılan/sıra dışı dosya reddi, şemaya özgü advisory lock (session mode), dosya başına transaction (`-- migrate:no-transaction` tek ifade için), hata mesajında dosya + satır. Lint: şema/ortam adı, search_path/rol, GRANT/REVOKE, şema DDL'i, BEGIN/COMMIT, RLS kapatma yasak. Migration rolü `bunker_<ortam>_migrator` (`dbsetup -role migrator`; USAGE+CREATE, 1min/5s, limit 2). Her `up` sonunda yetki eşitleme: uygulama rolüne DML, RLS her tabloda açık + yalnızca uygulama rolüne `app_access` politikası; anon/Data API erişemez (ADR-0012 — eski 'RLS açık, politika yok' kuralı uygulamanın kendisini de kilitlerdi). Testler: gerçek PostgreSQL'de up/dry-run/idempotentlik, uygulama rolü yetki sınırları, anon erişimi yok, hatalı dosyada tam geri alma, kilit, elle bozulan yetkilerin düzelmesi, depo migration'larının boş şemaya uygulanması; yerelde verify-full TLS ve süper kullanıcı olmayan yöneticiyle uçtan uca prova. Kapsama: migrate %88,3, dbsetup %90,3.
- **DB-002 (Supabase):** `bunker_test_migrator` kuruldu, `dbsetup check -role migrator` 12/12 ✓; ilk `migrate up` sürüm tablosunu kurdu ve yetkileri eşitledi; uygulama rolü denetimi hâlâ 12/12 ✓ (2026-11-03).
- **DB-003a:** `0001_identity_and_rooms.sql`: users, sessions, rooms, room_members, blocks. Veritabanında garanti edilenler: davet kodu biçimi ve açık odalar arasında tekilliği, kapasite {1,2}, durum/kapanış tutarlılığı, açık odanın sahibi, kullanıcı başına tek aktif oda, oda içi büyük/küçük harf duyarsız tekil takma ad (üyelikte takma ad kopyası), izleyicinin hazır olamaması, token hash'inin 32 bayt ve tekil olması, kendini engelleme yok. Kısıt testleri uygulama rolüyle gerçek PostgreSQL'de; her tabloda RLS + app_access doğrulandı. data-model.md'ye 'Kesinleşen tablolar' bölümü eklendi.
- **DB-003b:** `0002_catalog.sql`: scenarios (metin anahtar, tam 3 sığınak özelliği, uzunluk sınırları), card_definitions (açık tamsayı id, 5 kategori, kategori içinde tekil metin, gizli balance −1/0/+1, aktif kart indeksi). İçerik ayrı (CARD görevleri). Yetki eşitleme genişletildi: `@app:readonly` yorumlu tablolarda uygulama rolüne yalnızca SELECT + `for select` politikası; yanlış komut/rollü politika yeniden kurulur (ADR-0012 eki). Testler: salt okunur davranışı, işaretin kaldırılınca yazmanın açılması, anon'a açılan politikanın düzeltilmesi, katalog kısıtları.
- **DB-003c:** `0003_game.sql`: games (aşama/tur/sıra/deadline/state_version tutarlılık kısıtları, odada tek aktif oyun, oda silinse de oyun kalır), game_participants (koltuk sırası, durum ve eleme turu tutarlılığı, kazanan hayatta), player_cards (kart–kategori bileşik yabancı anahtarı, oyunda kart tekilliği, açılış alanları birlikte), votes (oy değiştirilemez, kendine oy yok), round_results, game_actions (idempotency anahtarı, sistem eylemleri actor_id null). `rooms.current_game_id` yabancı anahtarı eklendi. Tüm kısıtlar uygulama rolüyle gerçek PostgreSQL'de test edildi.
- **DB-003d:** `0004_chat_and_moderation.sql`: chat_messages (maskelenmiş metin 1–200, client_msg_id tekrar koruması, son 50 mesaj ve 30 gün temizlik indeksleri), reports (neden/not/durum tutarlılığı, ≤64 KB mesaj anlık görüntüsü, aynı kişi için tek açık rapor). Oda silinince sohbet gider, rapor ve kanıtı kalır. data-model.md'deki 15 tablonun tamamı kesinleşti (DB-003 bitti).
- **DB-004:** [queries.md](queries.md): 16 sıcak yol + 5 bakım sorgusu; her biri gerçekçi hacimde veriyle (20 bin kullanıcı, 3 bin oda, 2 bin oyun, 100 bin mesaj) uygulama rolüyle EXPLAIN testinde (tam tarama yok, beklenen indeks). Bulunanlar: toplu silmelerde `in (select)` tam tarama yapıyordu → `id = any(array(...))` biçimi; oda temizliği için iki kısmi indeks eksikti; yeni yabancı anahtar indeks testi `reports.room_id` eksiğini buldu → `0005_cleanup_indexes.sql`. Bilinçli indekssiz yabancı anahtarlar gerekçeli izin listesinde (liste eskirse test kırılır).
- **DB-005:** `persistence.DB` (Tx / Read yardımcıları; bağlantı fonksiyon dışına sızmaz, hata/panic → rollback). Havuzdan bağlantı `DB_ACQUIRE_TIMEOUT` (3 sn) ile sınırlı: önceden isteğin 10 sn'lik süresi boyunca bekliyordu → artık `ErrPoolExhausted` ile hızlı hata + WARNING; 500 ms üstü bekleme uyarısı. Supabase sınırları kaynaktan doğrulandı (Nano: 60 doğrudan, 200 pooler; session mode'da rol başına Pool Size). Bağlantı bütçesi architecture.md §11'de; `dbsetup check` artık rolün açık bağlantılarını, rol sınırını ve `-pool-size` ile panel değerinin yeterliliğini raporluyor. Testler: commit/rollback/panic, bağlantı sızıntısı yok, havuz doluyken 200 ms'de hata (istek 10 sn olsa da), iptal edilen isteğin ayrı hatası, geç alma uyarısı.
- **AUTH-001:** `internal/auth`: takma ad kuralları (NFC, 3–16, yalnızca Latin alfabesi + Türkçe harfler, rakam, boşluk, _; en az bir harf; ardışık boşluk yok; ayrılmış adlar; küfür kancası) ve Türkçe büyük/küçük harf duyarsız karşılaştırma (`IŞIK` = `ışık`, `I` ≠ `i`); opak token (32 bayt, base64url 43 karakter, SHA-256), biçimi bozuk token'ın DB'ye gitmeden reddi, `Bearer` ayrıştırma. security.md §1'e akışlar (oluşturma, doğrulama, yenileme, silme, istemci IP'si) ve 'son kullanım saatte bir yazılır' kararı. Küfür listesi için SAFE-013 eklendi. Testler %97,8 + fuzz (geçerli her ad DB kısıtını sağlar ve kararlıdır).
- **AUTH-002a:** `auth.Service`: CreateGuest (takma ad doğrulama → tek transaction'da users + sessions; ham token yalnızca yanıtta), Authenticate (biçim → 60 sn önbellek → `sessions ⨝ users`; süresi dolmuş / iptal / silinmiş kullanıcı → nedeni ayrıştırılmayan UNAUTHORIZED; son kullanım en fazla saatte bir), Refresh (tek transaction'da eskiyi iptal + yeni; eşzamanlı yenilemelerden yalnız biri başarılı), Me (aktif oda), ForgetUser. Önbellek yalnızca geçerli oturumları, oturum süresini aşmadan, üst sınırla tutar. `internal/testdb`: tüm migration'ları uygulanmış yalıtılmış şema + uygulama rolüyle DB (sonraki modüllerin testleri için). Kapsama %91,8.
- **AUTH-002b:** `POST /v1/sessions/guest` (201; IP başına 5/dk, sınır gövde doğrulamasından önce, IPv6 /64), `POST /v1/sessions/refresh` (boş gövde veya {}), `GET /v1/me` (active_room_id null/uuid); token yanıtları `Cache-Control: no-store`; `auth.Require` middleware (401 + `WWW-Authenticate: Bearer`, bağlama oturum sahibi ve user_id'li logger). `internal/ratelimit` (anahtar başına token bucket, bellek üst sınırlı; %100), `httpapi.ClientIP` + `RateKey`, `TRUSTED_PROXY_HOPS`. Uçtan uca testler (tam middleware zinciri + gerçek DB): sözleşme alanları, token'ın loglanmaması, sahte XFF ile sınırın atlatılamaması.

## Faz 3 — Go backend ve PostgreSQL

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| BE-001 | Go servis iskeleti | P0 | Done |
| BE-002 | HTTP router ve middleware | P0 | Done |
| BE-003 | Yapılandırılmış loglama | P0 | Done |
| BE-004 | Standart API hata formatı | P0 | Done |
| BE-005 | Health endpoint | P0 | Done |
| DB-001a | PostgreSQL bağlantı kodu, güvenlik denetimleri, CI entegrasyon testi | P0 | Done |
| DB-001b | Supabase kurulumu (şema/rol, Data API, gerçek bağlantı ve TLS doğrulaması) | P0 | Done |
| DB-002 | Migration altyapısı | P0 | Done |
| DB-003a | Şema: kimlik ve odalar (users, sessions, rooms, room_members, blocks) | P0 | Done |
| DB-003b | Şema: katalog (scenarios, card_definitions); içerik CARD-002…007'de | P0 | Done |
| DB-003c | Şema: oyun (games, game_participants, player_cards, votes, round_results, game_actions) | P0 | Done |
| DB-003d | Şema: sohbet ve moderasyon (chat_messages, reports) | P0 | Done |
| DB-004 | Sorgu ve indeks tasarımı | P0 | Done |
| DB-005 | Bağlantı havuzu ve limitleri | P0 | Done |
| AUTH-001 | Misafir oturumu tasarımı | P0 | Done |
| AUTH-002a | Oturum servisi: oluşturma, doğrulama, yenileme, önbellek | P0 | Done |
| AUTH-002b | Oturum uç noktaları (guest, refresh, me) ve IP sınırı | P0 | Awaiting User Approval |
| AUTH-003 | REST ve WebSocket yetkilendirmesi | P0 | Backlog |

## Faz 4 — Gerçek zamanlı altyapı

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| RT-001 | WebSocket sunucusu | P0 | Backlog |
| RT-002 | Bağlantı kimlik doğrulaması | P0 | Backlog |
| RT-003 | Oda üyeliği kontrolü | P0 | Backlog |
| RT-004 | Oda bazlı bağlantı yönetimi | P0 | Backlog |
| RT-005 | Mesaj şemaları ve doğrulama | P0 | Backlog |
| RT-006 | Oda içi olay yayını | P0 | Backlog |
| RT-007 | Bağlantı kopması ve yeniden bağlanma | P0 | Backlog |
| RT-008 | Bağlantı temizliği ve kaynak yönetimi | P0 | Backlog |
| RT-009 | Rate limiting | P0 | Backlog |
| RT-010 | Eşzamanlılık ve bağlantı testleri | P0 | Backlog |
| RT-011 | Cloud Run WebSocket kısıtlarının doğrulanması | P0 | Backlog |
| RT-012 | Gerçek zamanlı iletişim maliyet değerlendirmesi | P1 | Backlog |

## Faz 5 — Oda ve lobi

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| ROOM-001 | Oda modeli | P0 | Backlog |
| ROOM-002 | Davet kodu üretimi | P0 | Backlog |
| ROOM-003 | Oda oluşturma API'si | P0 | Backlog |
| ROOM-004 | Odaya katılma API'si | P0 | Backlog |
| ROOM-005 | Odadan ayrılma | P0 | Backlog |
| ROOM-006 | Oda üyeliği ve yetkilendirme | P0 | Backlog |
| ROOM-007 | Hazır ol mekanizması | P0 | Backlog |
| ROOM-008 | Oda ayarlarının senkronizasyonu (kapasite 1/2) | P0 | Backlog |
| ROOM-009 | Oda sahibi ayrıldığında yetki devri | P0 | Backlog |
| ROOM-010 | Boş oda temizliği | P1 | Backlog |
| ROOM-011 | Flutter oda akışları | P0 | Backlog |
| ROOM-012 | Çoklu oyuncu testleri | P0 | Backlog |

## Faz 6 — Kart ve senaryo sistemi

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| CARD-001 | Kart veri modeli (denge etiketi dahil) | P0 | Backlog |
| CARD-002 | Meslek kartları | P0 | Backlog |
| CARD-003 | Sağlık durumu kartları | P0 | Backlog |
| CARD-004 | Hobi ve beceri kartları | P0 | Backlog |
| CARD-005 | Eşya kartları | P0 | Backlog |
| CARD-006 | Gizli sır kartları | P0 | Backlog |
| CARD-007 | Felaket senaryosu modeli ve 5 senaryo metni | P0 | Backlog |
| CARD-008 | Rastgele kart dağıtımı | P0 | Backlog |
| CARD-009 | Oyuncuya özel gizli veri | P0 | Backlog |
| CARD-010 | Kart açıklama kuralları | P0 | Backlog |
| CARD-011 | İçerik doğrulama | P0 | Backlog |
| CARD-012 | Kart denge testleri | P1 | Backlog |

## Faz 7 — Oyun motoru

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| ENG-001 | Oyun durum makinesi | P0 | Backlog |
| ENG-002 | Durum geçişi doğrulama | P0 | Backlog |
| ENG-003 | Maç başlatma | P0 | Backlog |
| ENG-004 | Kart dağıtımı | P0 | Backlog |
| ENG-005 | Kart açıklama sırası | P0 | Backlog |
| ENG-006 | Tartışma aşaması | P0 | Backlog |
| ENG-007 | Tur zamanlayıcıları (kalıcı deadline) | P0 | Backlog |
| ENG-008 | Oylama | P0 | Backlog |
| ENG-009 | Oylama sonucu ve eşitlik bozma | P0 | Backlog |
| ENG-010 | Eleme | P0 | Backlog |
| ENG-011 | Kazanma koşulları | P0 | Backlog |
| ENG-012 | Maç sonu | P0 | Backlog |
| ENG-013 | Yeniden bağlanma ve oyun kurtarma | P0 | Backlog |
| ENG-014 | İdempotent eylemler | P0 | Backlog |
| ENG-015 | Gizli bilgi izolasyonu | P0 | Backlog |
| ENG-016 | Oyun motoru birim testleri | P0 | Backlog |

## Faz 8 — Oyun UI entegrasyonu

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| PLAY-001 | Lobi ve başlatma | P0 | Backlog |
| PLAY-002 | Kişisel karakter kartları | P0 | Backlog |
| PLAY-003 | Açıklanmış kartlar | P0 | Backlog |
| PLAY-004 | Tartışma ekranı | P0 | Backlog |
| PLAY-005 | Mesaj gönderimi | P0 | Backlog |
| PLAY-006 | Tur sayacı | P0 | Backlog |
| PLAY-007 | Oylama arayüzü | P0 | Backlog |
| PLAY-008 | Eleme sonucu | P0 | Backlog |
| PLAY-009 | Yeniden bağlantı deneyimi | P0 | Backlog |
| PLAY-010 | Maç sonu ekranı | P0 | Backlog |
| PLAY-011 | Hata ve yüklenme durumları | P0 | Backlog |
| PLAY-012 | Küçük ekran ve erişilebilirlik testleri | P1 | Backlog |

## Faz 9 — Güvenlik ve moderasyon

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| SAFE-001 | İstek ve girdi doğrulama | P0 | Backlog |
| SAFE-002 | Rate limiting | P0 | Backlog |
| SAFE-003 | Oda izolasyonu | P0 | Backlog |
| SAFE-004 | Gizli kart erişim testleri | P0 | Backlog |
| SAFE-005 | Oyuncu raporlama | P0 | Backlog |
| SAFE-006 | Oyuncu engelleme | P0 | Backlog |
| SAFE-007 | Oda sahibi moderasyonu (lobiden atma) | P0 | Backlog |
| SAFE-008 | Gizlilik politikası ve kullanım koşulları | P0 | Backlog |
| SAFE-009 | Veri saklama ve silme (sohbet 30 gün) | P0 | Backlog |
| SAFE-010 | Secret yönetimi | P0 | Backlog |
| SAFE-011 | Güvenlik logları | P1 | Backlog |
| SAFE-012 | Mağaza güvenlik ve yaş derecelendirmesi | P0 | Backlog |
| SAFE-013 | Küfür filtresi: Türkçe kelime listesi, takma ad reddi ve sohbet maskeleme (`auth.ProfanityChecker`) | P0 | Backlog |

## Faz 10 — Test, beta ve yayın

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| QA-001 | Oyun motoru testleri | P0 | Backlog |
| QA-002 | API entegrasyon testleri | P0 | Backlog |
| QA-003 | Çoklu cihaz testleri | P0 | Backlog |
| QA-004 | Ağ kesintisi testleri | P0 | Backlog |
| QA-005 | Arka plan ve yeniden bağlanma testleri | P0 | Backlog |
| QA-006 | Backend yeniden başlatma testleri | P0 | Backlog |
| QA-007 | Yük ve maliyet ölçümleri | P1 | Backlog |
| QA-008 | Güvenlik testleri | P0 | Backlog |
| REL-001 | Uygulama ikonları ve mağaza görselleri | P0 | Backlog |
| REL-002 | Gizlilik politikası ve destek sayfası | P0 | Backlog |
| REL-003 | Production konfigürasyonu | P0 | Backlog |
| REL-004 | Crash ve performans takibi | P1 | Backlog |
| REL-005 | Google Play test dağıtımı (TestFlight ertelendi) | P0 | Backlog |
| REL-006 | Kapalı beta | P0 | Backlog |
| REL-007 | Beta sorunlarının giderilmesi | P0 | Backlog |
| REL-008 | Yayın kontrol listesi | P0 | Backlog |

## Faz 11 — MVP sonrası (yalnızca beta değerlendirmesinden sonra)

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| POST-001 | iOS derlemesi ve TestFlight (bulut CI ile) | P2 | Backlog |
| POST-002 | Herkese açık oda listesi | P2 | Backlog |
| POST-003 | Arkadaş listesi | P2 | Backlog |
| POST-004 | Oyuncu profili ve istatistikleri | P2 | Backlog |
| POST-005 | Yeni senaryo paketleri ve kart kategorileri | P2 | Backlog |
| POST-006 | Sesli sohbet | P2 | Backlog |
| POST-007 | Paylaşılabilir maç sonuçları | P2 | Backlog |
| POST-008 | Kozmetik / premium özellikler | P2 | Backlog |
| POST-009 | Yayıncı modu, gelişmiş eşleştirme | P2 | Backlog |
