-- 0010 usernames and friends
-- 2026-11-08 kullanıcı kararı: takma ad küresel olarak benzersiz bir
-- kullanıcı adıdır; arkadaşlık (istek + onay) ve lobiye davet.

-- Benzersizlik anahtarı: Türkçe küçük harf (I→ı, İ→i), NFC; uygulama üretir
-- (auth.NicknameKey). Mevcut hesaplarda boş: bir sonraki açılışta kullanıcı
-- adını onaylar/seçer (GET /v1/me needs_nickname). Silinmiş hesapta boş.
alter table users add column nickname_key text;
alter table users add constraint users_nickname_key_shape check (
  nickname_key is null or char_length(nickname_key) between 3 and 16
);
create unique index users_nickname_key_uniq on users (nickname_key) where nickname_key is not null;

-- Arkadaşlık isteği (gönderen → alan). Kabul edilince silinir.
create table friend_requests (
  from_user_id uuid        not null references users (id) on delete cascade,
  to_user_id   uuid        not null references users (id) on delete cascade,
  created_at   timestamptz not null default now(),
  primary key (from_user_id, to_user_id),
  constraint friend_requests_not_self check (from_user_id <> to_user_id)
);
create index friend_requests_to_user_id_idx on friend_requests (to_user_id);

-- Arkadaşlık: iki yönlü, iki satır (her kullanıcı kendi listesini tek
-- sorguyla okur).
create table friendships (
  user_id    uuid        not null references users (id) on delete cascade,
  friend_id  uuid        not null references users (id) on delete cascade,
  created_at timestamptz not null default now(),
  primary key (user_id, friend_id),
  constraint friendships_not_self check (user_id <> friend_id)
);
create index friendships_friend_id_idx on friendships (friend_id);

-- Lobi daveti: arkadaşa, süreli. Oda kapanınca/silinince gider.
create table room_invites (
  room_id      uuid        not null references rooms (id) on delete cascade,
  to_user_id   uuid        not null references users (id) on delete cascade,
  from_user_id uuid        not null references users (id) on delete cascade,
  created_at   timestamptz not null default now(),
  expires_at   timestamptz not null,
  primary key (room_id, to_user_id),
  constraint room_invites_not_self check (from_user_id <> to_user_id)
);
create index room_invites_to_user_id_idx on room_invites (to_user_id, expires_at);
create index room_invites_from_user_id_idx on room_invites (from_user_id);
create index room_invites_expires_at_idx on room_invites (expires_at);
