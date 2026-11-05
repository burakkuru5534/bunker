package auth

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
	"github.com/burakkuru5534/bunker/backend/internal/observability"
	"github.com/burakkuru5534/bunker/backend/internal/testdb"
)

type apiTest struct {
	t       *testing.T
	handler http.Handler
	logs    *bytes.Buffer
}

func newAPI(t *testing.T, trustedHops int) *apiTest {
	t.Helper()
	env := testdb.New(t)
	var logs bytes.Buffer
	logger := observability.NewLogger(&logs, slog.LevelDebug, observability.FormatJSON)
	h := NewHTTP(NewService(env.DB, nil, logger), trustedHops)
	router := httpapi.NewRouter(httpapi.Options{
		Logger: logger, MinAppVersion: httpapi.Version{Major: 0, Minor: 1, Patch: 0},
		RequestTimeout: 5 * time.Second, Register: h.Register, TrustedProxyHops: trustedHops,
	})
	return &apiTest{t: t, handler: router, logs: &logs}
}

// events, verilen güvenlik olayının log satırlarıdır (SAFE-011).
func (a *apiTest) events(name string) []map[string]any {
	a.t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(a.logs.String()), "\n") {
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) == nil && m["security_event"] == name {
			out = append(out, m)
		}
	}
	return out
}

type resp struct {
	code   int
	header http.Header
	body   map[string]any
	raw    string
}

func (a *apiTest) do(method, path, token, body string, headers ...string) resp {
	a.t.Helper()
	var rd *strings.Reader
	if body != "" {
		rd = strings.NewReader(body)
	} else {
		rd = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set(httpapi.AppVersionHeader, "0.1.0")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.RemoteAddr = "192.0.2.10:5555"
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	out := resp{code: rec.Code, header: rec.Header(), raw: rec.Body.String()}
	_ = json.Unmarshal(rec.Body.Bytes(), &out.body)
	return out
}

func errCode(r resp) string {
	e, _ := r.body["error"].(map[string]any)
	c, _ := e["code"].(string)
	return c
}

func TestIntegrationHTTPGuestMeRefresh(t *testing.T) {
	a := newAPI(t, 0)

	r := a.do("POST", "/v1/sessions/guest", "", `{"nickname":" Işık "}`)
	if r.code != http.StatusCreated || r.header.Get("Cache-Control") != "no-store" {
		t.Fatalf("guest: %d %s", r.code, r.raw)
	}
	token, _ := r.body["token"].(string)
	user, _ := r.body["user"].(map[string]any)
	if len(token) != TokenLength || user["nickname"] != "Işık" || user["id"] == "" {
		t.Fatalf("guest gövdesi: %s", r.raw)
	}
	exp, err := time.Parse(time.RFC3339, r.body["expires_at"].(string))
	if err != nil || exp.Location() != time.UTC || time.Until(exp) < 89*24*time.Hour {
		t.Fatalf("expires_at RFC 3339 UTC, ~90 gün: %v %v", r.body["expires_at"], err)
	}
	if strings.Contains(a.logs.String(), token) {
		t.Fatal("token loglanmamalı")
	}

	me := a.do("GET", "/v1/me", token, "")
	if me.code != 200 || me.body["nickname"] != "Işık" || me.body["id"] != user["id"] {
		t.Fatalf("me: %d %s", me.code, me.raw)
	}
	if v, ok := me.body["active_room_id"]; !ok || v != nil {
		t.Fatalf("active_room_id null olmalı: %s", me.raw)
	}

	ref := a.do("POST", "/v1/sessions/refresh", token, "")
	newToken, _ := ref.body["token"].(string)
	if ref.code != 200 || newToken == "" || newToken == token || ref.body["user"] != nil || ref.header.Get("Cache-Control") != "no-store" {
		t.Fatalf("refresh: %d %s", ref.code, ref.raw)
	}
	if a.do("GET", "/v1/me", token, "").code != 401 {
		t.Fatal("eski token 401 olmalı")
	}
	if a.do("GET", "/v1/me", newToken, "").code != 200 {
		t.Fatal("yeni token 200 olmalı")
	}
	// {} gövdesiyle de yenilenir; bilinmeyen alan reddedilir.
	if r := a.do("POST", "/v1/sessions/refresh", newToken, `{"x":1}`); r.code != 400 || errCode(r) != "VALIDATION_FAILED" {
		t.Fatalf("bilinmeyen alan: %d %s", r.code, r.raw)
	}
	if r := a.do("POST", "/v1/sessions/refresh", newToken, `{}`); r.code != 200 {
		t.Fatalf("{} gövdeli yenileme: %d %s", r.code, r.raw)
	}
}

func TestIntegrationHTTPUnauthorized(t *testing.T) {
	a := newAPI(t, 0)
	for name, auth := range map[string]string{
		"başlık yok": "",
		"bozuk":      "Bearer abc",
		"basic":      "Basic dXNlcjpwYXNz",
		"bilinmeyen": "Bearer " + strings.Repeat("A", TokenLength),
	} {
		r := a.do("GET", "/v1/me", "", "", "Authorization", auth)
		if r.code != 401 || errCode(r) != "UNAUTHORIZED" || !strings.HasPrefix(r.header.Get("WWW-Authenticate"), "Bearer") {
			t.Errorf("%s: %d %s %q", name, r.code, r.raw, r.header.Get("WWW-Authenticate"))
		}
	}
	if r := a.do("POST", "/v1/sessions/refresh", "", ""); r.code != 401 {
		t.Fatalf("refresh yetkisiz: %d", r.code)
	}
	// Her ret güvenlik olayıdır; denenen token loglanmaz.
	ev := a.events("auth_failed")
	if len(ev) != 5 || ev[0]["client_ip"] != "192.0.2.10" || ev[0]["route"] != "GET /v1/me" {
		t.Fatalf("auth_failed olayları: %v", ev)
	}
	if strings.Contains(a.logs.String(), strings.Repeat("A", TokenLength)) || strings.Contains(a.logs.String(), "dXNlcjpwYXNz") {
		t.Fatal("denenen kimlik bilgisi loglandı")
	}
}

func TestIntegrationHTTPGuestValidation(t *testing.T) {
	cases := []struct {
		body, code, field string
	}{
		{`{"nickname":"ab"}`, "NICKNAME_INVALID", "too_short"},
		{`{"nickname":"Бурак"}`, "NICKNAME_INVALID", "invalid_chars"},
		{`{}`, "NICKNAME_INVALID", "required"},
		{`{"nickname":"Burak","admin":true}`, "VALIDATION_FAILED", "unknown_field"},
		{`{"nickname":5}`, "VALIDATION_FAILED", "invalid_type"},
	}
	// Her durumu ayrı IP'den gönder (sınıra takılmasın).
	a2 := newAPI(t, 1)
	for i, c := range cases {
		r := a2.do("POST", "/v1/sessions/guest", "", c.body, "X-Forwarded-For", "198.51.100."+strconv.Itoa(i+1))
		e, _ := r.body["error"].(map[string]any)
		d, _ := e["details"].(map[string]any)
		f, _ := d["fields"].(map[string]any)
		got := ""
		for _, v := range f {
			got, _ = v.(string)
		}
		if r.code != 400 || errCode(r) != c.code || got != c.field {
			t.Errorf("%s: %d %s", c.body, r.code, r.raw)
		}
	}
}

func TestIntegrationHTTPGuestRateLimit(t *testing.T) {
	a := newAPI(t, 1)
	send := func(xff string) resp {
		return a.do("POST", "/v1/sessions/guest", "", `{"nickname":"Oyuncu"}`, "X-Forwarded-For", xff)
	}
	for i := range GuestLimit {
		if r := send("203.0.113.7"); r.code != 201 {
			t.Fatalf("%d. istek 201 olmalı: %d %s", i+1, r.code, r.raw)
		}
	}
	r := send("203.0.113.7")
	retry, _ := strconv.Atoi(r.header.Get("Retry-After"))
	if r.code != 429 || errCode(r) != "RATE_LIMITED" || retry < 1 || retry > 12 {
		t.Fatalf("6. istek 429 + Retry-After: %d %q %s", r.code, r.header.Get("Retry-After"), r.raw)
	}
	// Açılan hesaplar ve ret, gerçek istemci IP'siyle güvenlik olayıdır.
	created := a.events("session_created")
	if len(created) != GuestLimit || created[0]["client_ip"] != "203.0.113.7" || created[0]["user_id"] == "" {
		t.Fatalf("session_created: %v", created)
	}
	limited := a.events("rate_limited")
	if len(limited) != 1 || limited[0]["client_ip"] != "203.0.113.7" || limited[0]["route"] != "POST /v1/sessions/guest" {
		t.Fatalf("rate_limited: %v", limited)
	}
	// İstemcinin yazdığı sol girdiler sınırı aşmaya yaramaz.
	if r := send("1.1.1.1, 203.0.113.7"); r.code != 429 {
		t.Fatalf("sahte XFF ile atlatılamamalı: %d", r.code)
	}
	// Başka IP etkilenmez; geçersiz denemeler de sayılır (önce sınır).
	if r := send("203.0.113.8"); r.code != 201 {
		t.Fatalf("başka IP: %d", r.code)
	}
	// Aynı IPv6 /64 tek anahtar.
	for i := range GuestLimit {
		send("2001:db8:1:2::" + strconv.Itoa(i+1))
	}
	if r := send("2001:db8:1:2::ffff"); r.code != 429 {
		t.Fatalf("aynı /64 sınırı paylaşmalı: %d", r.code)
	}
}

func TestIntegrationHTTPUserRateLimitAndHeaderOnly(t *testing.T) {
	a := newAPI(t, 1)
	guest := func(ip, nick string) string {
		r := a.do("POST", "/v1/sessions/guest", "", `{"nickname":"`+nick+`"}`, "X-Forwarded-For", ip)
		tok, _ := r.body["token"].(string)
		return tok
	}
	alice, bob := guest("198.51.100.1", "Alice"), guest("198.51.100.2", "Bobby")

	for i := range UserLimit {
		if r := a.do("GET", "/v1/me", alice, ""); r.code != 200 {
			t.Fatalf("%d. istek 200 olmalı: %d", i+1, r.code)
		}
	}
	r := a.do("GET", "/v1/me", alice, "")
	if r.code != 429 || errCode(r) != "RATE_LIMITED" || r.header.Get("Retry-After") == "" {
		t.Fatalf("31. istek 429 olmalı: %d %s", r.code, r.raw)
	}
	// Kullanıcı sınırı olayı kimin olduğunu taşır.
	if ev := a.events("rate_limited"); len(ev) != 1 || ev[0]["user_id"] == nil || ev[0]["route"] != "GET /v1/me" {
		t.Fatalf("kullanıcı sınırı olayı: %v", ev)
	}
	// Sınır kullanıcı başınadır.
	if r := a.do("GET", "/v1/me", bob, ""); r.code != 200 {
		t.Fatalf("başka kullanıcı etkilenmemeli: %d", r.code)
	}
	// Geçersiz token sınır kotasını tüketmez (önce kimlik).
	if r := a.do("GET", "/v1/me", "", ""); r.code != 401 {
		t.Fatalf("yetkisiz istek 401: %d", r.code)
	}
	// Token URL'den kabul edilmez.
	for _, q := range []string{"access_token", "token"} {
		if r := a.do("GET", "/v1/me?"+q+"="+bob, "", ""); r.code != 401 {
			t.Errorf("?%s ile kimlik doğrulanmamalı: %d", q, r.code)
		}
	}
}
