# Sistem Mimarisi — Sığınak: Son Kalanlar

> Kaynak: ARCH-001 (2026-11-02, kullanıcı onaylı). Kararların gerekçeleri: [decisions/](decisions/).

## 1. Bileşenler

```
┌──────────────┐   HTTPS (REST /v1)    ┌────────────────────────────┐   TCP/TLS   ┌──────────────────────┐
│ Flutter      │ ────────────────────► │ Cloud Run: bunker-api      │ ──────────► │ Supabase PostgreSQL  │
│ (Android)    │ ◄──── WSS /v1/ws ───► │ Go modüler monolith        │             │ (tek kalıcı kaynak)  │
└──────────────┘                       │ max 1 instance             │             └──────────────────────┘
                                       └────────────┬───────────────┘
                                                    │
                                      Secret Manager (DB bağlantısı, token anahtarı)
                                      Cloud Logging (JSON loglar)
```

- Tek Go servisi REST ve WebSocket'i birlikte sunar.
- MVP'de **en fazla 1 instance**; WebSocket bağlantıları bellekte tutulur, instance'lar arası koordinasyon yoktur ([ADR-0003](decisions/0003-single-instance-websocket-hub.md)).
- **PostgreSQL tek otoritedir.** Bellekteki her şey (bağlantılar, zamanlayıcılar) veritabanından yeniden kurulabilir.

## 2. Repository yapısı

```
bunker/
  apps/mobile/            Flutter uygulaması
  backend/                Go modülü (go.mod burada)
    cmd/server/
    internal/...
    tests/integration/
  migrations/postgres/    Sürümlü SQL migration'ları
  docs/
  scripts/
```

## 3. Backend modülleri

```
backend/
  cmd/server/            main: config, bağımlılık bağlama, graceful shutdown
  internal/
    config/              ortam değişkenleri ve doğrulama
    observability/       slog JSON loglama, request-id, gizli değer maskeleme
    httpapi/             router, middleware, standart hata formatı, health
    auth/                misafir oturumu, token üretimi/doğrulaması
    room/                oda, üyelik, davet kodu, hazır ol, kapasite, sahiplik devri
    game/
      engine/            SAF oyun motoru: (durum, eylem, zaman, rng) → (yeni durum, olaylar)
      service/           kullanım senaryoları: yetki, transaction, motor, yayın
      timers/            deadline zamanlayıcısı ve açılışta kurtarma
    cards/               kart ve senaryo kataloğu, dengeli dağıtım
    chat/                mesaj, sıklık sınırı, küfür filtresi, 30 gün saklama
    moderation/          raporlama, engelleme
    realtime/            WebSocket hub, oda kanalları, özel mesajlar
    persistence/         pgx havuzu, sqlc sorguları, transaction yardımcısı
  tests/integration/     PostgreSQL ve WebSocket entegrasyon testleri
```

### Master prompt yapısından sapmalar

| Değişiklik | Gerekçe |
|---|---|
| `lobby` → `room` | Lobi odanın bir durumu; modül oda yaşam döngüsünün tamamını yönetir |
| `voting` → `game/engine` | Oylama ve eşitlik bozma durum makinesinin parçası; ayrı modül kuralları ikiye böler |
| `chat` eklendi | Yazılı sohbet kararı (GAME-001) |
| `tests/integration` → `backend/tests/integration` | `go test ./...` ile doğrudan çalışır |

### Katman kuralı

```
transport (HTTP handler / WS mesaj işleyici)
      ▼
service   (kimlik, oda üyeliği, transaction, idempotency, yayın)
      ▼
engine    (saf iş kuralları; I/O yok, saat ve rng dışarıdan verilir)
```

- Handler'larda iş kuralı yoktur.
- Engine veritabanı, ağ ve sistem saatine erişmez → kurallar deterministik birim testleriyle doğrulanır ([ADR-0002](decisions/0002-modular-monolith-pure-engine.md)).

## 4. Eşzamanlılık ve tutarlılık

Ayrıntı: [ADR-0004](decisions/0004-db-authoritative-state.md).

| Konu | Yaklaşım |
|---|---|
| Eşzamanlı eylemler | Kısa transaction içinde `games` satırı `SELECT … FOR UPDATE`; ek olarak bellekte oda başına mutex |
| Sürüm | Her durum değişikliğinde `state_version` +1; olaylar sürümle gönderilir |
| İdempotency | Her eylemde istemci `action_id` (UUID) gönderir; `(game_id, player_id, action_id)` tekil. Oy için `(game_id, round, voter_id)` tekil |
| Yayın | Olaylar **commit'ten sonra** gönderilir |
| Transaction süresi | Yalnızca eylem işlenirken (ms). Tartışma veya WS boyunca açık transaction tutulmaz |

## 5. Zamanlayıcılar

Ayrıntı: [ADR-0005](decisions/0005-persistent-deadlines.md).

- Aşama bitişi `games.phase_deadline_at` alanında tutulur.
- Bellekteki zamanlayıcı süresi dolunca sistem adına "aşama süresi doldu" eylemi gönderir; beklenen `state_version` ile işlendiği için tekrar tetiklenmesi zararsızdır.
- Sunucu açılışında aktif oyunlar yüklenir; süresi geçmiş aşamalar hemen işlenir.

## 6. Gerçek zamanlı iletişim

- **Hub:** `oda → bağlantılar` eşlemesi. Her bağlantıda bir yazma goroutine'i ve sınırlı gönderim kuyruğu; kuyruğu dolan yavaş istemci kesilir ve yeniden bağlanır.
- **Mesaj türleri:** odaya genel yayın (açılmış kartlar, sayaçlar, sohbet) ve oyuncuya özel mesaj (kendi kartları).
- **Yeniden bağlanma:** İstemci son `state_version`'ı gönderir; sunucu oyuncuya özel **tam durum görüntüsü** ve son 50 sohbet mesajını yollar ([ADR-0006](decisions/0006-snapshot-on-reconnect.md)).
- **Protokol sürümü:** REST `/v1`; WS zarfı `{"v":1,"type":"...","payload":{...}}`.
- Mesaj şemaları ARCH-002 / ARCH-004'te kesinleşir.

## 7. Flutter yapısı

```
apps/mobile/lib/
  app/          router (go_router), tema (Material 3), başlangıç
  core/         network (Dio, WS istemcisi), güvenli depolama, hata modeli, l10n
  features/
    session/    misafir oturumu, takma ad
    home/       ana ekran, oda oluştur/katıl
    room/       lobi
    game/       kartlar, tartışma, oylama, eleme, sonuç
    chat/       sohbet bileşeni (lobi ve oyunda ortak)
    moderation/ raporla, engelle
```

- Her feature: `data/` (API, DTO) ve `presentation/` (ekran, widget, provider). `domain/` yalnızca gerektiğinde.
- İstemci yalnızca sunucu durumunu gösterir; oyun kuralı hesaplamaz.
- Kod üretici paketler (freezed, riverpod_generator vb.) MOB-003/004'te sürüm uyumluluğuyla birlikte değerlendirilir.

## 8. Ortamlar

| Ortam | Backend | Veritabanı |
|---|---|---|
| Yerel | Go, Windows'ta doğrudan | Docker Desktop'ta PostgreSQL konteyneri |
| Test | Cloud Run `bunker-api-test` (min 0, max 1) | Ayrı Supabase projesi |
| Production | Cloud Run `bunker-api` (max 1) | Ayrı Supabase projesi |

- Bölge hedefi: Frankfurt (Supabase `eu-central-1`, Cloud Run `europe-west3`) — kesin karar ARCH-005.
- Testler bulut çalışma alanında da aynı sürümlerle çalıştırılır (Go 1.27.0, Flutter 3.47.1, PostgreSQL).
- Bulut kaynakları ve ücretli değişiklikler ayrıca kullanıcı onayı gerektirir.

## 9. Bilinen sınırlamalar

- Her deploy ve instance değişiminde **tüm WebSocket bağlantıları kopar**; istemci otomatik yeniden bağlanıp tam durumu alır.
- Tek instance kapasitesi RT-011'de ölçülecek. Aşılırsa çoklu instance için yeni ADR gerekir (ör. PostgreSQL LISTEN/NOTIFY ile olay dağıtımı).
- Cloud Run istek zaman aşımı (en fazla 60 dk) WebSocket ömrünü sınırlar; istemci periyodik yeniden bağlanmayı sorunsuz yönetmelidir.

## 10. Veri akışı (ARCH-002)

### Komut ve olay kanalları

- **Tüm komutlar REST** ile gönderilir (kart açma, hazırım, oy, sohbet mesajı, oda işlemleri).
- **WebSocket yalnızca sunucu → istemci** olay ve durum taşır. İstemciden yalnızca `sync` (tam durum isteği) gelir; canlılık için WS ping/pong.
- İstemci ekranını HTTP yanıtına göre değil, WS olaylarına göre günceller (tek durum kaynağı).
- Gerekçe: [ADR-0007](decisions/0007-rest-commands-ws-push.md).

### a) Oyun eylemi

```
POST /v1/games/{id}/actions {action_id, type, payload}   Authorization: Bearer <token>
  1. Token doğrulama → user_id
  2. Katılımcı mı, eylem için uygun durumda mı (ör. hayatta)?
  3. BEGIN
     SELECT * FROM games WHERE id=$1 FOR UPDATE
     game_actions'ta (game_id, actor_id, action_id) var mı? → varsa kayıtlı sonucu döndür
     engine.Apply(durum, eylem, now, rng) → yeni durum + olaylar | kural hatası
     INSERT votes / UPDATE player_cards / UPDATE games (phase, deadline, state_version+1)
     INSERT game_actions
     COMMIT
  4. Commit sonrası: hub.Broadcast(oda, genel olaylar); hub.SendPrivate(oyuncu, özel olaylar);
     timers.Reschedule(oyun, deadline)
  5. HTTP 200 {state_version}
```

### b) Süre dolması

Zamanlayıcı → sistem eylemi `phase_timeout{expected_version}` → (a) ile aynı yol. Sürüm uyuşmazsa işlem yapılmaz.

### c) Sohbet mesajı

```
POST /v1/rooms/{id}/messages {client_msg_id, text}
  → üyelik (yalnızca hayattaki oyuncular / lobidekiler yazabilir)
  → sıklık sınırı (bellek içi, kullanıcı başına) → 200 karakter + küfür filtresi
  → INSERT chat_messages (oyun kilidi alınmaz)
  → hub.Broadcast(oda, chat.message) — engelleyen alıcılara gönderilmez
```

### d) Bağlanma / yeniden bağlanma

```
WSS /v1/ws?room_id=...   Authorization: Bearer <token>   (token URL'de değil → loglara düşmez)
  → token + oda üyeliği doğrulanır → bağlantı hub'a eklenir
  → session.snapshot (oyuncuya özel tam durum + son 50 mesaj)
  → sonraki olaylar state_version ile; sürüm atlaması → istemci {"type":"sync"} → yeni snapshot
```

### WebSocket olayları (sunucu → istemci)

Zarf: `{"v":1,"type":"...","room_id":"...","state_version":42,"ts":"...","payload":{...}}`

| Olay | Kime | İçerik |
|---|---|---|
| `session.snapshot` | Özel | Tam oda/oyun durumu, kendi kartları, son 50 mesaj |
| `room.member_joined` / `room.member_left` / `room.member_kicked` | Oda | Üye |
| `room.ready_changed` | Oda | Kullanıcı, hazır |
| `room.settings_changed` | Oda | Kapasite |
| `room.host_changed` | Oda | Yeni oda sahibi |
| `game.started` | Oda | Senaryo, katılımcılar, açıklama sırası |
| `game.your_cards` | Özel | Oyuncunun 5 kartı |
| `game.phase_changed` | Oda | Aşama, tur, deadline, sıradaki oyuncu |
| `game.card_revealed` | Oda | Oyuncu, kategori, kart metni |
| `game.discussion_ready` | Oda | "Hazırım" sayacı |
| `game.vote_progress` | Oda | x/y oy (hedef yok) |
| `game.tiebreak_started` | Oda | Eşit adaylar |
| `game.player_eliminated` | Oda | Elenen, yöntem, oy sayıları, tüm kartları |
| `game.player_connection` | Oda | Koptu / döndü / ayrıldı |
| `game.finished` | Oda | Kazananlar, herkesin tüm kartları |
| `room.returned_to_lobby` | Oda | Tekrar oyna |
| `chat.message` | Oda (engelleme filtresi) | Gönderen, metin, zaman |

Nihai payload şemaları ARCH-004'te ([api.md](api.md)) kesinleşir.

**Gizlilik kuralı:** Genel olaylarda açılmamış kart asla bulunmaz; özel olaylar yalnızca ilgili oyuncunun bağlantılarına gider (ENG-015, SAFE-004 ile test edilir).
