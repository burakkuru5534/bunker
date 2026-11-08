package social_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/account"
	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
	"github.com/burakkuru5534/bunker/backend/internal/moderation"
	"github.com/burakkuru5534/bunker/backend/internal/room"
	"github.com/burakkuru5534/bunker/backend/internal/social"
	"github.com/burakkuru5534/bunker/backend/internal/testdb"
)

type api struct {
	t       *testing.T
	env     *testdb.Env
	handler http.Handler
	svc     *social.Service
	ip      atomic.Int64
}

func newAPI(t *testing.T) *api {
	t.Helper()
	env := testdb.New(t)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	authSvc := auth.NewService(env.DB, nil, logger)
	authHTTP := auth.NewHTTP(authSvc, 1)
	roomSvc := room.NewService(env.DB, nil, nil, logger)
	svc := social.NewService(env.DB, logger)
	h := httpapi.NewRouter(httpapi.Options{
		Logger: logger, MinAppVersion: httpapi.Version{Minor: 1}, RequestTimeout: 5 * time.Second,
		Register: func(mux *http.ServeMux) {
			authHTTP.Register(mux)
			room.NewHTTP(roomSvc, authHTTP.Require).Register(mux)
			moderation.NewHTTP(moderation.NewService(env.DB, logger), authHTTP.Require).Register(mux)
			account.NewHTTP(account.NewService(env.DB, authSvc, roomSvc, logger), authHTTP.Require).Register(mux)
			social.NewHTTP(svc, authHTTP.Require).Register(mux)
		},
	})
	return &api{t: t, env: env, handler: h, svc: svc}
}

type resp struct {
	code int
	body map[string]any
	raw  string
}

func (a *api) do(method, path, token, body string) resp {
	a.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(httpapi.AppVersionHeader, "1.0.0")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	ip := a.ip.Add(1)
	req.Header.Set("X-Forwarded-For", "198.51.100."+strconv.Itoa(int(ip%250)+1))
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	out := resp{code: rec.Code, raw: rec.Body.String()}
	_ = json.Unmarshal(rec.Body.Bytes(), &out.body)
	return out
}

func errCode(r resp) string {
	e, _ := r.body["error"].(map[string]any)
	c, _ := e["code"].(string)
	return c
}

type user struct{ tok, id, nick string }

func (a *api) guest(nick string) user {
	a.t.Helper()
	r := a.do("POST", "/v1/sessions/guest", "", `{"nickname":"`+nick+`"}`)
	if r.code != 201 {
		a.t.Fatalf("guest %s: %d %s", nick, r.code, r.raw)
	}
	return user{r.body["token"].(string), r.body["user"].(map[string]any)["id"].(string), nick}
}

func (a *api) exec(sql string, args ...any) {
	a.t.Helper()
	if _, err := a.env.DB.Pool().Exec(context.Background(), sql, args...); err != nil {
		a.t.Fatalf("%s: %v", sql, err)
	}
}

func (a *api) friends(u user) (friends, incoming, outgoing []string) {
	a.t.Helper()
	r := a.do("GET", "/v1/friends", u.tok, "")
	if r.code != 200 {
		a.t.Fatalf("friends: %d %s", r.code, r.raw)
	}
	pick := func(k string) []string {
		var out []string
		for _, p := range r.body[k].([]any) {
			out = append(out, p.(map[string]any)["nickname"].(string))
		}
		return out
	}
	return pick("friends"), pick("incoming"), pick("outgoing")
}

func (a *api) request(from user, nick string) resp {
	return a.do("POST", "/v1/friends/requests", from.tok, `{"nickname":"`+nick+`"}`)
}

func (a *api) befriend(x, y user) {
	a.t.Helper()
	if r := a.request(x, y.nick); r.code != 201 {
		a.t.Fatalf("istek: %s", r.raw)
	}
	if r := a.do("POST", "/v1/friends/requests/"+x.id+"/accept", y.tok, ""); r.code != 204 {
		a.t.Fatalf("kabul: %d %s", r.code, r.raw)
	}
}

func eq(a, b []string) bool { return fmt.Sprint(a) == fmt.Sprint(b) }

func TestIntegrationNicknameUniqueness(t *testing.T) {
	a := newAPI(t)
	a.guest("Işık")
	for _, n := range []string{"IŞIK", "ışık", " Işık "} {
		r := a.do("POST", "/v1/sessions/guest", "", `{"nickname":"`+n+`"}`)
		f, _ := r.body["error"].(map[string]any)["details"].(map[string]any)["fields"].(map[string]any)
		if r.code != 409 || errCode(r) != "NICKNAME_TAKEN" || f["nickname"] != "taken" {
			t.Fatalf("%q: %d %s", n, r.code, r.raw)
		}
	}
	// "Isik" (noktasız değil) farklı addır.
	a.guest("Isik")

	b := a.guest("Mehmet")
	if r := a.do("GET", "/v1/me", b.tok, ""); r.body["needs_nickname"] != false {
		t.Fatalf("yeni hesap: %s", r.raw)
	}
	// Eski hesap: anahtar yok → adını onaylar.
	a.exec("update users set nickname_key = null where id = $1", b.id)
	if r := a.do("GET", "/v1/me", b.tok, ""); r.body["needs_nickname"] != true {
		t.Fatalf("eski hesap: %s", r.raw)
	}
	if r := a.do("PATCH", "/v1/me", b.tok, `{"nickname":"Mehmet"}`); r.code != 200 || r.body["nickname"] != "Mehmet" || r.body["needs_nickname"] != false {
		t.Fatalf("onay: %d %s", r.code, r.raw)
	}
	if r := a.do("GET", "/v1/me", b.tok, ""); r.body["needs_nickname"] != false {
		t.Fatalf("onay sonrası: %s", r.raw)
	}
	// Alınmış ada geçilemez; geçerli bir ada geçilir, önbellek tazelenir.
	if r := a.do("PATCH", "/v1/me", b.tok, `{"nickname":"IŞIK"}`); r.code != 409 || errCode(r) != "NICKNAME_TAKEN" {
		t.Fatalf("alınmış: %d %s", r.code, r.raw)
	}
	if r := a.do("PATCH", "/v1/me", b.tok, `{"nickname":"a"}`); r.code != 400 || errCode(r) != "NICKNAME_INVALID" {
		t.Fatalf("geçersiz: %d %s", r.code, r.raw)
	}
	if r := a.do("PATCH", "/v1/me", b.tok, `{"nickname":"Memo"}`); r.code != 200 {
		t.Fatalf("değiştir: %s", r.raw)
	}
	if r := a.do("GET", "/v1/me", b.tok, ""); r.body["nickname"] != "Memo" {
		t.Fatalf("önbellek: %s", r.raw)
	}
	// Eski ad boşaldı; silinen hesabın adı da boşalır.
	a.guest("Mehmet")
	c := a.guest("Zeynep")
	if r := a.do("DELETE", "/v1/me", c.tok, ""); r.code != 204 {
		t.Fatalf("sil: %s", r.raw)
	}
	a.guest("Zeynep")
}

func TestIntegrationFriendRequests(t *testing.T) {
	a := newAPI(t)
	ali, ayse, mert := a.guest("Ali"), a.guest("Ayşe"), a.guest("Mert")

	r := a.request(ali, "AYŞE") // büyük/küçük harf duyarsız
	if r.code != 201 || r.body["status"] != "requested" || r.body["user"].(map[string]any)["nickname"] != "Ayşe" {
		t.Fatalf("istek: %d %s", r.code, r.raw)
	}
	if r := a.request(ali, "Ayşe"); r.code != 201 { // tekrar: değişiklik yok
		t.Fatalf("tekrar istek: %s", r.raw)
	}
	if _, in, _ := a.friends(ayse); !eq(in, []string{"Ali"}) {
		t.Fatalf("gelen: %v", in)
	}
	if _, _, out := a.friends(ali); !eq(out, []string{"Ayşe"}) {
		t.Fatalf("giden: %v", out)
	}
	if r := a.do("GET", "/v1/inbox", ayse.tok, ""); r.body["incoming_requests"] != float64(1) {
		t.Fatalf("inbox: %s", r.raw)
	}
	if r := a.do("POST", "/v1/friends/requests/"+ali.id+"/accept", ayse.tok, ""); r.code != 204 {
		t.Fatalf("kabul: %d %s", r.code, r.raw)
	}
	if f, in, out := a.friends(ali); !eq(f, []string{"Ayşe"}) || len(in) != 0 || len(out) != 0 {
		t.Fatalf("ali: %v %v %v", f, in, out)
	}
	if f, _, _ := a.friends(ayse); !eq(f, []string{"Ali"}) {
		t.Fatalf("ayşe: %v", f)
	}
	if r := a.request(ayse, "Ali"); r.code != 409 || errCode(r) != "ALREADY_FRIENDS" {
		t.Fatalf("zaten: %d %s", r.code, r.raw)
	}
	// Karşılıklı istek doğrudan arkadaşlık.
	a.request(mert, "Ali")
	if r := a.request(ali, "Mert"); r.code != 201 || r.body["status"] != "friends" {
		t.Fatalf("karşılıklı: %s", r.raw)
	}
	if f, _, _ := a.friends(ali); !eq(f, []string{"Ayşe", "Mert"}) {
		t.Fatalf("liste: %v", f)
	}

	for name, c := range map[string]struct {
		u          user
		nick, code string
		status     int
	}{
		"yok":     {ali, "Kimseyok", "NOT_FOUND", 404},
		"kendisi": {ali, "ali", "VALIDATION_FAILED", 400},
		"boş":     {ali, "  ", "VALIDATION_FAILED", 400},
	} {
		if r := a.request(c.u, c.nick); r.code != c.status || errCode(r) != c.code {
			t.Errorf("%s: %d %s", name, r.code, r.raw)
		}
	}
	// Kabul edilecek istek yok.
	if r := a.do("POST", "/v1/friends/requests/"+mert.id+"/accept", ayse.tok, ""); r.code != 404 {
		t.Fatalf("istek yok: %d", r.code)
	}

	// Reddetme ve geri çekme.
	zey := a.guest("Zeynep")
	a.request(zey, "Ayşe")
	if r := a.do("DELETE", "/v1/friends/requests/"+zey.id, ayse.tok, ""); r.code != 204 {
		t.Fatal(r.raw)
	}
	if _, in, _ := a.friends(ayse); len(in) != 0 {
		t.Fatalf("reddedilen: %v", in)
	}
	a.request(zey, "Ayşe")
	a.do("DELETE", "/v1/friends/requests/"+ayse.id, zey.tok, "")
	if _, _, out := a.friends(zey); len(out) != 0 {
		t.Fatalf("geri çekilen: %v", out)
	}

	// Arkadaşlıktan çıkarma iki yönlü.
	if r := a.do("DELETE", "/v1/friends/"+mert.id, ali.tok, ""); r.code != 204 {
		t.Fatal(r.raw)
	}
	if f, _, _ := a.friends(mert); len(f) != 0 {
		t.Fatalf("çıkarılan: %v", f)
	}
}

func TestIntegrationBlockingAndDeletion(t *testing.T) {
	a := newAPI(t)
	ali, ayse, can := a.guest("Ali"), a.guest("Ayşe"), a.guest("Can")
	a.befriend(ali, ayse)
	a.request(can, "Ali")

	// Engel arkadaşlığı ve istekleri siler; engellenen "bulunamadı" görür.
	if r := a.do("POST", "/v1/blocks", ali.tok, `{"user_id":"`+ayse.id+`"}`); r.code != 204 {
		t.Fatalf("engel: %d %s", r.code, r.raw)
	}
	if f, _, _ := a.friends(ayse); len(f) != 0 {
		t.Fatalf("engel sonrası: %v", f)
	}
	if r := a.request(ayse, "Ali"); r.code != 404 {
		t.Fatalf("engellenen istek: %d %s", r.code, r.raw)
	}
	if r := a.request(ali, "Ayşe"); r.code != 404 {
		t.Fatalf("engelleyen istek: %d %s", r.code, r.raw)
	}

	// Hesap silinince arkadaşlıklar ve istekler gider.
	a.befriend(ali, a.guest("Deniz"))
	if r := a.do("DELETE", "/v1/me", ali.tok, ""); r.code != 204 {
		t.Fatal(r.raw)
	}
	if _, _, out := a.friends(can); len(out) != 0 {
		t.Fatalf("silinen hesaba istek kaldı: %v", out)
	}
	var n int
	if err := a.env.DB.Pool().QueryRow(context.Background(),
		`select (select count(*) from friendships where user_id = $1 or friend_id = $1)
		      + (select count(*) from friend_requests where from_user_id = $1 or to_user_id = $1)`, ali.id).Scan(&n); err != nil || n != 0 {
		t.Fatalf("kalan satır: %d %v", n, err)
	}
}

func TestIntegrationLobbyInvites(t *testing.T) {
	a := newAPI(t)
	host, friend, stranger := a.guest("Ali"), a.guest("Ayşe"), a.guest("Can")
	a.befriend(host, friend)
	r := a.do("POST", "/v1/rooms", host.tok, `{}`)
	roomID, code := r.body["id"].(string), r.body["invite_code"].(string)
	invite := func(from user, to string) resp {
		return a.do("POST", "/v1/rooms/"+roomID+"/invites", from.tok, `{"user_id":"`+to+`"}`)
	}
	inbox := func(u user) []any {
		r := a.do("GET", "/v1/inbox", u.tok, "")
		if r.code != 200 {
			t.Fatalf("inbox: %s", r.raw)
		}
		return r.body["invites"].([]any)
	}

	if r := invite(host, stranger.id); r.code != 403 || errCode(r) != "FORBIDDEN" {
		t.Fatalf("arkadaş değil: %d %s", r.code, r.raw)
	}
	if r := invite(stranger, friend.id); r.code != 404 {
		t.Fatalf("odada değil: %d %s", r.code, r.raw)
	}
	if r := invite(host, friend.id); r.code != 204 {
		t.Fatalf("davet: %d %s", r.code, r.raw)
	}
	inv := inbox(friend)
	if len(inv) != 1 {
		t.Fatalf("davet listesi: %v", inv)
	}
	v := inv[0].(map[string]any)
	if v["invite_code"] != code || v["room_id"] != roomID || v["players"] != float64(1) ||
		v["from"].(map[string]any)["nickname"] != "Ali" {
		t.Fatalf("davet: %v", v)
	}
	if len(inbox(stranger)) != 0 {
		t.Fatal("başkasına görünmemeli")
	}

	// Reddet → gider; yeniden davet → gelir; katılınca listelenmez.
	a.do("DELETE", "/v1/invites/"+roomID, friend.tok, "")
	if len(inbox(friend)) != 0 {
		t.Fatal("reddedilen davet")
	}
	invite(host, friend.id)
	if r := a.do("POST", "/v1/rooms/join", friend.tok, `{"invite_code":"`+code+`"}`); r.code != 200 {
		t.Fatalf("katıl: %s", r.raw)
	}
	if len(inbox(friend)) != 0 {
		t.Fatal("üye olunca davet listelenmez")
	}
	a.do("POST", "/v1/rooms/"+roomID+"/leave", friend.tok, "")

	// Süresi dolmuş davet ve lobide olmayan oda listelenmez.
	invite(host, friend.id)
	a.exec("update room_invites set expires_at = now() - interval '1 second' where room_id = $1", roomID)
	if len(inbox(friend)) != 0 {
		t.Fatal("süresi dolmuş davet")
	}
	invite(host, friend.id)
	a.exec("update rooms set status = 'finished' where id = $1", roomID)
	if len(inbox(friend)) != 0 {
		t.Fatal("lobide olmayan oda")
	}
	if r := invite(host, friend.id); r.code != 409 || errCode(r) != "INVALID_PHASE" {
		t.Fatalf("lobi dışı davet: %d %s", r.code, r.raw)
	}
	a.exec("update rooms set status = 'lobby' where id = $1", roomID)

	// Arkadaşlık bitince davet de gider.
	a.do("DELETE", "/v1/friends/"+friend.id, host.tok, "")
	if len(inbox(friend)) != 0 {
		t.Fatal("arkadaşlık bitince davet")
	}
}

func TestIntegrationFriendLimits(t *testing.T) {
	a := newAPI(t)
	me := a.guest("Ali")
	for i := range social.MaxOutgoing {
		u := a.guest(fmt.Sprintf("Oyuncu%02d", i))
		if r := a.request(me, u.nick); r.code != 201 {
			t.Fatalf("%d: %s", i, r.raw)
		}
	}
	extra := a.guest("Fazla")
	if r := a.request(me, extra.nick); r.code != 409 || errCode(r) != "FRIEND_LIMIT" {
		t.Fatalf("giden sınırı: %d %s", r.code, r.raw)
	}
}
