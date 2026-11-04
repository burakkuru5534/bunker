package moderation

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

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
	"github.com/burakkuru5534/bunker/backend/internal/room"
	"github.com/burakkuru5534/bunker/backend/internal/testdb"
)

type api struct {
	t       *testing.T
	env     *testdb.Env
	handler http.Handler
	svc     *Service
	ip      atomic.Int64
}

func newAPI(t *testing.T) *api {
	t.Helper()
	env := testdb.New(t)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	authHTTP := auth.NewHTTP(auth.NewService(env.DB, nil, logger), 1)
	roomHTTP := room.NewHTTP(room.NewService(env.DB, nil, nil, logger), authHTTP.Require)
	svc := NewService(env.DB, logger)
	modHTTP := NewHTTP(svc, authHTTP.Require)
	h := httpapi.NewRouter(httpapi.Options{
		Logger: logger, MinAppVersion: httpapi.Version{Minor: 1}, RequestTimeout: 5 * time.Second,
		Register: func(mux *http.ServeMux) { authHTTP.Register(mux); roomHTTP.Register(mux); modHTTP.Register(mux) },
	})
	return &api{t: t, env: env, handler: h, svc: svc}
}

type resp struct {
	code int
	body map[string]any
	list []any
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
	_ = json.Unmarshal(rec.Body.Bytes(), &out.list)
	return out
}

func errCode(r resp) string {
	e, _ := r.body["error"].(map[string]any)
	c, _ := e["code"].(string)
	return c
}

func fieldErr(r resp, f string) string {
	e, _ := r.body["error"].(map[string]any)
	d, _ := e["details"].(map[string]any)
	fs, _ := d["fields"].(map[string]any)
	s, _ := fs[f].(string)
	return s
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

func TestIntegrationBlocks(t *testing.T) {
	a := newAPI(t)
	tok, me := a.guest("Burak")
	_, x := a.guest("Ayse")
	_, y := a.guest("Mehmet")
	for _, target := range []string{x, y, x} { // tekrar: değişiklik yok
		if r := a.do("POST", "/v1/blocks", tok, `{"user_id":"`+target+`"}`); r.code != 204 {
			t.Fatalf("engelle: %d %s", r.code, r.raw)
		}
	}
	r := a.do("GET", "/v1/blocks", tok, "")
	if r.code != 200 || len(r.list) != 2 {
		t.Fatalf("liste: %d %s", r.code, r.raw)
	}
	first := r.list[0].(map[string]any)
	if first["user_id"] != y || first["nickname"] != "Mehmet" || first["created_at"] == nil {
		t.Fatalf("en yeni önce: %v", first)
	}
	for name, c := range map[string]struct {
		body, field, reason string
		code                int
	}{
		"kendini": {`{"user_id":"` + me + `"}`, "user_id", "not_allowed", 400},
		"bozuk":   {`{"user_id":"x"}`, "user_id", "invalid", 400},
		"boş":     {`{}`, "user_id", "required", 400},
	} {
		if r := a.do("POST", "/v1/blocks", tok, c.body); r.code != c.code || fieldErr(r, c.field) != c.reason {
			t.Errorf("%s: %d %s", name, r.code, r.raw)
		}
	}
	if r := a.do("POST", "/v1/blocks", tok, `{"user_id":"00000000-0000-4000-8000-000000000000"}`); r.code != 404 {
		t.Fatalf("olmayan kullanıcı: %d", r.code)
	}
	// Kaldır (iki kez de 204; bozuk kimlik de 204).
	for _, id := range []string{x, x, "bozuk"} {
		if r := a.do("DELETE", "/v1/blocks/"+id, tok, ""); r.code != 204 {
			t.Fatalf("kaldır %s: %d", id, r.code)
		}
	}
	if r := a.do("GET", "/v1/blocks", tok, ""); len(r.list) != 1 {
		t.Fatalf("kaldırma sonrası %d", len(r.list))
	}
	// Başkasının listesi ayrı; oturumsuz 401.
	tok2, _ := a.guest("Zeynep")
	if r := a.do("GET", "/v1/blocks", tok2, ""); r.code != 200 || len(r.list) != 0 {
		t.Fatalf("başkasının listesi: %s", r.raw)
	}
	if r := a.do("GET", "/v1/blocks", "", ""); r.code != 401 {
		t.Fatalf("oturumsuz: %d", r.code)
	}
}

// roomPair: aynı odada iki kullanıcı ve birkaç mesaj.
func (a *api) roomPair() (roomID, reporterTok, reporter, reported string) {
	a.t.Helper()
	reporterTok, reporter = a.guest("Burak")
	tok2, reported2 := a.guest("Kotu Adam")
	r := a.do("POST", "/v1/rooms", reporterTok, "")
	roomID = r.body["id"].(string)
	a.do("POST", "/v1/rooms/join", tok2, `{"invite_code":"`+r.body["invite_code"].(string)+`"}`)
	return roomID, reporterTok, reporter, reported2
}

func (a *api) seedMessages(roomID, user string, n int) {
	a.t.Helper()
	if _, err := a.env.Migrate.Exec(context.Background(), `insert into chat_messages (room_id, user_id, client_msg_id, body, created_at)
		select $1, $2, gen_random_uuid(), 'mesaj ' || lpad(i::text, 2, '0'), now() - (60 - i) * interval '1 second'
		from generate_series(1, $3) i`, roomID, user, n); err != nil {
		a.t.Fatal(err)
	}
}

func (a *api) report(tok, reported, roomID, reason, note string) resp {
	body := fmt.Sprintf(`{"reported_user_id":%q,"room_id":%q,"reason":%q`, reported, roomID, reason)
	if note != "" {
		body += fmt.Sprintf(`,"note":%q`, note)
	}
	return a.do("POST", "/v1/reports", tok, body+"}")
}

func TestIntegrationReport(t *testing.T) {
	a := newAPI(t)
	roomID, tok, reporter, reported := a.roomPair()
	a.seedMessages(roomID, reported, 55)

	r := a.report(tok, reported, roomID, "harassment", "  sürekli\nhakaret ediyor ")
	if r.code != 201 || r.body["id"] == nil {
		t.Fatalf("rapor: %d %s", r.code, r.raw)
	}
	id := r.body["id"].(string)
	var (
		rep, rptd, reason, note, status string
		snapshot                        []map[string]any
	)
	if err := a.env.DB.Pool().QueryRow(context.Background(), `select reporter_id::text, reported_id::text, reason, note, status,
		messages_snapshot from reports where id = $1`, id).Scan(&rep, &rptd, &reason, &note, &status, &snapshot); err != nil {
		t.Fatal(err)
	}
	if rep != reporter || rptd != reported || reason != "harassment" || note != "sürekli hakaret ediyor" || status != "open" {
		t.Fatalf("rapor satırı: %s %s %s %q %s", rep, rptd, reason, note, status)
	}
	if len(snapshot) != EvidenceMessages || snapshot[0]["text"] != "mesaj 06" || snapshot[49]["text"] != "mesaj 55" ||
		snapshot[0]["nickname"] != "Kotu Adam" || snapshot[0]["user_id"] != reported {
		t.Fatalf("kanıt: %d, ilk %v, son %v", len(snapshot), snapshot[0], snapshot[len(snapshot)-1])
	}
	// Kanıt, sohbet silinse de kalır.
	if _, err := a.env.Migrate.Exec(context.Background(), `delete from chat_messages where room_id = $1`, roomID); err != nil {
		t.Fatal(err)
	}
	if n := a.val(`select jsonb_array_length(messages_snapshot) from reports where id = $1`, id); n != int32(50) && n != int64(50) {
		t.Fatalf("silme sonrası kanıt %v", n)
	}
	// Aynı açık rapor: yeni kayıt yok, aynı kimlik.
	if r2 := a.report(tok, reported, roomID, "spam", ""); r2.code != 201 || r2.body["id"] != id {
		t.Fatalf("tekrar: %d %s", r2.code, r2.raw)
	}
	if n := a.val(`select count(*) from reports`).(int64); n != 1 {
		t.Fatalf("%d rapor", n)
	}
	// Rapor kapandıysa aynı kişi yeniden raporlanabilir.
	if _, err := a.env.Migrate.Exec(context.Background(), `update reports set status = 'dismissed', resolved_at = now() where id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if r3 := a.report(tok, reported, roomID, "spam", ""); r3.code != 201 || r3.body["id"] == id {
		t.Fatalf("kapanmış rapordan sonra: %d %s", r3.code, r3.raw)
	}
}

func TestIntegrationReportRules(t *testing.T) {
	a := newAPI(t)
	roomID, tok, reporter, reported := a.roomPair()
	_, stranger := a.guest("Yabanci")
	cases := map[string]struct {
		body         string
		field, value string
	}{
		"kendini":       {fmt.Sprintf(`{"reported_user_id":%q,"room_id":%q,"reason":"spam"}`, reporter, roomID), "reported_user_id", "not_allowed"},
		"neden yok":     {fmt.Sprintf(`{"reported_user_id":%q,"room_id":%q}`, reported, roomID), "reason", "required"},
		"bilinmeyen":    {fmt.Sprintf(`{"reported_user_id":%q,"room_id":%q,"reason":"kizdim"}`, reported, roomID), "reason", "not_allowed"},
		"oda bozuk":     {fmt.Sprintf(`{"reported_user_id":%q,"room_id":"x","reason":"spam"}`, reported), "room_id", "invalid"},
		"kullanıcı yok": {fmt.Sprintf(`{"room_id":%q,"reason":"spam"}`, roomID), "reported_user_id", "required"},
		"uzun not":      {fmt.Sprintf(`{"reported_user_id":%q,"room_id":%q,"reason":"spam","note":%q}`, reported, roomID, strings.Repeat("a", 201)), "note", "too_long"},
	}
	for name, c := range cases {
		if r := a.do("POST", "/v1/reports", tok, c.body); r.code != 400 || fieldErr(r, c.field) != c.value {
			t.Errorf("%s: %d %s", name, r.code, r.raw)
		}
	}
	// Ortak oda yok: yabancıyı raporlamak, olmayan oda, raporlayanın üye olmadığı oda → aynı 404.
	otherTok, _ := a.guest("Diger")
	otherRoom := a.do("POST", "/v1/rooms", otherTok, "").body["id"].(string)
	var bodies []string
	for _, c := range []struct{ user, room string }{
		{stranger, roomID}, {reported, "00000000-0000-4000-8000-000000000000"}, {reported, otherRoom},
	} {
		r := a.report(tok, c.user, c.room, "spam", "")
		if r.code != 404 {
			t.Fatalf("ortak oda yok: %d %s", r.code, r.raw)
		}
		e := r.body["error"].(map[string]any)
		delete(e, "request_id")
		raw, _ := json.Marshal(r.body)
		bodies = append(bodies, string(raw))
	}
	if bodies[0] != bodies[1] || bodies[1] != bodies[2] {
		t.Fatalf("404 yanıtları farklı: %v", bodies)
	}
	// Lobiden ayrılınca üyelik satırı silinir: raporlayanın o odada mesajı
	// yoksa ortaklık kanıtı kalmaz → 404. Mesaj yazmışsa ayrıldıktan sonra da
	// raporlayabilir.
	a.seedMessages(roomID, reported, 1)
	a.do("POST", "/v1/rooms/"+roomID+"/leave", tok, "")
	if r := a.report(tok, reported, roomID, "hate", ""); r.code != 404 {
		t.Fatalf("kanıtsız ayrılan raporlayabildi: %d %s", r.code, r.raw)
	}
	if n := a.val(`select count(*) from reports`).(int64); n != 0 {
		t.Fatalf("reddedilen raporlar yazıldı: %d", n)
	}
	a.seedMessages(roomID, reporter, 1)
	if r := a.report(tok, reported, roomID, "hate", ""); r.code != 201 {
		t.Fatalf("mesajı olan ayrılan raporlayamadı: %d %s", r.code, r.raw)
	}
}

func TestIntegrationReportRulesNoWrites(t *testing.T) {
	a := newAPI(t)
	roomID, tok, _, _ := a.roomPair()
	_, stranger := a.guest("Yabanci2")
	a.report(tok, stranger, roomID, "spam", "")
	if n := a.val(`select count(*) from reports`).(int64); n != 0 {
		t.Fatalf("reddedilen raporlar yazıldı: %d", n)
	}
}

// Saatte 10 rapor; sınır veritabanından önce uygulanır (reddedilen denemeler
// de sayılır: kullanıcı kimliklerini yoklamaya karşı).
func TestIntegrationReportRateLimit(t *testing.T) {
	a := newAPI(t)
	p := auth.Principal{UserID: "11111111-1111-4111-8111-111111111111"}
	req := ReportRequest{ReportedUserID: "22222222-2222-4222-8222-222222222222",
		RoomID: "33333333-3333-4333-8333-333333333333", Reason: "spam"}
	for i := range 10 {
		if _, err := a.svc.Report(context.Background(), p, req); apperr.CodeOf(err) != apperr.NotFound {
			t.Fatalf("%d: %v", i, err)
		}
	}
	if _, err := a.svc.Report(context.Background(), p, req); apperr.CodeOf(err) != apperr.RateLimited {
		t.Fatalf("11.: %v", err)
	}
	// Başka kullanıcı etkilenmez.
	other := auth.Principal{UserID: "44444444-4444-4444-8444-444444444444"}
	if _, err := a.svc.Report(context.Background(), other, req); apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("başka kullanıcı: %v", err)
	}
}
