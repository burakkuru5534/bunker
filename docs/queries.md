# Sorgular ve indeksler (DB-004)

> Kaynak: DB-004 (2026-11-03). Şema: [data-model.md](data-model.md), migration'lar: `migrations/postgres`.
> Bu listedeki her sorgu `backend/internal/migrate/query_plans_integration_test.go` içinde gerçekçi hacimde veriyle (20.000 kullanıcı, 3.000 oda, 2.000 oyun, 100.000 mesaj) **uygulama rolüyle EXPLAIN edilir**: büyük tablolarda tam tablo taraması olmamalı ve beklenen indeks kullanılmalıdır. CI'da PostgreSQL 17 ile çalışır.

## Sıcak yol

| Sorgu | Ne zaman | İndeks |
|---|---|---|
| Oturum doğrulama: `sessions` + `users`, `token_hash = $1` | Her REST/WS isteği (60 sn önbellekli) | `sessions_token_hash_key` |
| Kullanıcının aktif odası: `room_members` `user_id = $1 and left_at is null` | `GET /me`, oda oluşturma/katılma | `room_members_one_active_room_key` (kısmi tekil) |
| Davet koduyla oda `for update`: `invite_code = $1 and status <> 'closed'` | Katılma | `rooms_invite_code_open_key` (kısmi tekil) |
| Oda üyeleri: `room_id = $1 and left_at is null` | RoomView, yayın | `room_members_pkey` veya `room_members_active_nickname_key` |
| Üyelik / atılma: `(room_id, user_id)` | Her oda isteği | `room_members_pkey` |
| Oyun kilidi: `games` `id = $1 for update` | Her oyun eylemi (ADR-0004) | `games_pkey` |
| Katılımcılar (+ takma ad): `game_id = $1 order by seat` | Görüntü, sonuç | `game_participants_seat_key` / `_pkey` |
| Oyunun kartları (+ metin): `player_cards` `game_id = $1` | Görüntü, sonuç | `player_cards_pkey` / `_card_once_per_game_key` |
| Oy sayımı: `votes` `(game_id, round, kind)` | Oylama sonu | `votes_pkey` |
| Tur sonuçları: `round_results` `game_id = $1` | Görüntü, sonuç | `round_results_pkey` |
| Idempotency: `game_actions` `(game_id, actor_id, action_id)` | Her oyun eylemi | `game_actions_idempotency_key` |
| Son 50 mesaj: `chat_messages` `room_id = $1 order by created_at desc, id desc limit 50` | Yeniden bağlanma (ADR-0006), rapor | `chat_messages_room_created_idx` |
| Göndereni engelleyenler: `blocks` `blocked_id = $1` | Her sohbet yayını | `blocks_blocked_id_idx` |
| Engellenenler listesi: `blocks` `blocker_id = $1` | `GET /blocks` | `blocks_pkey` |
| Kullanıcının oturumlarını iptal: `sessions` `user_id = $1` | Çıkış, hesap silme | `sessions_user_id_idx` |
| Zamanlayıcı kurtarma: `games` `phase <> 'finished' order by phase_deadline_at` | Açılış (ADR-0005) | `games_active_deadline_idx` (kısmi) |

## Bakım işleri

Toplu silmeler küçük partilerle ve **`id = any(array(select id … order by … limit N))`** biçiminde yapılır: alt sorgu indeksle en fazla N kimlik seçer, silme birincil anahtarla yapılır. `id in (select …)` biçimi planlayıcıyı tüm tabloyu taramaya itebildiği için kullanılmaz (testle görüldü).

| İş | Sorgu | İndeks |
|---|---|---|
| Sohbet saklama (30 gün, SAFE-009) | `created_at < now() - 30 gün`, 1000'lik partiler | `chat_messages_created_at_idx` |
| Süresi dolan oturumlar | `expires_at < now()`, 1000'lik partiler | `sessions_expires_at_idx` |
| Hareketsiz açık odalar (ROOM-010) | `status <> 'closed' and updated_at < $1` | `rooms_open_updated_at_idx` (kısmi, 0005) |
| Eski kapalı odalar (ROOM-010) | `status = 'closed' and closed_at < $1`, 500'lük partiler | `rooms_closed_at_idx` (kısmi, 0005) |
| Moderasyon kuyruğu | `status in ('open', 'reviewing') order by created_at` | `reports_queue_idx` (kısmi) |

## Yabancı anahtar indeksleri

Üst satır silinince PostgreSQL alt tabloyu yabancı anahtar sütunlarıyla arar; indeks yoksa tüm tabloyu tarar ve bu `EXPLAIN`'de görünmez. `TestIntegrationForeignKeysHaveIndexes` her yabancı anahtarın destekleyici (kısmi olmayan) indeksi olduğunu denetler. Bilinçli istisnalar (üst satırın silinmediği ilişkiler; indeks yalnızca yazma maliyeti olurdu):

| Yabancı anahtar | Neden indeks yok |
|---|---|
| `games.scenario_id`, `player_cards (card_id, category)` | Katalog satırları silinmez (`active = false`) |
| `room_members.user_id`, `reports.reporter_id` | Kullanıcılar silinmez, anonimleştirilir |
| `votes` (oy veren, hedef), `round_results.eliminated_id` | Katılımcılar yalnızca oyunla birlikte silinir |
| `rooms.current_game_id`, `chat_messages.game_id` | Oyunlar yalnızca bakımda silinir |

Bu testte bulunan eksik `reports.room_id` (oda temizliği raporlara `set null` uygular) 0005'te eklendi.

## Yeni sorgu eklerken

1. Sorguyu `hotQueries` listesine beklenen indeksle ekleyin; gerekirse `seed`'e veri ekleyin.
2. Test başarısızsa önce sorguyu (ör. toplu silme biçimi), gerekmiyorsa yeni migration'la indeksi düzeltin.
3. Büyük tablolarda yeni indeks: `-- migrate:no-transaction` + `create index concurrently` (tek ifade).
