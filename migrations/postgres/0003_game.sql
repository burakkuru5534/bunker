-- 0003 game (DB-003c)
-- Oyun durumu, katılımcılar, dağıtılan kartlar, oylar, tur sonuçları ve
-- eylem kayıtları. Kaynak: game-design.md §2–§10, api.md (Oyun),
-- ADR-0004 (kilit, sürüm, idempotency), ADR-0005 (kalıcı deadline).
-- Oyunlar kalıcıdır: oda temizlense de sonuç katılımcılara açıktır.
-- Takma adlar kopyalanmaz; users'tan okunur (hesap silinince geçmişte de
-- "Silinmiş oyuncu" görünür, SAFE-009).

create table games (
  id                uuid        primary key default gen_random_uuid(),
  room_id           uuid        references rooms (id) on delete set null,
  scenario_id       text        not null references scenarios (id),
  capacity          smallint    not null,
  total_rounds      smallint    not null,
  phase             text        not null default 'dealing',
  round             smallint    not null default 0,
  -- REVEAL aşamasında sırası gelen oyuncunun koltuğu.
  turn_seat         smallint,
  phase_started_at  timestamptz not null default now(),
  -- Aşamanın bitiş zamanı; zamanlayıcı kurtarma bu alana dayanır (ADR-0005).
  phase_deadline_at timestamptz,
  -- Her durum değişikliğinde +1 (ADR-0004).
  state_version     bigint      not null default 0,
  started_at        timestamptz not null default now(),
  finished_at       timestamptz,
  end_reason        text,
  constraint games_capacity_range check (capacity in (1, 2)),
  -- tur sayısı = oyuncu − kapasite; 4–10 oyuncu → 2..9.
  constraint games_total_rounds_range check (total_rounds between 2 and 9),
  constraint games_phase_valid check (phase in (
    'dealing', 'scenario_intro', 'reveal', 'discussion', 'voting', 'tie_break', 'elimination', 'finished')),
  constraint games_round_range check (round between 0 and total_rounds),
  constraint games_round_phase check (
    case
      when phase in ('dealing', 'scenario_intro') then round = 0
      when phase = 'reveal' then round between 1 and 5
      when phase = 'finished' then true
      else round >= 1
    end),
  constraint games_turn_only_in_reveal check ((phase = 'reveal') = (turn_seat is not null)),
  constraint games_turn_seat_range check (turn_seat between 0 and 9),
  constraint games_deadline_unless_finished check ((phase = 'finished') = (phase_deadline_at is null)),
  constraint games_finished_consistent check (
    (phase = 'finished') = (finished_at is not null) and (phase = 'finished') = (end_reason is not null)),
  constraint games_end_reason_valid check (end_reason in ('capacity_reached', 'abandoned')),
  constraint games_state_version_nonnegative check (state_version >= 0)
);
-- Odada aynı anda tek aktif oyun.
create unique index games_room_active_key on games (room_id) where phase <> 'finished';
create index games_room_id_idx on games (room_id);
-- Açılışta süresi dolmuş aşamaları bulmak için (ADR-0005).
create index games_active_deadline_idx on games (phase_deadline_at) where phase <> 'finished';

alter table rooms
  add constraint rooms_current_game_id_fkey foreign key (current_game_id) references games (id) on delete set null;

-- Katılımcı. seat, oyun başında rastgele belirlenen açıklama sırasıdır.
-- Ayrılan oyuncu elenmiş sayılır (status = left, eliminated_round dolu).
create table game_participants (
  game_id                 uuid        not null references games (id) on delete cascade,
  user_id                 uuid        not null references users (id),
  seat                    smallint    not null,
  status                  text        not null default 'alive',
  eliminated_round        smallint,
  -- Tartışmada "Hazırım" dediği tur (tur başına sıfırlama yazısı gerekmez).
  discussion_ready_round  smallint,
  disconnected_at         timestamptz,
  is_winner               boolean     not null default false,
  primary key (game_id, user_id),
  constraint game_participants_seat_key unique (game_id, seat),
  constraint game_participants_seat_range check (seat between 0 and 9),
  constraint game_participants_status_valid check (status in ('alive', 'eliminated', 'left')),
  constraint game_participants_eliminated_round check (
    (status = 'alive') = (eliminated_round is null) and eliminated_round between 0 and 9),
  constraint game_participants_winner_alive check (not is_winner or status = 'alive'),
  constraint game_participants_ready_round_range check (discussion_ready_round between 1 and 9)
);
-- Kullanıcının oyunları ve "bu oyunun eski katılımcısı mı" sorgusu.
create index game_participants_user_id_idx on game_participants (user_id);

-- Kartın kategorisi yuvayla eşleşmeli: (card_id, category) bileşik anahtarı.
alter table card_definitions add constraint card_definitions_id_category_key unique (id, category);

-- Dağıtılan kartlar: oyuncu başına kategori başına 1, oyunda her kart en fazla 1 kez.
create table player_cards (
  game_id        uuid        not null,
  user_id        uuid        not null,
  category       text        not null,
  card_id        integer     not null,
  revealed_round smallint,
  -- player: oyuncu açtı; timeout: süre dolunca sunucu açtı; elimination:
  -- elenince/ayrılınca tümü açıldı.
  reveal_method  text,
  revealed_at    timestamptz,
  primary key (game_id, user_id, category),
  constraint player_cards_participant_fkey foreign key (game_id, user_id)
    references game_participants (game_id, user_id) on delete cascade,
  constraint player_cards_card_fkey foreign key (card_id, category)
    references card_definitions (id, category),
  constraint player_cards_card_once_per_game_key unique (game_id, card_id),
  constraint player_cards_category_valid check (category in ('profession', 'health', 'hobby', 'item', 'secret')),
  constraint player_cards_reveal_consistent check (
    (revealed_round is null) = (reveal_method is null) and (revealed_round is null) = (revealed_at is null)),
  constraint player_cards_reveal_method_valid check (reveal_method in ('player', 'timeout', 'elimination')),
  constraint player_cards_revealed_round_range check (revealed_round between 1 and 9)
);

-- Oylar yalnızca sunucuda kalır; istemciye yalnızca sayılar gider.
-- regular: hayattaki oyuncular; tiebreak: izleyiciler, eşit adaylar arasından.
create table votes (
  game_id    uuid        not null references games (id) on delete cascade,
  round      smallint    not null,
  kind       text        not null,
  voter_id   uuid        not null,
  target_id  uuid        not null,
  created_at timestamptz not null default now(),
  -- Oy değiştirilemez: tur ve tür başına oyuncu başına tek oy.
  primary key (game_id, round, kind, voter_id),
  constraint votes_voter_fkey foreign key (game_id, voter_id) references game_participants (game_id, user_id),
  constraint votes_target_fkey foreign key (game_id, target_id) references game_participants (game_id, user_id),
  constraint votes_round_range check (round between 1 and 9),
  constraint votes_kind_valid check (kind in ('regular', 'tiebreak')),
  constraint votes_not_self check (voter_id <> target_id)
);

-- Tur sonucu. tally: [{"user_id": "…", "votes": n}] (adaylara gelen oy sayıları).
create table round_results (
  game_id            uuid        not null references games (id) on delete cascade,
  round              smallint    not null,
  eliminated_id      uuid,
  method             text        not null,
  tally              jsonb       not null default '[]',
  lottery_candidates uuid[],
  decided_at         timestamptz not null default now(),
  primary key (game_id, round),
  constraint round_results_eliminated_fkey foreign key (game_id, eliminated_id)
    references game_participants (game_id, user_id),
  constraint round_results_round_range check (round between 1 and 9),
  constraint round_results_method_valid check (method in ('vote', 'spectators', 'lottery', 'left')),
  constraint round_results_eliminated_unless_left check (method = 'left' or eliminated_id is not null),
  constraint round_results_tally_array check (jsonb_typeof(tally) = 'array'),
  constraint round_results_lottery_candidates check (
    (method = 'lottery') = (lottery_candidates is not null)
    and (lottery_candidates is null or cardinality(lottery_candidates) >= 2))
);

-- Eylem kaydı: idempotency + denetim (ADR-0004). Aynı (oyun, oyuncu,
-- action_id) ikinci kez gelirse kaydedilen state_version döner (duplicate).
-- Sistem eylemleri (zamanlayıcı) actor_id = null ile kaydedilir.
create table game_actions (
  id            bigint      generated always as identity primary key,
  game_id       uuid        not null references games (id) on delete cascade,
  actor_id      uuid,
  action_id     uuid        not null,
  type          text        not null,
  payload       jsonb       not null default '{}',
  state_version bigint      not null,
  created_at    timestamptz not null default now(),
  constraint game_actions_idempotency_key unique (game_id, actor_id, action_id),
  constraint game_actions_actor_fkey foreign key (game_id, actor_id) references game_participants (game_id, user_id),
  constraint game_actions_type_valid check (type in (
    'reveal_card', 'discussion_ready', 'vote', 'tiebreak_vote', 'leave', 'disconnect_timeout', 'phase_timeout')),
  constraint game_actions_payload_object check (jsonb_typeof(payload) = 'object'),
  constraint game_actions_state_version_positive check (state_version > 0)
);
