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
- **AUTH-003:** `auth.Require` artık kullanıcı başına genel REST sınırını uygular (10 sn'de 30 → 429 + Retry-After; kimlik doğrulamasından sonra, kullanıcı başına); token yalnızca başlıktan (sorgu dizesindeki token yok sayılır, testli). WebSocket kararı: yetki/üyelik/sürüm hataları HTTP durumu değil, yükseltmeden sonra kapanış kodu (4001/4003/4026) — mobil istemcinin beklediği davranış; `/v1/ws` sürüm middleware'inden muaf, `httpapi.CheckAppVersion`, `VersionCloseCode`, `CloseCodeFor` ve kapanış kodu sabitleri WS işleyicisi (RT) için hazır. Kapsam notu: oda/oyun üyeliği denetimi oda modülüyle (ROOM, SAFE-003), kullanıcı başına 2 WS bağlantısı hub ile (RT-009) gelir.
- **ROOM-001/002:** `internal/room` saf model (DB/HTTP bilmez; ADR-0002): Join (kapalı/atılmış/zaten üye/oyun sürüyor/eski katılımcı izleyici/10 dolu/Türkçe duyarsız takma ad çakışması), Leave (lobide sil, oyunda left_at; sahiplik en eski aktif oyuncuya; son üye → kapat), SetReady (yalnız lobide, izleyici olamaz, aynı değer değişiklik değil), ChangeCapacity / Kick / CanStart (oda sahibi dahil herkes hazır, 4–10; details.reason) / Rematch. Davet kodu: kriptografik, modulo yanlılığı olmadan 31 karakterlik alfabe; giriş büyük/küçük harf, boşluk ve tire duyarsız. 20 bin kodla dağılım testi. api.md katılma/ayrılma/başlatma/atma kuralları netleşti. Kapsama %97,7.
- **ROOM-003:** `POST /v1/rooms` (kapasite ops., varsayılan 2; bilinmeyen alan reddi) → `201 RoomView` (+ `version`). Oda + sahibin oyuncu üyeliği tek transaction; `ALREADY_IN_ROOM` veritabanının 'tek aktif oda' kısıtından (yarışsız; 5 eşzamanlı istekte 1 oda) ve `details.room_id` ile; davet kodu çakışmasında en fazla 5 deneme. `room.Service`, `loadForUpdate` (satır kilidiyle oda + üyeler, sonraki işlemler için), `View`/`Presence` (connected RT gelene kadar false), `persistence.UniqueViolation`. Kapsama %95,8.
- **ROOM-004:** `POST /v1/rooms/join` (kullanıcı başına 10/dk, sınır kod denetiminden önce; kod büyük/küçük harf, boşluk, tire duyarsız) ve `GET /v1/rooms/{id}` (yalnız aktif üye; üye olmayan / olmayan / bozuk kimlik aynı 404). Katılma: açık oda kodla bulunup satır kilidiyle okunur → saf `room.Join` kararı → üyelik upsert (ayrılmış üye yeniden açılır) → sürüm +1 → commit sonrası `room.member_joined` (Publisher; RT gelene kadar yok sayılır). Yarış koruması: 14 eşzamanlı katılmada tam 9 kişi girer (10 oyuncu), sürüm ve olay sayısı tutarlı; takma ad / tek aktif oda yarışları DB kısıtından doğru hataya çevrilir. Oyun sürerken eski katılımcı izleyici olarak döner, yabancı GAME_IN_PROGRESS. Kapsama %94,2.
- **ROOM-005 + ROOM-009:** `POST /v1/rooms/{id}/leave` → 204, idempotent (üye olmayan / zaten ayrılmış / olmayan oda / bozuk kimlik: değişiklik yok, varlık sızmaz). Lobide satır silinir, oyunda `left_at` (eski katılımcı izleyici dönebilir; oyundaki 'ayrıldı' durumu GAME modülünde). Sahip ayrılırsa sahiplik en eski aktif oyuncuya; `room.member_left` + `room.host_changed` aynı sürümde, commit sonrası. Son üye → oda kapanır (kod yeniden kullanılabilir, olay yok). 5 üyenin eşzamanlı ayrılmasında oda tek kez, tutarlı kapanır. Kapsama %93,1.
- **ROOM-007 / ROOM-008 / SAFE-007:** Ortak `Service.mutate` yolu (kilitle → aktif üye değilse NOT_FOUND → saf karar → yaz → değişiklik varsa sürüm +1 → commit sonrası olay). `POST …/ready` (204; aynı değer değişiklik/olay yok; izleyici FORBIDDEN; lobi dışı INVALID_PHASE), `PATCH …/settings` (yalnız sahip; 200 RoomView; settings_changed), `POST …/kick` (yalnız sahip, lobide; atılan geri dönemez, odayı göremez, başka oda kurabilir; member_kicked olayı — atılanın WS'i RT'de 4003 ile kapanacak). Zorunlu alanlar eksikse alan hatası. Kapsama %92,6.
- **ROOM-006 / SAFE-003 (oda):** `room.LoadMembership` / `Service.Membership` (rol, sahip mi, oda durumu, aktif oyun; üye değil / kapalı / bozuk kimlik → NOT_FOUND) — oyun, sohbet ve WS yetkisi bunu kullanacak. Oda rotaları tek tabloda (`roomScopedRoutes`); yalıtım matrisi her rotada A odası üyelerinin B'ye isteğinin, var olmayan odanınkiyle birebir aynı yanıtı aldığını ve B'nin veritabanı halinin değişmediğini doğrular; örnek gövdesi verilmeyen yeni rota testi kırar. Mutasyon denemesi: üyelik denetimi kaldırılınca test varlık sızıntısını yakaladı.
- **ENG-001/002:** `internal/game/engine` saf motor (DB/HTTP/saat bilmez; ADR-0002): `NewGame` (4–10 oyuncu, kapasite 1/2, koltuk 0..n−1, her kategoriden bir kart, kart tekilliği), `Begin` (DEALING → SCENARIO_INTRO), `Timeout(durum, beklenen sürüm, şimdi, rng)` (ADR-0005; eski sürüm `ErrStale`, erken `ErrNotDue`, bitmiş oyun `ErrNoDeadline`; girdi durumu değişmez, her geçişte sürüm +1). Tur 1–5 REVEAL: sıra (tur−1). koltuktan halka, elenenler atlanır; süre dolunca tur 1 meslek, sonra açılmamış en düşük kategori otomatik açılır. Tur 6+ doğrudan tartışma (≤6 hayatta 90 sn, ≥7 120 sn). Oy sayımı: yalnız hayattakilerin hayattaki başka oyuncuya oyu; tek en yüksek → eleme; eşitlikte izleyici varsa TIE_BREAK (oy veren izleyicilerin hepsi aynı adayı seçtiyse o, yoksa kura), izleyici yoksa koltuk sıralı adaylardan kura. Eleme: tüm kartlar açılır, tur sonucu (sayım, kura adayları) yazılır; hayatta ≤ kapasite → FINISHED. Oyuncu eylemleri (kart açma, hazırım, oy, ayrılma) sonraki ENG adımlarında. Testler: 4–10 × kapasite 1/2 tam oyun (rastgele oylarla 14 senaryo), eşitlik/izleyici/kura, geçersiz oy, sıra kayması. Kapsama %94,3; mutasyon denemesi: oybirliği denetimi ve kendine oy filtresi kaldırılınca testler düştü.
- **ENG-004 (+ ENG-003 koltuk/senaryo kısmı):** `engine.DealGame(kullanıcılar, katalog, rng)` saf: koltuklar Fisher–Yates ile rastgele, senaryo aktiflerden rastgele, her kategoriden oyuncu başına bir kart (oyunda tekil). Denge: toplam olumsuz 2n’i aşarsa seçili olumsuz kart aynı kategoriden kullanılmamış olumsuz olmayanla takas edilir; olumsuzlar her kategoride o an en az olumsuzu olan oyunculara verilir (yük farkı ≤1 → oyuncu başına ≤2). Katalog yetersiz/tekrarlı/senaryosuz/kural sağlanamaz → `ErrCatalog`. Katalog satır sırası sonucu etkilemez. `CryptoRand` (crypto/rand, modulo yanlılığı yok). Testler: 4–10 oyuncu × 4 olumsuz yoğunluğu × 200 dağıtım (mümkün olmayan durumda hata beklenir), tam sınırda katalog, 20 bin dağıtımla koltuk/senaryo/kart dağılımı, CryptoRand aralık ve dağılım. Kapsama %95,4; mutasyon: dengeli atama, takas, karıştırma ve yanlı (Sattolo) karıştırma kaldırılınca testler düştü.
- **ENG-003:** `POST /v1/rooms/{id}/start` → `201 {game_id}`. `room.Service.Start` ortak `mutate` yolunda (oda satır kilidi → `CanStart` → `GameStarter.StartGame` aynı transaction’da → oda `in_game` + `current_game_id`, oda sürümü +1). `internal/game.Starter`: aktif katalogu okur (yalnız kimlik + denge), `DealGame` + `NewGame` + `Begin` (DEALING → SCENARIO_INTRO, 15 sn, state_version 1), `games` + `game_participants` + `player_cards` toplu (unnest) yazılır. Bağımlılık yönü game → room (`room.GameStarter` arayüzü). Olaylar: `game.started` (senaryo, katılımcılar+takma ad+koltuk, tur 1 açıklama sırası), oyuncu başına özel `game.your_cards` (`room.Event.To`), `game.phase_changed` (aşama adları istemcide BÜYÜK harf, `server_time`). Oyun olayları oyunun sürümünü taşır (`mutate` yalnızca boş sürümü oda sürümüyle doldurur). Testler (gerçek PostgreSQL): tam başlatma (oda/oyun/katılımcı/kart satırları, kart tekilliği, ≤2 olumsuz, olay içerikleri ve DB ile tutarlılık, odaya giden olaylarda açılmamış kart metni / denge / card_id yok), koşullar (az oyuncu, hazır değil, sahip değil 403, üye değil 404, oturumsuz 401, kapasite 1), 6 eşzamanlı başlatmada tek oyun ve tek `game.started`, boş katalogda 500 + tam geri alma. Yalıtım matrisi yeni rotayı kapsıyor. Mutasyon: koşul denetimi, özel kart yönlendirmesi, oyun sürümü ve DB’ye yazılan sürüm bozulunca testler düştü.
- **ENG-005 (+ ENG-014 temeli):** Motor: `Engine.Reveal` (aşama REVEAL, hayatta, sıra onda, tur 1 yalnız meslek, açık kart reddi; sonra sıradaki oyuncu veya tartışma). Eleme artık oyları tur sonuna kadar tutar (yüklenen durumla aynı). `game.loadState/saveState`: tam durum yükleme (FOR UPDATE ile) ve fark yazımı (games sürüm korumalı `where state_version = eski`, katılımcı durumu/kazanan, açılan kartlar `revealed_at` ile, tur sonuçları); eşitlik adayları saklanmaz, turun oylarından türetilir; oylar farktan değil eylemden açıkça yazılır (`insertVote`). `POST /v1/games/{id}/actions` (`game.Service.Act`): katı gövde, kilit → katılımcı değilse 404 → `(oyun, oyuncu, action_id)` daha önce işlendiyse kayıtlı sürüm + `duplicate:true` → motor → fark + `game_actions` → commit sonrası `game.card_revealed` (metinle, kart kimliksiz) + `game.phase_changed`. Testler: 4 tam oyunda (4/1, 6/2, 7/1, 10/2; rastgele oy, eşitlik bozma, kura) her geçişten sonra DB’den yüklenen durum motorunkiyle birebir; eski sürüm üzerine yazım reddi; kart açma kuralları uçtan uca; tekrar/duplicate; aynı action_id 6 eşzamanlı → 1 işlem + 5 duplicate; farklı id 6 eşzamanlı → 1 başarı; doğrulama hataları; erişim (yabancı/olmayan/bozuk aynı 404); odaya giden olaylarda açılmamış kart metni yok. Mutasyon: katılımcı denetimi, idempotency, satır kilidi, card_id temizliği, eleme kartı yazımı ve eşitlik türetmesi kaldırılınca testler düştü. Kapsama: game %85,4, engine %96,0.
- **ENG-007:** `game.Timers` (ADR-0005): bellekte oyun başına tek zamanlayıcı; olay akışından beslenir (`game.phase_changed` → planla, `game.finished` → bırak; `room.Event.GameID` iç alan), eski sürümlü olay yok sayılır. Süre dolunca `Service.Timeout(oyun, beklenen sürüm)`: kilit → sürüm farklı veya süre dolmamışsa yazmaz → `applyDue` (motor `Timeout`, fark, `phase_timeout` kaydı actor_id boş, olaylar) → commit sonrası yayın. Hata → artan bekleme (1 sn…30 sn); oyun yok → bırak. Açılışta `Start` bitmemiş oyunların bitişlerini yükler (geçmişler hemen; DB hazır değilse yeniden dener); kapanışta `Stop` HTTP’den sonra, bekleyen çağrıları bekler. Yeni aşama "şimdi"den başlar (uyuyan instance oyunculardan süre çalmaz). Ek güvence: eylem süresi geçmiş aşamaya gelirse önce süre dolumu ayrı transaction’da commit edilir, eylem güncel duruma uygulanır (reddedilse de süre dolumu kalır). Sunucu: tüm olaylar zamanlayıcı yayıncısından geçer (RT gelince gerçek yayıncı bağlanır). Testler: birim (sahte kaynak; zamanında tetikleme, sürüm sırası, hata→artan bekleme→yeniden planlama, erken/eski/olmayan oyun, olay gözlemi, kurtarma yeniden denemesi, Stop) ve gerçek PostgreSQL (zamanlayıcıyla 6 kişilik oyun müdahalesiz bitiyor, her sürüm bir süre dolumu kaydı; açılış kurtarması; eski sürüm/erken/bitmiş; 8 eşzamanlı süre dolumundan 1 işlem; süresi geçmiş aşamada eylem). 5 tekrar kararlı. Mutasyon: 7/7 yakalandı. Kapsama: game %86,8. Not: oyun bitince odanın `finished` olması ENG-012’de (kilit sırası oda → oyun olarak tasarlanacak).
- **ENG-006 + ENG-008:** Motor: `DiscussionReady` (DISCUSSION, hayatta; ikinci kez `ErrNoChange`; `game.discussion_ready {count,total}`; hayattakilerin hepsi hazır → VOTING) ve `Vote` (VOTING, hayatta, tur başına tek oy `ALREADY_VOTED`, `SELF_VOTE`, hedef hayatta değilse `INVALID_TARGET`; `game.vote_progress {cast,total}` hedef içermez; herkes oy verince sayım hemen — eleme, eşitlik bozma veya kura). Servis: eylem başına `persist` (oy `votes`’a açıkça yazılır), `ErrNoChange` → başarı, aynı sürüm, olay yok, kayıt (tekrar duplicate döner); oy hedefi `game_actions.payload`’a yazılmaz. Düzeltme: `game.player_eliminated.cards` artık kart kimliği yerine metin taşıyor (zamanlayıcı yolu dahil). Testler: motor birim + gerçek PostgreSQL (hazırım akışı ve tekrar, oy kuralları ve gizlilik, erken sayım ve eleme olayı, zamanlayıcı yolundaki eleme olayı, 8 eşzamanlı oyda 1–8 ilerleme ve tek tur sonucu, 5 kişilik oyunun yalnızca oyuncu eylemleriyle bitmesi). 3 tekrar kararlı; mutasyon 7/7. Kapsama: game %87,4, engine %96,5.
- **ENG-009:** Motor `TiebreakVote` (TIE_BREAK; izleyici değilse `NOT_SPECTATOR`; aday dışı/hayatta olmayan hedef `INVALID_TARGET`; tek oy `ALREADY_VOTED`; `game.tiebreak_progress {cast,total}` — total izleyici sayısı; tüm izleyiciler oy verince `resolveTieBreak` hemen: oybirliği → spectators, dağınık → kura). Servis: `tiebreak_vote` normal oyla ortak ayrıştırma, `votes.kind = tiebreak`, hedef eylem kaydında yok. Böylece oylama zinciri (oy → sayım → eşitlik → izleyici/kura → eleme) eylemlerle ve süre dolumuyla uçtan uca tamam. Testler: motor birim (kurallar, tek izleyici kararı, dağınık → kura) + gerçek PostgreSQL (eşitliğe HTTP oylarıyla düşme, tiebreak_started adayları, izleyici/aday/aşama hataları, tek izleyicinin kararıyla anında eleme ve DB yöntemi, eleme olayında metinler, iki izleyici aynı anda farklı oy → tek sonuç, kura). 5 tekrar kararlı; mutasyon 4/4. Kapsama: game %87,5, engine %96,7.
- **ENG-012 (+ rematch):** Kilit sırası her yerde oda → oyun (`loadState(lock)` önce oyunun odasını kilitler); deadlock önlenir, testle doğrulandı (oda kilitliyken eylem oyunu kilitlemeden bekler). Oyun bitince `saveState` aynı transaction’da odayı `finished` yapar (sürüm +1). `game.Result` / `GET /v1/games/{id}/result` (katılımcı; sürerken INVALID_PHASE; tüm kartlar metinle, kazananlar, tur sonuçları ve sayımlar; kimlik/denge/oy hedefi yok); `game.finished` payload’ı birebir aynı yapı. `POST /v1/rooms/{id}/rematch` (sahip, `finished`): oda lobiye, üyeler oyuncu ve hazır değil, `current_game_id` boş, `room.returned_to_lobby {room}`; yalıtım matrisine eklendi. Testler (gerçek PostgreSQL): zamanlayıcıyla biten oyunda oda durumu/sürümü, sonuç içeriği ve erişim (yabancı/olmayan/bozuk 404), olay = sonuç, rematch kuralları ve aynı odada yeni oyun, eski sonucun okunması, kilit sırası. Mutasyon 6/6. Kapsama: game %86,4, room %87,2.
- **ENG-010/011 (oyunda ayrılma, kazanma koşulları):** Motor `Leave`: hayattaki oyuncu `left` + elenmiş sayılır, kartları açılır (tanıtımda ayrılırsa kart turu 1), `game.player_eliminated {method: left}`; ardından hayatta ≤ kapasite → bitiş (0 → terk edildi; ayrılmalar tek tek işlendiği için pratikte ulaşılamaz, savunma amaçlı), REVEAL’da sırası gelen ayrıldıysa sıra geçer, kalanların hepsi hazırsa oylama, hepsi oy verdiyse sayım, eşitlikte tek aday kaldıysa sonuç. Hayatta değilse/bitmişse `ErrNoChange`. Bağlantı: `room.GameStarter.LeaveGame` — oda ayrılması oyun sürerken aynı transaction’da oyunu günceller (oda kilitli → oyun; kilit sırası korunur); `game.RoomHooks{Starter, Service}`; `leave` eylem kaydı. Olay sırası: oda olayları → oyun olayları. Testler: motor birim (tanıtım/sıra/tartışma/oylama/eşitlik/bitiş/terk) + gerçek PostgreSQL (sırası gelenin ayrılması ve olay sırası, izleyici olarak dönüş ve eylem reddi, ayrılmanın oyunu bitirmesi ve odanın finished olması, ayrılanın sonucu görmesi, oylamanın ayrılmayla tamamlanması, oda sahibinin oyunda ayrılması, kart açma + 3 ayrılma eşzamanlı ×5). Mutasyon 7/7. Kesintili test (rastgele koltukta oda sahibinin sırası gelince ek host_changed olayı) bulundu ve düzeltildi; 15 tekrar kararlı. Kapsama: game %86,1, engine %97,4, room %86,7. Bağlantı kopması (60 sn bekleme → `disconnect_timeout`) RT ile birlikte.
- **RT-001…004 + RT-006 (+ RT-009 kısmen):** `internal/realtime`, kütüphane `github.com/coder/websocket` v1.8.15 (govulncheck temiz). `GET /v1/ws?room_id=`: önce yükseltme, sonra sürüm (4026), token yalnız başlıktan (4001), kullanıcı başına bağlantı denemesi 10/dk (4008), üyelik (4003; hub’a kayıttan önce — üye olmayan hiç olay almaz). Kayıt → anlık görüntü yazılır → kuyruk; ilk mesaj her zaman `session.snapshot` (me, room; game ENG-013’te). `Hub`: oda → bağlantılar, kullanıcı başına en fazla 2 (fazlasında en eskisi 4008), `room.Publisher` (zarf v1, `To` → yalnız o kullanıcı; atılana önce olay sonra 4003, ayrılana 1000) ve `room.Presence` (oda görünümündeki `connected`). Bağlantı başına 64’lük kuyruk + yazıcı goroutine (dolunca 4009, yayın bloklanmaz), 25 sn ping, yanıtsız istemci el sıkışmasız kesilir; okuma bağlamı kapatmadan bağımsız (aksi halde kütüphane kodsuz koparıyordu). İstemciden yalnız `{"v":1,"type":"sync"}` (10 sn’de 5). Sunucu kapanışında 1012. Sunucuda olay zinciri zamanlayıcı → hub. Testler (gerçek sunucu + gerçek WS istemcisi): 7 ret senaryosu ve kodları, düz HTTP 426, anlık görüntü/olay/presence, özel kartların yalnız sahibine gitmesi (4 bağlantı), atılma/ayrılma kapanışları ve olay sırası, 2 bağlantı sınırı, sync ve geçersiz mesajlar, ping zaman aşımı, kapanış 1012; yavaş istemci birim testi. 5–8 tekrar kararlı (yarış dedektörüyle). Mutasyon 8/8. Kapsama %91,1. Test kodundaki bir hata (zaman aşımsız kanal okuma) bir çalıştırmayı süresiz bekletti; düzeltildi.
- **CI düzeltmesi:** `migrate` sorgu planı testindeki yalın `analyze` tüm veritabanını tarıyordu; paralel CI paketlerinin aynı anda kurup sildiği şemalarla deadlock (40P01). Artık yalnız testin kendi şemasındaki tablolar analiz ediliyor; paralel takım 3× temiz.
- **ENG-013 + ENG-015:** `game.Service.GameSnapshot(kullanıcı, oyun)` → `GameView` (`session.snapshot.game`): state_version, aşama (BÜYÜK), tur, süre, senaryo, sıradaki, katılımcılar (takma ad, koltuk, durum, `connected` hub’dan, açılmış kart metinleri), `my_cards` (yalnız kendi), `my_action_state`, hazır/oy/eşitlik sayaçları, son tur sonucu. Tek görünürlük kuralı: kart açıksa veya oyun bittiyse görünür; aksi halde yalnız sahibine. Oy hedefleri hiç yok. `RoomSnapshotter.Games` ile WS anlık görüntüsüne bağlandı (bağlanma ve `sync`). Testler: gizlilik özellik testi — 4 rastgele oyun (5–8 kişi; açma, oy, ayrılma, süre dolumu) her adımda tüm katılımcılar + bir yabancı için: başkasının kapalı kartı yok, kendi 5 kartı var, açık kartlar var, bitince hepsi var, kimlik/denge/hedef alanı yok. Alan testi (sıra, hazır/oy durumu ve sayaçlar, eleme sonucu, izleyici, eşitlik, bitiş), WS’de oyun sürerken bağlanan oyuncunun görünümü. Mutasyon 6/6 (biri ilk turda kaçtı → test tamamlandı). Kapsama: game %86,3, realtime %91,6.
- **RT-007:** `realtime.Hub` kullanıcının ilk bağlantısında / son bağlantısı kapanınca `ConnectionListener`’ı ayrı goroutine’de çağırır (kapanışta 1012 hariç; aynı bağlantının ikinci kez kaldırılması bildirim üretmez). `game.Disconnects`: oyun sürerken hayattaki oyuncuya `game_participants.disconnected_at` + `game.player_connection {disconnected}`; 60 sn içinde dönerse kayıt silinir + `{reconnected}`; dönmezse `room.Service.Leave` (sebep `disconnect_timeout` eylem kaydında) + `{left}`. Her kararda gerçek bağlantı durumu (presence) denetlenir (bildirimler sırasız gelebilir). Açılış kurtarması: aktif oyunlardaki hayattaki oyunculara süre tanınır, önceden kopmuşların kalan süresi korunur, bağlı oldukları anlaşılanların kaydı sessizce silinir. Sunucuda açılış/kapanış `lifecycle` ile. Testler (gerçek WS + PostgreSQL, süre 300 ms): kopma ve süre içinde dönüş, dönmeyenin odadan ve oyundan ayrılması ve 4003, lobide etkisizlik, açılış kurtarması (dönmeyen + kalan süreli ayrılır, bağlı 4 kalır). Paralel tam takım yükü altında 8 tur kararlı (yükte 2 sn okuma süresi yetmiyordu → test süreleri 5/8 sn). Mutasyon 5/5 (bir eşdeğer bozma davranışı değiştirmiyor). Not: `disconnects.go` gerçek zamanlı paketin testlerince sınanıyor; Go kapsamayı paket bazında ölçtüğü için game paketi tek başına %73,5 görünür.
- **Sohbet (+ SAFE-003 tamam):** `POST /v1/rooms/{id}/messages` oda modülünde (yalıtım matrisi kapsıyor): lobide/bitmiş odada her aktif üye, oyunda yalnız hayattakiler (403), üye değilse 404; `room.CleanMessage` (NFC, satır sonu → boşluk, kontrol/biçim/yön değiştirme karakterleri atılır, 1–200); `TextFilter` maskeleme bağlantı noktası (SAFE-013); sıklık 1/sn + 5/10 sn; `client_msg_id` tekrarı ilk sonucu döndürür, yayınlamaz, sınıra sayılmaz; `game_id` dolu (oyunda). `chat.message` olayı göndereni engelleyen üyelere gitmez (`room.Event.Exclude`, hub süzer); `session.snapshot.chat` son 50 mesaj (engellenenler hariç). SAFE-003: oda rotaları (sohbet dahil), oyun eylemleri/sonuç (katılımcı değilse aynı 404), WS (üyelik kayıttan önce) ve olaylar (oda + `To`/`Exclude`) yalıtılmış ve testli. Testler: metin temizleme birim; lobi (gönderim, maskeleme, tekrar, 429, doğrulama, 404, bitmiş oda), 10 sn sınırı, engelleme ve 50 mesaj geçmişi; oyunda hayattaki/elenen/izleyici; WS teslimi ve geçmiş. Mutasyon 7/7 (bir eşdeğer bozma gerçek bozmayla değiştirildi). Kapsama: room %87,0, realtime %92,0.
- **SAFE-013:** `internal/profanity` (liste `words.go`, gözden geçirilebilir): Normalize (NFC, Türkçe küçük harf, @/3/1/0/5/$ → harf, harf dışı atılır; tekrarlar korunur). Eşleşme: kök kelime başında (ekli haller), kısaltmalar yalnız tam kelime, istisnalar harf harf (sikke ≠ sike…). Tek yönlü sadeleştirme: metindeki düz harf listedeki Türkçe harfin yerine geçer ("serefsiz" ~ "şerefsiz"), tersi geçmez ("şikayet", "sıkıcı" ≁ "sik"). Harf tekrarı esnek ("siiiktir"), aralanmış harfler birleştirilir ("s i k t i r", "s_i_k"). Takma adda red (`NICKNAME_INVALID` / `profanity`), sohbette kelime `***`. Testler: 28 küfürlü örnek (maske + tespit), 61 masum örnek (sıkıcı, şikayet, götürmek, amca, amin, I am, dolu, klasik, class, assassin, Dickens…), takma adlar, bağlantı (auth/room arayüzleri). Kapsama %98,9; mutasyon 7/7. Test bulduğu hata: istisnaların tekrar sıkıştırılması "sikke"yi "sike" yapıp "sikeyim"i geçiriyordu — düzeltildi. Listeden çıkarılanlar: "döl" (dolu/dolar/dolap), "am"/"amın" (I am, amin). Bilinen sınırlar: küfrü bilerek ı ile yazmak; istisnayı taklit eden yazım (ör. "sikke…" ile başlayan) — raporlama kapatır.
- **SAFE-005 + SAFE-006:** `internal/moderation`: `POST/GET /v1/blocks`, `DELETE /v1/blocks/{id}` (kendini engelleme 400, olmayan kullanıcı 404, tekrar/olmayan kaldırma 204) ve `POST /v1/reports` (neden listesi, not temizliği, ortak oda kanıtı — üyelik/oyun/mesaj — yoksa aynı 404, son 50 mesaj kanıt olarak kopyalanır, açık rapor tekrarında aynı id, saatte 10 / kullanıcı, sınır DB’den önce). Moderasyon paneli kapsam dışı (raporlar DB’de ve günlükte). Testler (gerçek PostgreSQL): engelleme akışı ve kuralları, rapor satırı ve kanıt (55 mesajdan son 50, sohbet silinince kalır), tekrar ve kapanmış rapor, kurallar ve 404 eşitliği, lobiden ayrılınca kanıt kuralı, sıklık sınırı. Mutasyon 6/6 (bir eşdeğer bozma gerçek bozmayla değiştirildi). Kapsama %88,3.
- **RT-007 dayanıklılık düzeltmesi:** kopma/geri dönme kayıtları geçici veritabanı hatalarında artan aralıkla yeniden denenir (önceden hata yalnızca günlüğe düşüyor, kopan oyuncu hiç "ayrıldı" sayılmayabiliyordu). Paralel tam takım yükünde bir kopma testi aralıklı düşüyordu; kök neden kesin kanıtlanamadı (en olası: test veritabanı havuzunun dolması), olay bekleme süresi 15 sn yapıldı; yük altında 64 çalıştırma temiz.
- **SAFE-009 (temizlik) + ROOM-010:** `internal/maintenance.Job` sunucu içinde saatte bir (ilk 2 dk sonra), her adım partiler (500) ve çalıştırma başına 20 parti üst sınırla, `id = any(array(...))` biçiminde: sohbet 30 gün, süresi dolan oturumlar, terk edilmiş odalar (lobide/bitmiş, 6 sa hareketsiz, aktif üyelerden kimse bağlı değil; kilitli yeniden denetim; üyeler serbest bırakılır — tek aktif oda kısıtı), kapalı odalar 7 gün sonra silinir (oyunlar/raporlar kalır, bağ boşalır), sonuçlanan raporlar 180 gün. Oyundaki odalara dokunulmaz. Migration 0006 `reports_resolved_at_idx` (sorgu planı testi indeks olmadan tam taramayı gösterdi). Testler (gerçek PostgreSQL): tüm adımlar sınır değerleriyle (31/29 gün, 7/1 sa, 8/1 gün, 181/179 gün, açık rapor), bağlı üyeli oda, kapanan odanın üyelerinin yeni odaya girebilmesi, oyun/rapor korunması, idempotent ikinci çalıştırma, parti ve üst sınır, kilitli yeniden denetim, başlat/durdur. Mutasyon 6/6. Kapsama %85,4. Testin yakaladığı hata: ikinci Stop’u güvenli yapmak için alanları boşaltınca döngü nil kanalda sonsuza bekliyordu — döngü kanalları yerel kopyadan okur.

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
| AUTH-002b | Oturum uç noktaları (guest, refresh, me) ve IP sınırı | P0 | Done |
| AUTH-003 | REST ve WebSocket yetkilendirmesi | P0 | Done |

## Faz 4 — Gerçek zamanlı altyapı

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| RT-001 | WebSocket sunucusu | P0 | Awaiting User Approval |
| RT-002 | Bağlantı kimlik doğrulaması | P0 | Awaiting User Approval |
| RT-003 | Oda üyeliği kontrolü | P0 | Awaiting User Approval |
| RT-004 | Oda bazlı bağlantı yönetimi | P0 | Awaiting User Approval |
| RT-005 | Mesaj şemaları ve doğrulama | P0 | Backlog |
| RT-006 | Oda içi olay yayını | P0 | Awaiting User Approval |
| RT-007 | Bağlantı kopması ve yeniden bağlanma | P0 | Done |
| RT-008 | Bağlantı temizliği ve kaynak yönetimi | P0 | Backlog |
| RT-009 | Rate limiting | P0 | Backlog |
| RT-010 | Eşzamanlılık ve bağlantı testleri | P0 | Backlog |
| RT-011 | Cloud Run WebSocket kısıtlarının doğrulanması | P0 | Backlog |
| RT-012 | Gerçek zamanlı iletişim maliyet değerlendirmesi | P1 | Backlog |

## Faz 5 — Oda ve lobi

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| ROOM-001 | Oda modeli | P0 | Done |
| ROOM-002 | Davet kodu üretimi | P0 | Done |
| ROOM-003 | Oda oluşturma API'si | P0 | Done |
| ROOM-004 | Odaya katılma API'si | P0 | Done |
| ROOM-005 | Odadan ayrılma | P0 | Done |
| ROOM-006 | Oda üyeliği ve yetkilendirme | P0 | Done |
| ROOM-007 | Hazır ol mekanizması | P0 | Done |
| ROOM-008 | Oda ayarlarının senkronizasyonu (kapasite 1/2) | P0 | Done |
| ROOM-009 | Oda sahibi ayrıldığında yetki devri | P0 | Done |
| ROOM-010 | Boş oda temizliği | P1 | Awaiting User Approval |
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
| ENG-001 | Oyun durum makinesi | P0 | Done |
| ENG-002 | Durum geçişi doğrulama | P0 | Done |
| ENG-003 | Maç başlatma | P0 | Done |
| ENG-004 | Kart dağıtımı | P0 | Done |
| ENG-005 | Kart açıklama sırası | P0 | Done |
| ENG-006 | Tartışma aşaması | P0 | Done |
| ENG-007 | Tur zamanlayıcıları (kalıcı deadline) | P0 | Done |
| ENG-008 | Oylama | P0 | Done |
| ENG-009 | Oylama sonucu ve eşitlik bozma | P0 | Done |
| ENG-010 | Eleme | P0 | Done |
| ENG-011 | Kazanma koşulları | P0 | Done |
| ENG-012 | Maç sonu | P0 | Done |
| ENG-013 | Yeniden bağlanma ve oyun kurtarma | P0 | Done |
| ENG-014 | İdempotent eylemler | P0 | Backlog |
| ENG-015 | Gizli bilgi izolasyonu | P0 | Done |
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
| SAFE-003 | Oda izolasyonu | P0 | Done |
| SAFE-004 | Gizli kart erişim testleri | P0 | Backlog |
| SAFE-005 | Oyuncu raporlama | P0 | Done |
| SAFE-006 | Oyuncu engelleme | P0 | Done |
| SAFE-007 | Oda sahibi moderasyonu (lobiden atma) | P0 | Done |
| SAFE-008 | Gizlilik politikası ve kullanım koşulları | P0 | Backlog |
| SAFE-009 | Veri saklama ve silme (sohbet 30 gün) | P0 | In Progress (temizlik işi tamam; hesap silme `DELETE /v1/me` sonraki adım) |
| SAFE-010 | Secret yönetimi | P0 | Backlog |
| SAFE-011 | Güvenlik logları | P1 | Backlog |
| SAFE-012 | Mağaza güvenlik ve yaş derecelendirmesi | P0 | Backlog |
| SAFE-013 | Küfür filtresi: Türkçe kelime listesi, takma ad reddi ve sohbet maskeleme (`auth.ProfanityChecker`) | P0 | Done |

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
