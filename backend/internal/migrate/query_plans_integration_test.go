package migrate

// DB-004: sık çalışan sorguların planları. Gerçekçi hacimde veriyle
// (on binlerce satır) her sorgu uygulama rolüyle EXPLAIN edilir; büyük
// tablolarda tam tablo taraması (Seq Scan) olmamalı ve beklenen indeks
// kullanılmalıdır. Sorgular docs/queries.md ile aynıdır.

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

// planQuery, katalogdaki bir sorgudur.
type planQuery struct {
	name  string
	sql   string
	args  func(s *seedIDs) []any
	index string // planda bulunması gereken indeks ("|" ile alternatifler)
}

type seedIDs struct {
	tokenHash        []byte
	user, room, game string
	inviteCode       string
	actor, action    string
	cutoff           time.Time
}

// hotQueries: uygulamanın sıcak yolu ve bakım işleri (docs/queries.md).
var hotQueries = []planQuery{
	{"oturum doğrulama", `select s.user_id, s.expires_at, s.revoked_at, u.nickname, u.deleted_at
		from sessions s join users u on u.id = s.user_id where s.token_hash = $1`,
		func(s *seedIDs) []any { return []any{s.tokenHash} }, "sessions_token_hash_key"},
	{"kullanıcının aktif odası", `select room_id from room_members where user_id = $1 and left_at is null`,
		func(s *seedIDs) []any { return []any{s.user} }, "room_members_one_active_room_key"},
	{"davet koduyla oda (kilitli)", `select id, status, capacity, current_game_id from rooms
		where invite_code = $1 and status <> 'closed' for update`,
		func(s *seedIDs) []any { return []any{s.inviteCode} }, "rooms_invite_code_open_key"},
	{"oda üyeleri", `select m.user_id, m.nickname, m.role, m.is_ready, m.joined_at from room_members m
		where m.room_id = $1 and m.left_at is null order by m.joined_at`,
		func(s *seedIDs) []any { return []any{s.room} }, "room_members_pkey|room_members_active_nickname_key"},
	{"üyelik/atılma kontrolü", `select left_at, kicked_at from room_members where room_id = $1 and user_id = $2`,
		func(s *seedIDs) []any { return []any{s.room, s.user} }, "room_members_pkey"},
	{"oyun kilidi", `select phase, round, turn_seat, phase_deadline_at, state_version from games where id = $1 for update`,
		func(s *seedIDs) []any { return []any{s.game} }, "games_pkey"},
	{"katılımcılar", `select p.user_id, u.nickname, p.seat, p.status, p.eliminated_round
		from game_participants p join users u on u.id = p.user_id where p.game_id = $1 order by p.seat`,
		func(s *seedIDs) []any { return []any{s.game} }, "game_participants_pkey|game_participants_seat_key"},
	{"oyunun kartları", `select pc.user_id, pc.category, c.text, pc.revealed_round from player_cards pc
		join card_definitions c on c.id = pc.card_id where pc.game_id = $1`,
		func(s *seedIDs) []any { return []any{s.game} }, "player_cards_pkey|player_cards_card_once_per_game_key"},
	{"oy sayımı", `select target_id, count(*) from votes where game_id = $1 and round = $2 and kind = 'regular' group by target_id`,
		func(s *seedIDs) []any { return []any{s.game, 1} }, "votes_pkey"},
	{"tur sonuçları", `select round, eliminated_id, method, tally from round_results where game_id = $1 order by round`,
		func(s *seedIDs) []any { return []any{s.game} }, "round_results_pkey"},
	{"idempotency", `select state_version from game_actions where game_id = $1 and actor_id = $2 and action_id = $3`,
		func(s *seedIDs) []any { return []any{s.game, s.actor, s.action} }, "game_actions_idempotency_key"},
	{"son 50 mesaj", `select id, user_id, body, created_at from chat_messages where room_id = $1
		order by created_at desc, id desc limit 50`,
		func(s *seedIDs) []any { return []any{s.room} }, "chat_messages_room_created_idx"},
	{"göndereni engelleyenler", `select blocker_id from blocks where blocked_id = $1`,
		func(s *seedIDs) []any { return []any{s.user} }, "blocks_blocked_id_idx"},
	{"engellenenler listesi", `select b.blocked_id, u.nickname, b.created_at from blocks b
		join users u on u.id = b.blocked_id where b.blocker_id = $1 order by b.created_at desc`,
		func(s *seedIDs) []any { return []any{s.user} }, "blocks_pkey"},
	{"kullanıcının oturumları (iptal)", `update sessions set revoked_at = now() where user_id = $1 and revoked_at is null`,
		func(s *seedIDs) []any { return []any{s.user} }, "sessions_user_id_idx"},
	{"zamanlayıcı kurtarma", `select id, phase_deadline_at, state_version from games
		where phase <> 'finished' order by phase_deadline_at`,
		func(*seedIDs) []any { return nil }, "games_active_deadline_idx"},
	// Toplu silmeler "id = any(array(...))" biçimindedir: alt sorgu indeksle
	// en fazla N kimlik seçer, silme birincil anahtarla yapılır ("in (select)"
	// biçimi planlayıcıyı tüm tabloyu taramaya itebiliyor).
	{"sohbet saklama (toplu)", `delete from chat_messages where id = any(array(
		select id from chat_messages where created_at < $1 order by created_at limit 1000))`,
		func(s *seedIDs) []any { return []any{s.cutoff} }, "chat_messages_created_at_idx"},
	{"süresi dolan oturumlar (toplu)", `delete from sessions where id = any(array(
		select id from sessions where expires_at < now() order by expires_at limit 1000))`,
		func(*seedIDs) []any { return nil }, "sessions_expires_at_idx"},
	{"terk edilmiş açık odalar", `select id from rooms where status <> 'closed' and status <> 'in_game'
		and updated_at < $1 order by updated_at limit 100`,
		func(s *seedIDs) []any { return []any{s.cutoff} }, "rooms_open_updated_at_idx"},
	{"eski kapalı odalar (toplu)", `delete from rooms where id = any(array(
		select id from rooms where status = 'closed' and closed_at < $1 order by closed_at limit 500))`,
		func(s *seedIDs) []any { return []any{s.cutoff} }, "rooms_closed_at_idx"},
	{"sonuçlanmış raporlar (toplu)", `delete from reports where id = any(array(
		select id from reports where status in ('actioned', 'dismissed') and resolved_at < $1
		order by resolved_at limit 500))`,
		func(s *seedIDs) []any { return []any{s.cutoff} }, "reports_resolved_at_idx"},
	{"moderasyon kuyruğu", `select id, reason, created_at from reports where status in ('open', 'reviewing')
		order by created_at limit 50`,
		func(*seedIDs) []any { return nil }, "reports_queue_idx"},
}

// seed, yönetici bağlantısıyla gerçekçi hacimde veri yükler ve ANALYZE eder.
func seed(t *testing.T, e *env) *seedIDs {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	s := e.schema
	stmts := []string{
		`set search_path = ` + s,
		`insert into scenarios (id, title, disaster, duration_label, shelter_features, outside_world)
		 values ('s1', 'Nükleer Kış', repeat('k', 30), '3 yıl', array['a','b','c'], repeat('d', 30))`,
		`insert into card_definitions (id, category, text, balance)
		 select c.n * 1000 + i, c.cat, c.cat || ' ' || i, (i % 3) - 1
		   from (values (1, 'profession'), (2, 'health'), (3, 'hobby'), (4, 'item'), (5, 'secret')) c (n, cat),
		        generate_series(1, 30) i`,
		// 20.000 kullanıcı, her birine bir oturum (yarısının süresi dolmuş).
		// Geçici tablolar (rn → id) eşleştirmeyi doğrusal yapar.
		`insert into users (nickname) select 'oyuncu' || i from generate_series(1, 20000) i`,
		`create temp table tu on commit drop as select id, row_number() over (order by id) rn from users`,
		`insert into sessions (user_id, token_hash, created_at, expires_at)
		 select id, sha256(id::text::bytea), now() - interval '100 days' * (rn % 2),
		        now() - interval '100 days' * (rn % 2) + interval '90 days' from tu`,
		// 3.000 oda: %80'i kapalı.
		`insert into rooms (invite_code, host_user_id, status, closed_at, updated_at)
		 select translate(upper(substr(md5(i::text), 1, 6)), '01', '23'), tu.id,
		        case when i % 5 = 0 then 'lobby' else 'closed' end,
		        case when i % 5 = 0 then null else now() - interval '1 day' * (i % 60) end,
		        now() - interval '1 hour' * (i % 2000)
		   from generate_series(1, 3000) i join tu on tu.rn = i`,
		`create temp table tr on commit drop as select id, status, row_number() over (order by id) rn from rooms`,
		// Her odaya 4 üye; açık odalarda aktif, kapalılarda ayrılmış.
		`insert into room_members (room_id, user_id, nickname, role, left_at)
		 select tr.id, tu.id, 'oyuncu' || tr.rn || '_' || k, 'player', case when tr.status = 'closed' then now() end
		   from tr cross join generate_series(0, 3) k join tu on tu.rn = (tr.rn * 4 + k) % 20000 + 1`,
		// 2.000 oyun, %95'i bitmiş; her birinde 8 katılımcı, 5'er kart.
		`insert into games (room_id, scenario_id, capacity, total_rounds, round, phase, phase_deadline_at, finished_at, end_reason)
		 select tr.id, 's1', 2, 6, 1,
		        case when i % 20 = 0 then 'voting' else 'finished' end,
		        case when i % 20 = 0 then now() + interval '30 seconds' end,
		        case when i % 20 = 0 then null else now() end,
		        case when i % 20 = 0 then null else 'capacity_reached' end
		   from generate_series(1, 2000) i join tr on tr.rn = i`,
		`create temp table tg on commit drop as select id, row_number() over (order by id) rn from games`,
		`insert into game_participants (game_id, user_id, seat)
		 select tg.id, tu.id, k from tg cross join generate_series(0, 7) k join tu on tu.rn = (tg.rn * 8 + k) % 20000 + 1`,
		`insert into player_cards (game_id, user_id, category, card_id)
		 select p.game_id, p.user_id, c.cat, c.n * 1000 + p.seat + 1
		   from game_participants p,
		        (values (1, 'profession'), (2, 'health'), (3, 'hobby'), (4, 'item'), (5, 'secret')) c (n, cat)`,
		`insert into votes (game_id, round, kind, voter_id, target_id)
		 select a.game_id, 1, 'regular', a.user_id, b.user_id
		   from game_participants a join game_participants b on b.game_id = a.game_id and b.seat = (a.seat + 1) % 8`,
		`insert into round_results (game_id, round, eliminated_id, method)
		 select game_id, 1, user_id, 'vote' from game_participants where seat = 0`,
		`insert into game_actions (game_id, actor_id, action_id, type, state_version)
		 select game_id, user_id, gen_random_uuid(), 'vote', 1 from game_participants
		 union all
		 select game_id, user_id, gen_random_uuid(), 'reveal_card', 2 from game_participants`,
		// 100.000 mesaj; ~%5'i 30 günden eski.
		`create temp table tm on commit drop as select room_id, user_id, row_number() over () rn from room_members`,
		`insert into chat_messages (room_id, user_id, client_msg_id, body, created_at)
		 select tm.room_id, tm.user_id, gen_random_uuid(), 'mesaj ' || i,
		        now() - case when i % 20 = 0 then interval '40 days' else interval '1 minute' * (i % 10000) end
		   from generate_series(1, 100000) i join tm on tm.rn = i % 12000 + 1`,
		`insert into blocks (blocker_id, blocked_id)
		 select a.id, b.id from tu a join tu b on b.rn = a.rn + 1 where a.rn <= 5000`,
		`insert into reports (reporter_id, reported_id, room_id, reason, status, resolved_at)
		 select a.id, b.id, tr.id, 'spam', case when a.rn % 10 = 0 then 'open' else 'dismissed' end,
		        case when a.rn % 10 = 0 then null else now() end
		   from tu a join tu b on b.rn = a.rn + 7 join tr on tr.rn = a.rn where a.rn <= 2000`,
	}
	conn, err := e.admin.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range stmts {
		if _, err := conn.Exec(ctx, q); err != nil {
			_ = conn.Rollback(ctx)
			t.Fatalf("%.80s: %v", q, err)
		}
	}
	if err := conn.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	// Yalnızca bu testin şemasındaki tablolar analiz edilir. Yalın "analyze"
	// tüm veritabanını tarar; CI'da paralel paketlerin aynı anda kurup
	// sildiği şemalara kilit alıp deadlock'a yol açıyordu.
	var tables string
	if err := e.admin.QueryRow(ctx, `select string_agg(format('%I.%I', schemaname, tablename), ', ')
		from pg_tables where schemaname = $1`, s).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if _, err := e.admin.Exec(ctx, "analyze "+tables); err != nil {
		t.Fatal(err)
	}

	ids := &seedIDs{cutoff: time.Now().Add(-30 * 24 * time.Hour)}
	q := func(sql string, dst ...any) {
		if err := e.admin.QueryRow(ctx, sql).Scan(dst...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	q(`select m.user_id, m.room_id, r.invite_code from `+s+`.room_members m join `+s+`.rooms r on r.id = m.room_id
	   where m.left_at is null limit 1`, &ids.user, &ids.room, &ids.inviteCode)
	q(`select token_hash from `+s+`.sessions where user_id = '`+ids.user+`'`, &ids.tokenHash)
	q(`select game_id, actor_id, action_id from `+s+`.game_actions limit 1`, &ids.game, &ids.actor, &ids.action)
	return ids
}

type planNode struct {
	NodeType     string     `json:"Node Type"`
	RelationName string     `json:"Relation Name"`
	IndexName    string     `json:"Index Name"`
	Plans        []planNode `json:"Plans"`
}

func walk(n planNode, fn func(planNode)) {
	fn(n)
	for _, c := range n.Plans {
		walk(c, fn)
	}
}

// smallTables: tam taranması sorun olmayan küçük kataloglar.
var smallTables = []string{"scenarios", "card_definitions"}

func TestIntegrationQueryPlans(t *testing.T) {
	e := schemaEnv(t)
	ids := seed(t, e)
	ctx := context.Background()

	for _, hq := range hotQueries {
		t.Run(hq.name, func(t *testing.T) {
			// EXPLAIN (ANALYZE yok) değişiklik yapmaz; DML'ler de yalnızca planlanır.
			tx, err := e.appPool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(ctx) }()
			var raw []byte
			if err := tx.QueryRow(ctx, "explain (format json) "+hq.sql, hq.args(ids)...).Scan(&raw); err != nil {
				t.Fatalf("explain: %v", err)
			}
			var plans []struct{ Plan planNode }
			if err := json.Unmarshal(raw, &plans); err != nil || len(plans) != 1 {
				t.Fatalf("plan çözülemedi: %v", err)
			}
			var seq, used []string
			walk(plans[0].Plan, func(n planNode) {
				if n.NodeType == "Seq Scan" && !slices.Contains(smallTables, n.RelationName) {
					seq = append(seq, n.RelationName)
				}
				if n.IndexName != "" {
					used = append(used, n.IndexName)
				}
			})
			if len(seq) > 0 {
				t.Errorf("tam tablo taraması: %s", strings.Join(seq, ", "))
			}
			if !slices.ContainsFunc(strings.Split(hq.index, "|"), func(ix string) bool { return slices.Contains(used, ix) }) {
				t.Errorf("%s kullanılmalı; kullanılan: %s", hq.index, fmt.Sprint(used))
			}
		})
	}
}

// fkWithoutIndexAllowed: destekleyici indeksi bilerek olmayan yabancı
// anahtarlar. Üst satırı silinmeyen/güncellenmeyen ilişkilerde indeks
// yalnızca yazma maliyetidir.
var fkWithoutIndexAllowed = map[string]string{
	"games_scenario_id_fkey":        "katalog satırları silinmez (active=false)",
	"player_cards_card_fkey":        "katalog satırları silinmez",
	"room_members_user_id_fkey":     "kullanıcılar silinmez, anonimleştirilir (SAFE-009)",
	"reports_reporter_id_fkey":      "kullanıcılar silinmez",
	"votes_voter_fkey":              "katılımcılar yalnızca oyunla birlikte silinir",
	"votes_target_fkey":             "katılımcılar yalnızca oyunla birlikte silinir",
	"round_results_eliminated_fkey": "katılımcılar yalnızca oyunla birlikte silinir",
	"rooms_current_game_id_fkey":    "oyunlar yalnızca bakımda silinir",
	"chat_messages_game_id_fkey":    "oyunlar yalnızca bakımda silinir",
}

// Üst satır silinince/ güncellenince PostgreSQL alt tabloyu yabancı anahtar
// sütunlarıyla arar; indeks yoksa tüm tabloyu tarar (EXPLAIN'de görünmez).
func TestIntegrationForeignKeysHaveIndexes(t *testing.T) {
	e := schemaEnv(t)
	rows, err := e.admin.Query(context.Background(), `
		select c.conname::text
		  from pg_constraint c
		  join pg_namespace n on n.oid = c.connamespace
		 where c.contype = 'f' and n.nspname = $1
		   and not exists (
		     select 1 from pg_index i
		      where i.indrelid = c.conrelid and i.indpred is null
		        and (i.indkey::int2[])[0:array_length(c.conkey, 1) - 1] @> c.conkey
		        and (i.indkey::int2[])[0:array_length(c.conkey, 1) - 1] <@ c.conkey)
		 order by 1`, e.schema)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	missing := map[string]bool{}
	for rows.Next() {
		var name string
		_ = rows.Scan(&name)
		missing[name] = true
		if _, ok := fkWithoutIndexAllowed[name]; !ok {
			t.Errorf("%s: destekleyici indeks yok (üst satır silinince tam tarama)", name)
		}
	}
	// İzin listesi eskimesin: listede olup artık indeksi olan / var olmayan anahtar.
	for name := range fkWithoutIndexAllowed {
		if !missing[name] {
			t.Errorf("%s izin listesinde ama indeksi var veya böyle bir anahtar yok; listeden çıkarın", name)
		}
	}
}
