package migrate

// Depodaki migration'larla kurulan şemanın kısıtları, uygulama rolüyle
// gerçek PostgreSQL'de sınanır (DB-003). TEST_DATABASE_ADMIN_URL gerekir.

import (
	"context"
	"crypto/sha256"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// schemaEnv, depo migration'ları uygulanmış rastgele bir şemadır.
func schemaEnv(t *testing.T) *env {
	t.Helper()
	e := setup(t)
	migs, err := Load(filepath.Join("..", "..", "..", "migrations", "postgres"))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Up(ctx10(t), e.migPool, e.opts(), migs, false); err != nil {
		t.Fatal(err)
	}
	return e
}

// exec, uygulama rolüyle çalıştırır; wantConstraint boş değilse o kısıtın
// ihlal edilmesini bekler.
func (e *env) expect(t *testing.T, wantConstraint, sql string, args ...any) {
	t.Helper()
	_, err := e.appPool.Exec(context.Background(), sql, args...)
	if wantConstraint == "" {
		if err != nil {
			t.Fatalf("başarılı olmalı: %s: %v", sql, err)
		}
		return
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.ConstraintName != wantConstraint {
		t.Fatalf("%s kısıtı bekleniyordu: %s: %v", wantConstraint, sql, err)
	}
}

func (e *env) newUser(t *testing.T, nick string) string {
	t.Helper()
	var id string
	if err := e.appPool.QueryRow(context.Background(), "insert into users (nickname) values ($1) returning id", nick).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (e *env) newRoom(t *testing.T, code, host string) string {
	t.Helper()
	var id string
	if err := e.appPool.QueryRow(context.Background(),
		"insert into rooms (invite_code, host_user_id) values ($1, $2) returning id", code, host).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestIntegrationSchemaUsersAndSessions(t *testing.T) {
	e := schemaEnv(t)
	e.expect(t, "", "insert into users (nickname) values ('Burak')")
	e.expect(t, "", "insert into users (nickname) values ('Silinmiş oyuncu')")
	e.expect(t, "users_nickname_shape", "insert into users (nickname) values ('ab')")
	e.expect(t, "users_nickname_shape", "insert into users (nickname) values ('ÇokUzunBirTakmaAd1')")
	e.expect(t, "users_nickname_shape", "insert into users (nickname) values (' Burak')")

	u := e.newUser(t, "Ayşe")
	hash := sha256.Sum256([]byte("token"))
	e.expect(t, "", "insert into sessions (user_id, token_hash, expires_at) values ($1, $2, now() + interval '90 days')", u, hash[:])
	e.expect(t, "sessions_token_hash_key", "insert into sessions (user_id, token_hash, expires_at) values ($1, $2, now() + interval '1 day')", u, hash[:])
	e.expect(t, "sessions_token_hash_sha256", "insert into sessions (user_id, token_hash, expires_at) values ($1, '\\x00', now() + interval '1 day')", u)
	e.expect(t, "sessions_expiry_after_create", "insert into sessions (user_id, token_hash, expires_at) values ($1, $2, now() - interval '1 day')", u, make([]byte, 32))
	e.expect(t, "sessions_user_id_fkey", "insert into sessions (user_id, token_hash, expires_at) values (gen_random_uuid(), $1, now() + interval '1 day')", make([]byte, 32))
}

func TestIntegrationSchemaRooms(t *testing.T) {
	e := schemaEnv(t)
	host := e.newUser(t, "Burak")
	room := e.newRoom(t, "K7M2QX", host)

	var capacity int
	var status string
	var version int64
	if err := e.appPool.QueryRow(context.Background(), "select capacity, status, version from rooms where id = $1", room).Scan(&capacity, &status, &version); err != nil ||
		capacity != 2 || status != "lobby" || version != 0 {
		t.Fatalf("varsayılanlar: %d %s %d %v", capacity, status, version, err)
	}

	for _, bad := range []string{"k7m2qx", "K7M2Q", "K7M2QX1", "O7M2QX", "17M2QX", "L7M2QX"} {
		e.expect(t, "rooms_invite_code_shape", "insert into rooms (invite_code, host_user_id) values ($1, $2)", bad, host)
	}
	e.expect(t, "rooms_invite_code_open_key", "insert into rooms (invite_code, host_user_id) values ('K7M2QX', $1)", host)
	e.expect(t, "rooms_capacity_range", "insert into rooms (invite_code, host_user_id, capacity) values ('AAAAAA', $1, 3)", host)
	e.expect(t, "rooms_status_valid", "update rooms set status = 'playing' where id = $1", room)
	e.expect(t, "rooms_open_has_host", "insert into rooms (invite_code) values ('BBBBBB')")
	e.expect(t, "rooms_closed_consistent", "update rooms set status = 'closed' where id = $1", room)
	e.expect(t, "rooms_version_nonnegative", "update rooms set version = -1 where id = $1", room)

	// Kapanan odanın kodu yeniden kullanılabilir; kapalı odada sahip gerekmez.
	e.expect(t, "", "update rooms set status = 'closed', closed_at = now(), host_user_id = null where id = $1", room)
	e.newRoom(t, "K7M2QX", host)
}

func TestIntegrationSchemaRoomMembers(t *testing.T) {
	e := schemaEnv(t)
	burak, ayse := e.newUser(t, "Burak"), e.newUser(t, "Ayşe")
	r1, r2 := e.newRoom(t, "AAAAAA", burak), e.newRoom(t, "BBBBBB", ayse)
	join := "insert into room_members (room_id, user_id, nickname, role) values ($1, $2, $3, $4)"

	e.expect(t, "", join, r1, burak, "Burak", "player")
	e.expect(t, "room_members_pkey", join, r1, burak, "Burak", "player")
	// Aynı anda tek aktif oda.
	e.expect(t, "room_members_one_active_room_key", join, r2, burak, "Burak", "player")
	// Odada aynı takma ad (büyük/küçük harf duyarsız).
	e.expect(t, "room_members_active_nickname_key", join, r1, ayse, "BURAK", "player")
	e.expect(t, "room_members_role_valid", join, r1, ayse, "Ayşe", "host")
	e.expect(t, "room_members_nickname_shape", join, r1, ayse, "A", "player")
	e.expect(t, "", join, r1, ayse, "Ayşe", "spectator")
	e.expect(t, "room_members_spectator_not_ready", "update room_members set is_ready = true where user_id = $1", ayse)
	e.expect(t, "room_members_kicked_has_left", "update room_members set kicked_at = now() where user_id = $1", ayse)

	// Ayrılınca başka odaya katılabilir; eski takma ad serbest kalır.
	e.expect(t, "", "update room_members set left_at = now(), kicked_at = now() where room_id = $1 and user_id = $2", r1, ayse)
	e.expect(t, "", join, r2, ayse, "Ayşe", "player")
	carol := e.newUser(t, "Ayşe")
	e.expect(t, "", join, r1, carol, "Ayşe", "player")

	// Oda silinince üyelikler gider (temizlik, ROOM-010).
	e.expect(t, "", "delete from rooms where id = $1", r2)
	var n int
	_ = e.appPool.QueryRow(context.Background(), "select count(*) from room_members where room_id = $1", r2).Scan(&n)
	if n != 0 {
		t.Fatal("üyelikler odayla silinmeli")
	}
}

func TestIntegrationSchemaBlocksAndUserDelete(t *testing.T) {
	e := schemaEnv(t)
	a, b := e.newUser(t, "Burak"), e.newUser(t, "Ayşe")
	e.expect(t, "", "insert into blocks (blocker_id, blocked_id) values ($1, $2)", a, b)
	e.expect(t, "blocks_pkey", "insert into blocks (blocker_id, blocked_id) values ($1, $2)", a, b)
	e.expect(t, "blocks_not_self", "insert into blocks (blocker_id, blocked_id) values ($1, $1)", a)

	// Sahibi silinen oda sahipsiz kalamaz (açıkken); önce devir/kapanış gerekir.
	room := e.newRoom(t, "CCCCCC", a)
	e.expect(t, "rooms_open_has_host", "delete from users where id = $1", a)
	e.expect(t, "", "update rooms set status = 'closed', closed_at = now() where id = $1", room)
	e.expect(t, "", "delete from users where id = $1", a)
	var host *string
	_ = e.appPool.QueryRow(context.Background(), "select host_user_id from rooms where id = $1", room).Scan(&host)
	var blocks int
	_ = e.appPool.QueryRow(context.Background(), "select count(*) from blocks").Scan(&blocks)
	if host != nil || blocks != 0 {
		t.Fatalf("silinen kullanıcının bağları temizlenmeli: host=%v blocks=%d", host, blocks)
	}
}

func TestIntegrationSchemaRLSOnEveryTable(t *testing.T) {
	e := schemaEnv(t)
	rows, err := e.admin.Query(context.Background(), `
		select c.relname, c.relrowsecurity,
		       exists (select 1 from pg_policies p where p.schemaname = $1 and p.tablename = c.relname and p.policyname = 'app_access')
		  from pg_class c join pg_namespace n on n.oid = c.relnamespace
		 where n.nspname = $1 and c.relkind in ('r', 'p')`, e.schema)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	seen := map[string]bool{}
	for rows.Next() {
		var name string
		var rls, pol bool
		_ = rows.Scan(&name, &rls, &pol)
		seen[name] = true
		if !rls {
			t.Errorf("%s: RLS kapalı", name)
		}
		if pol != (name != TrackingTable) {
			t.Errorf("%s: app_access politikası = %v", name, pol)
		}
	}
	for _, want := range []string{"users", "sessions", "rooms", "room_members", "blocks", "scenarios", "card_definitions",
		"games", "game_participants", "player_cards", "votes", "round_results", "game_actions"} {
		if !seen[want] {
			t.Errorf("%s tablosu yok (görülen: %s)", want, strings.Join(keys(seen), ","))
		}
	}
	// Kataloglar salt okunur, diğerleri okuma/yazma.
	for table, want := range map[string]string{"scenarios": "SELECT", "card_definitions": "SELECT", "users": "ALL", "rooms": "ALL"} {
		if got := e.policyCmd(t, table); got != want {
			t.Errorf("%s politikası %s olmalı: %s", table, want, got)
		}
	}
}

func keys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// mig, migration rolüyle çalıştırır (katalog verisini yalnızca o yazabilir).
func (e *env) asMigrator(t *testing.T, wantConstraint, sql string, args ...any) {
	t.Helper()
	_, err := e.migPool.Exec(context.Background(), sql, args...)
	if wantConstraint == "" {
		if err != nil {
			t.Fatalf("başarılı olmalı: %s: %v", sql, err)
		}
		return
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.ConstraintName != wantConstraint {
		t.Fatalf("%s kısıtı bekleniyordu: %s: %v", wantConstraint, sql, err)
	}
}

func TestIntegrationSchemaCatalog(t *testing.T) {
	e := schemaEnv(t)
	insScenario := `insert into scenarios (id, title, disaster, duration_label, shelter_features, outside_world)
		values ($1, $2, $3, $4, $5, $6)`
	disaster := "Gökyüzü aylardır kül rengi; güneş ışığı yere ulaşmıyor."
	outside := "Dışarıda sıcaklık eksi kırkın altında; yollar kapalı."
	feats := []string{"Erzak tam", "Isıtma bakım istiyor", "Tıbbi malzeme az"}

	e.asMigrator(t, "", insScenario, "nuclear_winter", "Nükleer Kış", disaster, "3 yıl", feats, outside)
	e.asMigrator(t, "scenarios_pkey", insScenario, "nuclear_winter", "Başka", disaster, "3 yıl", feats, outside)
	e.asMigrator(t, "scenarios_id_shape", insScenario, "Nuclear-Winter", "Nükleer Kış", disaster, "3 yıl", feats, outside)
	e.asMigrator(t, "scenarios_title_shape", insScenario, "x1", " Nükleer", disaster, "3 yıl", feats, outside)
	e.asMigrator(t, "scenarios_disaster_len", insScenario, "x2", "Başlık", "kısa", "3 yıl", feats, outside)
	e.asMigrator(t, "scenarios_three_features", insScenario, "x3", "Başlık", disaster, "3 yıl", feats[:2], outside)
	e.asMigrator(t, "scenarios_three_features", insScenario, "x4", "Başlık", disaster, "3 yıl", []*string{&feats[0], nil, &feats[1]}, outside)
	e.asMigrator(t, "scenarios_outside_len", insScenario, "x5", "Başlık", disaster, "3 yıl", feats, "kısa")

	insCard := "insert into card_definitions (id, category, text, balance) values ($1, $2, $3, $4)"
	e.asMigrator(t, "", insCard, 1001, "profession", "Elektrik mühendisi", 1)
	e.asMigrator(t, "", insCard, 5001, "secret", "Ehliyeti sahte", -1)
	e.asMigrator(t, "card_definitions_pkey", insCard, 1001, "profession", "Doktor", 1)
	e.asMigrator(t, "card_definitions_category_text_key", insCard, 1002, "profession", "Elektrik mühendisi", 0)
	e.asMigrator(t, "card_definitions_category_valid", insCard, 6001, "action", "Kapıyı kilitle", 0)
	e.asMigrator(t, "card_definitions_balance_valid", insCard, 1003, "profession", "Aşçı", 2)
	e.asMigrator(t, "card_definitions_text_shape", insCard, 1004, "profession", "Aşçı ", 0)
	e.asMigrator(t, "card_definitions_id_positive", insCard, 0, "profession", "Aşçı", 0)

	// Uygulama rolü okur, yazamaz.
	var n int
	if err := e.appPool.QueryRow(context.Background(), "select count(*) from card_definitions where active").Scan(&n); err != nil || n != 2 {
		t.Fatalf("uygulama kataloğu okuyabilmeli: %d %v", n, err)
	}
	for _, q := range []string{
		"insert into card_definitions (id, category, text, balance) values (9, 'item', 'Fener', 1)",
		"update card_definitions set balance = 1",
		"delete from scenarios",
	} {
		if _, err := e.appPool.Exec(context.Background(), q); err == nil {
			t.Errorf("uygulama katalogu değiştirememeli: %s", q)
		}
	}
}

// gameFixture: senaryo, kartlar, oda ve 4 kullanıcılı bir oyun kurar.
type gameFixture struct {
	game, room string
	users      []string
}

func (e *env) newGame(t *testing.T) gameFixture {
	t.Helper()
	ctx := context.Background()
	e.asMigrator(t, "", `insert into scenarios (id, title, disaster, duration_label, shelter_features, outside_world)
		values ('nuclear_winter', 'Nükleer Kış', 'Gökyüzü aylardır kül rengi; güneş ışığı yere ulaşmıyor.', '3 yıl',
		        array['Erzak tam', 'Isıtma bakım istiyor', 'Tıbbi malzeme az'], 'Dışarıda sıcaklık eksi kırkın altında.')`)
	e.asMigrator(t, "", `insert into card_definitions (id, category, text, balance) values
		(1001, 'profession', 'Elektrik mühendisi', 1), (1002, 'profession', 'Aşçı', 0),
		(2001, 'health', 'Sağlıklı', 1), (5001, 'secret', 'Ehliyeti sahte', -1)`)
	var f gameFixture
	for _, n := range []string{"Burak", "Ayşe", "Mehmet", "Zeynep"} {
		f.users = append(f.users, e.newUser(t, n))
	}
	f.room = e.newRoom(t, "GGGGGG", f.users[0])
	if err := e.appPool.QueryRow(ctx, `insert into games (room_id, scenario_id, capacity, total_rounds, phase_deadline_at)
		values ($1, 'nuclear_winter', 2, 2, now() + interval '5 seconds') returning id`, f.room).Scan(&f.game); err != nil {
		t.Fatal(err)
	}
	for i, u := range f.users {
		e.expect(t, "", "insert into game_participants (game_id, user_id, seat) values ($1, $2, $3)", f.game, u, i)
	}
	e.expect(t, "", "update rooms set status = 'in_game', current_game_id = $2 where id = $1", f.room, f.game)
	return f
}

func TestIntegrationSchemaGames(t *testing.T) {
	e := schemaEnv(t)
	f := e.newGame(t)
	g := f.game
	ins := `insert into games (room_id, scenario_id, capacity, total_rounds, phase_deadline_at) values ($1, $2, $3, $4, now())`

	// Odada tek aktif oyun.
	e.expect(t, "games_room_active_key", ins, f.room, "nuclear_winter", 2, 2)
	e.expect(t, "games_scenario_id_fkey", ins, nil, "yok_boyle", 2, 2)
	e.expect(t, "games_capacity_range", ins, nil, "nuclear_winter", 3, 2)
	e.expect(t, "games_total_rounds_range", ins, nil, "nuclear_winter", 2, 1)
	e.expect(t, "games_deadline_unless_finished", `insert into games (scenario_id, capacity, total_rounds) values ('nuclear_winter', 2, 2)`)

	upd := "update games set phase = $2, round = $3, turn_seat = $4 where id = $1"
	e.expect(t, "games_phase_valid", upd, g, "lobby", 0, nil)
	e.expect(t, "games_round_phase", upd, g, "scenario_intro", 1, nil)
	e.expect(t, "games_turn_only_in_reveal", upd, g, "reveal", 1, nil)
	e.expect(t, "games_turn_only_in_reveal", upd, g, "discussion", 1, 0)
	e.expect(t, "games_round_range", upd, g, "voting", 3, nil)
	e.expect(t, "", upd, g, "reveal", 1, 0)
	e.expect(t, "", upd, g, "voting", 2, nil)
	e.expect(t, "games_state_version_nonnegative", "update games set state_version = -1 where id = $1", g)

	// Bitiş: deadline yok, zaman ve neden dolu.
	e.expect(t, "games_deadline_unless_finished", "update games set phase = 'finished', finished_at = now(), end_reason = 'abandoned' where id = $1", g)
	e.expect(t, "games_finished_consistent", "update games set phase = 'finished', phase_deadline_at = null where id = $1", g)
	e.expect(t, "games_end_reason_valid", "update games set phase = 'finished', phase_deadline_at = null, finished_at = now(), end_reason = 'won' where id = $1", g)
	e.expect(t, "", "update games set phase = 'finished', phase_deadline_at = null, finished_at = now(), end_reason = 'capacity_reached' where id = $1", g)
	// Bitince odada yeni oyun açılabilir.
	e.expect(t, "", ins, f.room, "nuclear_winter", 1, 3)

	// Oda silinse de oyun kalır (sonuç kalıcı).
	e.expect(t, "", "update rooms set status = 'closed', closed_at = now(), current_game_id = null where id = $1", f.room)
	e.expect(t, "", "delete from rooms where id = $1", f.room)
	var room *string
	if err := e.appPool.QueryRow(context.Background(), "select room_id from games where id = $1", g).Scan(&room); err != nil || room != nil {
		t.Fatalf("oyun kalmalı, oda bağı boşalmalı: %v %v", room, err)
	}
	e.expect(t, "rooms_current_game_id_fkey", "insert into rooms (invite_code, host_user_id, current_game_id) values ('HHHHHH', $1, gen_random_uuid())", f.users[0])
}

func TestIntegrationSchemaParticipantsAndCards(t *testing.T) {
	e := schemaEnv(t)
	f := e.newGame(t)
	g, u := f.game, f.users
	extra := e.newUser(t, "Can")

	e.expect(t, "game_participants_seat_key", "insert into game_participants (game_id, user_id, seat) values ($1, $2, 0)", g, extra)
	e.expect(t, "game_participants_seat_range", "insert into game_participants (game_id, user_id, seat) values ($1, $2, 10)", g, extra)
	e.expect(t, "game_participants_eliminated_round", "update game_participants set status = 'eliminated' where game_id = $1 and user_id = $2", g, u[3])
	e.expect(t, "game_participants_eliminated_round", "update game_participants set eliminated_round = 1 where game_id = $1 and user_id = $2", g, u[3])
	e.expect(t, "game_participants_status_valid", "update game_participants set status = 'dead', eliminated_round = 1 where game_id = $1 and user_id = $2", g, u[3])
	e.expect(t, "", "update game_participants set status = 'left', eliminated_round = 0 where game_id = $1 and user_id = $2", g, u[3])
	e.expect(t, "game_participants_winner_alive", "update game_participants set is_winner = true where game_id = $1 and user_id = $2", g, u[3])
	e.expect(t, "", "update game_participants set is_winner = true where game_id = $1 and user_id = $2", g, u[0])

	deal := "insert into player_cards (game_id, user_id, category, card_id) values ($1, $2, $3, $4)"
	e.expect(t, "", deal, g, u[0], "profession", 1001)
	e.expect(t, "player_cards_pkey", deal, g, u[0], "profession", 1002)
	// Aynı kart oyunda ikinci kez dağıtılamaz.
	e.expect(t, "player_cards_card_once_per_game_key", deal, g, u[1], "profession", 1001)
	// Kart, yuvanın kategorisinde olmalı.
	e.expect(t, "player_cards_card_fkey", deal, g, u[1], "health", 1002)
	e.expect(t, "player_cards_card_fkey", deal, g, u[1], "profession", 9999)
	e.expect(t, "player_cards_participant_fkey", deal, g, extra, "profession", 1002)
	e.expect(t, "", deal, g, u[1], "profession", 1002)

	reveal := "update player_cards set revealed_round = $3, reveal_method = $4, revealed_at = $5 where game_id = $1 and user_id = $2 and category = 'profession'"
	e.expect(t, "player_cards_reveal_consistent", reveal, g, u[0], 1, "player", nil)
	e.expect(t, "player_cards_reveal_method_valid", reveal, g, u[0], 1, "magic", time.Now())
	e.expect(t, "player_cards_revealed_round_range", reveal, g, u[0], 0, "player", time.Now())
	e.expect(t, "", reveal, g, u[0], 1, "player", time.Now())

	// Katalogdaki kart, dağıtılmışken silinemez (geçmiş korunur).
	e.asMigrator(t, "player_cards_card_fkey", "delete from card_definitions where id = 1001")
}

func TestIntegrationSchemaVotesResultsActions(t *testing.T) {
	e := schemaEnv(t)
	f := e.newGame(t)
	g, u := f.game, f.users
	outsider := e.newUser(t, "Yabancı")

	vote := "insert into votes (game_id, round, kind, voter_id, target_id) values ($1, $2, $3, $4, $5)"
	e.expect(t, "", vote, g, 1, "regular", u[0], u[1])
	// Oy değiştirilemez / tekrar edilemez.
	e.expect(t, "votes_pkey", vote, g, 1, "regular", u[0], u[2])
	e.expect(t, "votes_not_self", vote, g, 1, "regular", u[1], u[1])
	e.expect(t, "votes_kind_valid", vote, g, 1, "secret", u[1], u[0])
	e.expect(t, "votes_round_range", vote, g, 0, "regular", u[1], u[0])
	e.expect(t, "votes_voter_fkey", vote, g, 1, "regular", outsider, u[0])
	e.expect(t, "votes_target_fkey", vote, g, 1, "regular", u[1], outsider)
	// Aynı turda eşitlik oyu ayrı türdür.
	e.expect(t, "", vote, g, 1, "tiebreak", u[0], u[2])

	res := "insert into round_results (game_id, round, eliminated_id, method, tally, lottery_candidates) values ($1, $2, $3, $4, $5, $6)"
	tally := `[{"user_id": "` + u[1] + `", "votes": 3}]`
	e.expect(t, "", res, g, 1, u[1], "vote", tally, nil)
	e.expect(t, "round_results_pkey", res, g, 1, u[2], "vote", tally, nil)
	e.expect(t, "round_results_method_valid", res, g, 2, u[2], "dice", tally, nil)
	e.expect(t, "round_results_eliminated_unless_left", res, g, 2, nil, "vote", tally, nil)
	e.expect(t, "round_results_tally_array", res, g, 2, u[2], "vote", `{"a": 1}`, nil)
	e.expect(t, "round_results_lottery_candidates", res, g, 2, u[2], "lottery", tally, nil)
	e.expect(t, "round_results_lottery_candidates", res, g, 2, u[2], "lottery", tally, []string{u[2]})
	e.expect(t, "round_results_lottery_candidates", res, g, 2, u[2], "vote", tally, []string{u[2], u[3]})
	e.expect(t, "round_results_eliminated_fkey", res, g, 2, outsider, "vote", tally, nil)
	e.expect(t, "", res, g, 2, u[2], "lottery", tally, []string{u[2], u[3]})

	act := "insert into game_actions (game_id, actor_id, action_id, type, payload, state_version) values ($1, $2, $3, $4, $5, $6)"
	aid := "6f1c2c1e-6a0b-4b39-9d6e-0c5f8b6f6a01"
	e.expect(t, "", act, g, u[0], aid, "vote", `{"target_user_id": "x"}`, 3)
	// Tekrarlanan istek: aynı (oyun, oyuncu, action_id).
	e.expect(t, "game_actions_idempotency_key", act, g, u[0], aid, "vote", `{}`, 4)
	// Başka oyuncu aynı action_id'yi kullanabilir.
	e.expect(t, "", act, g, u[1], aid, "vote", `{}`, 4)
	// Sistem eylemleri (actor yok) tekrarlanabilir; sürüm denetimi korur.
	e.expect(t, "", act, g, nil, aid, "phase_timeout", `{}`, 5)
	e.expect(t, "", act, g, nil, aid, "phase_timeout", `{}`, 6)
	e.expect(t, "game_actions_type_valid", act, g, u[0], "6f1c2c1e-6a0b-4b39-9d6e-0c5f8b6f6a02", "cheat", `{}`, 7)
	e.expect(t, "game_actions_payload_object", act, g, u[0], "6f1c2c1e-6a0b-4b39-9d6e-0c5f8b6f6a03", "vote", `[1]`, 7)
	e.expect(t, "game_actions_state_version_positive", act, g, u[0], "6f1c2c1e-6a0b-4b39-9d6e-0c5f8b6f6a04", "vote", `{}`, 0)
	e.expect(t, "game_actions_actor_fkey", act, g, outsider, aid, "vote", `{}`, 7)

	// Oyun silinirse (yalnızca bakım) bağlı kayıtlar da gider.
	e.expect(t, "", "update rooms set current_game_id = null where id = $1", f.room)
	e.expect(t, "", "delete from games where id = $1", g)
	for _, table := range []string{"game_participants", "player_cards", "votes", "round_results", "game_actions"} {
		var n int
		_ = e.appPool.QueryRow(context.Background(), "select count(*) from "+table+" where game_id = $1", g).Scan(&n)
		if n != 0 {
			t.Errorf("%s oyunla silinmeli", table)
		}
	}
}
