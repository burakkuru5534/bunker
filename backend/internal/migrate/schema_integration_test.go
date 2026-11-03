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
	for _, want := range []string{"users", "sessions", "rooms", "room_members", "blocks"} {
		if !seen[want] {
			t.Errorf("%s tablosu yok (görülen: %s)", want, strings.Join(keys(seen), ","))
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
