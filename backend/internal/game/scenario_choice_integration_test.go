package game

import (
	"context"
	"testing"

	"github.com/burakkuru5534/bunker/backend/internal/room"
)

func fieldCode(r resp, field string) string {
	e, _ := r.body["error"].(map[string]any)
	d, _ := e["details"].(map[string]any)
	f, _ := d["fields"].(map[string]any)
	c, _ := f[field].(string)
	return c
}

func (a *api) settings(l lobby, tok, body string) resp {
	return a.do("PATCH", "/v1/rooms/"+l.roomID+"/settings", tok, body)
}

func (a *api) lastSettingsEvent() map[string]any {
	evs := a.events.all()
	for i := len(evs) - 1; i >= 0; i-- {
		if evs[i].Type == room.EventSettingsChanged {
			return evs[i].Payload.(map[string]any)
		}
	}
	return nil
}

func TestIntegrationScenarioList(t *testing.T) {
	a := newAPI(t, true)
	tok, _ := a.guest("Burak")
	r := a.do("GET", "/v1/scenarios", tok, "")
	if r.code != 200 {
		t.Fatalf("liste: %d %s", r.code, r.raw)
	}
	list, _ := r.body["scenarios"].([]any)
	var ids []string
	for _, s := range list {
		m := s.(map[string]any)
		if m["title"] != "Senaryo "+m["id"].(string) || len(m) != 2 {
			t.Fatalf("satır: %v", m)
		}
		ids = append(ids, m["id"].(string))
	}
	// Başlığa göre sıralı; yalnızca aktifler.
	if want := []string{"meteor", "nuclear_winter", "red_cough"}; len(ids) != 3 || ids[0] != want[0] || ids[1] != want[1] || ids[2] != want[2] {
		t.Fatalf("sıra: %v", ids)
	}
	if _, err := a.env.Migrate.Exec(context.Background(), `update scenarios set active = false where id = 'meteor'`); err != nil {
		t.Fatal(err)
	}
	if r := a.do("GET", "/v1/scenarios", tok, ""); len(r.body["scenarios"].([]any)) != 2 {
		t.Fatalf("pasif listelenmez: %s", r.raw)
	}
	if r := a.do("GET", "/v1/scenarios", "", ""); r.code != 401 {
		t.Fatalf("oturumsuz: %d", r.code)
	}
}

func TestIntegrationScenarioSettings(t *testing.T) {
	a := newAPI(t, true)
	l := a.lobby(4, 2, false)
	host := l.tokens[0]

	if r := a.do("GET", "/v1/rooms/"+l.roomID, host, ""); r.body["scenario_id"] != nil {
		t.Fatalf("varsayılan rastgele: %s", r.raw)
	}
	r := a.settings(l, host, `{"scenario_id":"meteor"}`)
	if r.code != 200 || r.body["scenario_id"] != "meteor" || r.body["capacity"] != float64(2) {
		t.Fatalf("seç: %d %s", r.code, r.raw)
	}
	v := r.body["version"]
	if e := a.lastSettingsEvent(); e == nil || *(e["scenario_id"].(*string)) != "meteor" || e["capacity"] != 2 {
		t.Fatalf("olay: %v", e)
	}
	n := len(a.events.all())
	if r := a.settings(l, host, `{"scenario_id":"meteor"}`); r.code != 200 || r.body["version"] != v || len(a.events.all()) != n {
		t.Fatal("aynı seçim: değişiklik yok")
	}
	// Yalnızca kapasite: seçim korunur.
	if r := a.settings(l, host, `{"capacity":1}`); r.body["scenario_id"] != "meteor" || r.body["capacity"] != float64(1) {
		t.Fatalf("kapasite seçimi silmemeli: %s", r.raw)
	}
	// null → rastgele.
	if r := a.settings(l, host, `{"scenario_id":null}`); r.code != 200 || r.body["scenario_id"] != nil {
		t.Fatalf("rastgele: %d %s", r.code, r.raw)
	}
	if e := a.lastSettingsEvent(); e["scenario_id"].(*string) != nil {
		t.Fatalf("olay rastgele: %v", e)
	}
	// Birlikte.
	if r := a.settings(l, host, `{"capacity":2,"scenario_id":"red_cough"}`); r.body["scenario_id"] != "red_cough" || r.body["capacity"] != float64(2) {
		t.Fatalf("birlikte: %s", r.raw)
	}

	for name, c := range map[string]struct {
		tok, body, code, field, fcode string
		status                        int
	}{
		"bilinmeyen":       {host, `{"scenario_id":"yok"}`, "VALIDATION_FAILED", "scenario_id", "not_allowed", 400},
		"yanlış tür":       {host, `{"scenario_id":5}`, "VALIDATION_FAILED", "scenario_id", "invalid_type", 400},
		"boş gövde":        {host, `{}`, "VALIDATION_FAILED", "capacity", "required", 400},
		"oda sahibi değil": {l.tokens[1], `{"scenario_id":"meteor"}`, "FORBIDDEN", "", "", 403},
	} {
		r := a.settings(l, c.tok, c.body)
		if r.code != c.status || errCode(r) != c.code || (c.field != "" && fieldCode(r, c.field) != c.fcode) {
			t.Errorf("%s: %d %s", name, r.code, r.raw)
		}
	}
	if _, err := a.env.Migrate.Exec(context.Background(), `update scenarios set active = false where id = 'meteor'`); err != nil {
		t.Fatal(err)
	}
	if r := a.settings(l, host, `{"scenario_id":"meteor"}`); r.code != 400 || fieldCode(r, "scenario_id") != "not_allowed" {
		t.Fatalf("pasif senaryo: %d %s", r.code, r.raw)
	}
}

func TestIntegrationStartWithChosenScenario(t *testing.T) {
	a := newAPI(t, true)
	for _, id := range []string{"red_cough", "meteor", "nuclear_winter"} {
		l := a.lobby(4, 2, true)
		if r := a.settings(l, l.tokens[0], `{"scenario_id":"`+id+`"}`); r.code != 200 {
			t.Fatalf("seç: %s", r.raw)
		}
		r := a.start(l, l.tokens[0])
		if r.code != 201 {
			t.Fatalf("başlat: %d %s", r.code, r.raw)
		}
		if got := a.scalar(`select scenario_id from games where id = $1`, r.body["game_id"]); got != id {
			t.Fatalf("seçilen %s, oyun %v", id, got)
		}
		// Tekrar oynamada da seçim odada kalır.
		if got := a.scalar(`select scenario_id from rooms where id = $1`, l.roomID); got != id {
			t.Fatalf("oda seçimi: %v", got)
		}
	}
}

func TestIntegrationStartFallsBackWhenChoiceInactive(t *testing.T) {
	a := newAPI(t, true)
	l := a.lobby(4, 2, true)
	if r := a.settings(l, l.tokens[0], `{"scenario_id":"meteor"}`); r.code != 200 {
		t.Fatal(r.raw)
	}
	if _, err := a.env.Migrate.Exec(context.Background(), `update scenarios set active = false where id = 'meteor'`); err != nil {
		t.Fatal(err)
	}
	r := a.start(l, l.tokens[0])
	if r.code != 201 {
		t.Fatalf("pasif seçimde başlatma rastgeleye düşmeli: %d %s", r.code, r.raw)
	}
	if got := a.scalar(`select scenario_id from games where id = $1`, r.body["game_id"]); got == "meteor" {
		t.Fatal("pasif senaryo kullanılmamalı")
	}
}
