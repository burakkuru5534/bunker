package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
	"github.com/burakkuru5534/bunker/backend/internal/testdb"
)

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func newTestService(t *testing.T) (*Service, *testdb.Env, *clock) {
	t.Helper()
	env := testdb.New(t)
	svc := NewService(env.DB, nil, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	c := &clock{t: time.Now()}
	svc.now = c.now
	return svc, env, c
}

func exec(t *testing.T, db *persistence.DB, sql string, args ...any) {
	t.Helper()
	if err := db.Read(context.Background(), func(q persistence.Querier) error {
		_, err := q.Exec(context.Background(), sql, args...)
		return err
	}); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func scalar[T any](t *testing.T, db *persistence.DB, sql string, args ...any) T {
	t.Helper()
	var v T
	if err := db.Read(context.Background(), func(q persistence.Querier) error {
		return q.QueryRow(context.Background(), sql, args...).Scan(&v)
	}); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return v
}

func isUnauthorized(err error) bool { return apperr.CodeOf(err) == apperr.Unauthorized }

func TestIntegrationCreateGuest(t *testing.T) {
	svc, env, c := newTestService(t)
	ctx := context.Background()

	iss, err := svc.CreateGuest(ctx, "  Ayşe ")
	if err != nil {
		t.Fatal(err)
	}
	if iss.User.Nickname != "Ayşe" || iss.User.ID == "" || len(iss.Token) != TokenLength {
		t.Fatalf("yanıt: %+v", iss)
	}
	if d := iss.ExpiresAt.Sub(c.now()); d < SessionTTL-2*time.Second || d > SessionTTL {
		t.Fatalf("süre 90 gün olmalı: %v", d)
	}
	// Ham token saklanmaz; yalnızca hash.
	hash, _ := HashToken(iss.Token)
	if n := scalar[int](t, env.DB, "select count(*) from sessions where token_hash = $1 and user_id = $2", hash, iss.User.ID); n != 1 {
		t.Fatal("oturum hash ile kaydedilmeli")
	}
	if n := scalar[int](t, env.DB, "select count(*) from sessions where encode(token_hash, 'escape') like '%' || $1 || '%'", iss.Token); n != 0 {
		t.Fatal("ham token veritabanında olmamalı")
	}

	// Geçersiz ad: hiçbir satır oluşmaz.
	before := scalar[int](t, env.DB, "select count(*) from users")
	_, err = svc.CreateGuest(ctx, "Бурак")
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.NicknameInvalid || ae.Fields["nickname"] != apperr.FieldInvalidChars {
		t.Fatalf("NICKNAME_INVALID bekleniyordu: %v", err)
	}
	if after := scalar[int](t, env.DB, "select count(*) from users"); after != before {
		t.Fatal("geçersiz adla kullanıcı oluşmamalı")
	}
}

func TestIntegrationAuthenticate(t *testing.T) {
	svc, env, c := newTestService(t)
	ctx := context.Background()
	iss, _ := svc.CreateGuest(ctx, "Burak")

	p, err := svc.Authenticate(ctx, iss.Token)
	if err != nil || p.UserID != iss.User.ID || p.Nickname != "Burak" || p.SessionID == "" {
		t.Fatalf("doğrulama: %+v %v", p, err)
	}

	other, _ := NewToken()
	for name, raw := range map[string]string{"boş": "", "bozuk": "abc", "bilinmeyen": other.Raw} {
		if _, err := svc.Authenticate(ctx, raw); !isUnauthorized(err) {
			t.Errorf("%s token: UNAUTHORIZED bekleniyordu: %v", name, err)
		}
	}

	// Süresi dolmuş / iptal / silinmiş kullanıcı (önbellek dışı yeni oturumlarla).
	cases := map[string]func(userID, sessionID string){
		"süresi dolmuş": func(_, sid string) {
			exec(t, env.DB, "update sessions set created_at = now() - interval '91 days', expires_at = now() - interval '1 day' where id = $1", sid)
		},
		"iptal": func(_, sid string) { exec(t, env.DB, "update sessions set revoked_at = now() where id = $1", sid) },
		"silinmiş kullanıcı": func(uid, _ string) {
			exec(t, env.DB, "update users set nickname = $2, deleted_at = now() where id = $1", uid, DeletedNickname)
		},
	}
	n := 0
	for name, mutate := range cases {
		n++
		is, _ := svc.CreateGuest(ctx, fmt.Sprintf("Oyuncu%d", n))
		h, _ := HashToken(is.Token)
		sid := scalar[string](t, env.DB, "select id::text from sessions where token_hash = $1", h)
		mutate(is.User.ID, sid)
		if _, err := svc.Authenticate(ctx, is.Token); !isUnauthorized(err) {
			t.Errorf("%s: UNAUTHORIZED bekleniyordu: %v", name, err)
		}
	}

	// Önbellek: iptal veritabanında yapılsa da ≤ 60 sn geçerli kalır, sonra düşer.
	exec(t, env.DB, "update sessions set revoked_at = now() where id = $1", p.SessionID)
	if _, err := svc.Authenticate(ctx, iss.Token); err != nil {
		t.Fatalf("önbellekten gelmeli: %v", err)
	}
	c.advance(CacheTTL + time.Second)
	if _, err := svc.Authenticate(ctx, iss.Token); !isUnauthorized(err) {
		t.Fatalf("önbellek süresi dolunca iptal görülmeli: %v", err)
	}
}

func TestIntegrationTouchAtMostHourly(t *testing.T) {
	svc, env, c := newTestService(t)
	ctx := context.Background()
	iss, _ := svc.CreateGuest(ctx, "Burak")
	lastUsed := func() *time.Time {
		return scalar[*time.Time](t, env.DB, "select last_used_at from sessions where user_id = $1", iss.User.ID)
	}

	if _, err := svc.Authenticate(ctx, iss.Token); err != nil {
		t.Fatal(err)
	}
	first := lastUsed()
	if first == nil {
		t.Fatal("ilk kullanım yazılmalı")
	}
	// Bir saat içinde (önbellek düşse bile) yeniden yazılmaz.
	for range 3 {
		c.advance(20 * time.Minute)
		if _, err := svc.Authenticate(ctx, iss.Token); err != nil {
			t.Fatal(err)
		}
	}
	if !lastUsed().Equal(*first) {
		t.Fatal("bir saat dolmadan yazılmamalı")
	}
	// Veritabanındaki gerçek zaman ileri gitmez; saati ileri alıp eski kaydı taklit et.
	exec(t, env.DB, "update sessions set last_used_at = now() - interval '2 hours' where user_id = $1", iss.User.ID)
	c.advance(2 * time.Minute) // önbellek düşer, eski last_used okunur
	if _, err := svc.Authenticate(ctx, iss.Token); err != nil {
		t.Fatal(err)
	}
	if time.Since(*lastUsed()) > time.Minute {
		t.Fatal("bir saatten eskiyse yeniden yazılmalı")
	}
	if seen := scalar[time.Time](t, env.DB, "select last_seen_at from users where id = $1", iss.User.ID); time.Since(seen) > time.Minute {
		t.Fatal("kullanıcının son görülmesi de güncellenmeli")
	}
}

func TestIntegrationRefresh(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()
	iss, _ := svc.CreateGuest(ctx, "Burak")
	p, _ := svc.Authenticate(ctx, iss.Token)

	next, err := svc.Refresh(ctx, p)
	if err != nil || next.Token == iss.Token || next.User.ID != iss.User.ID {
		t.Fatalf("yenileme: %+v %v", next, err)
	}
	// Eski token hemen geçersiz (önbellekten de düşer), yenisi geçerli.
	if _, err := svc.Authenticate(ctx, iss.Token); !isUnauthorized(err) {
		t.Fatalf("eski token geçersiz olmalı: %v", err)
	}
	if _, err := svc.Authenticate(ctx, next.Token); err != nil {
		t.Fatalf("yeni token geçerli olmalı: %v", err)
	}
	// Aynı eski oturumla ikinci yenileme reddedilir.
	if _, err := svc.Refresh(ctx, p); !isUnauthorized(err) {
		t.Fatalf("ikinci yenileme UNAUTHORIZED olmalı: %v", err)
	}

	// Eşzamanlı yenilemelerden yalnızca biri başarılı olur.
	p2, _ := svc.Authenticate(ctx, next.Token)
	var ok, denied int
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			_, err := svc.Refresh(ctx, p2)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				ok++
			case isUnauthorized(err):
				denied++
			default:
				t.Errorf("beklenmeyen hata: %v", err)
			}
		})
	}
	wg.Wait()
	if ok != 1 || denied != 4 {
		t.Fatalf("eşzamanlı yenileme: %d başarılı, %d reddedildi", ok, denied)
	}
}

func TestIntegrationMeAndForgetUser(t *testing.T) {
	svc, env, _ := newTestService(t)
	ctx := context.Background()
	iss, _ := svc.CreateGuest(ctx, "Burak")
	p, _ := svc.Authenticate(ctx, iss.Token)

	me, err := svc.Me(ctx, p)
	if err != nil || me.ActiveRoomID != nil || me.User.Nickname != "Burak" {
		t.Fatalf("aktif oda yok: %+v %v", me, err)
	}
	room := scalar[string](t, env.DB, "insert into rooms (invite_code, host_user_id) values ('K7M2QX', $1) returning id::text", p.UserID)
	exec(t, env.DB, "insert into room_members (room_id, user_id, nickname, role) values ($1, $2, 'Burak', 'player')", room, p.UserID)
	me, _ = svc.Me(ctx, p)
	if me.ActiveRoomID == nil || *me.ActiveRoomID != room {
		t.Fatalf("aktif oda dönmeli: %+v", me)
	}

	// Hesap silinince önbellek beklenmeden düşürülebilir.
	exec(t, env.DB, "update users set nickname = $2, deleted_at = now() where id = $1", p.UserID, DeletedNickname)
	if _, err := svc.Authenticate(ctx, iss.Token); err != nil {
		t.Fatal("önbellekte hâlâ geçerli")
	}
	svc.ForgetUser(p.UserID)
	if _, err := svc.Authenticate(ctx, iss.Token); !isUnauthorized(err) {
		t.Fatalf("ForgetUser sonrası geçersiz olmalı: %v", err)
	}
}
