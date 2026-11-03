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

- Tablo backend'de `internal/apperr` (kodlar) ve `internal/httpapi` (HTTP durumu) ile birebir aynıdır; mobilde her kodun l10n metni vardır. İki taraftaki testler bu tabloyu okur — kod eklerken önce buraya ekleyin.
- `500 INTERNAL` yanıtında `message` her zaman genel metindir ve `details` boştur; asıl hata yalnızca loglanır.
- `429 RATE_LIMITED` yanıtında `Retry-After` (saniye, ≥ 1) başlığı bulunur.
- Koda özgü ek bilgiler `details` içindedir (ör. `UPGRADE_REQUIRED` → `details.min_version`).

### Alan hataları (`details.fields`)

`VALIDATION_FAILED` yanıtında `details.fields` alan adı → alan hatası kodu eşlemesidir. Alan adı JSON yoludur (iç içe alanlar noktalı: `settings.capacity`); gövdenin tamamı için `body`, başlıklar için başlık adı kullanılır. Her alan için yalnızca ilk hata döner.

```json
{ "error": { "code": "VALIDATION_FAILED", "message": "doğrulama başarısız", "request_id": "…",
  "details": { "fields": { "nickname": "too_short", "capacity": "not_allowed" } } } }
```

| Alan hatası | Anlamı |
|---|---|
| `required` | Eksik (boş gövde dahil) |
| `invalid` | Biçim veya değer geçersiz (bozuk JSON, birden fazla JSON değeri, yanlış `Content-Type` dahil) |
| `invalid_type` | JSON türü yanlış (ör. sayı yerine metin) |
| `too_short` / `too_long` | Uzunluk sınırı dışında |
| `too_large` | Gövde 4 KB sınırını aşıyor (`details.max_bytes`) |
| `unknown_field` | Beklenmeyen alan (sunucu bilinmeyen alanları reddeder) |
| `not_allowed` | İzin verilen değerlerden biri değil (ör. `capacity: 3`) |
| `profanity` | Küfür filtresine takıldı |
| `invalid_chars` | İzin verilmeyen karakter |

İstemci bilinmeyen bir alan hatası kodunu genel "geçersiz" metnine düşürür.

## 3. Ortak doğrulama kuralları

| Alan | Kural |
|---|---|
| `nickname` | Unicode NFC'ye çevrilir, baştaki/sondaki boşluk kırpılır; 3–16 karakter; **Latin alfabesi harfleri (Türkçe ç ğ ı İ ö ş ü dahil)**, 0–9, boşluk, `_`; en az bir harf; ardışık boşluk yok; ayrılmış adlar ("Silinmiş oyuncu", "Sistem", "Admin"…) yok; küfür filtresi. Hata: `NICKNAME_INVALID` + `details.fields.nickname` ∈ `required`, `too_short`, `too_long`, `invalid_chars`, `invalid` (harf yok / ardışık boşluk), `not_allowed`, `profanity`. Oda içi tekillik Türkçe büyük/küçük harf duyarsızdır ("IŞIK" = "ışık") |
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

- `X-App-Version` gerekmez; `HEAD` de desteklenir; yanıtlar `Cache-Control: no-store`.
- `/health/ready` başarısızsa `503 {"status":"not_ready"}` döner; hangi denetimin başarısız olduğu yanıtta yer almaz, yalnızca loglanır. Denetimler paralel çalışır, her biri en çok 2 sn.
- `/health` Cloud Run başlangıç/canlılık denetimidir; `/health/ready` probe olarak kullanılmaz (DB kesintisi konteyneri yeniden başlatmamalı).

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

- Token içeren yanıtlar `Cache-Control: no-store` taşır. Token loglanmaz.
- Sınır, istek gövdesi doğrulanmadan önce uygulanır (geçersiz denemeler de sayılır). IPv6'da sınır `/64` ağı başınadır.
- Yetki gerektiren her uç nokta geçersiz/eksik token'da `401 UNAUTHORIZED` + `WWW-Authenticate: Bearer realm="siginak"` döner; neden (süresi dolmuş, iptal, bilinmeyen) ayrıştırılmaz.

**POST `/v1/sessions/refresh`** — Yetki: Bearer. Gövde boş veya `{}`. Mevcut token'ı iptal eder, yenisini verir; aynı token'la eşzamanlı ikinci yenileme `401` alır (istemci yeni token'ı kullanmalı).
```json
// 200
{ "token": "…", "expires_at": "…" }
```

**GET `/v1/me`** → `200 { "id", "nickname", "active_room_id": "uuid|null" }`
Uygulama açılışında aktif odaya geri dönmek için kullanılır.

**PATCH `/v1/me`** — `{ "nickname": "…" }` → `200 { "id", "nickname" }`. Oyun sürerken (`in_game` odada) değiştirilemez → `INVALID_PHASE`.

**DELETE `/v1/me`** — Hesabı ve oturumları siler; oyun geçmişinde kullanıcı anonimleştirilir ("Silinmiş oyuncu"). Aktif odadan ayrılma uygulanır. → `204`. (Google Play hesap silme gerekliliği; SAFE-009.)

### Oda

**Oda görünümü (RoomView)** — birçok yanıtta ortak. `members` yalnızca aktif üyeleri katılma sırasıyla içerir; `version` lobi olaylarının sürümüdür (WS `state_version` ile aynı sayaç). `connected`, gerçek zamanlı katman (RT) gelene kadar `false`'tur:
```json
{
  "id": "uuid",
  "invite_code": "K7M2QX",
  "status": "lobby | in_game | finished",
  "host_user_id": "uuid",
  "capacity": 2,
  "current_game_id": "uuid | null",
  "version": 0,
  "members": [
    { "user_id": "uuid", "nickname": "Burak", "role": "player | spectator",
      "is_ready": false, "connected": true }
  ]
}
```

| Yöntem | Yol | Gövde | Yetki / koşul | Başarı | Önemli hatalar |
|---|---|---|---|---|---|
| POST | `/v1/rooms` | `{ "capacity": 2 }` (ops., varsayılan 2) | Kullanıcı başka aktif odada değil | `201 RoomView` (oluşturan = oda sahibi, oyuncu) | `ALREADY_IN_ROOM` (`details.room_id` = mevcut aktif oda), `VALIDATION_FAILED` (`fields.capacity = not_allowed`) |
| POST | `/v1/rooms/join` | `{ "invite_code": "K7M2QX" }` | Sınır 10/dk | `200 RoomView` | `INVITE_CODE_NOT_FOUND`, `ROOM_FULL`, `GAME_IN_PROGRESS`, `KICKED`, `NICKNAME_TAKEN_IN_ROOM`, `ALREADY_IN_ROOM` |
| GET | `/v1/rooms/{roomId}` | — | Aktif üye | `200 RoomView` | `NOT_FOUND` (üye olmayan, olmayan oda ve bozuk kimlik aynı yanıtı alır; varlık sızdırılmaz) |
| POST | `/v1/rooms/{roomId}/leave` | — | Üye | `204` | — (**idempotent**: üye olmayan, zaten ayrılmış veya olmayan oda da `204`, değişiklik yok) |
| POST | `/v1/rooms/{roomId}/ready` | `{ "ready": true }` | Oyuncu, `lobby` | `204` (aynı değer: değişiklik/olay yok) | `INVALID_PHASE`, `FORBIDDEN` (izleyici) |
| PATCH | `/v1/rooms/{roomId}/settings` | `{ "capacity": 1 }` | Oda sahibi, `lobby` | `200 RoomView` (aynı değer: değişiklik/olay yok) | `FORBIDDEN`, `INVALID_PHASE`, `VALIDATION_FAILED` |
| POST | `/v1/rooms/{roomId}/kick` | `{ "user_id": "uuid" }` | Oda sahibi, `lobby` | `204` | `FORBIDDEN`, `INVALID_PHASE`, `NOT_FOUND` (hedef aktif üye değil), `VALIDATION_FAILED` (kendini / bozuk kimlik) |
| POST | `/v1/rooms/{roomId}/start` | — | Oda sahibi, `lobby`, 4–10 oyuncu, herkes hazır | `201 { "game_id" }` | `START_CONDITIONS_NOT_MET` |
| POST | `/v1/rooms/{roomId}/rematch` | — | Oda sahibi, `finished` | `200 RoomView` (`lobby`, hazırlar sıfırlanır) | `FORBIDDEN`, `INVALID_PHASE` |
| POST | `/v1/rooms/{roomId}/messages` | `{ "client_msg_id": "uuid", "text": "…" }` | Lobide herkes; oyunda yalnızca hayattaki oyuncular | `201 { "id", "created_at" }` | `MESSAGE_INVALID`, `RATE_LIMITED`, `FORBIDDEN` |

**Katılma kuralları:**
- `lobby` ve yer var → oyuncu olarak katılır.
- `in_game` ve kullanıcı bu oyunun eski katılımcısı (ayrılmış) → **izleyici** olarak döner.
- `in_game` ve kullanıcı katılımcı değil → `GAME_IN_PROGRESS`.
- Atılmış kullanıcı aynı odaya tekrar katılamaz → `KICKED`.
- Bir kullanıcı aynı anda yalnızca **bir aktif odada** olabilir.
- `finished` odaya katılan oyuncu olarak girer (oda sahibi "Tekrar oyna" deyince lobide olur); 10 oyuncu sınırı geçerlidir.
- Zaten aktif üye olan kullanıcının tekrar katılması değişiklik yapmaz, `200 RoomView` döner.
- Ayrılmış (atılmamış) kullanıcı lobiye yeniden katılabilir.
- Davet kodu girişi büyük/küçük harf duyarsızdır; boşluk ve tire yok sayılır. Biçimi bozuk kod `400 VALIDATION_FAILED` (`fields.invite_code = invalid`), geçerli biçimde ama açık odası olmayan kod `404 INVITE_CODE_NOT_FOUND`.

**Ayrılma kuralları:** Lobide (ve sonuç ekranında) üyelik silinir. Oyunda üyelik `left_at` ile kapanır, katılımcı "ayrıldı" olur ve elenmiş sayılır. Oda sahibi ayrılırsa sahiplik **en eski aktif oyuncuya**, oyuncu yoksa en eski üyeye geçer (`room.member_left` + `room.host_changed`, aynı sürüm). Son üye ayrılırsa oda kapanır (olay yok; kodu yeniden kullanılabilir).

**Başlatma:** Oda sahibi, `lobby`, 4–10 oyuncu ve **oda sahibi dahil** tüm oyuncular hazır (izleyiciler sayılmaz). Değilse `409 START_CONDITIONS_NOT_MET` + `details.reason` ∈ `too_few_players`, `not_all_ready` (+ `details.players`); oda sahibi değilse `403 FORBIDDEN`, oyun zaten başladıysa `409 INVALID_PHASE` (eşzamanlı başlatmalardan yalnızca biri başarılı olur). Başarılı başlatma tek transaction’dır (koltuk, senaryo ve kartlar sunucuda rastgele; oda `in_game`, oda sürümü +1); commit sonrası sırasıyla `game.started` (oda), her oyuncuya `game.your_cards` (özel) ve `game.phase_changed` (`SCENARIO_INTRO`) yayınlanır, hepsi oyunun `state_version`’ı (1) ile. Kart kataloğu yetersizse `500 INTERNAL` (hiçbir şey yazılmaz). **Tekrar oyna:** tüm aktif üyeler (izleyiciler dahil) oyuncu olur, hazırlar sıfırlanır, `current_game_id` boşalır; yanıt ve `room.returned_to_lobby` aynı `RoomView`’ı (yeni sürümle) taşır. Biten oyunun sonucu okunmaya devam eder. Bitmiş odaya katılan yeni oyuncularla 10’u aşılırsa başlatma `too_many_players` ile reddedilir.

**Atma:** Oda sahibi kendini atamaz (`VALIDATION_FAILED`); aktif olmayan hedef `NOT_FOUND`.

**Oda işlemlerinde üyelik:** `{roomId}` içeren işlemlerde (görüntüleme, hazır, ayarlar, atma) çağıran odanın aktif üyesi değilse — oda yoksa veya kimlik bozuksa da — yanıt `404 NOT_FOUND`'dır; odanın varlığı sızdırılmaz. Üyeler arasında kural hataları (`FORBIDDEN`, `INVALID_PHASE`) olduğu gibi döner. Her değişiklik oda `version`'ını bir artırır ve commit sonrası ilgili `room.*` olayı yayınlanır.

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

- `action_id` oyuncu başına tekildir (iki oyuncu aynı kimliği kullanabilir). Yalnızca **başarılı** eylemler kaydedilir; reddedilen eylem (ör. `NOT_YOUR_TURN`) aynı `action_id` ile yeniden denenebilir.
- Çağıran oyunun katılımcısı değilse — oyun yoksa veya kimlik bozuksa da — `404 NOT_FOUND` (oyunun varlığı sızdırılmaz).
- Gövde katıdır: `payload` nesne olmalı, bilinmeyen alan reddedilir (`VALIDATION_FAILED`, `details.fields`: `payload.category` vb.). Henüz desteklenmeyen türler (`discussion_ready`, `vote`, `tiebreak_vote`; ENG-006/008/009) `type: not_allowed` döner.
- `game.card_revealed` ve `game.player_eliminated` (`cards`) kart metnini taşır; kart kimliği ve denge etiketi istemciye gitmez.
- `discussion_ready`: aynı turda ikinci kez gelirse başarıdır ama etkisizdir (sürüm artmaz, olay yok). Hayattaki herkes hazır olunca `VOTING` hemen başlar.
- `vote`: hedef hayattaki başka bir katılımcı olmalı (değilse `INVALID_TARGET`; biçimi bozuksa `VALIDATION_FAILED`). Hedef yalnızca `votes` tablosunda tutulur; eylem kaydına ve olaylara yazılmaz. Hayattaki herkes oy verince sayım hemen yapılır.
- `tiebreak_vote`: yalnızca izleyiciler (elenen/ayrılan katılımcılar; hayattaki oyuncu `NOT_SPECTATOR`), yalnızca eşit oy alan ve hâlâ hayattaki adaylardan birine (`INVALID_TARGET`), bir kez (`ALREADY_VOTED`). Tüm izleyiciler oy verince sonuç hemen: hepsi aynı aday → o elenir (`method: spectators`), farklıysa kura. Hedef yine yalnızca `votes` tablosunda.
- Aday listesi (`candidates`, `lottery_candidates`) koltuk sırasındadır.

**Oyun sırasında ayrılma** (`POST /v1/rooms/{id}/leave`, oyun sürerken): hayattaki oyuncu oyunda `left` olur, elenmiş sayılır ve tüm kartları açılır; odaya `room.member_left` (+ gerekirse `room.host_changed`) ardından `game.player_eliminated` (`method: "left"`, `tally`/`lottery_candidates` boş, kartlar metinle) gider. Tur sonucu yazılmaz; turun elemesi yine oylamayla yapılır. Hayatta kalan kapasiteye inerse oyun hemen biter (`game.finished`); sıradaki kart açıcı ayrılırsa sıra geçer; kalanların hepsi hazırsa/oy verdiyse aşama erken biter; eşitlikte tek aday kalırsa sonuç hemen belirlenir. İzleyicinin veya bitmiş oyunda ayrılma oyunu değiştirmez. Ayrılan aynı odaya yalnızca izleyici olarak döner.

**GET `/v1/games/{gameId}/result`** — Yalnızca `FINISHED` oyunlar (sürerken `409 INVALID_PHASE`); yetki: oyunun katılımcısı (değilse, oyun yoksa veya kimlik bozuksa `404`). `game.finished` olayının payload’ı bu yanıtla birebir aynıdır. Oyuncular koltuk sırasında, kartlar kategori sırasında; takma ad güncel `users` kaydından (silinmiş hesap "Silinmiş oyuncu"). Oyun bitince oda `finished` olur (oda sürümü +1; ayrı oda olayı yok, `game.finished` bildirir).
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
- **Yetki, üyelik ve sürüm hataları HTTP durumu (401/403/426) olarak dönmez:** sunucu bağlantıyı yükseltir ve hemen ilgili kapanış koduyla (4001 / 4003 / 4026) kapatır; mobil istemci yükseltme öncesi HTTP durumunu güvenilir okuyamaz. Bu yüzden `/v1/ws` sürüm middleware'inden muaftır; denetimi WS işleyicisi yapar (`httpapi.CheckAppVersion`, `httpapi.CloseCodeFor`). Eksik/biçimsiz sürüm başlığı da 4026'dır; geçersiz `room_id` 4003'tür.
- Token yalnızca `Authorization` başlığından okunur; sorgu dizesindeki token yok sayılır.
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
