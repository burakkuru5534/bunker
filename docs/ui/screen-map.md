# Ekran Haritası — Sığınak: Son Kalanlar

> Kaynak: UI-002 (2026-11-02). İlgili: [brand.md](brand.md), [../api.md](../api.md), [../game-design.md](../game-design.md).

## 1. Temel ilke

Oyun aşamalarını **sunucu** yönetir. Bu yüzden oda ve oyunun tüm aşamaları tek bir rota (`/room/:roomId`) altında, **sunucudan gelen duruma göre** değişen görünümlerle gösterilir. Her aşama için ayrı rota açılmaz: aşama değişince ekran geçmişi (back stack) şişmez, yeniden bağlanmada doğru görünüm kendiliğinden seçilir.

## 2. Rotalar (go_router)

| Rota | Ekran | Not |
|---|---|---|
| `/` | Açılış (Splash) | Yönlendirme kararı verir; kullanıcıya ~1 sn'den fazla görünmez |
| `/welcome` | Hoş geldin + takma ad | İlk açılış; misafir hesap uyarısı |
| `/home` | Ana ekran | Oda oluştur / Koda katıl |
| `/join` | Koda katıl | 6 karakter kod girişi |
| `/room/:roomId` | Oda kabuğu (RoomShell) | Lobi, oyun, sonuç görünümleri buradan |
| `/settings` | Ayarlar | Takma ad, engellenenler, gizlilik, hesabı sil |
| `/settings/blocks` | Engellenenler | Listele / engeli kaldır |
| `/update` | Güncelleme gerekli | `426` veya WS `4026` sonrası |

### Yönlendirme kuralları (redirect)

```
Uygulama açıldı → /
  token yok                          → /welcome
  token var → GET /v1/me
     401                             → token sil → /welcome
     426                             → /update
     active_room_id var              → /room/:id   (oyuna geri dönüş)
     yok                             → /home
```

- `/room/:id` dışındaki rotalara giderken kullanıcı aktif bir odadaysa önce "Odadan ayrıl" onayı istenir.
- `/update` açıkken diğer tüm rotalar kilitlidir.

## 3. Ekranlar

### 3.1 Hoş geldin (`/welcome`)
- Başlık: "Sığınağa kim girecek?" + kısa oyun açıklaması (3 madde).
- Takma ad alanı (3–16 karakter, anlık doğrulama).
- Bilgi notu: "Hesabın bu cihaza bağlı. Uygulamayı silersen geri getirilemez."
- Gizlilik politikası ve kullanım koşulları bağlantısı.
- Düğme: **Başla** → `POST /v1/sessions/guest` → `/home`.

### 3.2 Ana ekran (`/home`)
- Üstte takma ad + ayarlar ikonu.
- Büyük düğme: **Oda oluştur** → `POST /v1/rooms` → `/room/:id`.
- İkincil düğme: **Koda katıl** → `/join`.
- "Nasıl oynanır?" kısa açılır bölüm.

### 3.3 Koda katıl (`/join`)
- 6 kutulu kod girişi (otomatik büyük harf, yapıştırma desteği, belirsiz karakterler kabul edilmez).
- **Katıl** → `POST /v1/rooms/join` → `/room/:id`.
- Hata kodlarına göre satır içi mesaj (`INVITE_CODE_NOT_FOUND`, `ROOM_FULL`, `GAME_IN_PROGRESS`, `KICKED`, `NICKNAME_TAKEN_IN_ROOM` → "Takma adını değiştir" kısayolu).

### 3.4 Oda kabuğu (`/room/:roomId`)

WebSocket bağlantısını ve `session.snapshot` durumunu yönetir; içerik `room.status` ve `game.phase`'e göre seçilir:

| Durum | Görünüm |
|---|---|
| `room.status = lobby` | **Lobi görünümü** |
| `in_game`, `SCENARIO_INTRO` | **Senaryo görünümü** |
| `in_game`, `REVEAL` / `DISCUSSION` / `VOTING` / `TIE_BREAK` / `ELIMINATION` | **Oyun görünümü** (aşamaya göre eylem alanı değişir) |
| `finished` | **Sonuç görünümü** |

Tüm görünümlerde ortak:
- **Bağlantı şeridi** (üstte): "Bağlantı koptu, yeniden bağlanıyoruz…" / "Bağlandı".
- **Sohbet**: alttan açılan panel; okunmamış mesaj rozeti. İzleyicide giriş alanı yerine "İzleyiciler yazamaz" notu.
- **Geri tuşu**: Lobide "Odadan ayrıl?" onayı; oyunda "Ayrılırsan elenmiş sayılırsın" uyarılı onay.

#### a) Lobi görünümü
- Davet kodu (büyük) + **Kopyala** düğmesi.
- Kapasite seçici (1 / 2) — yalnızca oda sahibi değiştirebilir, diğerleri görür.
- Oyuncu listesi (4–10): avatar, takma ad, hazır işareti, oda sahibi tacı, bağlantı durumu. Oda sahibinde her satırda "At" menüsü.
- **Hazırım** anahtarı.
- Oda sahibinde **Başlat** (4–10 oyuncu ve herkes hazırsa etkin; değilse nedeni yazılı).

#### b) Senaryo görünümü (30 sn; anlatımlı animasyon, bkz. [scenario-intro-assets.md](scenario-intro-assets.md))
- Senaryo başlığı, felaket metni, kalış süresi, sığınak özellikleri, dış dünya.
- Sayaç. Ardından oyun görünümüne geçer; senaryo, oyun görünümünde üstteki çipten tekrar açılabilir.

#### c) Oyun görünümü
```
┌─────────────────────────────────────────┐
│ [Senaryo çipi]   Tur 2/8   TARTIŞMA 01:12│  ← üst çubuk: aşama + sayaç
├─────────────────────────────────────────┤
│  Oyuncu ızgarası (2 sütun)              │
│  ┌────────┐ ┌────────┐                  │
│  │ Ayşe   │ │ Mehmet │  her kutu: avatar,│
│  │ 🔵 Cerrah│ │ 🔵 Aşçı │  durum, açılmış   │
│  │ 🩷 Astım │ │        │  kart çipleri     │
│  └────────┘ └────────┘                  │
├─────────────────────────────────────────┤
│  Eylem alanı (aşamaya göre)             │
├─────────────────────────────────────────┤
│ [Kartlarım]                  [Sohbet 3] │
└─────────────────────────────────────────┘
```
- Oyuncu kutusuna dokununca: açılmış kartlarının tamamı + **Raporla / Engelle**.
- **Kartlarım** (alt sayfa): kendi 5 kartım, açık/kapalı işaretli.

| Aşama | Eylem alanı |
|---|---|
| `REVEAL` — sıra bende | "Sıra sende. Bir kartını aç." + açılmamış kartlarım; 1. turda yalnızca meslek seçilebilir; seçince onay penceresi |
| `REVEAL` — sıra başkasında | "Ayşe kartını açıyor…" + sayaç |
| `DISCUSSION` | "İkna etme zamanı." + **Hazırım** düğmesi + "5/7 hazır" |
| `VOTING` | Oyuncu listesinden hedef seç (kendim seçilemez) → onay penceresi ("Oy değiştirilemez") → "Oyun verildi. 4/7" |
| `TIE_BREAK` — izleyiciysem | Eşit adaylardan seç → onay |
| `TIE_BREAK` — hayattaysam | "Eşitlik! İzleyiciler karar veriyor…" |
| `ELIMINATION` (8 sn) | Elenen oyuncu, yöntem (oy / izleyiciler / kura), oy sayıları, tüm kartları |
| İzleyiciysem (her aşama) | Üstte "İzleyicisin" şeridi; eylem alanı yalnızca bilgi |

- Elendiğim an: tam ekran "Kapı kapandı. Artık izleyicisin." (2 sn) → izleyici moduna geçer.

#### d) Sonuç görünümü
- "Sığınağa girenler" (kazananlar, turkuaz) ve "Dışarıda kalanlar".
- Her oyuncunun 5 kartı açık; tur tur eleme özeti (yöntem + oy sayıları).
- Oda sahibinde **Tekrar oyna** → `POST …/rematch` → lobi görünümü.
- Herkeste **Ana ekrana dön** (odadan ayrılır).

### 3.5 Ayarlar (`/settings`)
- Takma adı değiştir (oyun sürerken kapalı).
- Engellenenler → `/settings/blocks`.
- Gizlilik politikası, kullanım koşulları, destek (dış bağlantı).
- **Hesabımı sil** → iki adımlı onay → `DELETE /v1/me` → `/welcome`.
- Uygulama sürümü.

### 3.6 Güncelleme gerekli (`/update`)
- "Yeni bir sürüm var" + Google Play'e git düğmesi.

## 4. Diyaloglar ve geçici öğeler

| Öğe | Tetikleyen |
|---|---|
| Kart açma onayı | Kart seçildi ("Bu kart geri kapatılamaz") |
| Oy onayı | Hedef seçildi ("Oy değiştirilemez") |
| Odadan ayrıl onayı | Geri tuşu / Ayrıl |
| Oyuncu atma onayı | Oda sahibi "At" |
| Raporla | Oyuncu kutusu menüsü (neden + ops. not) |
| Engelle onayı | Oyuncu kutusu menüsü |
| Atıldın bildirimi | WS `4003` / `room.member_kicked` (ben) → `/home` |
| Hata snackbar'ı | REST hata kodları (l10n metni) |

## 5. Akışlar

```
İlk açılış:   / → /welcome → (guest) → /home
Oda kurma:    /home → (POST rooms) → /room/:id [lobi]
Katılma:      /home → /join → (POST join) → /room/:id [lobi | izleyici]
Oyun:         [lobi] → Başlat → [senaryo] → [oyun: tur döngüsü] → [sonuç]
Tekrar:       [sonuç] → Tekrar oyna → [lobi]
Çıkış:        [sonuç] → Ana ekrana dön → /home
Geri dönüş:   uygulama kapandı/açıldı → / → GET /me (active_room) → /room/:id → snapshot
Arka plan:    uygulama arka plana → WS kapanabilir → öne gelince yeniden bağlan + sync
Atılma:       [lobi] → 4003 → /home + bildirim
```

## 6. Uygulama yaşam döngüsü

- Ön plana gelince: WS bağlı değilse hemen yeniden bağlan; bağlıysa `sync` gönder.
- Arka planda WS açık tutulmaya zorlanmaz (pil); 60 sn bekleme süresi (GAME-003) kullanıcıyı korur.
- Bildirim (push) yok (MVP dışı).

## 7. MVP dışı

Derin bağlantı (deep link) ile davet, paylaşım menüsü (yalnızca kopyala var), maç geçmişi ekranı, profil ekranı, açık tema.
