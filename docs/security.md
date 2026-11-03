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
| Token biçimi (AUTH-001) | 32 bayt → base64url **dolgusuz, 43 karakter**. Hash ham 32 bayt üzerinden SHA-256. Biçimi bozuk token veritabanına sorulmadan `401` alır (`auth.HashToken`). `Bearer` şema adı büyük/küçük harf duyarsız |
| Takma ad (AUTH-001) | NFC + kırpma; 3–16 karakter; **yalnızca Latin alfabesi** harfleri (Türkçe dahil), 0–9, boşluk, `_`; en az bir harf; ardışık boşluk yok; ayrılmış adlar yok; küfür filtresi (SAFE-013). Latin'e benzeyen başka alfabe harfleriyle (ör. Kiril "а") başkasını taklit etme ve birleşik işaret yığınları engellenir. Karşılaştırma Türkçe kurallarıyla (`IŞIK` = `ışık`, `I` ≠ `i`) |
| Son kullanım yazımı (AUTH-001) | `sessions.last_used_at` ve `users.last_seen_at` her istekte değil, **en fazla saatte bir** güncellenir (istek başına yazma olmasın) |

### Akışlar (AUTH-001; uygulama AUTH-002/003)

| Akış | Adımlar |
|---|---|
| Oluşturma `POST /v1/sessions/guest` | IP sınırı (5/dk) → takma ad doğrula → tek transaction: `users` + `sessions` (hash, `expires_at = now + 90 gün`) → `201 { token, expires_at, user }`. Ham token yalnızca bu yanıtta bulunur |
| Doğrulama (her istek) | `Bearer` al → biçim → hash → önbellek (≤ 60 sn) veya `sessions ⨝ users` (`token_hash`): süresi dolmuş, iptal edilmiş ya da kullanıcısı silinmiş → `401 UNAUTHORIZED`. Hata nedeni istemciye ayrıştırılmaz |
| Yenileme `POST /v1/sessions/refresh` | Geçerli oturumla: tek transaction'da eskisini iptal et + yenisini oluştur → `200 { token, expires_at }`. Ağ hatasında istemci eski token'la tekrar dener; eski token iptal edildiyse yeniden misafir oturumu açar |
| Hesap silme `DELETE /v1/me` | Aktif odadan ayrıl → takma ad `Silinmiş oyuncu`, `deleted_at` → tüm oturumlar silinir → `204` |
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
| Raporlar | SAFE-009'da belirlenir. Rapor, odanın son mesajlarının kopyasını içerir; sohbetin 30 günlük silinmesinden etkilenmez |

- Veriler AB bölgesinde (Supabase) tutulur. **KVKK** kapsamında yurt dışına veri aktarımı ve aydınlatma metni gereklilikleri SAFE-008'de ele alınır; hukuki metinler için bir uzmana danışılması önerilir.
- Google Play "Veri güvenliği" formu SAFE-012 / REL-008'de doldurulur.
