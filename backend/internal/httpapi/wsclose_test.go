package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
)

func TestCloseCodeFor(t *testing.T) {
	cases := map[error]int{
		apperr.New(apperr.Unauthorized, ""):    4001,
		apperr.New(apperr.NotAMember, ""):      4003,
		apperr.New(apperr.Kicked, ""):          4003,
		apperr.New(apperr.NotFound, ""):        4003,
		apperr.New(apperr.UpgradeRequired, ""): 4026,
		apperr.Field("room_id", "invalid"):     4003,
		apperr.RateLimit(0):                    4008,
		errors.New("db"):                       1011,
		apperr.New(apperr.Internal, ""):        1011,
	}
	for err, want := range cases {
		if got := CloseCodeFor(err); got != want {
			t.Errorf("%v → %d, beklenen %d", err, got, want)
		}
	}
}

func TestWebSocketPathSkipsVersionMiddleware(t *testing.T) {
	var reached bool
	h := NewRouter(Options{
		MinAppVersion: testMin,
		Register: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /v1/ws", func(w http.ResponseWriter, r *http.Request) {
				reached = true
				// İşleyici sürümü kendisi denetler.
				if err := CheckAppVersion(r, testMin); VersionCloseCode(err) != CloseUpgradeRequired {
					t.Errorf("eski sürüm 4026'ya eşlenmeli: %v", err)
				}
				w.WriteHeader(http.StatusSwitchingProtocols)
			})
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/ws?room_id=x", nil)
	req.Header.Set(AppVersionHeader, "0.0.1") // eski
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !reached || rec.Code != http.StatusSwitchingProtocols {
		t.Fatalf("WS yolu sürüm middleware'ine takılmamalı: %d", rec.Code)
	}
	// Diğer /v1 yolları hâlâ 426 alır.
	req = httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.Header.Set(AppVersionHeader, "0.0.1")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUpgradeRequired {
		t.Fatalf("REST 426 almalı: %d", rec.Code)
	}
}

func TestCheckAppVersion(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	err := CheckAppVersion(r, testMin)
	if apperr.CodeOf(err) != apperr.ValidationFailed || VersionCloseCode(err) != CloseUpgradeRequired {
		t.Fatal("eksik başlık: VALIDATION_FAILED, WS'de 4026")
	}
	if VersionCloseCode(nil) != 0 {
		t.Fatal("hata yoksa 0")
	}
	r.Header.Set(AppVersionHeader, "0.2.0")
	if CheckAppVersion(r, testMin) != nil {
		t.Fatal("geçerli sürüm")
	}
}
