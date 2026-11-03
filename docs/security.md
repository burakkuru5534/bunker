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
| Kurtarma | Uygulama silinirse / cihaz değişirse misafir hesabı kurtarılamaz; kullanıcıya ilk açılışta bildirilir |

## 2. Veritabanı erişimi (Supabase)

Supabase, `public` şemasındaki tabloları varsayılan olarak Data API (PostgREST) üzerinden dışarı açabilir. Uygulamamız bu API'yi kullanmaz.

- Tek Supabase projesinde iki şema: **`bunker_test`** ve **`bunker_prod`** ([ADR-0009](decisions/0009-single-supabase-schema-isolation.md)). Bu şemalar Data API'ye açılmaz.
- Tüm tablolarda **RLS etkin, politika yok** (varsayılan reddet) — ikinci savunma hattı.
- Supabase panelinde Data API kapatılır (**Integrations → Data API → Enable Data API** kapalı) veya en azından "Exposed schemas" listesinde `bunker_*` yoktur (MANUAL — DB-001b). Ayrıca şemalarda `anon`, `authenticated`, `service_role` ve `PUBLIC` yetkisi kaldırılır; `dbsetup check` bunu doğrular.
- **Database Settings → SSL Configuration → Enforce SSL on incoming connections** açıktır (MANUAL — DB-001b). Backend test/prod'da `sslmode=verify-full` + Supabase CA (`backend/certs/supabase-ca.crt`) olmadan başlamaz.
- Rol şifresi Supabase'e düz metin gitmez: `cmd/dbsetup` şifreyi yerelde üretir, `.env`'e yazar, SQL'e yalnızca **SCRAM-SHA-256 doğrulayıcısı** koyar. Rol: yalnızca `LOGIN`, bağlantı sınırı 10, `statement_timeout 5s`, `lock_timeout 3s`, `idle_in_transaction_session_timeout 10s`; şemada `CREATE` yetkisi yoktur (DDL migration rolünde, DB-002).
- Her ortam yalnızca kendi şemasında yetkili **ayrı bir veritabanı rolü** ile bağlanır (`bunker_test_app`, `bunker_prod_app`); roller birbirinin şemasına erişemez. `postgres`/service-role kullanılmaz.
- Supabase `anon` / `service_role` anahtarları mobil uygulamaya **asla** konmaz; backend'in de bunlara ihtiyacı yoktur.
- Tüm sorgular parametreli (pgx/sqlc) → SQL injection yok.

## 3. Gizli değerler

| Değer | Yerel | Test / Prod |
|---|---|---|
| `DATABASE_URL` | `.env` (git'e girmez) | Google Secret Manager |
| Diğer | — | Gerektiğinde Secret Manager |

- Cloud Run servis hesabı yalnızca kendi secret'ları için `roles/secretmanager.secretAccessor` iznine sahiptir.
- Test ve prod ayrı servis hesapları ve ayrı secret'lar kullanır.
- `.env.example` yalnızca anahtar adlarını içerir, değer içermez.

## 4. Tehditler ve önlemler

| Tehdit | Önlem | Görev |
|---|---|---|
| Token hırsızlığı | Keystore'da saklama, TLS, token loglanmaz, hash'li saklama, döndürme, iptal | AUTH-002 |
| Başka odanın verisine erişim (IDOR) | Her REST/WS isteğinde oda üyeliği ve oyun katılımı kontrolü | ROOM-006, SAFE-003 |
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
| Oturumlar | Süresi dolduktan / iptalden sonra temizlenir |
| Raporlar | SAFE-009'da belirlenir |

- Veriler AB bölgesinde (Supabase) tutulur. **KVKK** kapsamında yurt dışına veri aktarımı ve aydınlatma metni gereklilikleri SAFE-008'de ele alınır; hukuki metinler için bir uzmana danışılması önerilir.
- Google Play "Veri güvenliği" formu SAFE-012 / REL-008'de doldurulur.
