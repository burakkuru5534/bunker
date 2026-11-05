package maintenance

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/testdb"
)

type presenceSet map[string]bool

func (p presenceSet) Connected(id string) bool { return p[id] }

type env struct {
	t  *testing.T
	db *testdb.Env
}

func (e env) exec(sql string, args ...any) {
	e.t.Helper()
	if _, err := e.db.Migrate.Exec(context.Background(), sql, args...); err != nil {
		e.t.Fatalf("%.60s: %v", sql, err)
	}
}

func (e env) id(sql string, args ...any) string {
	e.t.Helper()
	var id string
	if err := e.db.Migrate.QueryRow(context.Background(), sql, args...).Scan(&id); err != nil {
		e.t.Fatalf("%.60s: %v", sql, err)
	}
	return id
}

func (e env) count(sql string, args ...any) int64 {
	e.t.Helper()
	var n int64
	if err := e.db.Migrate.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		e.t.Fatalf("%.60s: %v", sql, err)
	}
	return n
}

func (e env) user(nick string) string {
	return e.id(`insert into users (nickname) values ($1) returning id::text`, nick)
}

// room: oda + aktif üyeler; updated_at verilen kadar geride.
func (e env) room(code, status string, age time.Duration, members ...string) string {
	id := e.id(`insert into rooms (invite_code, host_user_id, status, updated_at) values ($1, $2, $3, now() - $4::interval)
		returning id::text`, code, members[0], status, age.String())
	for i, m := range members {
		e.exec(`insert into room_members (room_id, user_id, nickname, role) values ($1, $2, $3, 'player')`, id, m, "Uye"+string(rune('A'+i)))
	}
	return id
}

func newJob(t *testing.T, db *testdb.Env, presence presenceSet) *Job {
	return New(db.DB, presence, DefaultRetention, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
}

func TestIntegrationRunOnce(t *testing.T) {
	db := testdb.New(t)
	e := env{t, db}
	u1, u2, u3, u4, u5 := e.user("Ali"), e.user("Ayse"), e.user("Can"), e.user("Deniz"), e.user("Ece")

	// Sohbet: 31 günlük silinir, 29 günlük kalır.
	chatRoom := e.room("CHATAA", "lobby", time.Minute, u1)
	e.exec(`insert into chat_messages (room_id, user_id, client_msg_id, body, created_at) values
		($1, $2, gen_random_uuid(), 'eski', now() - interval '31 days'),
		($1, $2, gen_random_uuid(), 'yeni', now() - interval '29 days')`, chatRoom, u1)

	// Oturumlar: süresi dolmuş silinir.
	e.exec(`insert into sessions (user_id, token_hash, expires_at, created_at) values
		($1, sha256('a'::bytea), now() - interval '1 hour', now() - interval '2 days'),
		($1, sha256('b'::bytea), now() + interval '1 day', now())`, u1)

	// Terk edilmiş: 7 saat hareketsiz lobi (kapanır), bağlı üyesi olan (kalır),
	// 1 saatlik (kalır), oyundaki (kalır), bitmiş (kapanır).
	abandoned := e.room("ABNDAA", "lobby", 7*time.Hour, u2)
	connected := e.room("CNNNAA", "lobby", 7*time.Hour, u3)
	recent := e.room("RCNTAA", "lobby", time.Hour, u4)
	finished := e.room("FNSHAA", "finished", 8*time.Hour, u5)

	// Eski kapalı oda: silinir; oyunu ve raporu kalır (oda bağı boşalır).
	oldClosed := e.id(`insert into rooms (invite_code, host_user_id, status, closed_at) values ('XNDCAA', $1, 'closed', now() - interval '8 days')
		returning id::text`, u1)
	e.exec(`insert into scenarios (id, title, disaster, duration_label, shelter_features, outside_world) values
		('meteor', 'Göktaşı', 'Uzun bir felaket metni burada yer alır, en az yirmi karakter.', '2 yıl', array['a','b','c'],
		 'Dış dünya hakkında yeterince uzun bir açıklama metni.')`)
	game := e.id(`insert into games (room_id, scenario_id, capacity, total_rounds, phase, phase_deadline_at, finished_at, end_reason)
		values ($1, 'meteor', 1, 3, 'finished', null, now(), 'capacity_reached') returning id::text`, oldClosed)
	report := e.id(`insert into reports (reporter_id, reported_id, room_id, reason) values ($1, $2, $3, 'spam') returning id::text`, u1, u2, oldClosed)
	newClosed := e.id(`insert into rooms (invite_code, host_user_id, status, closed_at) values ('NEWCAA', $1, 'closed', now() - interval '1 day')
		returning id::text`, u1)

	// Raporlar: 181 gün önce sonuçlanmış silinir; 179 gün ve açık olan kalır.
	e.exec(`insert into reports (reporter_id, reported_id, reason, status, resolved_at) values
		($1, $2, 'spam', 'dismissed', now() - interval '181 days'),
		($1, $3, 'spam', 'actioned', now() - interval '179 days')`, u3, u4, u5)

	job := newJob(t, db, presenceSet{u3: true})
	st, err := job.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := Stats{ChatMessages: 1, Sessions: 1, AbandonedRooms: 2, ClosedRooms: 1, Reports: 1}
	if st != want {
		t.Fatalf("istatistik %+v, beklenen %+v", st, want)
	}
	if e.count(`select count(*) from chat_messages where body = 'yeni'`) != 1 || e.count(`select count(*) from chat_messages`) != 1 {
		t.Fatal("sohbet saklama yanlış")
	}
	if e.count(`select count(*) from sessions`) != 1 {
		t.Fatal("oturum temizliği yanlış")
	}
	for id, wantStatus := range map[string]string{abandoned: "closed", connected: "lobby", recent: "lobby", finished: "closed"} {
		var status string
		if err := db.Migrate.QueryRow(context.Background(), `select status from rooms where id = $1`, id).Scan(&status); err != nil || status != wantStatus {
			t.Errorf("oda %s: %s (beklenen %s) %v", id, status, wantStatus, err)
		}
	}
	// Kapanan odanın üyeleri serbest: yeni odaya girebilirler (tek aktif oda kısıtı).
	if e.count(`select count(*) from room_members where room_id = $1 and left_at is null`, abandoned) != 0 {
		t.Fatal("kapanan odada aktif üye kaldı")
	}
	e.room("YENAAA", "lobby", 0, u2) // kısıt ihlali olsaydı burada hata verirdi
	// Eski kapalı oda silindi; oyun ve rapor duruyor, bağları boş.
	if e.count(`select count(*) from rooms where id = $1`, oldClosed) != 0 || e.count(`select count(*) from rooms where id = $1`, newClosed) != 1 {
		t.Fatal("kapalı oda silme yanlış")
	}
	if e.count(`select count(*) from games where id = $1 and room_id is null`, game) != 1 ||
		e.count(`select count(*) from reports where id = $1 and room_id is null`, report) != 1 {
		t.Fatal("oyun veya rapor oda silinince kayboldu")
	}
	if e.count(`select count(*) from reports`) != 2 || e.count(`select count(*) from reports where status = 'open'`) != 1 {
		t.Fatal("rapor saklama yanlış")
	}
	// İkinci çalıştırma: yapılacak iş yok (idempotent).
	if st, err := job.RunOnce(context.Background()); err != nil || st != (Stats{}) {
		t.Fatalf("ikinci çalıştırma %+v %v", st, err)
	}
}

// Partiler: büyük birikim parti parti silinir, çalıştırma başına üst sınırda durur.
func TestIntegrationBatchesAndCap(t *testing.T) {
	db := testdb.New(t)
	e := env{t, db}
	u := e.user("Ali")
	r := e.room("CHATBB", "lobby", time.Minute, u)
	e.exec(`insert into chat_messages (room_id, user_id, client_msg_id, body, created_at)
		select $1, $2, gen_random_uuid(), 'm', now() - interval '40 days' from generate_series(1, 25)`, r, u)
	job := newJob(t, db, nil)
	job.ret.Batch, job.ret.MaxBatches = 4, 3
	st, err := job.RunOnce(context.Background())
	if err != nil || st.ChatMessages != 12 {
		t.Fatalf("ilk çalıştırma %d (4×3 beklenir) %v", st.ChatMessages, err)
	}
	for range 2 {
		_, _ = job.RunOnce(context.Background())
	}
	if n := e.count(`select count(*) from chat_messages`); n != 0 {
		t.Fatalf("%d mesaj kaldı", n)
	}
}

// Kapanmadan hemen önce odada hareket olursa (kilitli yeniden denetim) kapanmaz.
func TestIntegrationAbandonedRecheck(t *testing.T) {
	db := testdb.New(t)
	e := env{t, db}
	u := e.user("Ali")
	r := e.room("RECHAA", "lobby", 7*time.Hour, u)
	job := newJob(t, db, nil)
	cutoff := time.Now().Add(-DefaultRetention.AbandonedRoom)
	e.exec(`update rooms set updated_at = now() where id = $1`, r) // seçimden sonra biri katıldı
	if ok, err := job.closeRoom(context.Background(), r, cutoff); err != nil || ok {
		t.Fatalf("hareketlenen oda kapandı: %v %v", ok, err)
	}
	e.exec(`update rooms set status = 'in_game', updated_at = now() - interval '7 hours' where id = $1`, r)
	if ok, _ := job.closeRoom(context.Background(), r, cutoff); ok {
		t.Fatal("oyundaki oda kapandı")
	}
}

func TestStartStop(t *testing.T) {
	db := testdb.New(t)
	job := newJob(t, db, nil)
	job.Start(10*time.Millisecond, 10*time.Millisecond)
	job.Start(time.Hour, time.Hour) // ikinci Start etkisiz
	time.Sleep(50 * time.Millisecond)
	if err := job.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := job.Stop(context.Background()); err != nil { // ikinci Stop güvenli
		t.Fatal(err)
	}
}

// SAFE-014: 90 gündür kullanılmayan, oturumu ve aktif odası kalmamış hesap
// DELETE /v1/me ile aynı şekilde anonimleştirilir; diğerlerine dokunulmaz.
func TestIntegrationInactiveUsers(t *testing.T) {
	db := testdb.New(t)
	e := env{t, db}
	old := func(nick string, days int) string {
		return e.id(`insert into users (nickname, created_at, last_seen_at)
			values ($1, now() - $2::interval, now() - $2::interval) returning id::text`, nick, fmt.Sprintf("%d days", days))
	}
	orphan := old("Yetim", 91)          // silinir
	expired := old("Suresi", 95)        // süresi dolmuş oturumu var: önce oturum, sonra hesap silinir
	recent := old("Yakin", 89)          // 90 gün dolmadı
	withSession := old("Oturumlu", 120) // geçerli oturumu var (ör. saat farkı): kalır
	inRoom := old("Odada", 120)         // aktif odası var: kalır
	friend := e.user("Arkadas")
	already := old("Silinmis", 200)
	e.exec(`update users set deleted_at = now() - interval '1 day' where id = $1`, already)

	e.exec(`insert into sessions (user_id, token_hash, created_at, expires_at) values
		($1, sha256('x'::bytea), now() - interval '95 days', now() - interval '5 days'),
		($2, sha256('y'::bytea), now(), now() + interval '1 day')`, expired, withSession)
	r := e.room("KNACAA", "in_game", 0, inRoom, friend)
	e.exec(`insert into room_members (room_id, user_id, nickname, role, left_at) values ($1, $2, 'EskiAd', 'player', now() - interval '100 days')`,
		e.room("XYDRAA", "lobby", 0, e.user("Evsahibi")), orphan)
	e.exec(`insert into chat_messages (room_id, user_id, client_msg_id, body) values ($1, $2, gen_random_uuid(), 'merhaba')`, r, inRoom)
	e.exec(`insert into blocks (blocker_id, blocked_id) values ($1, $2), ($2, $1), ($3, $2)`, orphan, friend, recent)

	job := newJob(t, db, nil)
	st, err := job.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.InactiveUsers != 2 || st.Sessions != 1 {
		t.Fatalf("istatistik %+v; 2 hesap ve 1 oturum beklenir", st)
	}
	deleted := func(id string) bool {
		return e.count(`select count(*) from users where id = $1 and deleted_at is not null and nickname = 'Silinmiş oyuncu'`, id) == 1
	}
	for id, want := range map[string]bool{orphan: true, expired: true, recent: false, withSession: false, inRoom: false, friend: false} {
		if deleted(id) != want {
			t.Errorf("kullanıcı %s silindi=%v, beklenen %v", id, !want, want)
		}
	}
	// Anonimleştirme DELETE /v1/me ile aynı: geçmiş üyelikteki ad, engellemeler.
	if e.count(`select count(*) from room_members where user_id = $1 and nickname = 'Silinmiş oyuncu'`, orphan) != 1 {
		t.Error("geçmiş üyelikteki takma ad anonimleşmedi")
	}
	if e.count(`select count(*) from blocks where blocker_id = $1 or blocked_id = $1`, orphan) != 0 ||
		e.count(`select count(*) from blocks`) != 1 {
		t.Error("engellemeler yanlış silindi")
	}
	if e.count(`select count(*) from chat_messages where user_id = $1`, inRoom) != 1 {
		t.Error("odadaki kullanıcının mesajı silindi")
	}
	// Daha önce silinmiş hesabın silinme zamanı değişmez (yeniden işlenmez).
	if e.count(`select count(*) from users where id = $1 and deleted_at < now() - interval '12 hours'`, already) != 1 {
		t.Error("silinmiş hesap yeniden işlendi")
	}
	if st, err := job.RunOnce(context.Background()); err != nil || st != (Stats{}) {
		t.Fatalf("ikinci çalıştırma %+v %v", st, err)
	}
}

// Seçimden sonra hesap kullanılırsa (kilitli yeniden denetim) silinmez.
func TestIntegrationInactiveUserRecheck(t *testing.T) {
	db := testdb.New(t)
	e := env{t, db}
	u := e.id(`insert into users (nickname, last_seen_at) values ('Geri', now() - interval '100 days') returning id::text`)
	job := newJob(t, db, nil)
	cutoff := time.Now().Add(-DefaultRetention.InactiveUser)
	e.exec(`update users set last_seen_at = now() where id = $1`, u)
	if ok, err := job.deleteInactiveUser(context.Background(), u, cutoff); err != nil || ok {
		t.Fatalf("kullanılan hesap silindi: %v %v", ok, err)
	}
	e.exec(`update users set last_seen_at = now() - interval '100 days' where id = $1`, u)
	e.exec(`insert into sessions (user_id, token_hash, expires_at) values ($1, sha256('z'::bytea), now() + interval '1 day')`, u)
	if ok, _ := job.deleteInactiveUser(context.Background(), u, cutoff); ok {
		t.Fatal("oturumu olan hesap silindi")
	}
	e.exec(`delete from sessions where user_id = $1`, u)
	if ok, err := job.deleteInactiveUser(context.Background(), u, cutoff); err != nil || !ok {
		t.Fatalf("aday hesap silinmedi: %v %v", ok, err)
	}
}

// Uygun olmayan hesaplar aday sorgusuna girmemeli: girseydi en eski oldukları
// için partiyi doldurur, "ilerleme yok" ile çalıştırma erken biterdi.
func TestIntegrationInactiveUsersSkipIneligibleCandidates(t *testing.T) {
	db := testdb.New(t)
	e := env{t, db}
	user := func(nick string, days int) string {
		return e.id(`insert into users (nickname, last_seen_at) values ($1, now() - $2::interval) returning id::text`,
			nick, fmt.Sprintf("%d days", days))
	}
	gone := user("Silinmis", 300)
	e.exec(`update users set deleted_at = now() - interval '200 days' where id = $1`, gone)
	withSession := user("Oturumlu", 250)
	e.exec(`insert into sessions (user_id, token_hash, expires_at) values ($1, sha256('s'::bytea), now() + interval '1 day')`, withSession)
	e.room("SKPAAA", "in_game", 0, user("Odada", 200))
	for i, d := range []int{120, 110, 100} {
		user(fmt.Sprintf("Aday%d", i), d)
	}

	job := newJob(t, db, nil)
	job.ret.Batch, job.ret.MaxBatches = 1, 5
	st, err := job.RunOnce(context.Background())
	if err != nil || st.InactiveUsers != 3 {
		t.Fatalf("%d hesap silindi (3 beklenir) %v", st.InactiveUsers, err)
	}
}
