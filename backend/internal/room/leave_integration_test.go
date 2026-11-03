package room

import (
	"context"
	"sync"
	"testing"

	"github.com/burakkuru5534/bunker/backend/internal/auth"
)

func (a *api) join(tok, code string) resp {
	a.t.Helper()
	r := a.do("POST", "/v1/rooms/join", tok, `{"invite_code":"`+code+`"}`)
	if r.code != 200 {
		a.t.Fatalf("join: %d %s", r.code, r.raw)
	}
	return r
}

func (a *api) leave(tok, roomID string) {
	a.t.Helper()
	if r := a.do("POST", "/v1/rooms/"+roomID+"/leave", tok, ""); r.code != 204 {
		a.t.Fatalf("leave: %d %s", r.code, r.raw)
	}
}

func (a *api) roomRow(roomID string) (status, host string, version int64, closed bool) {
	a.t.Helper()
	if err := a.env.Admin.QueryRow(context.Background(), `select status, coalesce(host_user_id::text, ''), version, closed_at is not null
		from `+a.env.Schema+`.rooms where id = $1`, roomID).Scan(&status, &host, &version, &closed); err != nil {
		a.t.Fatal(err)
	}
	return
}

func eventTypes(evs []Event) []string {
	var out []string
	for _, e := range evs {
		out = append(out, e.Type)
	}
	return out
}

func TestIntegrationLeaveLobby(t *testing.T) {
	a := newAPI(t)
	hostTok, roomID, code := a.room("Burak")
	tok, uid := a.guest("Ayşe")
	a.join(tok, code)
	before := len(a.events.all())

	a.leave(tok, roomID)
	var rows int
	_ = a.env.Admin.QueryRow(context.Background(), "select count(*) from "+a.env.Schema+".room_members where room_id = $1 and user_id = $2", roomID, uid).Scan(&rows)
	if rows != 0 {
		t.Fatal("lobide üyelik satırı silinmeli")
	}
	_, host, version, _ := a.roomRow(roomID)
	evs := a.events.all()[before:]
	if version != 2 || len(evs) != 1 || evs[0].Type != EventMemberLeft || evs[0].Version != 2 {
		t.Fatalf("sürüm 2, tek member_left: %d %v", version, eventTypes(evs))
	}
	if host == uid {
		t.Fatal("sahiplik değişmemeli")
	}
	if me := a.do("GET", "/v1/me", tok, ""); me.body["active_room_id"] != nil {
		t.Fatalf("aktif oda kalmamalı: %s", me.raw)
	}
	if g := a.do("GET", "/v1/rooms/"+roomID, tok, ""); g.code != 404 {
		t.Fatal("ayrılan odayı göremez")
	}

	// İdempotent: tekrar ayrılma değişiklik yapmaz.
	a.leave(tok, roomID)
	if _, _, v, _ := a.roomRow(roomID); v != 2 || len(a.events.all()) != before+1 {
		t.Fatal("ikinci ayrılma sürüm/olay üretmemeli")
	}
	// Yeniden katılabilir, başka oda da kurabilir.
	a.join(tok, code)
	a.leave(tok, roomID)
	if r := a.do("POST", "/v1/rooms", tok, ""); r.code != 201 {
		t.Fatalf("ayrıldıktan sonra oda kurabilmeli: %d", r.code)
	}
	_ = hostTok
}

func TestIntegrationLeaveHostTransferAndClose(t *testing.T) {
	a := newAPI(t)
	hostTok, roomID, code := a.room("Burak")
	t2, u2 := a.guest("Ayşe")
	t3, _ := a.guest("Mehmet")
	a.join(t2, code)
	a.join(t3, code)
	before := len(a.events.all())

	a.leave(hostTok, roomID)
	_, host, _, _ := a.roomRow(roomID)
	evs := a.events.all()[before:]
	if host != u2 || len(evs) != 2 || evs[0].Type != EventMemberLeft || evs[1].Type != EventHostChanged ||
		evs[1].Payload.(map[string]any)["host_user_id"] != u2 || evs[0].Version != evs[1].Version {
		t.Fatalf("sahiplik en eski oyuncuya, iki olay aynı sürümde: host=%s %v", host, eventTypes(evs))
	}
	if g := a.do("GET", "/v1/rooms/"+roomID, t3, ""); g.body["host_user_id"] != u2 {
		t.Fatalf("görünüm yeni sahibi gösterir: %s", g.raw)
	}

	a.leave(t3, roomID)
	before = len(a.events.all())
	a.leave(t2, roomID) // son üye
	status, _, _, closed := a.roomRow(roomID)
	if status != "closed" || !closed || len(a.events.all()) != before {
		t.Fatalf("son üye: oda kapanır, olay yok: %s %v", status, closed)
	}
	// Kapalı odanın kodu artık bulunamaz.
	t4, _ := a.guest("Zeynep")
	if r := a.do("POST", "/v1/rooms/join", t4, `{"invite_code":"`+code+`"}`); r.code != 404 {
		t.Fatalf("kapalı oda kodu: %d", r.code)
	}
}

func TestIntegrationLeaveNoOpCases(t *testing.T) {
	a := newAPI(t)
	_, roomID, _ := a.room("Burak")
	outsider, _ := a.guest("Ayşe")
	for _, id := range []string{roomID, "00000000-0000-0000-0000-000000000000", "bozuk"} {
		if r := a.do("POST", "/v1/rooms/"+id+"/leave", outsider, ""); r.code != 204 {
			t.Errorf("%s: 204 bekleniyordu: %d %s", id, r.code, r.raw)
		}
	}
	if _, _, v, _ := a.roomRow(roomID); v != 0 {
		t.Fatal("üye olmayanın ayrılması değişiklik yapmaz")
	}
	if r := a.do("POST", "/v1/rooms/"+roomID+"/leave", "", ""); r.code != 401 {
		t.Fatalf("yetkisiz: %d", r.code)
	}
}

func TestIntegrationLeaveDuringGame(t *testing.T) {
	a := newAPI(t)
	hostTok, roomID, code := a.room("Burak")
	tok, uid := a.guest("Ayşe")
	a.join(tok, code)
	a.exec(`insert into scenarios (id, title, disaster, duration_label, shelter_features, outside_world)
		values ('s1', 'Nükleer Kış', repeat('k', 30), '3 yıl', array['a','b','c'], repeat('d', 30))`)
	var gameID string
	_ = a.env.Admin.QueryRow(context.Background(), `insert into `+a.env.Schema+`.games
		(room_id, scenario_id, capacity, total_rounds, phase_deadline_at) values ($1, 's1', 2, 2, now()) returning id::text`, roomID).Scan(&gameID)
	_, hostID, _, _ := a.roomRow(roomID)
	a.exec("insert into game_participants (game_id, user_id, seat) values ($1, $2, 0), ($1, $3, 1)", gameID, hostID, uid)
	a.exec("update rooms set status = 'in_game', current_game_id = $2 where id = $1", roomID, gameID)

	a.leave(tok, roomID)
	var leftAt bool
	if err := a.env.Admin.QueryRow(context.Background(), "select left_at is not null from "+a.env.Schema+".room_members where room_id = $1 and user_id = $2",
		roomID, uid).Scan(&leftAt); err != nil || !leftAt {
		t.Fatalf("oyunda satır kalır, left_at dolar: %v %v", leftAt, err)
	}
	// Eski katılımcı izleyici olarak döner.
	r := a.join(tok, code)
	for _, m := range members(r) {
		if m["user_id"] == uid && m["role"] != "spectator" {
			t.Fatalf("izleyici dönmeli: %s", r.raw)
		}
	}
	_ = hostTok
}

func TestIntegrationConcurrentLeaveClosesOnce(t *testing.T) {
	a := newAPI(t)
	hostTok, roomID, code := a.room("Kurucu")
	toks := []string{hostTok}
	for _, n := range []string{"Ayşe", "Mehmet", "Zeynep", "Can"} {
		tok, _ := a.guest(n)
		a.join(tok, code)
		toks = append(toks, tok)
	}
	svcAuth := auth.NewService(a.env.DB, nil, nil)
	var wg sync.WaitGroup
	for _, tok := range toks {
		p, err := svcAuth.Authenticate(context.Background(), tok)
		if err != nil {
			t.Fatal(err)
		}
		wg.Go(func() {
			if err := a.svc.Leave(context.Background(), p, roomID); err != nil {
				t.Errorf("ayrılma: %v", err)
			}
		})
	}
	wg.Wait()
	status, _, _, closed := a.roomRow(roomID)
	var active int
	_ = a.env.Admin.QueryRow(context.Background(), "select count(*) from "+a.env.Schema+".room_members where room_id = $1 and left_at is null", roomID).Scan(&active)
	if status != "closed" || !closed || active != 0 {
		t.Fatalf("hepsi ayrılınca oda bir kez kapanmalı: %s %v aktif=%d", status, closed, active)
	}
}
