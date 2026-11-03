package room

import (
	"context"
	"net/url"
	"strconv"
	"sync"
	"testing"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/auth"
)

func (a *api) exec(sql string, args ...any) {
	a.t.Helper()
	if _, err := a.env.Admin.Exec(context.Background(), "set search_path = "+a.env.Schema+"; "+sql, args...); err != nil {
		// Çoklu ifade + argüman desteklenmez; search_path'i ayrıca ayarla.
		if _, err2 := a.env.Admin.Exec(context.Background(), "set search_path = "+a.env.Schema); err2 != nil {
			a.t.Fatal(err2)
		}
		if _, err := a.env.Admin.Exec(context.Background(), sql, args...); err != nil {
			a.t.Fatalf("%s: %v", sql, err)
		}
	}
}

// room, yeni bir kullanıcıyla oda kurar.
func (a *api) room(host string) (token, roomID, code string) {
	a.t.Helper()
	tok, _ := a.guest(host)
	r := a.do("POST", "/v1/rooms", tok, "")
	return tok, r.body["id"].(string), r.body["invite_code"].(string)
}

func members(r resp) []map[string]any {
	var out []map[string]any
	for _, m := range r.body["members"].([]any) {
		out = append(out, m.(map[string]any))
	}
	return out
}

func TestIntegrationJoin(t *testing.T) {
	a := newAPI(t)
	hostTok, roomID, code := a.room("Burak")
	tok, uid := a.guest("Ayşe")

	r := a.do("POST", "/v1/rooms/join", tok, `{"invite_code":"`+code+`"}`)
	if r.code != 200 || r.body["id"] != roomID || r.body["version"] != float64(1) {
		t.Fatalf("katılma: %d %s", r.code, r.raw)
	}
	ms := members(r)
	if len(ms) != 2 || ms[1]["user_id"] != uid || ms[1]["role"] != "player" || ms[1]["is_ready"] != false {
		t.Fatalf("üyeler katılma sırasıyla: %s", r.raw)
	}
	ev := a.events.all()
	if len(ev) != 1 || ev[0].Type != EventMemberJoined || ev[0].RoomID != roomID || ev[0].Version != 1 {
		t.Fatalf("tek olay, commit sonrası: %+v", ev)
	}

	// Tekrar katılma: değişiklik yok, olay yok; küçük harf + tireli kod da olur.
	again := a.do("POST", "/v1/rooms/join", tok, `{"invite_code":"`+code[:3]+"-"+toLower(code[3:])+`"}`)
	if again.code != 200 || again.body["version"] != float64(1) || len(a.events.all()) != 1 {
		t.Fatalf("tekrar katılma değişiklik yapmamalı: %d %s", again.code, again.raw)
	}

	// Görüntüleme: üyeler görür.
	if g := a.do("GET", "/v1/rooms/"+roomID, hostTok, ""); g.code != 200 || len(members(g)) != 2 {
		t.Fatalf("üye görür: %d %s", g.code, g.raw)
	}
}

func toLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

func TestIntegrationJoinErrors(t *testing.T) {
	a := newAPI(t)
	_, roomID, code := a.room("Burak")

	tok, _ := a.guest("Ayşe")
	cases := []struct {
		body, status, errCode string
	}{
		{`{"invite_code":"K7M"}`, "400", "VALIDATION_FAILED"},
		{`{"invite_code":"O0O0O0"}`, "400", "VALIDATION_FAILED"},
		{`{}`, "400", "VALIDATION_FAILED"},
		{`{"invite_code":"ZZZZZZ"}`, "404", "INVITE_CODE_NOT_FOUND"},
	}
	for _, c := range cases {
		r := a.do("POST", "/v1/rooms/join", tok, c.body)
		if strconv.Itoa(r.code) != c.status || errCode(r) != c.errCode {
			t.Errorf("%s: %d %s", c.body, r.code, r.raw)
		}
	}

	// Takma ad çakışması (Türkçe duyarsız).
	dup, _ := a.guest("BURAK")
	if r := a.do("POST", "/v1/rooms/join", dup, `{"invite_code":"`+code+`"}`); r.code != 409 || errCode(r) != "NICKNAME_TAKEN_IN_ROOM" {
		t.Fatalf("takma ad: %d %s", r.code, r.raw)
	}
	// Başka odadaki kullanıcı.
	other, otherRoom, _ := a.room("Mehmet")
	r := a.do("POST", "/v1/rooms/join", other, `{"invite_code":"`+code+`"}`)
	d, _ := r.body["error"].(map[string]any)["details"].(map[string]any)
	if r.code != 409 || errCode(r) != "ALREADY_IN_ROOM" || d["room_id"] != otherRoom {
		t.Fatalf("başka oda: %d %s", r.code, r.raw)
	}
	// Atılan kullanıcı.
	kickedTok, kickedID := a.guest("Zeynep")
	a.do("POST", "/v1/rooms/join", kickedTok, `{"invite_code":"`+code+`"}`)
	a.exec("update room_members set left_at = now(), kicked_at = now() where room_id = $1 and user_id = $2", roomID, kickedID)
	if r := a.do("POST", "/v1/rooms/join", kickedTok, `{"invite_code":"`+code+`"}`); r.code != 403 || errCode(r) != "KICKED" {
		t.Fatalf("atılan: %d %s", r.code, r.raw)
	}
	// Kapalı oda kodu bulunamaz.
	a.exec("update room_members set left_at = now() where room_id = $1", roomID)
	a.exec("update rooms set status = 'closed', closed_at = now() where id = $1", roomID)
	if r := a.do("POST", "/v1/rooms/join", tok, `{"invite_code":"`+code+`"}`); r.code != 404 || errCode(r) != "INVITE_CODE_NOT_FOUND" {
		t.Fatalf("kapalı oda: %d %s", r.code, r.raw)
	}
}

func TestIntegrationJoinRateLimit(t *testing.T) {
	a := newAPI(t)
	tok, _ := a.guest("Burak")
	for i := range JoinLimit {
		if r := a.do("POST", "/v1/rooms/join", tok, `{"invite_code":"ZZZZZ`+string("23456789AB"[i])+`"}`); r.code != 404 {
			t.Fatalf("%d. deneme 404: %d", i+1, r.code)
		}
	}
	if r := a.do("POST", "/v1/rooms/join", tok, `{"invite_code":"ZZZZZC"}`); r.code != 429 {
		t.Fatalf("11. deneme 429: %d %s", r.code, r.raw)
	}
}

func TestIntegrationJoinFullRoomUnderConcurrency(t *testing.T) {
	a := newAPI(t)
	_, roomID, code := a.room("Kurucu")
	svcAuth := auth.NewService(a.env.DB, nil, nil)
	var principals []auth.Principal
	for i := range 14 {
		tok, _ := a.guest("Oyuncu" + strconv.Itoa(i))
		p, err := svcAuth.Authenticate(context.Background(), tok)
		if err != nil {
			t.Fatal(err)
		}
		principals = append(principals, p)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok, full := 0, 0
	for _, p := range principals {
		wg.Go(func() {
			_, err := a.svc.Join(context.Background(), p, code)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				ok++
			case apperr.CodeOf(err) == apperr.RoomFull:
				full++
			default:
				t.Errorf("beklenmeyen: %v", err)
			}
		})
	}
	wg.Wait()
	var players int
	_ = a.env.Admin.QueryRow(context.Background(), "select count(*) from "+a.env.Schema+".room_members where room_id = $1 and left_at is null", roomID).Scan(&players)
	if ok != MaxPlayers-1 || full != 14-(MaxPlayers-1) || players != MaxPlayers {
		t.Fatalf("eşzamanlı: %d katıldı, %d dolu, odada %d", ok, full, players)
	}
	var version int64
	_ = a.env.Admin.QueryRow(context.Background(), "select version from "+a.env.Schema+".rooms where id = $1", roomID).Scan(&version)
	if version != int64(MaxPlayers-1) || len(a.events.all()) != MaxPlayers-1 {
		t.Fatalf("her katılma bir sürüm ve bir olay: sürüm %d, olay %d", version, len(a.events.all()))
	}
}

func TestIntegrationJoinDuringGame(t *testing.T) {
	a := newAPI(t)
	hostTok, roomID, code := a.room("Burak")
	partTok, partID := a.guest("Ayşe")
	a.do("POST", "/v1/rooms/join", partTok, `{"invite_code":"`+code+`"}`)
	_ = hostTok

	// Oyunu doğrudan kur: senaryo, oyun, katılımcılar; Ayşe oyundan ayrılmış.
	a.exec(`insert into scenarios (id, title, disaster, duration_label, shelter_features, outside_world)
		values ('s1', 'Nükleer Kış', repeat('k', 30), '3 yıl', array['a','b','c'], repeat('d', 30))`)
	var gameID string
	if err := a.env.Admin.QueryRow(context.Background(), `insert into `+a.env.Schema+`.games
		(room_id, scenario_id, capacity, total_rounds, phase_deadline_at) values ($1, 's1', 2, 2, now()) returning id::text`, roomID).Scan(&gameID); err != nil {
		t.Fatal(err)
	}
	var hostID string
	_ = a.env.Admin.QueryRow(context.Background(), "select host_user_id::text from "+a.env.Schema+".rooms where id = $1", roomID).Scan(&hostID)
	a.exec("insert into game_participants (game_id, user_id, seat) values ($1, $2, 0), ($1, $3, 1)", gameID, hostID, partID)
	a.exec("update game_participants set status = 'left', eliminated_round = 0 where user_id = $1", partID)
	a.exec("update room_members set left_at = now() where user_id = $1", partID)
	a.exec("update rooms set status = 'in_game', current_game_id = $2 where id = $1", roomID, gameID)

	// Eski katılımcı izleyici olarak döner.
	r := a.do("POST", "/v1/rooms/join", partTok, `{"invite_code":"`+code+`"}`)
	if r.code != 200 {
		t.Fatalf("eski katılımcı: %d %s", r.code, r.raw)
	}
	for _, m := range members(r) {
		if m["user_id"] == partID && m["role"] != "spectator" {
			t.Fatalf("izleyici olmalı: %s", r.raw)
		}
	}
	// Yabancı katılamaz.
	stranger, _ := a.guest("Mehmet")
	if r := a.do("POST", "/v1/rooms/join", stranger, `{"invite_code":"`+code+`"}`); r.code != 409 || errCode(r) != "GAME_IN_PROGRESS" {
		t.Fatalf("yabancı: %d %s", r.code, r.raw)
	}
}

func TestIntegrationGetRoomAccess(t *testing.T) {
	a := newAPI(t)
	hostTok, roomID, _ := a.room("Burak")
	outsider, _ := a.guest("Ayşe")
	for name, c := range map[string]struct{ tok, id string }{
		"üye olmayan":  {outsider, roomID},
		"olmayan oda":  {hostTok, "00000000-0000-0000-0000-000000000000"},
		"bozuk kimlik": {hostTok, "abc"},
		"sql benzeri":  {hostTok, "1' or '1'='1"},
	} {
		if r := a.do("GET", "/v1/rooms/"+url.PathEscape(c.id), c.tok, ""); r.code != 404 || errCode(r) != "NOT_FOUND" {
			t.Errorf("%s: %d %s", name, r.code, r.raw)
		}
	}
	if r := a.do("GET", "/v1/rooms/"+roomID, "", ""); r.code != 401 {
		t.Fatalf("yetkisiz: %d", r.code)
	}
}
