package room

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
	"github.com/burakkuru5534/bunker/backend/internal/testdb"
)

type api struct {
	t       *testing.T
	env     *testdb.Env
	svc     *Service
	handler http.Handler
	ip      int
}

func newAPI(t *testing.T) *api {
	t.Helper()
	env := testdb.New(t)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	authHTTP := auth.NewHTTP(auth.NewService(env.DB, nil, logger), 1)
	svc := NewService(env.DB, nil, logger)
	roomHTTP := NewHTTP(svc, authHTTP.Require)
	h := httpapi.NewRouter(httpapi.Options{
		Logger: logger, MinAppVersion: httpapi.Version{Minor: 1}, RequestTimeout: 5 * time.Second,
		Register: func(mux *http.ServeMux) { authHTTP.Register(mux); roomHTTP.Register(mux) },
	})
	return &api{t: t, env: env, svc: svc, handler: h}
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
	a.ip++
	req.Header.Set("X-Forwarded-For", "198.51.100."+strconv.Itoa(a.ip%250+1))
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	out := resp{code: rec.Code, raw: rec.Body.String()}
	_ = json.Unmarshal(rec.Body.Bytes(), &out.body)
	return out
}

// guest, misafir oturumu açıp token ve kullanıcı kimliğini döndürür.
func (a *api) guest(nick string) (token, userID string) {
	a.t.Helper()
	r := a.do("POST", "/v1/sessions/guest", "", `{"nickname":"`+nick+`"}`)
	if r.code != 201 {
		a.t.Fatalf("guest: %d %s", r.code, r.raw)
	}
	u := r.body["user"].(map[string]any)
	return r.body["token"].(string), u["id"].(string)
}

func errCode(r resp) string {
	e, _ := r.body["error"].(map[string]any)
	c, _ := e["code"].(string)
	return c
}

func TestIntegrationCreateRoom(t *testing.T) {
	a := newAPI(t)
	tok, uid := a.guest("Burak")

	r := a.do("POST", "/v1/rooms", tok, "")
	if r.code != 201 {
		t.Fatalf("oda: %d %s", r.code, r.raw)
	}
	code, _ := r.body["invite_code"].(string)
	if _, err := NormalizeInviteCode(code); err != nil || r.body["status"] != "lobby" || r.body["capacity"] != float64(2) ||
		r.body["host_user_id"] != uid || r.body["current_game_id"] != nil || r.body["version"] != float64(0) {
		t.Fatalf("RoomView: %s", r.raw)
	}
	members := r.body["members"].([]any)
	m := members[0].(map[string]any)
	if len(members) != 1 || m["user_id"] != uid || m["nickname"] != "Burak" || m["role"] != "player" ||
		m["is_ready"] != false || m["connected"] != false {
		t.Fatalf("üye: %s", r.raw)
	}

	// /me aktif odayı gösterir.
	if me := a.do("GET", "/v1/me", tok, ""); me.body["active_room_id"] != r.body["id"] {
		t.Fatalf("me aktif oda: %s", me.raw)
	}

	// İkinci oda: ALREADY_IN_ROOM + mevcut oda kimliği.
	again := a.do("POST", "/v1/rooms", tok, `{"capacity":1}`)
	details, _ := again.body["error"].(map[string]any)["details"].(map[string]any)
	if again.code != 409 || errCode(again) != "ALREADY_IN_ROOM" || details["room_id"] != r.body["id"] {
		t.Fatalf("ikinci oda: %d %s", again.code, again.raw)
	}
	var rooms int
	_ = a.env.Admin.QueryRow(context.Background(), "select count(*) from "+a.env.Schema+".rooms").Scan(&rooms)
	if rooms != 1 {
		t.Fatalf("başarısız denemede oda satırı kalmamalı: %d", rooms)
	}
}

func TestIntegrationCreateRoomCapacityAndValidation(t *testing.T) {
	a := newAPI(t)
	for i, c := range []struct {
		body string
		code int
		cap  float64
		err  string
	}{
		{`{"capacity":1}`, 201, 1, ""},
		{`{}`, 201, 2, ""},
		{`{"capacity":3}`, 400, 0, "VALIDATION_FAILED"},
		{`{"capacity":"iki"}`, 400, 0, "VALIDATION_FAILED"},
		{`{"capacity":2,"public":true}`, 400, 0, "VALIDATION_FAILED"},
	} {
		tok, _ := a.guest("Oyuncu" + strconv.Itoa(i))
		r := a.do("POST", "/v1/rooms", tok, c.body)
		if r.code != c.code || (c.cap != 0 && r.body["capacity"] != c.cap) || (c.err != "" && errCode(r) != c.err) {
			t.Errorf("%s: %d %s", c.body, r.code, r.raw)
		}
	}
	if r := a.do("POST", "/v1/rooms", "", ""); r.code != 401 {
		t.Fatalf("yetkisiz: %d", r.code)
	}
}

func TestIntegrationInviteCodeCollisionRetry(t *testing.T) {
	a := newAPI(t)
	tokA, _ := a.guest("Burak")
	tokB, _ := a.guest("Ayşe")
	first := a.do("POST", "/v1/rooms", tokA, "")
	taken := first.body["invite_code"].(string)

	// Üretici önce açık bir odanın kodunu, sonra yenisini verir.
	calls := 0
	a.svc.newCode = func() (string, error) {
		calls++
		if calls == 1 {
			return taken, nil
		}
		return NewInviteCode()
	}
	r := a.do("POST", "/v1/rooms", tokB, "")
	if r.code != 201 || r.body["invite_code"] == taken || calls != 2 {
		t.Fatalf("çakışmada yeniden denenmeli: %d %s (çağrı %d)", r.code, r.raw, calls)
	}

	// Hep çakışırsa sınırlı deneme sonrası hata (sonsuz döngü yok).
	tokC, _ := a.guest("Mehmet")
	a.svc.newCode = func() (string, error) { return taken, nil }
	if r := a.do("POST", "/v1/rooms", tokC, ""); r.code != 500 {
		t.Fatalf("deneme sınırı: %d %s", r.code, r.raw)
	}
}

func TestIntegrationConcurrentCreateSameUser(t *testing.T) {
	a := newAPI(t)
	tok, _ := a.guest("Burak")
	p, err := auth.NewService(a.env.DB, nil, nil).Authenticate(context.Background(), tok)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok, already := 0, 0
	for range 5 {
		wg.Go(func() {
			_, err := a.svc.Create(context.Background(), p, 2)
			mu.Lock()
			defer mu.Unlock()
			switch apperr.CodeOf(err) {
			case apperr.AlreadyInRoom:
				already++
			default:
				if err == nil {
					ok++
				} else {
					t.Errorf("beklenmeyen: %v", err)
				}
			}
		})
	}
	wg.Wait()
	if ok != 1 || already != 4 {
		t.Fatalf("eşzamanlı: %d oda, %d ALREADY_IN_ROOM", ok, already)
	}
}

func TestIntegrationLoadForUpdate(t *testing.T) {
	a := newAPI(t)
	tok, uid := a.guest("Burak")
	created := a.do("POST", "/v1/rooms", tok, `{"capacity":1}`)
	id := created.body["id"].(string)
	err := a.env.DB.Tx(context.Background(), func(tx pgx.Tx) error {
		r, err := loadForUpdate(context.Background(), tx, id)
		if err != nil {
			return err
		}
		if r.HostUserID != uid || r.Capacity != 1 || r.Status != StatusLobby || len(r.Members) != 1 || r.Members[0].Role != RolePlayer {
			t.Errorf("okunan oda: %+v", r)
		}
		_, err = loadForUpdate(context.Background(), tx, "00000000-0000-0000-0000-000000000000")
		if err != errRoomNotFound {
			t.Errorf("olmayan oda: %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
