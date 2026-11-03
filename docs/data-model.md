# Veri Modeli (TASLAK) — Sığınak: Son Kalanlar

> Kaynak: ARCH-002 (2026-11-02, kullanıcı onaylı taslak). **Kesin şema DB-003'te** belirlenir; sütun tipleri ve indeksler orada netleşir.
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

## Tutarlılık kuralları

- Her oyun eylemi `games` satırı `FOR UPDATE` ile kilitlenmiş kısa bir transaction içinde işlenir; `state_version` her değişiklikte +1 ([ADR-0004](decisions/0004-db-authoritative-state.md)).
- Aynı kart bir oyunda iki kez dağıtılamaz: tekil `(game_id, card_id)`.
- Aynı oyuncu aynı turda aynı türde iki kez oy veremez: tekil `(game_id, round, kind, voter_id)`.
- Tekrarlanan eylem: tekil `(game_id, actor_id, action_id)`.

## Yazma yükü tahmini

10 oyuncu, kapasite 2, 8 tur: kart açma + oylama + aşama geçişleri ≈ 400 yazma; sohbet ≈ 200 INSERT → **maç başına ≈ 600 kısa yazma**. Ölçüm QA-007'de.
