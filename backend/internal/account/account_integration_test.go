package account

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

	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/game"
	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
	"github.com/burakkuru5534/bunker/backend/internal/moderation"
	"github.com/burakkuru5534/bunker/backend/internal/room"
	"github.com/burakkuru5534/bunker/backend/internal/testdb"
)

type api struct {
	t       *testing.T
	env     *testdb.Env
	handler http.Handler
	ip      atomic.Int64
}

func newAPI(t *testing.T) *api {
	t.Helper()
	env := testdb.New(t)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	authSvc := auth.NewService(env.DB, nil, logger)
	authHTTP := auth.NewHTTP(authSvc, 1)
	gameSvc := game.NewService(env.DB, nil, logger)
	roomSvc := room.NewService(env.DB, nil, nil, logger).WithGames(game.RoomHooks{Starter: game.NewStarter(logger), Service: gameSvc})
	h := httpapi.NewRouter(httpapi.Options{
		Logger: logger, MinAppVersion: httpapi.Version{Minor: 1}, RequestTimeout: 5 * time.Second,
		Register: func(mux *http.ServeMux) {
			authHTTP.Register(mux)
			room.NewHTTP(roomSvc, authHTTP.Require).Register(mux)
			game.NewHTTP(gameSvc, authHTTP.Require).Register(mux)
			moderation.NewHTTP(moderation.NewService(env.DB, logger), authHTTP.Require).Register(mux)
			NewHTTP(NewService(env.DB, authSvc, roomSvc, logger), authHTTP.Require).Register(mux)
		},
	})
	return &api{t: t, env: env, handler: h}
}

type resp struct {
	code int
	body map[string]any
	raw  string
}

func (a *api) do(method, path, token, body string) resp {
	a.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(httpapi.AppVersionHeader, "0.1.0")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("X-Forwarded-For", "198.51.100."+strconv.Itoa(int(a.ip.Add(1)%250)+1))
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	out := resp{code: rec.Code, raw: rec.Body.String()}
	_ = json.Unmarshal(rec.Body.Bytes(), &out.body)
	return out
}

func (a *api) guest(nick string) (string, string) {
	a.t.Helper()
	r := a.do("POST", "/v1/sessions/guest", "", `{"nickname":"`+nick+`"}`)
	if r.code != 201 {
		a.t.Fatalf("guest: %d %s", r.code, r.raw)
	}
	return r.body["token"].(string), r.body["user"].(map[string]any)["id"].(string)
}

func (a *api) val(sql string, args ...any) any {
	a.t.Helper()
	var v any
	if err := a.env.DB.Pool().QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		a.t.Fatalf("%s: %v", sql, err)
	}
	return v
}

func (a *api) exec(sql string, args ...any) {
	a.t.Helper()
	if _, err := a.env.Migrate.Exec(context.Background(), sql, args...); err != nil {
		a.t.Fatalf("%.60s: %v", sql, err)
	}
}

// Lobide: oda sahibi hesabını siler → odadan ayrılır, sahiplik devredilir;
// kişisel veriler silinir; başkalarının verisi kalır.
func TestIntegrationDeleteInLobby(t *testing.T) {
	a := newAPI(t)
	tok, me := a.guest("Burak")
	tok2, other := a.guest("Ayse")
	_, third := a.guest("Can")
	r := a.do("POST", "/v1/rooms", tok, "")
	roomID := r.body["id"].(string)
	a.do("POST", "/v1/rooms/join", tok2, `{"invite_code":"`+r.body["invite_code"].(string)+`"}`)
	a.do("POST", "/v1/rooms/"+roomID+"/messages", tok, `{"client_msg_id":"00000000-0000-4000-8000-000000000001","text":"benim mesajım"}`)
	a.do("POST", "/v1/rooms/"+roomID+"/messages", tok2, `{"client_msg_id":"00000000-0000-4000-8000-000000000002","text":"onun mesajı"}`)
	a.do("POST", "/v1/blocks", tok, `{"user_id":"`+third+`"}`)  // benim engelim
	a.do("POST", "/v1/blocks", tok2, `{"user_id":"`+me+`"}`)    // beni engelleyen
	a.do("POST", "/v1/blocks", tok2, `{"user_id":"`+third+`"}`) // başkasının engeli (kalmalı)
	if rr := a.do("POST", "/v1/reports", tok2, fmt.Sprintf(`{"reported_user_id":%q,"room_id":%q,"reason":"spam"}`, me, roomID)); rr.code != 201 {
		t.Fatalf("rapor: %s", rr.raw)
	}
	// İkinci geçerli token: yenileme (eski token da önbellekte olabilir).
	refreshed := a.do("POST", "/v1/sessions/refresh", tok, "").body["token"].(string)

	if r := a.do("DELETE", "/v1/me", refreshed, ""); r.code != 204 {
		t.Fatalf("sil: %d %s", r.code, r.raw)
	}
	// Oturumlar geçersiz.
	for _, tk := range []string{tok, refreshed} {
		if r := a.do("GET", "/v1/me", tk, ""); r.code != 401 {
			t.Fatalf("silinen oturum: %d", r.code)
		}
	}
	if a.val(`select count(*) from sessions where user_id = $1`, me).(int64) != 0 {
		t.Fatal("oturum satırları kaldı")
	}
	// Anonimleşme.
	if a.val(`select nickname from users where id = $1`, me) != auth.DeletedNickname ||
		a.val(`select deleted_at is not null from users where id = $1`, me) != true {
		t.Fatal("kullanıcı anonimleşmedi")
	}
	// Odadan ayrıldı, sahiplik devredildi; oda açık.
	if a.val(`select host_user_id::text from rooms where id = $1`, roomID) != other ||
		a.val(`select count(*) from room_members where room_id = $1 and user_id = $2 and left_at is null`, roomID, me).(int64) != 0 {
		t.Fatal("odadan ayrılma/sahiplik")
	}
	// Kendi mesajları ve iki yönlü engeller silindi; başkalarınınki kaldı.
	if a.val(`select count(*) from chat_messages where user_id = $1`, me).(int64) != 0 ||
		a.val(`select count(*) from chat_messages where user_id = $1`, other).(int64) != 1 {
		t.Fatal("mesaj silme yanlış")
	}
	if a.val(`select count(*) from blocks where blocker_id = $1 or blocked_id = $1`, me).(int64) != 0 ||
		a.val(`select count(*) from blocks where blocker_id = $1 and blocked_id = $2`, other, third).(int64) != 1 {
		t.Fatal("engel silme yanlış")
	}
	// Rapor moderasyon için kalır.
	if a.val(`select count(*) from reports where reported_id = $1`, me).(int64) != 1 {
		t.Fatal("rapor silindi")
	}
	// Aynı takma adla yeni misafir hesap açılabilir.
	if _, newID := a.guest("Burak"); newID == me {
		t.Fatal("aynı kimlik")
	}
}

// Oyunda: hesap silen oyuncu oyunda elenmiş sayılır, oyun sürer; sonuçta
// "Silinmiş oyuncu" görünür.
func TestIntegrationDeleteDuringGame(t *testing.T) {
	a := newAPI(t)
	seedCatalog(t, a)
	var toks, users []string
	for i := range 5 {
		tk, u := a.guest(fmt.Sprintf("Oyuncu%d", i))
		toks, users = append(toks, tk), append(users, u)
	}
	r := a.do("POST", "/v1/rooms", toks[0], `{"capacity":1}`)
	roomID := r.body["id"].(string)
	for _, tk := range toks[1:] {
		a.do("POST", "/v1/rooms/join", tk, `{"invite_code":"`+r.body["invite_code"].(string)+`"}`)
	}
	for _, tk := range toks {
		a.do("POST", "/v1/rooms/"+roomID+"/ready", tk, `{"ready":true}`)
	}
	gameID := a.do("POST", "/v1/rooms/"+roomID+"/start", toks[0], "").body["game_id"].(string)

	if r := a.do("DELETE", "/v1/me", toks[2], ""); r.code != 204 {
		t.Fatalf("sil: %d %s", r.code, r.raw)
	}
	if a.val(`select status from game_participants where game_id = $1 and user_id = $2`, gameID, users[2]) != "left" ||
		a.val(`select phase from games where id = $1`, gameID) == "finished" {
		t.Fatal("oyundaki silme")
	}
	if n := a.val(`select count(*) from game_actions where game_id = $1 and actor_id = $2 and type = 'leave'`, gameID, users[2]).(int64); n != 1 {
		t.Fatalf("ayrılma kaydı %d", n)
	}
	// Oyunu bitir (kalan 4 kişiden 3'ü ayrılır) ve sonucu oku.
	for _, i := range []int{1, 3, 4} {
		a.do("POST", "/v1/rooms/"+roomID+"/leave", toks[i], "")
	}
	res := a.do("GET", "/v1/games/"+gameID+"/result", toks[0], "")
	if res.code != 200 {
		t.Fatalf("sonuç: %d %s", res.code, res.raw)
	}
	found := false
	for _, p := range res.body["players"].([]any) {
		pm := p.(map[string]any)
		if pm["user_id"] == users[2] {
			found = pm["nickname"] == auth.DeletedNickname && pm["status"] == "left"
		}
	}
	if !found {
		t.Fatalf("sonuçta anonim değil: %s", res.raw)
	}
	if a.val(`select nickname from room_members where room_id = $1 and user_id = $2`, roomID, users[2]) != auth.DeletedNickname {
		t.Fatal("oda üyeliğindeki takma ad anonimleşmedi")
	}
}

func TestIntegrationDeleteRequiresSession(t *testing.T) {
	a := newAPI(t)
	if r := a.do("DELETE", "/v1/me", "", ""); r.code != 401 {
		t.Fatalf("oturumsuz: %d", r.code)
	}
	if r := a.do("DELETE", "/v1/me", "bozuk", ""); r.code != 401 {
		t.Fatalf("bozuk token: %d", r.code)
	}
}

func seedCatalog(t *testing.T, a *api) {
	t.Helper()
	a.exec(`insert into scenarios (id, title, disaster, duration_label, shelter_features, outside_world)
		values ('meteor', 'Göktaşı', 'Uzun bir felaket metni burada yer alır, en az yirmi karakter.', '2 yıl',
		array['a1','b2','c3'], 'Dış dünya hakkında yeterince uzun bir açıklama metni.')`)
	for ci, c := range []string{"profession", "health", "hobby", "item", "secret"} {
		for i := range 12 {
			a.exec(`insert into card_definitions (id, category, text, balance) values ($1, $2, $3, $4)`,
				(ci+1)*1000+i, c, fmt.Sprintf("%s kartı (%02d)", c, i), []int{-1, 0, 1}[i%3])
		}
	}
}
