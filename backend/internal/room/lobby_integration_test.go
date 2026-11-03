package room

import (
	"context"
	"testing"
)

func lastEvent(a *api) Event {
	evs := a.events.all()
	return evs[len(evs)-1]
}

func TestIntegrationReady(t *testing.T) {
	a := newAPI(t)
	hostTok, roomID, code := a.room("Burak")
	tok, uid := a.guest("Ayşe")
	a.join(tok, code)
	before := len(a.events.all())

	if r := a.do("POST", "/v1/rooms/"+roomID+"/ready", tok, `{"ready":true}`); r.code != 204 {
		t.Fatalf("hazır: %d %s", r.code, r.raw)
	}
	e := lastEvent(a)
	if len(a.events.all()) != before+1 || e.Type != EventReadyChanged || e.Payload.(map[string]any)["user_id"] != uid ||
		e.Payload.(map[string]any)["is_ready"] != true || e.Version != 2 {
		t.Fatalf("ready_changed: %+v", e)
	}
	g := a.do("GET", "/v1/rooms/"+roomID, hostTok, "")
	for _, m := range members(g) {
		if m["user_id"] == uid && m["is_ready"] != true {
			t.Fatalf("görünümde hazır: %s", g.raw)
		}
	}
	// Aynı değer: değişiklik yok.
	a.do("POST", "/v1/rooms/"+roomID+"/ready", tok, `{"ready":true}`)
	if len(a.events.all()) != before+1 {
		t.Fatal("aynı değer olay üretmemeli")
	}
	if r := a.do("POST", "/v1/rooms/"+roomID+"/ready", tok, `{"ready":false}`); r.code != 204 || lastEvent(a).Version != 3 {
		t.Fatal("hazırlığı kaldırma")
	}

	for name, c := range map[string]struct {
		body, want string
		status     int
	}{
		"gövde yok":  {`{}`, "VALIDATION_FAILED", 400},
		"yanlış tür": {`{"ready":"evet"}`, "VALIDATION_FAILED", 400},
	} {
		if r := a.do("POST", "/v1/rooms/"+roomID+"/ready", tok, c.body); r.code != c.status || errCode(r) != c.want {
			t.Errorf("%s: %d %s", name, r.code, r.raw)
		}
	}
	outsider, _ := a.guest("Mehmet")
	if r := a.do("POST", "/v1/rooms/"+roomID+"/ready", outsider, `{"ready":true}`); r.code != 404 {
		t.Fatalf("üye olmayan 404: %d", r.code)
	}
	a.exec("update rooms set status = 'finished' where id = $1", roomID)
	if r := a.do("POST", "/v1/rooms/"+roomID+"/ready", tok, `{"ready":true}`); r.code != 409 || errCode(r) != "INVALID_PHASE" {
		t.Fatalf("lobi dışında: %d %s", r.code, r.raw)
	}
}

func TestIntegrationSettings(t *testing.T) {
	a := newAPI(t)
	hostTok, roomID, code := a.room("Burak")
	tok, _ := a.guest("Ayşe")
	a.join(tok, code)

	r := a.do("PATCH", "/v1/rooms/"+roomID+"/settings", hostTok, `{"capacity":1}`)
	if r.code != 200 || r.body["capacity"] != float64(1) || r.body["version"] != float64(2) {
		t.Fatalf("kapasite: %d %s", r.code, r.raw)
	}
	if e := lastEvent(a); e.Type != EventSettingsChanged || e.Payload.(map[string]any)["capacity"] != 1 {
		t.Fatalf("settings_changed: %+v", e)
	}
	n := len(a.events.all())
	if r := a.do("PATCH", "/v1/rooms/"+roomID+"/settings", hostTok, `{"capacity":1}`); r.code != 200 || r.body["version"] != float64(2) || len(a.events.all()) != n {
		t.Fatal("aynı kapasite: değişiklik yok, görünüm döner")
	}
	cases := []struct {
		tok, body, want string
		status          int
	}{
		{tok, `{"capacity":2}`, "FORBIDDEN", 403},
		{hostTok, `{"capacity":3}`, "VALIDATION_FAILED", 400},
		{hostTok, `{}`, "VALIDATION_FAILED", 400},
	}
	for _, c := range cases {
		if r := a.do("PATCH", "/v1/rooms/"+roomID+"/settings", c.tok, c.body); r.code != c.status || errCode(r) != c.want {
			t.Errorf("%s: %d %s", c.body, r.code, r.raw)
		}
	}
	if r := a.do("POST", "/v1/rooms/"+roomID+"/settings", hostTok, `{"capacity":2}`); r.code != 404 {
		t.Fatalf("yanlış yöntem 404: %d", r.code)
	}
}

func TestIntegrationKick(t *testing.T) {
	a := newAPI(t)
	hostTok, roomID, code := a.room("Burak")
	tok, uid := a.guest("Ayşe")
	other, otherID := a.guest("Mehmet")
	a.join(tok, code)
	a.join(other, code)

	if r := a.do("POST", "/v1/rooms/"+roomID+"/kick", tok, `{"user_id":"`+otherID+`"}`); r.code != 403 || errCode(r) != "FORBIDDEN" {
		t.Fatalf("sahip olmayan atamaz: %d %s", r.code, r.raw)
	}
	if r := a.do("POST", "/v1/rooms/"+roomID+"/kick", hostTok, `{"user_id":"`+uid+`"}`); r.code != 204 {
		t.Fatalf("atma: %d %s", r.code, r.raw)
	}
	if e := lastEvent(a); e.Type != EventMemberKicked || e.Payload.(map[string]any)["user_id"] != uid {
		t.Fatalf("member_kicked: %+v", e)
	}
	// Atılan: odayı göremez, geri dönemez, başka oda kurabilir.
	if g := a.do("GET", "/v1/rooms/"+roomID, tok, ""); g.code != 404 {
		t.Fatal("atılan odayı göremez")
	}
	if r := a.do("POST", "/v1/rooms/join", tok, `{"invite_code":"`+code+`"}`); r.code != 403 || errCode(r) != "KICKED" {
		t.Fatalf("atılan geri dönemez: %d %s", r.code, r.raw)
	}
	if r := a.do("POST", "/v1/rooms", tok, ""); r.code != 201 {
		t.Fatalf("atılan başka oda kurabilir: %d", r.code)
	}
	// Atılan biri tekrar atılamaz; kendini atamaz; bozuk kimlik.
	cases := []struct {
		body, want string
		status     int
	}{
		{`{"user_id":"` + uid + `"}`, "NOT_FOUND", 404},
		{`{"user_id":"bozuk"}`, "VALIDATION_FAILED", 400},
		{`{}`, "VALIDATION_FAILED", 400},
	}
	for _, c := range cases {
		if r := a.do("POST", "/v1/rooms/"+roomID+"/kick", hostTok, c.body); r.code != c.status || errCode(r) != c.want {
			t.Errorf("%s: %d %s", c.body, r.code, r.raw)
		}
	}
	var hostID string
	_ = a.env.Admin.QueryRow(context.Background(), "select host_user_id::text from "+a.env.Schema+".rooms where id = $1", roomID).Scan(&hostID)
	if r := a.do("POST", "/v1/rooms/"+roomID+"/kick", hostTok, `{"user_id":"`+hostID+`"}`); r.code != 400 {
		t.Fatalf("kendini atamaz: %d %s", r.code, r.raw)
	}
	a.exec("update rooms set status = 'finished' where id = $1", roomID)
	if r := a.do("POST", "/v1/rooms/"+roomID+"/kick", hostTok, `{"user_id":"`+otherID+`"}`); r.code != 409 || errCode(r) != "INVALID_PHASE" {
		t.Fatalf("yalnızca lobide: %d %s", r.code, r.raw)
	}
}
