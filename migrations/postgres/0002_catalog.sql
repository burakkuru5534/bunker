-- 0002 catalog (DB-003b)
-- Senaryo ve kart katalogları. İçerik (metinler) ayrı migration'larla yüklenir
-- (CARD-002…007); bu dosya yalnızca yapıyı kurar.
-- Uygulama rolü katalogları yalnızca okur (@app:readonly, ADR-0012).
-- Kayıtlar silinmez, active = false ile emekliye ayrılır: geçmiş oyunlar
-- bunlara başvurur.

-- Senaryo: mekanik etkisi yok, tartışmanın zemini (game-design.md §7).
-- id, içerik migration'larının başvurduğu kalıcı anahtardır (ör. nuclear_winter).
create table scenarios (
  id               text        primary key,
  title            text        not null,
  disaster         text        not null,
  duration_label   text        not null,
  shelter_features text[]      not null,
  outside_world    text        not null,
  active           boolean     not null default true,
  created_at       timestamptz not null default now(),
  constraint scenarios_id_shape check (id ~ '^[a-z][a-z0-9_]{1,39}$'),
  constraint scenarios_title_shape check (title = btrim(title) and char_length(title) between 2 and 40),
  constraint scenarios_disaster_len check (char_length(btrim(disaster)) between 20 and 600),
  constraint scenarios_duration_len check (char_length(btrim(duration_label)) between 2 and 20),
  constraint scenarios_three_features check (
    cardinality(shelter_features) = 3 and array_position(shelter_features, null) is null
  ),
  constraint scenarios_outside_len check (char_length(btrim(outside_world)) between 20 and 400)
);
comment on table scenarios is '@app:readonly Senaryo kataloğu (game-design.md §7)';

-- Kart: her oyuncuya her kategoriden 1 kart (game-design.md §6).
-- balance gizli denge etiketidir (+1 olumlu, 0 nötr, -1 olumsuz); istemciye
-- gönderilmez. id içerik migration'larında açıkça verilir ve değişmez
-- (öneri: meslek 1xxx, sağlık 2xxx, hobi 3xxx, eşya 4xxx, sır 5xxx).
create table card_definitions (
  id         integer     primary key,
  category   text        not null,
  text       text        not null,
  balance    smallint    not null,
  active     boolean     not null default true,
  created_at timestamptz not null default now(),
  constraint card_definitions_id_positive check (id > 0),
  constraint card_definitions_category_valid check (category in ('profession', 'health', 'hobby', 'item', 'secret')),
  constraint card_definitions_text_shape check (text = btrim(text) and char_length(text) between 2 and 120),
  constraint card_definitions_balance_valid check (balance in (-1, 0, 1)),
  constraint card_definitions_category_text_key unique (category, text)
);
comment on table card_definitions is '@app:readonly Kart kataloğu (game-design.md §6); balance istemciye gönderilmez';

-- Dağıtım sorgusu: kategoriye göre aktif kartlar.
create index card_definitions_active_category_idx on card_definitions (category, balance) where active;
