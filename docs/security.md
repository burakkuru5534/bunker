# Güvenlik — Sığınak: Son Kalanlar

> Kaynak: ARCH-003 (2026-11-02). Uygulama görevleri: AUTH-*, RT-009, SAFE-*.

## 1. Kimlik ve oturum modeli

Ayrıntı: [ADR-0008](decisions/0008-opaque-session-tokens.md).

| Konu | Karar |
|---|---|
| Oturum türü | Misafir. `POST /v1/sessions/guest` → yeni `users` satırı + oturum |
| Token | **Opak**, 32 bayt kriptografik rastgele (`crypto/rand`), base64url. JWT kullanılmaz. |
| Saklama (sunucu) | Yalnızca `SHA-256(token)` → `sessions.token_hash`. Ham token veritabanında yok. |
| Saklama (istemci) | `flutter_secure_storage` (Android Keystore, AES-GCM). Android yedekleme ve cihazdan cihaza aktarım kapalı (`allowBackup=false`, `data_extraction_rules.xml`) |
| Süre | 90 gün; `POST /v1/sessions/refresh` token'ı **döndürür** (yenisini verir, eskisini iptal eder) |
| İptal | `sessions.revoked_at`; doğrulama her istekte sunucu tarafında yapılır |
| Doğrulama maliyeti | Bellekte kısa ömürlü (≤60 sn) önbellek; iptal edilen oturum en geç 60 sn içinde geçersiz olur |
| Taşıma | REST: `Authorization: Bearer`. WS: `Authorization` başlığı. **Token URL'de asla yer almaz.** |
| Kurtarma | Uygulama silinirse / cihaz değişirse misafir hesabı kurtarılamaz; kullanıcıya ilk açılışta bildirilir. Erişilemeyen hesap 90 gün sonra otomatik silinir (§6) |
| Token biçimi (AUTH-001) | 32 bayt → base64url **dolgusuz, 43 karakter**. Hash ham 32 bayt üzerinden SHA-256. Biçimi bozuk token veritabanına sorulmadan `401` alır (`auth.HashToken`). `Bearer` şema adı büyük/küçük harf duyarsız |
| Takma ad (AUTH-001) | NFC + kırpma; 3–16 karakter; **yalnızca Latin alfabesi** harfleri (Türkçe dahil), 0–9, boşluk, `_`; en az bir harf; ardışık boşluk yok; ayrılmış adlar yok; küfür filtresi (SAFE-013). Latin'e benzeyen başka alfabe harfleriyle (ör. Kiril "а") başkasını taklit etme ve birleşik işaret yığınları engellenir. Karşılaştırma Türkçe kurallarıyla (`IŞIK` = `ışık`, `I` ≠ `i`) |
| Son kullanım yazımı (AUTH-001) | `sessions.last_used_at` ve `users.last_seen_at` her istekte değil, **en fazla saatte bir** güncellenir (istek başına yazma olmasın) |

### Akışlar (AUTH-001; uygulama AUTH-002/003)

| Akış | Adımlar |
|---|---|
| Oluşturma `POST /v1/sessions/guest` | IP sınırı (5/dk) → takma ad doğrula → tek transaction: `users` + `sessions` (hash, `expires_at = now + 90 gün`) → `201 { token, expires_at, user }`. Ham token yalnızca bu yanıtta bulunur |
| Doğrulama (her istek) | `Bearer` al → biçim → hash → önbellek (≤ 60 sn) veya `sessions ⨝ users` (`token_hash`): süresi dolmuş, iptal edilmiş ya da kullanıcısı silinmiş → `401 UNAUTHORIZED`. Hata nedeni istemciye ayrıştırılmaz |
| Yenileme `POST /v1/sessions/refresh` | Geçerli oturumla: tek transaction'da eskisini iptal et + yenisini oluştur → `200 { token, expires_at }`. Ağ hatasında istemci eski token'la tekrar dener; eski token iptal edildiyse yeniden misafir oturumu açar |
| Hesap silme `DELETE /v1/me` | Aktif odadan ayrıl → takma ad `Silinmiş oyuncu` (oda üyelik kayıtlarında da), `deleted_at` → tüm oturumlar, kendi sohbet mesajları ve engellemeler silinir → `204`. Raporlar moderasyon kanıtı olarak kalır (180 gün kuralı) |
| İstemci IP'si | `httpapi.ClientIP`: yerelde `RemoteAddr`; bulutta `X-Forwarded-For`'un **sağdan `TRUSTED_PROXY_HOPS`'uncu** girdisi (varsayılan 1); istemcinin yazdığı soldaki girdilere güvenilmez. IPv6 sınır anahtarı `/64`. Cloud Run'daki kesin biçim ilk dağıtımda doğrulanır |

## 2. Veritabanı erişimi (Supabase)

Supabase, `public` şemasındaki tabloları varsayılan olarak Data API (PostgREST) üzerinden dışarı açabilir. Uygulamamız bu API'yi kullanmaz.

- Tek Supabase projesinde iki şema: **`bunker_test`** ve **`bunker_prod`** ([ADR-0009](decisions/0009-single-supabase-schema-isolation.md)). Bu şemalar Data API'ye açılmaz.
- Tüm tablolarda **RLS etkin**; tek politika `app_access` yalnızca uygulama rolüne izin verir. `anon` / `authenticated` / `service_role` / `PUBLIC` için yetki ve politika yok (varsayılan reddet) — ikinci savunma hattı ([ADR-0012](decisions/0012-migration-role-and-rls-policy.md)). Tabloların sahibi ayrı migration rolüdür; uygulama rolü DDL yapamaz.
- Supabase panelinde Data API kapatılır (**Integrations → Data API → Enable Data API** kapalı) veya en azından "Exposed schemas" listesinde `bunker_*` yoktur (MANUAL — DB-001b). Ayrıca şemalarda `anon`, `authenticated`, `service_role` ve `PUBLIC` yetkisi kaldırılır; `dbsetup check` bunu doğrular.
- **Database Settings → SSL Configuration → Enforce SSL on incoming connections** açıktır (MANUAL — DB-001b). Backend test/prod'da `sslmode=verify-full` + Supabase CA (`backend/certs/supabase-ca.crt`) olmadan başlamaz.
- Rol şifresi Supabase'e düz metin gitmez: `cmd/dbsetup` şifreyi yerelde üretir, `.env`'e yazar, SQL'e yalnızca **SCRAM-SHA-256 doğrulayıcısı** koyar. Rol: yalnızca `LOGIN`, bağlantı sınırı 10, `statement_timeout 5s`, `lock_timeout 3s`, `idle_in_transaction_session_timeout 10s`; şemada `CREATE` yetkisi yoktur (DDL migration rolünde, DB-002).
- Her ortam yalnızca kendi şemasında yetkili **ayrı bir veritabanı rolü** ile bağlanır (`bunker_test_app`, `bunker_prod_app`); roller birbirinin şemasına erişemez. `postgres`/service-role kullanılmaz.
- Supabase `anon` / `service_role` anahtarları mobil uygulamaya **asla** konmaz; backend'in de bunlara ihtiyacı yoktur.
- Tüm sorgular parametreli (pgx/sqlc) → SQL injection yok.

## 3. Gizli değerler (SAFE-010)

### Envanter

| Değer | Kullanan | Yerel | CI | Test / Prod (Cloud Run) |
|---|---|---|---|---|
| `DATABASE_URL` (uygulama rolü `bunker_<env>_app`) | sunucu | `backend/.env` (prod: `.env.prod`) | Yok — CI geçici PostgreSQL kullanır (gizli değil) | Cloud Run ortam değişkeni (Secret Manager kullanılmaz) |
| `MIGRATION_DATABASE_URL` (migration rolü `bunker_<env>_migrator`) | `cmd/migrate` | Aynı `.env` | Yok | **Cloud Run'a verilmez**; migration geliştirici makinesinden çalışır |
| Supabase veritabanı yönetici (`postgres`) şifresi | yalnızca rol kurulumu (SQL editörü) | Hiçbir dosyada değil; Supabase panelinde | — | — |
| Supabase API anahtarları (publishable / `sb_secret_…`) | **kullanılmıyor** (uygulama Supabase API'sine bağlanmaz) | — | — | — |

- Mobil uygulamada gizli değer yoktur: yalnızca API/WS adresleri ve yasal sayfa adresleri (`--dart-define`, gizli değil). Oturum token'ı cihazda Keystore'da tutulur (§1).
- Yapılandırma gizli değerleri yalnızca ortam değişkeninden okur (`config.Secret`: loglarda ve hata mesajlarında maskeli). `.env` yalnızca `APP_ENV` boş/`local` iken okunur; bulutta dosya okunmaz.
- `.env*` (`.env.example` hariç), anahtar/sertifika dosyaları ve servis hesabı JSON'ları `.gitignore`'dadır. `.env.example` yalnızca anahtar adlarını içerir.

### Depoya sızmaya karşı

- `backend/cmd/secretscan` bilinen biçimleri arar: Supabase gizli anahtarı, JWT, parolalı PostgreSQL adresi, özel anahtar, servis hesabı JSON'u, `*_SECRET_KEY=` / `PGPASSWORD=` atamaları. Bulguda değer yazılmaz (dosya:satır:tür). Test/belge yer tutucuları (`cokgizli`, `<parola>`, `${…}`) bulgu değildir; bilinçli bir örnek satıra `secretscan:ignore` yazılır.
- **CI:** `.github/workflows/secrets.yml` her push ve PR'da izlenen tüm dosyaları tarar.
- **Commit öncesi (isteğe bağlı):** `git config core.hooksPath .githooks` → `.githooks/pre-commit` commit'e hazırlanan içeriği tarar, bulgu varsa commit durur.
- Geçmiş taraması (5 Kasım 2026, 83 commit): gerçek gizli değer bulunmadı; eşleşmelerin hepsi test yer tutucusu.

### Cloud Run'da (REL-003; kaynak oluşturma onayla)

**Karar (2026-11-05, kullanıcı): Secret Manager kullanılmaz.** `DATABASE_URL` Cloud Run servisine doğrudan ortam değişkeni olarak verilir. Sonuçları ve önlemleri:

- Değer, Cloud Run revizyon yapılandırmasında düz metin durur: projede servis yapılandırmasını okuyabilen herkes (`run.services.get`; **Viewer rolü dahil**) görebilir. → GCP projesine başka kişi veya rol eklenmez; eklenirse önce şifre döndürülür ve bu karar yeniden değerlendirilir.
- Değer komut satırına yazılmaz (terminal geçmişine düşmesin): dağıtımda git'e girmeyen bir dosyadan verilir — `gcloud run deploy … --env-vars-file backend/.env.cloudrun.yaml` (`.env.*` kalıbıyla `.gitignore`'da; `secretscan` de yakalar).
- Değerin kendisi bir uygulama rolü bağlantısıdır: rol yalnızca kendi şemasında yetkili, `CREATE` yetkisi yok (§2); sızarsa etkisi o şemayla sınırlı ve rol şifresi dakikalar içinde döndürülebilir (aşağıda).
- Ortam başına ayrı servis ve ayrı rol (test / prod); test servisine prod bağlantısı verilmez.
- Döndürmede yeni değer yeni revizyonla devreye girer (`--env-vars-file` ile yeniden dağıtım); eski revizyonlar trafik almaz ama yapılandırmalarında eski (artık geçersiz) değer kalır — sorun değildir, eski şifre zaten iptal edilmiştir.

### Döndürme (rotation)

| Ne | Ne zaman | Nasıl |
|---|---|---|
| Uygulama / migration rolü şifresi | Sızma şüphesi; ekipten ayrılan biri; yılda bir | `go run ./cmd/dbsetup init -env <test\|prod> -role <app\|migrator> …` yeni şifre üretir, `.env`'e yazar ve Supabase SQL editöründe çalıştırılacak `ALTER ROLE … PASSWORD 'SCRAM-SHA-256$…'` ifadesini verir (düz şifre Supabase'e gitmez). Bulutta: `.env.cloudrun.yaml`'ı güncelle → yeniden dağıt. `dbsetup check` ile doğrula |
| Supabase `postgres` yönetici şifresi | Sızma şüphesi | Supabase panelinde Project Settings → Database → "Reset database password" (menü adları değişebilir). Uygulama bu rolü kullanmadığı için başka değişiklik gerekmez |
| Supabase API gizli anahtarı | Sızma şüphesi | Panelde Project Settings → API Keys: anahtarı iptal et / yenisini oluştur. Uygulama kullanmadığı için başka değişiklik gerekmez |
| Kullanıcı oturum token'ları | Sunucu tarafı sızma şüphesi | `update sessions set revoked_at = now()` (tümü) — herkes yeniden misafir hesap açar; token'lar yalnızca SHA-256 özetiyle saklandığı için veritabanı sızıntısı tek başına token vermez |

**Açık madde (kullanıcı):** Geliştirme sırasında Supabase `postgres` şifresi ve `SUPABASE_SECRET_KEY` bir sohbet ortamına yazıldı. Bu değerler hiçbir dosyada ve git geçmişinde yok (tarandı), ancak **ikisi de yukarıdaki tabloya göre yenilenmelidir.** Yenileme tamamlanınca bu madde kaldırılır.

## 4. Tehditler ve önlemler

| Tehdit | Önlem | Görev |
|---|---|---|
| Token hırsızlığı | Keystore'da saklama, TLS, token loglanmaz, hash'li saklama, döndürme, iptal | AUTH-002 |
| Başka odanın verisine erişim (IDOR) | Her REST/WS isteğinde oda üyeliği ve oyun katılımı kontrolü; üye olmayana yanıt, var olmayan odanınkiyle **aynı** (varlık sızmaz). Oda rotaları tek tabloda (`roomScopedRoutes`) ve yalıtım testi hepsini dolaşır; diğer modüller `room.LoadMembership` kullanır | ROOM-006, SAFE-003 (oda kısmı tamam; oyun/sohbet/WS kendi görevlerinde) |
| Gizli kart sızıntısı | Genel yayınlarda açılmamış kart yok; özel kanal; snapshot üretimi testli | ENG-015, SAFE-004 |
| Oy gizliliğinin ihlali | `votes` yalnızca sunucuda; istemciye yalnızca sayılar | ENG-009 |
| Değiştirilmiş istemciyle hile | Sunucu otoriter; istemci yalnızca niyet gönderir | ENG-002 |
| Tekrarlanan istekler | `action_id` tekilliği, `state_version` | ENG-014 |
| Davet kodu tahmini | 6 karakter, belirsiz harfler hariç 31 karakterlik alfabe (~887 milyon); katılma denemesi sınırı | ROOM-002, SAFE-002 |
| Sohbet spam / taciz | Sıklık sınırı, 200 karakter, küfür filtresi, engelleme, raporlama | SAFE-005/006 |
| Toplu hesap açma | `POST /sessions/guest` IP başına sınır | SAFE-002 |
| Kaynak tüketme (DoS) | Gövde ≤ 4 KB; kullanıcı başına en fazla 2 WS bağlantısı; WS kuyruğu dolunca kesme; max-instances=1 maliyet tavanı | RT-008, RT-009 |
| Loglarda kişisel veri | Token, `Authorization` başlığı, sohbet içeriği loglanmaz; IP yalnızca güvenlik olaylarında | BE-003, SAFE-011 |

## 5. Sıklık sınırları (başlangıç değerleri)

Bellek içi token bucket (tek instance olduğu için yeterli).

| Kapsam | Sınır |
|---|---|
| Misafir oturumu oluşturma | IP başına 5 / dk |
| Odaya katılma (davet kodu denemesi) | Kullanıcı başına 10 / dk |
| Genel REST | Kullanıcı başına 30 / 10 sn |
| Sohbet | 1 / sn ve 5 / 10 sn |
| WS bağlantısı | Kullanıcı başına eşzamanlı 2 |

## 6. Gizlilik ve veri saklama

| Veri | Saklama |
|---|---|
| Sohbet mesajları | 30 gün |
| Oyun sonuçları, kartlar, tur sonuçları | Kalıcı (kullanıcı silme talebinde anonimleştirilir) |
| Oylar (`votes`) | Kalıcı, yalnızca sunucu |
| Oturumlar | Süresi dolunca silinir |
| Hesaplar | Kullanıcı silene kadar (`DELETE /v1/me`). **90 gündür kullanılmayan** (`last_seen_at`), oturumu ve aktif odası kalmamış hesap — artık hiçbir cihazdan erişilemez — bakım işinde `DELETE /v1/me` ile aynı şekilde anonimleştirilir (SAFE-014; `account.Anonymize`) |
| Odalar | Lobide/bitmiş, 6 saattir hareketsiz ve kimse bağlı değilse kapanır (üyeler serbest bırakılır); kapandıktan 7 gün sonra silinir. Oyunlar ve raporlar kalır, oda bağları boşalır |
| Raporlar | Açık/incelemedeki raporlar silinmez; sonuçlanan (actioned/dismissed) rapor sonuçlandıktan **180 gün** sonra silinir. Rapor, odanın son 50 mesajının kopyasını içerir; sohbetin 30 günlük silinmesinden etkilenmez |

- Temizlik sunucu içinde saatte bir çalışır (`internal/maintenance`; açılıştan 2 dk sonra ilk çalıştırma), partiler hâlinde ve çalıştırma başına üst sınırla; harici zamanlayıcı ve ek maliyet yok. Sunucu kapalıyken (min-instances 0) çalışmaz, ilk açılışta birikeni temizler.
- Veriler AB bölgesinde (Supabase) tutulur. **KVKK** kapsamında yurt dışına veri aktarımı ve aydınlatma metni gereklilikleri SAFE-008'de ele alınır; hukuki metinler için bir uzmana danışılması önerilir.
- Google Play "Veri güvenliği" formu SAFE-012 / REL-008'de doldurulur.

## 7. Güvenlik logları (SAFE-011)

Güvenlik açısından anlamlı olaylar tek tip alanlarla loglanır (`observability.Security`). Cloud Logging'de `jsonPayload.security_event` alanıyla süzülür; olay adları sabittir (panel ve uyarılar bunlara dayanır).

| Olay | Seviye | Ne zaman | Ek alanlar |
|---|---|---|---|
| `auth_failed` | WARNING | Geçersiz, süresi dolmuş veya iptal edilmiş token (REST 401, WS 4001) | `route`, `method`/`close_code` |
| `access_denied` | WARNING | Yetki, üyelik veya atılma reddi (REST 403 `FORBIDDEN`/`NOT_A_MEMBER`/`KICKED`, WS 4003) | `user_id`, `route`, (WS) `room_id` |
| `rate_limited` | WARNING | Sıklık sınırı (REST 429, WS bağlantı denemesi 4008) | `user_id` (oturumluysa), `route` |
| `session_created` | INFO | Misafir hesap açıldı | `user_id` |
| `account_deleted` | INFO | `DELETE /v1/me` | `user_id`, `left_room` |
| `report_created` | INFO | Yeni rapor (açık rapor tekrarı olay değildir) | `report_id`, `user_id`, `reported_id`, `room_id`, `reason` |
| `user_blocked` | INFO | Yeni engel (tekrarı olay değildir) | `user_id`, `blocked_id` |
| `member_kicked` | INFO | Oda sahibi üyeyi attı | `user_id` (sahip), `room_id`, `target_user_id` |

- Her olayda istek varsa `request_id` ve izleme alanı, oturum varsa `user_id` bulunur. **`client_ip` yalnızca güvenlik olaylarında** yazılır (`httpapi.ClientIPContext`, güvenilir vekil sayısına göre); erişim logu ve diğer loglar IP yazmaz.
- **Loglanmayanlar:** token ve `Authorization` başlığı (denenen geçersiz token dahil), sohbet içeriği, rapor notu ve rapordaki mesaj kopyası, takma ad. Gizli anahtar adları ayrıca maskelenir (`[REDACTED]`).
- `route` rota kalıbıdır (`POST /v1/rooms/{roomId}/kick`); kimlikler kalıpta yer tutucudur.
- Oyun kuralı hataları (409), doğrulama (400) ve bulunamadı (404) güvenlik olayı değildir. Üye olmayanın oda isteği REST'te 404 döndüğü için (varlık sızmasın) olay üretmez; WS'te 4003 `access_denied` olarak görünür. Kapanmış bir odaya yeniden bağlanan meşru istemciler de `access_denied` üretir; tek başına saldırı göstergesi değildir.
- Kullanılmayan hesapların otomatik silinmesi (SAFE-014) tek tek değil, temizlik işinin toplam sayısıyla loglanır.

Saklama ve maliyet (Kasım 2026, Google Cloud Observability fiyat sayfası): Cloud Logging'de proje başına aylık ilk 50 GiB ücretsiz, fazlası $0,50/GiB; `_Default` kovası 30 gün saklar (30 günden uzun saklama ücretli). Güvenlik olayları ayrı kovaya alınmaz; 30 gün yeterli kabul edilir. Kullanıcı tanımlı log tabanlı metrikler (uyarı için) Cloud Monitoring özel metriği olarak **ücretlidir**; kurulmadan önce onay alınır (REL-004).

Örnek sorgu (Logs Explorer): `jsonPayload.security_event="auth_failed"` veya `jsonPayload.security_event=~"rate_limited|access_denied" AND jsonPayload.client_ip="203.0.113.7"`.
