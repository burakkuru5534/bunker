# API Sözleşmeleri — v1

> Kaynak: ARCH-004 (2026-11-02). İlgili: [architecture.md §10](architecture.md), [ADR-0007](decisions/0007-rest-commands-ws-push.md), [security.md](security.md).
> Komutlar REST ile gönderilir; WebSocket yalnızca sunucu → istemci olay taşır.

## 1. Genel kurallar

| Konu | Kural |
|---|---|
| Taban URL | `https://<cloud-run-host>`; tüm uç noktalar `/v1` altında (health hariç) |
| Biçim | JSON, UTF-8. İstek gövdesi ≤ 4 KB |
| Kimlik | `Authorization: Bearer <token>` (oturum oluşturma ve health hariç tümünde zorunlu) |
| Uygulama sürümü | `X-App-Version: 1.0.0` başlığı. Desteklenmeyen sürüm → `426 UPGRADE_REQUIRED` |
| Kimlikler | UUID dizgesi |
| Zaman | RFC 3339, UTC (`2026-11-02T18:00:00Z`) |
| İdempotency | Oyun eylemlerinde `action_id`, sohbette `client_msg_id` (UUID, istemci üretir). Aynı değerle tekrar → ilk sonuç döner |
| Sürüm uyumluluğu | v1 içinde yalnızca **ekleme** yapılır (yeni alan/olay). İstemci bilinmeyen alan ve olayları yok sayar. Kırıcı değişiklik → `/v2` |
| Sıklık sınırı | Aşıldığında `429 RATE_LIMITED` + `Retry-After` başlığı |

## 2. Hata biçimi

```json
{
  "error": {
    "code": "ROOM_FULL",
    "message": "Oda dolu.",
    "request_id": "01JC…",
    "details": {}
  }
}
```

- `code` makine için sabittir; istemci mesajı `code`'a göre kendi l10n metninden gösterir. `message` yalnızca hata ayıklama içindir.
- `request_id` loglarla eşleştirme içindir.

### Hata kodları

| HTTP | code | Anlamı |
|---|---|---|
| 400 | `VALIDATION_FAILED` | Gövde/parametre geçersiz (`details.fields`) |
| 400 | `NICKNAME_INVALID` | Takma ad kurallara uymuyor |
| 400 | `MESSAGE_INVALID` | Mesaj boş, çok uzun veya filtreden geçemedi |
| 401 | `UNAUTHORIZED` | Token yok, geçersiz, süresi dolmuş veya iptal edilmiş |
| 403 | `FORBIDDEN` | İşlem için yetki yok (ör. oda sahibi değil) |
| 403 | `NOT_A_MEMBER` | Oda/oyun üyesi değil |
| 403 | `KICKED` | Bu odadan atıldı, tekrar katılamaz |
| 404 | `NOT_FOUND` | Kaynak yok (veya üyelik yok — varlık sızdırılmaz) |
| 404 | `INVITE_CODE_NOT_FOUND` | Davet kodu geçersiz veya oda kapalı |
| 409 | `ROOM_FULL` | 10 oyuncu dolu |
| 409 | `GAME_IN_PROGRESS` | Oyun sürüyor; daha önce katılımcı olmayan kullanıcı katılamaz |
| 409 | `NICKNAME_TAKEN_IN_ROOM` | Odada aynı takma ad var |
| 409 | `ALREADY_IN_ROOM` | Kullanıcı başka bir aktif odada |
| 409 | `INVALID_PHASE` | Eylem bu aşamada yapılamaz |
| 409 | `NOT_YOUR_TURN` | Kart açma sırası başkasında |
| 409 | `CARD_ALREADY_REVEALED` | Kart zaten açık |
| 409 | `CATEGORY_NOT_ALLOWED` | 1. turda yalnızca meslek açılabilir |
| 409 | `ALREADY_VOTED` | Bu turda oy verildi (oy değiştirilemez) |
| 409 | `SELF_VOTE` | Kendine oy verilemez |
| 409 | `INVALID_TARGET` | Hedef hayatta değil / aday değil |
| 409 | `NOT_ALIVE` | Elenmiş/ayrılmış oyuncu bu eylemi yapamaz |
| 409 | `NOT_SPECTATOR` | Eşitlik oyu yalnızca izleyicilere açık |
| 409 | `START_CONDITIONS_NOT_MET` | 4–10 oyuncu ve herkes hazır değil |
| 409 | `STALE_STATE` | Beklenen `state_version` uyuşmuyor (yalnızca sistem içi) |
| 426 | `UPGRADE_REQUIRED` | Uygulama sürümü çok eski |
| 429 | `RATE_LIMITED` | Sıklık sınırı |
| 500 | `INTERNAL` | Beklenmeyen hata (ayrıntı loglarda) |

## 3. Ortak doğrulama kuralları

| Alan | Kural |
|---|---|
| `nickname` | Baştaki/sondaki boşluk kırpılır; 3–16 karakter; harf (Türkçe dahil), rakam, boşluk, `_`; ardışık boşluk yok; küfür filtresi |
| `invite_code` | 6 karakter, alfabe `23456789ABCDEFGHJKMNPQRSTUVWXYZ` (31 karakter; 0/O, 1/I/L yok). Büyük/küçük harf duyarsız |
| `capacity` | 1 veya 2 |
| `text` (sohbet) | Kırpılır; 1–200 karakter; küfürler `***` ile maskelenir |
| `category` | `profession`, `health`, `hobby`, `item`, `secret` |

---

## 4. REST uç noktaları

### Sağlık

| Yöntem | Yol | Yetki | Yanıt |
|---|---|---|---|
| GET | `/health` | Yok | `200 {"status":"ok"}` — süreç ayakta (DB'ye bakmaz) |
| GET | `/health/ready` | Yok | `200 {"status":"ready"}` / `503` — DB erişilebilir mi |

### Oturum ve kullanıcı

**POST `/v1/sessions/guest`** — Yetki: yok. Sınır: IP başına 5/dk.
```json
// İstek
{ "nickname": "Burak" }
// 201
{ "token": "…", "expires_at": "2027-01-31T18:00:00Z",
  "user": { "id": "uuid", "nickname": "Burak" } }
```
Hatalar: `VALIDATION_FAILED`, `NICKNAME_INVALID`, `RATE_LIMITED`.

**POST `/v1/sessions/refresh`** — Mevcut token'ı iptal eder, yenisini verir.
```json
// 200
{ "token": "…", "expires_at": "…" }
```

**GET `/v1/me`** → `200 { "id", "nickname", "active_room_id": "uuid|null" }`
Uygulama açılışında aktif odaya geri dönmek için kullanılır.

**PATCH `/v1/me`** — `{ "nickname": "…" }` → `200 { "id", "nickname" }`. Oyun sürerken (`in_game` odada) değiştirilemez → `INVALID_PHASE`.

**DELETE `/v1/me`** — Hesabı ve oturumları siler; oyun geçmişinde kullanıcı anonimleştirilir ("Silinmiş oyuncu"). Aktif odadan ayrılma uygulanır. → `204`. (Google Play hesap silme gerekliliği; SAFE-009.)

### Oda

**Oda görünümü (RoomView)** — birçok yanıtta ortak:
```json
{
  "id": "uuid",
  "invite_code": "K7M2QX",
  "status": "lobby | in_game | finished",
  "host_user_id": "uuid",
  "capacity": 2,
  "current_game_id": "uuid | null",
  "members": [
    { "user_id": "uuid", "nickname": "Burak", "role": "player | spectator",
      "is_ready": false, "connected": true }
  ]
}
```

| Yöntem | Yol | Gövde | Yetki / koşul | Başarı | Önemli hatalar |
|---|---|---|---|---|---|
| POST | `/v1/rooms` | `{ "capacity": 2 }` (ops., varsayılan 2) | Kullanıcı başka aktif odada değil | `201 RoomView` (oluşturan = oda sahibi, oyuncu) | `ALREADY_IN_ROOM` |
| POST | `/v1/rooms/join` | `{ "invite_code": "K7M2QX" }` | Sınır 10/dk | `200 RoomView` | `INVITE_CODE_NOT_FOUND`, `ROOM_FULL`, `GAME_IN_PROGRESS`, `KICKED`, `NICKNAME_TAKEN_IN_ROOM`, `ALREADY_IN_ROOM` |
| GET | `/v1/rooms/{roomId}` | — | Üye | `200 RoomView` | `NOT_FOUND` |
| POST | `/v1/rooms/{roomId}/leave` | — | Üye | `204` | — |
| POST | `/v1/rooms/{roomId}/ready` | `{ "ready": true }` | Oyuncu, `lobby` | `204` | `INVALID_PHASE` |
| PATCH | `/v1/rooms/{roomId}/settings` | `{ "capacity": 1 }` | Oda sahibi, `lobby` | `200 RoomView` | `FORBIDDEN`, `INVALID_PHASE` |
| POST | `/v1/rooms/{roomId}/kick` | `{ "user_id": "uuid" }` | Oda sahibi, `lobby` | `204` | `FORBIDDEN`, `INVALID_PHASE` |
| POST | `/v1/rooms/{roomId}/start` | — | Oda sahibi, `lobby`, 4–10 oyuncu, herkes hazır | `201 { "game_id" }` | `START_CONDITIONS_NOT_MET` |
| POST | `/v1/rooms/{roomId}/rematch` | — | Oda sahibi, `finished` | `200 RoomView` (`lobby`, hazırlar sıfırlanır) | `FORBIDDEN`, `INVALID_PHASE` |
| POST | `/v1/rooms/{roomId}/messages` | `{ "client_msg_id": "uuid", "text": "…" }` | Lobide herkes; oyunda yalnızca hayattaki oyuncular | `201 { "id", "created_at" }` | `MESSAGE_INVALID`, `RATE_LIMITED`, `FORBIDDEN` |

**Katılma kuralları:**
- `lobby` ve yer var → oyuncu olarak katılır.
- `in_game` ve kullanıcı bu oyunun eski katılımcısı (ayrılmış) → **izleyici** olarak döner.
- `in_game` ve kullanıcı katılımcı değil → `GAME_IN_PROGRESS`.
- Atılmış kullanıcı aynı odaya tekrar katılamaz → `KICKED`.
- Bir kullanıcı aynı anda yalnızca **bir aktif odada** olabilir.

**Ayrılma kuralları:** Lobide üyelik silinir. Oyunda katılımcı "ayrıldı" olur ve elenmiş sayılır. Oda sahibi ayrılırsa sahiplik en eski üyeye geçer. Son üye ayrılırsa oda kapanır.

### Oyun

**POST `/v1/games/{gameId}/actions`** — Tüm oyun eylemleri (oy dahil) tek uç noktadan gider.

> Not: Master prompt'taki ayrı `/votes` uç noktası bu uç noktaya birleştirildi; tüm eylemler aynı kilit, sürüm ve idempotency yolundan geçer.

```json
{ "action_id": "uuid", "type": "reveal_card", "payload": { "category": "health" } }
```

| `type` | `payload` | Kim / ne zaman | Hatalar |
|---|---|---|---|
| `reveal_card` | `{ "category": "…" }` | Sırası gelen hayattaki oyuncu, `REVEAL` | `NOT_YOUR_TURN`, `CARD_ALREADY_REVEALED`, `CATEGORY_NOT_ALLOWED` |
| `discussion_ready` | `{}` | Hayattaki oyuncu, `DISCUSSION` | `INVALID_PHASE`, `NOT_ALIVE` |
| `vote` | `{ "target_user_id": "uuid" }` | Hayattaki oyuncu, `VOTING` | `ALREADY_VOTED`, `SELF_VOTE`, `INVALID_TARGET` |
| `tiebreak_vote` | `{ "target_user_id": "uuid" }` | İzleyici, `TIE_BREAK`, hedef eşit adaylardan | `NOT_SPECTATOR`, `ALREADY_VOTED`, `INVALID_TARGET` |

```json
// 200
{ "state_version": 43, "duplicate": false }
```
`duplicate: true` → aynı `action_id` daha önce işlenmiş; durum değişmedi. Ekran WS olaylarından güncellenir.

**GET `/v1/games/{gameId}/result`** — Yalnızca `FINISHED` oyunlar; yetki: oyunun katılımcısı.
```json
{
  "game_id": "uuid", "finished_at": "…", "end_reason": "capacity_reached | abandoned",
  "scenario": { "id": "nuclear_winter", "title": "Nükleer Kış" },
  "capacity": 2,
  "players": [
    { "user_id": "uuid", "nickname": "Burak", "is_winner": true,
      "status": "alive | eliminated | left", "eliminated_round": null,
      "cards": [ { "category": "profession", "text": "Cerrah" } ] }
  ],
  "rounds": [
    { "round": 1, "eliminated_user_id": "uuid",
      "method": "vote | spectators | lottery | left",
      "tally": [ { "user_id": "uuid", "votes": 4 } ] }
  ]
}
```

### Moderasyon

| Yöntem | Yol | Gövde | Başarı | Not |
|---|---|---|---|---|
| POST | `/v1/reports` | `{ "reported_user_id", "room_id", "reason": "harassment\|hate\|spam\|cheating\|other", "note": "≤200" }` | `201` | Ortak odada olunmalı; odanın son 50 mesajı rapora eklenir |
| GET | `/v1/blocks` | — | `200 [ { "user_id", "nickname", "created_at" } ]` | |
| POST | `/v1/blocks` | `{ "user_id": "uuid" }` | `204` | Kendini engelleyemez |
| DELETE | `/v1/blocks/{userId}` | — | `204` | |

---

## 5. WebSocket

### Bağlantı

```
GET /v1/ws?room_id=<uuid>
Authorization: Bearer <token>
X-App-Version: 1.0.0
```

- Kimlik ve oda üyeliği doğrulanır; kullanıcı başına en fazla 2 eşzamanlı bağlantı.
- Bağlanınca ilk mesaj her zaman `session.snapshot`.
- Sunucu 25 sn'de bir WS ping gönderir; 60 sn yanıt yoksa bağlantı kapanır.
- Cloud Run istek zaman aşımı (≤60 dk) dolunca bağlantı kapanır; istemci hemen yeniden bağlanır.

### Kapanış kodları

| Kod | Anlamı | İstemci davranışı |
|---|---|---|
| 1000 | Normal | — |
| 1001 / 1006 / 1012 | Sunucu yeniden başladı / ağ koptu / deploy | Üstel geri çekilmeyle yeniden bağlan (0,5 sn → en fazla 10 sn) |
| 4001 | Yetkisiz | Token'ı yenile; olmazsa yeni oturum |
| 4003 | Oda üyesi değil / atıldı | Ana ekrana dön |
| 4008 | Çok fazla bağlantı | En eski bağlantı kapatılır |
| 4009 | Yavaş istemci (kuyruk doldu) | Yeniden bağlan |
| 4026 | Sürüm eski | Güncelleme ekranı |

### Zarf

```json
{ "v": 1, "type": "game.card_revealed", "room_id": "uuid",
  "state_version": 42, "ts": "2026-11-02T18:00:00Z", "payload": { } }
```

- `state_version` oyun durumunu değiştiren olaylarda bulunur. İstemci, bildiği sürümden **1 fazlası dışında** bir sürüm görürse `sync` gönderir.
- Oyun sürmüyorsa (lobi) `state_version` oda sürümüdür.

### İstemci → sunucu

```json
{ "v": 1, "type": "sync" }
```
Yanıt: yeni `session.snapshot`. Başka mesaj kabul edilmez (gelirse yok sayılır).

### Sunucu → istemci olayları

**`session.snapshot`** (özel)
```json
{
  "server_time": "…",
  "me": { "user_id": "uuid", "role": "player | spectator" },
  "room": { /* RoomView */ },
  "game": null,
  "chat": [ { "id": 1, "user_id": "uuid", "nickname": "Ayşe", "text": "…", "created_at": "…" } ]
}
```
`game` (oyun sürüyorsa):
```json
{
  "id": "uuid", "phase": "REVEAL", "round": 2, "total_rounds": 8,
  "phase_deadline_at": "…",
  "scenario": { "id": "…", "title": "…", "disaster": "…", "duration": "…",
                "bunker": ["…","…","…"], "outside": "…" },
  "turn_user_id": "uuid | null",
  "participants": [
    { "user_id": "uuid", "nickname": "…", "seat": 0,
      "status": "alive | eliminated | left", "connected": true,
      "revealed_cards": [ { "category": "profession", "text": "Cerrah" } ] }
  ],
  "my_cards": [ { "category": "profession", "text": "Cerrah", "revealed": true } ],
  "my_action_state": { "voted": false, "discussion_ready": false, "tiebreak_voted": false },
  "discussion_ready_count": 3,
  "vote_progress": { "cast": 2, "total": 7 },
  "tiebreak": { "candidates": ["uuid","uuid"], "progress": { "cast": 1, "total": 2 } },
  "last_round_result": { /* game.player_eliminated payload */ }
}
```
- `my_cards` yalnızca katılımcıya, kendi kartlarıdır; izleyicide yalnızca kendi (elenmiş) kartları.
- Kalan süre istemcide `phase_deadline_at − server_time` farkıyla hesaplanır (saat kayması etkilenmez).

**Oda olayları** (odadaki herkese)

| type | payload |
|---|---|
| `room.member_joined` | `{ "member": { user_id, nickname, role, is_ready, connected } }` |
| `room.member_left` | `{ "user_id" }` |
| `room.member_kicked` | `{ "user_id" }` (atılan kullanıcıya ayrıca 4003 ile kapanış) |
| `room.ready_changed` | `{ "user_id", "is_ready" }` |
| `room.settings_changed` | `{ "capacity" }` |
| `room.host_changed` | `{ "host_user_id" }` |
| `room.returned_to_lobby` | `{ "room": RoomView }` |
| `chat.message` | `{ "id", "user_id", "nickname", "text", "created_at" }` — engelleyen alıcılara gönderilmez |

**Oyun olayları**

| type | Kime | payload |
|---|---|---|
| `game.started` | Oda | `{ "game_id", "scenario", "capacity", "total_rounds", "participants": [{user_id,nickname,seat}], "reveal_order": ["uuid"] }` |
| `game.your_cards` | **Özel** | `{ "cards": [ { "category", "text", "revealed": false } ] }` |
| `game.phase_changed` | Oda | `{ "phase", "round", "phase_deadline_at", "server_time", "turn_user_id" }` |
| `game.card_revealed` | Oda | `{ "user_id", "category", "text", "auto": false }` (`auto`: süre dolunca sunucu açtı) |
| `game.discussion_ready` | Oda | `{ "count", "total" }` |
| `game.vote_progress` | Oda | `{ "cast", "total" }` — kimin kime verdiği yok |
| `game.tiebreak_started` | Oda | `{ "candidates": ["uuid"], "phase_deadline_at" }` |
| `game.tiebreak_progress` | Oda | `{ "cast", "total" }` |
| `game.player_eliminated` | Oda | `{ "round", "user_id", "method": "vote\|spectators\|lottery\|left", "tally": [{user_id,votes}], "lottery_candidates": ["uuid"], "cards": [ {category,text} ] }` |
| `game.player_connection` | Oda | `{ "user_id", "state": "disconnected\|reconnected\|left" }` |
| `game.finished` | Oda | `GET /result` yanıtıyla aynı yapı |

### Gizlilik kuralları (test edilir: ENG-015, SAFE-004)

- Oda olaylarında hayattaki oyuncunun **açılmamış** kartı asla bulunmaz.
- `game.your_cards` ve `session.snapshot.game.my_cards` yalnızca ilgili kullanıcının bağlantılarına gider.
- `votes` hedefleri hiçbir olayda veya yanıtta yer almaz; yalnızca `tally` (oy sayıları) gösterilir.
