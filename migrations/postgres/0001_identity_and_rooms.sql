-- 0001 identity and rooms (DB-003a)
-- Kullanıcılar, oturumlar, odalar, oda üyelikleri, engellemeler.
-- Kaynak: docs/data-model.md, docs/api.md (Oturum, Oda, Moderasyon), docs/security.md §1.
-- Şemadan bağımsızdır; yetkiler ve RLS 'migrate up' sonunda eşitlenir (ADR-0012).
-- Zaman damgaları uygulama tarafından güncellenir (tetikleyici yok).

-- Misafir kullanıcı. Hesap silinince satır kalır (oyun geçmişi), takma ad
-- "Silinmiş oyuncu" olur ve deleted_at dolar (SAFE-009).
create table users (
  id           uuid        primary key default gen_random_uuid(),
  nickname     text        not null,
  created_at   timestamptz not null default now(),
  last_seen_at timestamptz not null default now(),
  deleted_at   timestamptz,
  -- Ayrıntılı kural (karakterler, küfür) uygulamada; burada emniyet ağı.
  constraint users_nickname_shape check (
    nickname = btrim(nickname) and char_length(nickname) between 3 and 16
  )
);

-- Oturum: yalnızca SHA-256(token) saklanır, ham token veritabanında yok (ADR-0008).
create table sessions (
  id           uuid        primary key default gen_random_uuid(),
  user_id      uuid        not null references users (id) on delete cascade,
  token_hash   bytea       not null,
  created_at   timestamptz not null default now(),
  expires_at   timestamptz not null,
  last_used_at timestamptz,
  revoked_at   timestamptz,
  constraint sessions_token_hash_sha256 check (octet_length(token_hash) = 32),
  constraint sessions_expiry_after_create check (expires_at > created_at),
  constraint sessions_token_hash_key unique (token_hash)
);
create index sessions_user_id_idx on sessions (user_id);
create index sessions_expires_at_idx on sessions (expires_at);

-- Oda. Lobide state_version yerine 'version' kullanılır (api.md WS olayları).
-- current_game_id'nin yabancı anahtarı games tablosuyla eklenir (DB-003c).
create table rooms (
  id              uuid        primary key default gen_random_uuid(),
  invite_code     text        not null,
  host_user_id    uuid        references users (id) on delete set null,
  capacity        smallint    not null default 2,
  status          text        not null default 'lobby',
  current_game_id uuid,
  version         bigint      not null default 0,
  created_at      timestamptz not null default now(),
  updated_at      timestamptz not null default now(),
  closed_at       timestamptz,
  -- 31 karakterlik alfabe: 0/O ve 1/I/L yok; büyük harfle saklanır (api.md §3).
  constraint rooms_invite_code_shape check (invite_code ~ '^[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{6}$'),
  constraint rooms_capacity_range check (capacity in (1, 2)),
  constraint rooms_status_valid check (status in ('lobby', 'in_game', 'finished', 'closed')),
  constraint rooms_closed_consistent check ((status = 'closed') = (closed_at is not null)),
  constraint rooms_open_has_host check (status = 'closed' or host_user_id is not null),
  constraint rooms_version_nonnegative check (version >= 0)
);
-- Davet kodu yalnızca açık odalar arasında tekil; kapanan odanın kodu yeniden kullanılabilir.
create unique index rooms_invite_code_open_key on rooms (invite_code) where status <> 'closed';
create index rooms_host_user_id_idx on rooms (host_user_id);

-- Oda üyeliği. Aktif üyelik: left_at boş. Lobiden ayrılan satırı silinir;
-- atılan (kicked_at) ve oyundan ayrılan (left_at) satırı kalır.
-- nickname, katılma anındaki takma adın kopyasıdır: oda içi tekillik
-- yarış durumuna karşı veritabanında garanti edilir (NICKNAME_TAKEN_IN_ROOM).
create table room_members (
  room_id   uuid        not null references rooms (id) on delete cascade,
  user_id   uuid        not null references users (id) on delete cascade,
  nickname  text        not null,
  role      text        not null,
  is_ready  boolean     not null default false,
  joined_at timestamptz not null default now(),
  left_at   timestamptz,
  kicked_at timestamptz,
  primary key (room_id, user_id),
  constraint room_members_role_valid check (role in ('player', 'spectator')),
  constraint room_members_spectator_not_ready check (not (is_ready and role = 'spectator')),
  constraint room_members_kicked_has_left check (kicked_at is null or left_at is not null),
  constraint room_members_nickname_shape check (
    nickname = btrim(nickname) and char_length(nickname) between 3 and 16
  )
);
-- Kullanıcı aynı anda yalnızca bir aktif odada (ALREADY_IN_ROOM).
create unique index room_members_one_active_room_key on room_members (user_id) where left_at is null;
-- Aktif üyeler arasında takma ad tekil, büyük/küçük harf duyarsız.
create unique index room_members_active_nickname_key on room_members (room_id, lower(nickname)) where left_at is null;

-- Engelleme: engellenenin mesajları engelleyene gösterilmez (oyunu etkilemez).
create table blocks (
  blocker_id uuid        not null references users (id) on delete cascade,
  blocked_id uuid        not null references users (id) on delete cascade,
  created_at timestamptz not null default now(),
  primary key (blocker_id, blocked_id),
  constraint blocks_not_self check (blocker_id <> blocked_id)
);
create index blocks_blocked_id_idx on blocks (blocked_id);
