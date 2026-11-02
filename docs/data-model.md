# Veri Modeli (TASLAK) — Sığınak: Son Kalanlar

> Kaynak: ARCH-002 (2026-11-02, kullanıcı onaylı taslak). **Kesin şema DB-003'te** belirlenir; sütun tipleri ve indeksler orada netleşir.
> Tek kalıcı veri kaynağı: Supabase PostgreSQL.

## Sadeleştirmeler (master prompt listesine göre)

- `player_profiles` ayrı tablo değil → `users` içinde.
- `game_summaries` ayrı tablo değil → kalıcı sonuç `games` + `game_participants` + `round_results`'tan okunur (veri tekrarı yok).

## Tablolar

| Tablo | Amaç | Önemli alanlar / kısıtlar | Erişim | Saklama |
|---|---|---|---|---|
| `users` | Misafir kullanıcı | `id`, `nickname`, `created_at`, `last_seen_at` | Sahibi; takma ad oda üyelerine görünür | Hesap silinene kadar (SAFE-009) |
| `sessions` | Oturumlar | `user_id`, `token_hash`, `expires_at`, `revoked_at` | Yalnızca sunucu | Süresi dolunca temizlenir (AUTH-001) |
| `rooms` | Oda | `invite_code` (aktif odalarda tekil), `host_user_id`, `capacity` ∈ {1,2}, `status`, `current_game_id` | Oda üyeleri | Kapanınca temizlik (ROOM-010) |
| `room_members` | Üyelik | PK `(room_id, user_id)`; `role` (player/spectator), `is_ready`, `joined_at`, `left_at` | Oda üyeleri | Oda ile |
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

## Tutarlılık kuralları

- Her oyun eylemi `games` satırı `FOR UPDATE` ile kilitlenmiş kısa bir transaction içinde işlenir; `state_version` her değişiklikte +1 ([ADR-0004](decisions/0004-db-authoritative-state.md)).
- Aynı kart bir oyunda iki kez dağıtılamaz: tekil `(game_id, card_id)`.
- Aynı oyuncu aynı turda aynı türde iki kez oy veremez: tekil `(game_id, round, kind, voter_id)`.
- Tekrarlanan eylem: tekil `(game_id, actor_id, action_id)`.

## Yazma yükü tahmini

10 oyuncu, kapasite 2, 8 tur: kart açma + oylama + aşama geçişleri ≈ 400 yazma; sohbet ≈ 200 INSERT → **maç başına ≈ 600 kısa yazma**. Ölçüm QA-007'de.
