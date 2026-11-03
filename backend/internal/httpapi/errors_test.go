package httpapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/observability"
)

// docsCodes, docs/api.md §2 hata tablosunu okur: code → HTTP durumu.
func docsCodes(t *testing.T) map[apperr.Code]int {
	t.Helper()
	path := filepath.Join("..", "..", "..", "docs", "api.md")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("docs/api.md açılamadı (%s): %v", path, err)
	}
	defer f.Close()

	row := regexp.MustCompile("^\\| (\\d{3}) \\| `([A-Z_]+)` \\|")
	out := map[apperr.Code]int{}
	sc := bufio.NewScanner(f)
	inTable := false
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "### Hata kodları"):
			inTable = true
		case inTable && strings.HasPrefix(line, "## "):
			inTable = false
		case inTable:
			if m := row.FindStringSubmatch(line); m != nil {
				status, _ := strconv.Atoi(m[1])
				out[apperr.Code(m[2])] = status
			}
		}
	}
	if len(out) == 0 {
		t.Fatal("docs/api.md hata tablosu bulunamadı")
	}
	return out
}

func TestErrorCodesMatchDocs(t *testing.T) {
	docs := docsCodes(t)
	for code, status := range docs {
		got, ok := statusByCode[code]
		if !ok {
			t.Errorf("%s belgede var, kodda yok", code)
		} else if got != status {
			t.Errorf("%s: belge %d, kod %d", code, status, got)
		}
	}
	for code := range statusByCode {
		if _, ok := docs[code]; !ok {
			t.Errorf("%s kodda var, belgede yok", code)
		}
	}
	for _, code := range apperr.All {
		if _, ok := statusByCode[code]; !ok {
			t.Errorf("%s için HTTP durumu tanımlı değil", code)
		}
	}
	if len(apperr.All) != len(statusByCode) {
		t.Errorf("apperr.All (%d) ile durum tablosu (%d) farklı", len(apperr.All), len(statusByCode))
	}
}

func TestStatusOfUnknownIs500(t *testing.T) {
	if StatusOf("YOK_BOYLE") != http.StatusInternalServerError {
		t.Fatal("bilinmeyen kod 500 olmalı")
	}
}

type errResp struct {
	rec  *httptest.ResponseRecorder
	body ErrorDetail
	logs string
}

func writeErr(t *testing.T, ctx context.Context, err error) errResp {
	t.Helper()
	var logs bytes.Buffer
	logger := observability.NewLogger(&logs, slog.LevelDebug, observability.FormatJSON)
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = observability.WithLogger(ctx, logger)
	r := httptest.NewRequest(http.MethodGet, "/v1/x", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	WriteAppError(rec, r, err)

	var res errResp
	res.rec, res.logs = rec, logs.String()
	if rec.Body.Len() > 0 {
		var b ErrorBody
		if jerr := json.Unmarshal(rec.Body.Bytes(), &b); jerr != nil {
			t.Fatalf("hata gövdesi çözülemedi: %v (%s)", jerr, rec.Body.String())
		}
		res.body = b.Error
	}
	return res
}

func TestWriteAppErrorDomainError(t *testing.T) {
	err := fmt.Errorf("room: %w", apperr.New(apperr.RoomFull, "10/10 dolu").WithDetail("max_players", 10))
	res := writeErr(t, nil, err)

	if res.rec.Code != http.StatusConflict || res.body.Code != "ROOM_FULL" || res.body.Message != "10/10 dolu" {
		t.Fatalf("yanıt yanlış: %d %+v", res.rec.Code, res.body)
	}
	if res.body.Details["max_players"] != float64(10) {
		t.Fatalf("ayrıntı korunmalı: %v", res.body.Details)
	}
	if _, ok := res.body.Details["fields"]; ok {
		t.Fatal("alan hatası yoksa fields yazılmamalı")
	}
}

func TestWriteAppErrorFields(t *testing.T) {
	var fe apperr.FieldErrors
	fe.Add("nickname", apperr.FieldTooShort)
	fe.Add("capacity", apperr.FieldNotAllowed)
	res := writeErr(t, nil, fe.Err())

	if res.rec.Code != http.StatusBadRequest || res.body.Code != "VALIDATION_FAILED" {
		t.Fatalf("yanıt yanlış: %d %+v", res.rec.Code, res.body)
	}
	fields, _ := res.body.Details["fields"].(map[string]any)
	if fields["nickname"] != "too_short" || fields["capacity"] != "not_allowed" {
		t.Fatalf("details.fields yanlış: %v", res.body.Details)
	}
}

func TestWriteAppErrorRetryAfter(t *testing.T) {
	cases := map[time.Duration]string{1500 * time.Millisecond: "2", 0: "1", 30 * time.Second: "30"}
	for d, want := range cases {
		res := writeErr(t, nil, apperr.RateLimit(d))
		if res.rec.Code != http.StatusTooManyRequests || res.rec.Header().Get("Retry-After") != want {
			t.Errorf("%v: %d Retry-After=%q, %q bekleniyordu", d, res.rec.Code, res.rec.Header().Get("Retry-After"), want)
		}
	}
	if writeErr(t, nil, apperr.New(apperr.RoomFull, "")).rec.Header().Get("Retry-After") != "" {
		t.Fatal("yalnızca RATE_LIMITED Retry-After almalı")
	}
}

func TestWriteAppErrorHidesInternalDetails(t *testing.T) {
	for name, err := range map[string]error{
		"düz hata":        errors.New("pq: password=gizli bağlantı hatası"),
		"INTERNAL apperr": apperr.Wrap(apperr.Internal, "db sorgusu: users tablosu", errors.New("x")),
		"bilinmeyen kod":  apperr.New("YOK_BOYLE", "iç ayrıntı"),
	} {
		res := writeErr(t, nil, err)
		if res.rec.Code != 500 || res.body.Code != "INTERNAL" || res.body.Message != "beklenmeyen hata" {
			t.Errorf("%s: %d %+v", name, res.rec.Code, res.body)
		}
		if len(res.body.Details) != 0 {
			t.Errorf("%s: ayrıntı sızmamalı: %v", name, res.body.Details)
		}
		if !strings.Contains(res.logs, `"severity":"ERROR"`) {
			t.Errorf("%s: ERROR olarak loglanmalı: %s", name, res.logs)
		}
	}
}

func TestWriteAppErrorCauseNotSentToClient(t *testing.T) {
	res := writeErr(t, nil, apperr.Wrap(apperr.NotFound, "oda yok", errors.New("sql: no rows")))
	if strings.Contains(res.rec.Body.String(), "sql") {
		t.Fatal("alttaki neden istemciye gitmemeli")
	}
	if !strings.Contains(res.logs, "sql: no rows") {
		t.Fatal("alttaki neden debug olarak loglanmalı")
	}
}

func TestWriteAppErrorContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res := writeErr(t, ctx, ctx.Err())
	if res.rec.Body.Len() != 0 {
		t.Fatal("istemci gittiyse yanıt yazılmamalı")
	}

	// Bağlam iptal edilmemişken gelen Canceled sıradan bir hatadır.
	if writeErr(t, nil, context.Canceled).rec.Code != 500 {
		t.Fatal("bağlam canlıyken 500 olmalı")
	}

	res = writeErr(t, nil, fmt.Errorf("sorgu: %w", context.DeadlineExceeded))
	if res.rec.Code != 500 || !strings.Contains(res.logs, `"severity":"WARNING"`) {
		t.Fatalf("süre aşımı 500 + WARNING olmalı: %d %s", res.rec.Code, res.logs)
	}
}

func TestHandleAdapter(t *testing.T) {
	h := Handle(func(w http.ResponseWriter, r *http.Request) error {
		if r.URL.Query().Get("ok") == "1" {
			WriteJSON(w, http.StatusOK, map[string]string{"durum": "tamam"})
			return nil
		}
		return apperr.New(apperr.Forbidden, "oda sahibi değil")
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?ok=1", nil))
	if rec.Code != 200 {
		t.Fatalf("başarı: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), `"FORBIDDEN"`) {
		t.Fatalf("hata: %d %s", rec.Code, rec.Body.String())
	}
}

type joinReq struct {
	InviteCode string `json:"invite_code"`
	Settings   struct {
		Capacity int `json:"capacity"`
	} `json:"settings"`
}

func decodeReq(body, contentType string, limit int64) error {
	r := httptest.NewRequest(http.MethodPost, "/v1/x", strings.NewReader(body))
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	if limit > 0 {
		r.Body = http.MaxBytesReader(httptest.NewRecorder(), r.Body, limit)
	}
	var dst joinReq
	return DecodeJSON(r, &dst)
}

func TestDecodeJSONSuccess(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"invite_code":"K7M2QX","settings":{"capacity":2}}`+"\n"))
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	var dst joinReq
	if err := DecodeJSON(r, &dst); err != nil || dst.InviteCode != "K7M2QX" || dst.Settings.Capacity != 2 {
		t.Fatalf("çözülmeli: %v %+v", err, dst)
	}
	// Content-Type başlığı yoksa da kabul edilir.
	if err := decodeReq(`{}`, "", 0); err != nil {
		t.Fatalf("başlıksız JSON kabul edilmeli: %v", err)
	}
}

func TestDecodeJSONErrors(t *testing.T) {
	cases := []struct {
		name, body, ct string
		limit          int64
		field, reason  string
	}{
		{"boş gövde", "", "application/json", 0, "body", apperr.FieldRequired},
		{"söz dizimi", `{"invite_code":`, "application/json", 0, "body", apperr.FieldInvalid},
		{"bozuk", `{x}`, "application/json", 0, "body", apperr.FieldInvalid},
		{"bilinmeyen alan", `{"invite_code":"A","admin":true}`, "", 0, "admin", apperr.FieldUnknown},
		{"yanlış tür", `{"invite_code":5}`, "", 0, "invite_code", apperr.FieldInvalidType},
		{"iç içe yanlış tür", `{"settings":{"capacity":"iki"}}`, "", 0, "settings.capacity", apperr.FieldInvalidType},
		{"dizi gövde", `[1]`, "", 0, "body", apperr.FieldInvalidType},
		{"ikinci değer", `{} {}`, "", 0, "body", apperr.FieldInvalid},
		{"çöp kuyruk", `{}x`, "", 0, "body", apperr.FieldInvalid},
		{"çok büyük", `{"invite_code":"` + strings.Repeat("A", 100) + `"}`, "", 32, "body", apperr.FieldTooLarge},
		{"kuyrukta büyük", `{}` + strings.Repeat(" ", 40), "", 32, "body", apperr.FieldTooLarge},
		{"yanlış içerik türü", `{}`, "text/plain", 0, "Content-Type", apperr.FieldInvalid},
		{"bozuk içerik türü", `{}`, ";;", 0, "Content-Type", apperr.FieldInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := decodeReq(tc.body, tc.ct, tc.limit)
			e, ok := errors.AsType[*apperr.Error](err)
			if !ok || e.Code != apperr.ValidationFailed {
				t.Fatalf("VALIDATION_FAILED bekleniyordu: %v", err)
			}
			if e.Fields[tc.field] != tc.reason || len(e.Fields) != 1 {
				t.Fatalf("alanlar = %v, %s=%s bekleniyordu", e.Fields, tc.field, tc.reason)
			}
		})
	}
}

func TestDecodeJSONNilBody(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Body = nil
	var dst joinReq
	e, _ := errors.AsType[*apperr.Error](DecodeJSON(r, &dst))
	if e == nil || e.Fields["body"] != apperr.FieldRequired {
		t.Fatalf("gövde yoksa body=required: %v", e)
	}
}

// Uçtan uca: yönlendirici + Handle + DecodeJSON, gövde sınırı ve alan hatası.
func TestRouterHandleDecodeEndToEnd(t *testing.T) {
	h := NewRouter(Options{
		Logger:        slog.New(slog.DiscardHandler),
		MinAppVersion: testMin,
		Register: func(mux *http.ServeMux) {
			mux.Handle("POST /v1/rooms/join", Handle(func(w http.ResponseWriter, r *http.Request) error {
				var req joinReq
				if err := DecodeJSON(r, &req); err != nil {
					return err
				}
				var fe apperr.FieldErrors
				if len(req.InviteCode) != 6 {
					fe.Add("invite_code", apperr.FieldInvalid)
				}
				if err := fe.Err(); err != nil {
					return err
				}
				return apperr.New(apperr.InviteCodeNotFound, "")
			}))
		},
	})

	send := func(body string, chunked bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/v1/rooms/join", strings.NewReader(body))
		r.Header.Set(AppVersionHeader, "0.2.0")
		r.Header.Set("Content-Type", "application/json")
		if chunked {
			r.ContentLength = -1
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec
	}

	if rec := send(`{"invite_code":"K7"}`, false); rec.Code != 400 || !strings.Contains(rec.Body.String(), `"fields":{"invite_code":"invalid"}`) {
		t.Fatalf("alan hatası: %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(`{"invite_code":"K7M2QX"}`, false); rec.Code != 404 || !strings.Contains(rec.Body.String(), "INVITE_CODE_NOT_FOUND") {
		t.Fatalf("domain hatası: %d %s", rec.Code, rec.Body.String())
	}
	big := `{"invite_code":"` + strings.Repeat("A", MaxBodyBytes) + `"}`
	for _, chunked := range []bool{false, true} {
		rec := send(big, chunked)
		if rec.Code != 400 || !strings.Contains(rec.Body.String(), `"body":"too_large"`) {
			t.Fatalf("büyük gövde (chunked=%v): %d %s", chunked, rec.Code, rec.Body.String())
		}
	}
}
