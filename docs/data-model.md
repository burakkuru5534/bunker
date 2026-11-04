# Veri Modeli — Sığınak: Son Kalanlar

> Kaynak: ARCH-002 (2026-11-02, kullanıcı onaylı taslak); **kesin şema DB-003a–d (2026-11-03)**, aşağıdaki "Kesinleşen tablolar" bölümünde ve `migrations/postgres` dosyalarında. Üstteki tablo genel bakıştır; çelişki olursa migration dosyaları geçerlidir.
> Tek kalıcı veri kaynağı: Supabase PostgreSQL.

## Sadeleştirmeler (master prompt listesine göre)

- `player_profiles` ayrı tablo değil → `users` içinde.
- `game_summaries` ayrı tablo değil → kalıcı sonuç `games` + `game_participants` + `round_results`'tan okunur (veri tekrarı yok).

## Tablolar

| Tablo | Amaç | Önemli alanlar / kısıtlar | Erişim | Saklama |
|---|---|---|---|---|
| `users` | Misafir kullanıcı | `id`, `nickname`, `created_at`, `last_seen_at`, `deleted_at` (silinince takma ad "Silinmiş oyuncu" olarak anonimleştirilir) | Sahibi; takma ad oda üyelerine görünür | Hesap silinene kadar (SAFE-009) |
| `sessions` | Oturumlar | `user_id`, `token_hash`, `expires_at`, `revoked_at` | Yalnızca sunucu | Süresi dolunca temizlenir (AUTH-001) |
| `rooms` | Oda | `invite_code` (aktif odalarda tekil), `host_user_id`, `capacity` ∈ {1,2}, `status` (lobby/in_game/finished/closed), `current_game_id`; kullanıcı aynı anda tek aktif odada | Oda üyeleri | Kapanınca temizlik (ROOM-010) |
| `room_members` | Üyelik | PK `(room_id, user_id)`; `role` (player/spectator), `is_ready`, `joined_at`, `left_at`, `kicked_at` (atılan tekrar katılamaz) | Oda üyeleri | Oda ile |
| `card_definitions` | Kart kataloğu | `category`, `text`, `balance` (+/0/−), `active` | Yalnızca sunucu (metin açılınca gönderilir) | Kalıcı; migration ile yüklenir |
| `scenarios` | Senaryolar | başlık, felaket, süre, sığınak, dış dünya | Herkese (oyun başlayınca) | Kalıcı |
| `games` | Oyun durumu | `room_id`, `scenario_id`, `status`, `phase`, `round`, `phase_deadline_at`, `state_version`, `turn_index`, `capacity`, `started_at`, `finished_at` | Katılımcılar | Kalıcı (sonuç) |
| `game_participants` | Katılımcılar | PK `(game_id, user_id)`; `seat`, `status` (alive/eliminated/left), `eliminated_round`, `disconnected_at`, `is_winner` | Katılımcılar | Kalıcı |
| `player_cards` | Dağıtılan kartlar | PK `(game_id, user_id, category)`; `card_id`, `revealed_round`; tekil `(game_id, card_id)` | Sahibi; açılınca herkes | Kalıcı |
| `votes` | Oylar | tekil `(game_id, round, kind, voter_id)`; `kind` (regular/tiebreak), `target_id` | **Yalnızca sunucu** (kimin kime oy verdiği gösterilmez) | Kalıcı (denetim) |
| `round_results` | Tur sonucu | PK `(game_id, round)`; `eliminated_id`, `method` (vote/spectators/lottery/left), `tally` (aday→oy sayısı), `lottery_candidates` | Katılımcılar (oy sayıları) | Kalıcı |
| `game_actions` | İdempotency + denetim | tekil `(game_id, actor_id, action_id)`; `type`, `payload`, `result`, `state_version` | Yalnızca sunucu | Kalıcı (gözden geçirilecek) |
| `chat_messages` | Sohbet | indeks `(room_id, created_at)`; `game_id` (ops.), `user_id`, `body` | Oda üyeleri (engelleme filtresi) | **30 gün** |
| `reports` | Raporlar | `reporter_id`, `reported_id`, `room_id`, `reason`, `messages_snapshot`, `status` | Yalnızca sunucu/moderasyon | SAFE-009'da belirlenir |
| `blocks` | Engellemeler | PK `(blocker_id, blocked_id)` | Sahibi | Kaldırılana kadar |

## Kesinleşen tablolar

Migration'lar: [`migrations/postgres`](../migrations/postgres/README.md). Tüm tablolarda RLS açık; yalnızca uygulama rolüne `app_access` politikası (ADR-0012). Kimlikler `uuid` (`gen_random_uuid()`), zamanlar `timestamptz`; zaman damgaları uygulama tarafından güncellenir (tetikleyici yok).

### 0001 — kimlik ve odalar (DB-003a)

| Tablo | Kesin kararlar |
|---|---|
| `users` | `nickname` 3–16 karakter, baş/son boşluk yok (ayrıntılı kural ve küfür filtresi uygulamada). Hesap silinince satır kalır: takma ad "Silinmiş oyuncu", `deleted_at` dolu. |
| `sessions` | `token_hash bytea` = SHA-256 (32 bayt, tekil). `expires_at > created_at`. Kullanıcı silinirse oturumlar silinir. İndeks: `user_id`, `expires_at` (temizlik). |
| `rooms` | `invite_code` 31 karakterlik alfabeden 6 büyük harf; **yalnızca açık odalar arasında tekil** (kapanan odanın kodu yeniden kullanılabilir). `capacity` ∈ {1,2}, varsayılan 2. `status` ∈ lobby/in_game/finished/closed; `closed_at` yalnızca `closed`'da dolu; açık odanın sahibi olmak zorunda. `version` = lobi olaylarının sürümü. `current_game_id`'nin yabancı anahtarı `games` ile gelir (DB-003c). |
| `room_members` | PK `(room_id, user_id)`. `role` player/spectator; izleyici hazır olamaz; atılan (`kicked_at`) aynı zamanda ayrılmıştır. **Aktif üyelik = `left_at` boş.** Kullanıcı başına tek aktif üyelik (ALREADY_IN_ROOM) ve oda içinde aktif üyeler arasında büyük/küçük harf duyarsız tekil `nickname` (NICKNAME_TAKEN_IN_ROOM) veritabanında garanti edilir; bunun için katılma anındaki takma adın kopyası `room_members.nickname`'de tutulur (PATCH /me lobi üyeliğini de günceller). Oda silinince üyelikler silinir. |
| `blocks` | PK `(blocker_id, blocked_id)`, kendini engelleme yok; ters yön için `blocked_id` indeksi. |

### 0002 — katalog (DB-003b)

| Tablo | Kesin kararlar |
|---|---|
| `scenarios` | `id` metin anahtar (ör. `nuclear_winter`); `title` ≤ 40, `disaster` 20–600, `duration_label` (ör. "3 yıl"), `shelter_features` tam 3 madde, `outside_world` 20–400 karakter. |
| `card_definitions` | `id` içerik migration'larında açıkça verilir ve değişmez (öneri: meslek 1xxx, sağlık 2xxx, hobi 3xxx, eşya 4xxx, sır 5xxx). `category` 5 değerden biri, `text` 2–120 karakter ve kategori içinde tekil, `balance` ∈ {−1, 0, +1} (gizli denge etiketi, istemciye gönderilmez). İndeks: aktif kartlar `(category, balance)`. |

Her iki tablo **salt okunur** (`@app:readonly`): uygulama rolü yalnızca okur; içerik CARD-002…007'de migration'larla yüklenir. Kayıtlar silinmez, `active = false` ile emekliye ayrılır (geçmiş oyunlar başvurur).

### 0003 — oyun (DB-003c)

| Tablo | Kesin kararlar |
|---|---|
| `games` | `phase` ∈ dealing, scenario_intro, reveal, discussion, voting, tie_break, elimination, finished (ayrı `status` yok; aktif oyun = `phase <> 'finished'`). `round` 0 (dağıtım/giriş), REVEAL yalnızca tur 1–5, `round ≤ total_rounds` (2–9). `turn_seat` yalnızca REVEAL'da. `phase_deadline_at` bitmemiş her aşamada dolu (ADR-0005). `finished_at` ve `end_reason` (capacity_reached / abandoned) yalnızca bitince. **Odada tek aktif oyun** (kısmi tekil indeks). Oda silinirse oyun kalır (`room_id` boşalır). `rooms.current_game_id` → `games` yabancı anahtarı burada eklendi. İndeks: aktif oyunların deadline'ı (açılışta kurtarma). |
| `game_participants` | PK `(game_id, user_id)`; `seat` 0–9 oyunda tekil = rastgele açıklama sırası. `status` alive/eliminated/left; elenen/ayrılanın `eliminated_round`'u dolu (oyun başlamadan ayrılma = 0). Kazanan hayatta olmalı. "Hazırım" için `discussion_ready_round` (tur başına sıfırlama gerekmez). Takma ad kopyalanmaz (silinen hesap geçmişte de anonim görünür). |
| `player_cards` | PK `(game_id, user_id, category)`. **Kart kategorisi yuvayla eşleşir** (`card_definitions (id, category)` bileşik yabancı anahtarı). **Oyunda her kart en fazla bir kez** (`(game_id, card_id)` tekil). Açılış: `revealed_round`, `reveal_method` (player / timeout / elimination), `revealed_at` birlikte dolu. Dağıtılmış kart katalogdan silinemez. |
| `votes` | PK `(game_id, round, kind, voter_id)` → oy değiştirilemez; `kind` regular/tiebreak; kendine oy yok; oy veren ve hedef oyunun katılımcısı. Yalnızca sunucuda okunur. |
| `round_results` | PK `(game_id, round)`; `method` vote/spectators/lottery/left; `left` dışında elenen dolu; `tally` JSON dizi; `lottery_candidates` yalnızca kurada ve en az 2 aday. |
| `game_actions` | Idempotency: `(game_id, actor_id, action_id)` tekil; tekrar gelen istek kaydedilen `state_version`'ı döndürür. Sistem eylemleri `actor_id = null` (zamanlayıcı; sürüm denetimi korur). `type`: reveal_card, discussion_ready, vote, tiebreak_vote, leave, disconnect_timeout, phase_timeout. |

Oyun silinirse (yalnızca bakım) bağlı tüm kayıtlar silinir.

### 0004 — sohbet ve moderasyon (DB-003d)

| Tablo | Kesin kararlar |
|---|---|
| `chat_messages` | `bigint` kimlik (sıralama). `body` küfür filtresinden geçmiş (maskelenmiş) metin, 1–200 karakter, baş/son boşluksuz; maskelenmemiş metin saklanmaz. Tekrar koruması `(room_id, user_id, client_msg_id)`. `game_id` lobide boş. Oda silinince mesajlar silinir. İndeksler: `(room_id, created_at desc, id desc)` son 50 mesaj; `created_at` 30 günlük temizlik (SAFE-009). |
| `reports` | `reason` harassment/hate/spam/cheating/other; `note` ≤ 200; kendini raporlama yok. `messages_snapshot` rapor anındaki son mesajlar (JSON dizi, ≤ 64 KB) — sohbet silinse ve oda temizlense de kanıt kalır. `status` open → reviewing → actioned/dismissed; kapanınca `resolved_at`. Aynı kişi aynı odada aynı kişi için tek açık rapor. Sonuçlanan rapor 180 gün sonra silinir (SAFE-009); açık raporlar kalır. |

### 0005 — temizlik indeksleri (DB-004)

`rooms_open_updated_at_idx`, `rooms_closed_at_idx` (ROOM-010) ve `reports_room_id_idx`. Sorgular ve indeks gerekçeleri: [queries.md](queries.md).

### 0006 — rapor saklama indeksi (SAFE-009)

`reports_resolved_at_idx` (kısmi: `status in (actioned, dismissed)`): sonuçlanmış raporların 180 günlük toplu silmesi; indeks olmadan her parti raporlar tablosunu tarar (sorgu planı testi gösterdi).

## Tutarlılık kuralları

- Her oyun eylemi `games` satırı `FOR UPDATE` ile kilitlenmiş kısa bir transaction içinde işlenir; `state_version` her değişiklikte +1 ([ADR-0004](decisions/0004-db-authoritative-state.md)).
- Aynı kart bir oyunda iki kez dağıtılamaz: tekil `(game_id, card_id)`.
- Aynı oyuncu aynı turda aynı türde iki kez oy veremez: tekil `(game_id, round, kind, voter_id)`.
- Tekrarlanan eylem: tekil `(game_id, actor_id, action_id)`.

## Yazma yükü tahmini

10 oyuncu, kapasite 2, 8 tur: kart açma + oylama + aşama geçişleri ≈ 400 yazma; sohbet ≈ 200 INSERT → **maç başına ≈ 600 kısa yazma**. Ölçüm QA-007'de.
