-- 0004 chat and moderation (DB-003d)
-- Yazılı sohbet ve raporlar. Kaynak: product-requirements.md §4,
-- api.md (Oda: messages, Moderasyon), architecture.md §c, security.md §6.

-- Sohbet mesajı. body, küfür filtresinden geçmiş (maskelenmiş) metindir;
-- yani diğer oyuncuların gördüğüdür. 30 gün sonra silinir (SAFE-009,
-- created_at indeksiyle toplu silme).
create table chat_messages (
  id            bigint      generated always as identity primary key,
  room_id       uuid        not null references rooms (id) on delete cascade,
  game_id       uuid        references games (id) on delete set null,
  user_id       uuid        not null references users (id),
  -- İstemcinin ürettiği kimlik: ağ tekrarında aynı mesaj iki kez kaydedilmez.
  client_msg_id uuid        not null,
  body          text        not null,
  created_at    timestamptz not null default now(),
  constraint chat_messages_idempotency_key unique (room_id, user_id, client_msg_id),
  constraint chat_messages_body_shape check (body = btrim(body) and char_length(body) between 1 and 200)
);
-- Yeniden bağlanmada odanın son 50 mesajı (ADR-0006) ve rapor kanıtı.
create index chat_messages_room_created_idx on chat_messages (room_id, created_at desc, id desc);
-- 30 günlük saklama temizliği.
create index chat_messages_created_at_idx on chat_messages (created_at);
create index chat_messages_user_id_idx on chat_messages (user_id);

-- Rapor. messages_snapshot, rapor anındaki odanın son mesajlarıdır
-- ([{"id", "user_id", "nickname", "text", "created_at"}]); sohbet 30 günde
-- silinse de kanıt korunur. Saklama süresi SAFE-009'da belirlenir.
create table reports (
  id                uuid        primary key default gen_random_uuid(),
  reporter_id       uuid        not null references users (id),
  reported_id       uuid        not null references users (id),
  room_id           uuid        references rooms (id) on delete set null,
  reason            text        not null,
  note              text,
  messages_snapshot jsonb       not null default '[]',
  status            text        not null default 'open',
  created_at        timestamptz not null default now(),
  resolved_at       timestamptz,
  constraint reports_not_self check (reporter_id <> reported_id),
  constraint reports_reason_valid check (reason in ('harassment', 'hate', 'spam', 'cheating', 'other')),
  constraint reports_note_len check (note is null or char_length(note) between 1 and 200),
  constraint reports_snapshot_array check (jsonb_typeof(messages_snapshot) = 'array'),
  -- 50 mesaj × ~300 bayt; kötüye kullanıma karşı üst sınır.
  constraint reports_snapshot_size check (octet_length(messages_snapshot::text) <= 65536),
  constraint reports_status_valid check (status in ('open', 'reviewing', 'actioned', 'dismissed')),
  constraint reports_resolved_consistent check ((status in ('actioned', 'dismissed')) = (resolved_at is not null))
);
-- Aynı kişi aynı odada aynı kişiyi açık raporla bir kez raporlar (spam önleme).
create unique index reports_open_once_key on reports (reporter_id, reported_id, room_id) where status = 'open';
-- Moderasyon kuyruğu ve bir kullanıcı hakkındaki raporlar.
create index reports_queue_idx on reports (created_at) where status in ('open', 'reviewing');
create index reports_reported_id_idx on reports (reported_id);
