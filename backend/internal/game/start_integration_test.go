package game

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
	"github.com/burakkuru5534/bunker/backend/internal/room"
	"github.com/burakkuru5534/bunker/backend/internal/testdb"
)

type recorder struct {
	mu     sync.Mutex
	events []room.Event
}

func (r *recorder) Publish(e room.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
}
func (r *recorder) all() []room.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.events)
}

type api struct {
	t       *testing.T
	env     *testdb.Env
	events  *recorder
	handler http.Handler
	starter *Starter
	svc     *Service
	timers  *Timers
	logger  *slog.Logger
	ip      atomic.Int64
}

func newAPI(t *testing.T, seed bool) *api {
	t.Helper()
	env := testdb.New(t)
	if seed {
		seedCatalog(t, env)
	}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	rec := &recorder{}
	starter := NewStarter(logger)
	authHTTP := auth.NewHTTP(auth.NewService(env.DB, nil, logger), 1)
	timers := NewTimers(logger)
	t.Cleanup(func() { _ = timers.Stop(context.Background()) })
	pub := timers.Publisher(rec)
	roomHTTP := room.NewHTTP(room.NewService(env.DB, nil, pub, logger).WithGames(starter), authHTTP.Require)
	svc := NewService(env.DB, pub, logger)
	timers.Bind(svc)
	gameHTTP := NewHTTP(svc, authHTTP.Require)
	h := httpapi.NewRouter(httpapi.Options{
		Logger: logger, MinAppVersion: httpapi.Version{Minor: 1}, RequestTimeout: 5 * time.Second,
		Register: func(mux *http.ServeMux) { authHTTP.Register(mux); roomHTTP.Register(mux); gameHTTP.Register(mux) },
	})
	return &api{t: t, env: env, events: rec, handler: h, starter: starter, svc: svc, timers: timers, logger: logger}
}

// seedCatalog: 3 senaryo; kategori başına 12 kart, her kategoride 4 olumsuz.
func seedCatalog(t *testing.T, env *testdb.Env) {
	t.Helper()
	ctx := context.Background()
	for _, id := range []string{"nuclear_winter", "red_cough", "meteor"} {
		if _, err := env.Migrate.Exec(ctx, `insert into scenarios (id, title, disaster, duration_label, shelter_features, outside_world)
			values ($1, $2, $3, '3 yıl', array['Erzak tam','Isıtma bakım istiyor','Tıbbi malzeme az'], $4)`,
			id, "Senaryo "+id, "Uzun bir felaket metni burada yer alır, en az yirmi karakter.",
			"Dış dünya hakkında yeterince uzun bir açıklama metni."); err != nil {
			t.Fatal(err)
		}
	}
	for ci, c := range engine.Categories {
		for i := range 12 {
			balance := []int{-1, 0, 1}[i%3]
			if _, err := env.Migrate.Exec(ctx, `insert into card_definitions (id, category, text, balance) values ($1, $2, $3, $4)`,
				(ci+1)*1000+i, string(c), fmt.Sprintf("%s kartı (%02d)", c, i), balance); err != nil {
				t.Fatal(err)
			}
		}
	}
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
	ip := a.ip.Add(1)
	req.Header.Set("X-Forwarded-For", "198.51.100."+strconv.Itoa(int(ip%250)+1))
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	out := resp{code: rec.Code, raw: rec.Body.String()}
	_ = json.Unmarshal(rec.Body.Bytes(), &out.body)
	return out
}

func errCode(r resp) string {
	e, _ := r.body["error"].(map[string]any)
	c, _ := e["code"].(string)
	return c
}

func (a *api) guest(nick string) (token, userID string) {
	a.t.Helper()
	r := a.do("POST", "/v1/sessions/guest", "", `{"nickname":"`+nick+`"}`)
	if r.code != 201 {
		a.t.Fatalf("guest: %d %s", r.code, r.raw)
	}
	return r.body["token"].(string), r.body["user"].(map[string]any)["id"].(string)
}

type lobby struct {
	roomID string
	tokens []string // 0: oda sahibi
	users  []string
}

// lobby: n oyunculu oda; ready ise herkes hazır.
func (a *api) lobby(n, capacity int, ready bool) lobby {
	a.t.Helper()
	var l lobby
	for i := range n {
		tok, uid := a.guest(fmt.Sprintf("Oyuncu%d", i))
		l.tokens, l.users = append(l.tokens, tok), append(l.users, uid)
	}
	r := a.do("POST", "/v1/rooms", l.tokens[0], fmt.Sprintf(`{"capacity":%d}`, capacity))
	if r.code != 201 {
		a.t.Fatalf("oda: %d %s", r.code, r.raw)
	}
	l.roomID = r.body["id"].(string)
	code := r.body["invite_code"].(string)
	for i := 1; i < n; i++ {
		if r := a.do("POST", "/v1/rooms/join", l.tokens[i], `{"invite_code":"`+code+`"}`); r.code != 200 {
			a.t.Fatalf("katıl: %d %s", r.code, r.raw)
		}
	}
	if ready {
		for _, tok := range l.tokens {
			if r := a.do("POST", "/v1/rooms/"+l.roomID+"/ready", tok, `{"ready":true}`); r.code != 204 {
				a.t.Fatalf("hazır: %d %s", r.code, r.raw)
			}
		}
	}
	return l
}

func (a *api) start(l lobby, tok string) resp {
	return a.do("POST", "/v1/rooms/"+l.roomID+"/start", tok, "")
}

func (a *api) scalar(sql string, args ...any) any {
	a.t.Helper()
	var v any
	if err := a.env.DB.Pool().QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		a.t.Fatalf("%s: %v", sql, err)
	}
	return v
}

func TestIntegrationStartGame(t *testing.T) {
	a := newAPI(t, true)
	l := a.lobby(6, 2, true)
	roomVersion := a.scalar(`select version from rooms where id = $1`, l.roomID).(int64)
	before := len(a.events.all())

	r := a.start(l, l.tokens[0])
	if r.code != 201 {
		t.Fatalf("başlat: %d %s", r.code, r.raw)
	}
	gameID, _ := r.body["game_id"].(string)

	// Oda.
	var status string
	var current *string
	var version int64
	if err := a.env.DB.Pool().QueryRow(context.Background(), `select status, current_game_id::text, version from rooms where id = $1`,
		l.roomID).Scan(&status, &current, &version); err != nil {
		t.Fatal(err)
	}
	if status != "in_game" || current == nil || *current != gameID || version != roomVersion+1 {
		t.Fatalf("oda: %s %v v%d", status, current, version)
	}

	// Oyun satırı.
	var (
		phase                    string
		round, totalRounds, capa int16
		turn                     *int16
		stateVersion             int64
		startedAt, deadline      time.Time
	)
	if err := a.env.DB.Pool().QueryRow(context.Background(), `select phase, round, total_rounds, capacity, turn_seat,
		state_version, phase_started_at, phase_deadline_at from games where id = $1`, gameID).
		Scan(&phase, &round, &totalRounds, &capa, &turn, &stateVersion, &startedAt, &deadline); err != nil {
		t.Fatal(err)
	}
	if phase != "scenario_intro" || round != 0 || totalRounds != 4 || capa != 2 || turn != nil || stateVersion != 1 ||
		deadline.Sub(startedAt) != engine.DefaultDurations.ScenarioIntro {
		t.Fatalf("oyun: %s r%d t%d c%d v%d %v", phase, round, totalRounds, capa, stateVersion, deadline.Sub(startedAt))
	}

	// Katılımcılar ve kartlar.
	rows, err := a.env.DB.Pool().Query(context.Background(), `select p.user_id::text, p.seat, p.status,
		array_agg(c.card_id order by c.card_id), array_agg(d.text order by c.card_id),
		count(*) filter (where d.balance < 0), count(distinct c.category)
		from game_participants p join player_cards c using (game_id, user_id)
		join card_definitions d on d.id = c.card_id
		where p.game_id = $1 group by p.user_id, p.seat, p.status order by p.seat`, gameID)
	if err != nil {
		t.Fatal(err)
	}
	type dbPlayer struct {
		user   string
		seat   int16
		status string
		cards  []int32
		texts  []string
		neg    int64
		cats   int64
	}
	var players []dbPlayer
	for rows.Next() {
		var p dbPlayer
		if err := rows.Scan(&p.user, &p.seat, &p.status, &p.cards, &p.texts, &p.neg, &p.cats); err != nil {
			t.Fatal(err)
		}
		players = append(players, p)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(players) != 6 {
		t.Fatalf("%d katılımcı", len(players))
	}
	var seated []string
	allCards := map[int32]bool{}
	for i, p := range players {
		if int(p.seat) != i || p.status != "alive" || p.cats != 5 || len(p.cards) != 5 || p.neg > 2 {
			t.Fatalf("katılımcı %+v", p)
		}
		for _, c := range p.cards {
			allCards[c] = true
		}
		seated = append(seated, p.user)
	}
	if len(allCards) != 30 {
		t.Fatalf("tekil kart %d", len(allCards))
	}
	if !slices.Equal(slices.Sorted(slices.Values(seated)), slices.Sorted(slices.Values(l.users))) {
		t.Fatal("katılımcılar odadaki oyuncular değil")
	}

	// Olaylar: game.started (oda), 6 × game.your_cards (özel), phase_changed.
	evs := a.events.all()[before:]
	if len(evs) != 8 {
		t.Fatalf("%d olay", len(evs))
	}
	started := evs[0]
	sp := started.Payload.(map[string]any)
	if started.Type != EventStarted || started.To != "" || started.Version != 1 || started.RoomID != l.roomID ||
		sp["game_id"] != gameID || sp["total_rounds"] != 4 || sp["capacity"] != 2 {
		t.Fatalf("game.started: %+v", started)
	}
	if !slices.Equal(sp["reveal_order"].([]string), seated) {
		t.Fatalf("reveal_order %v, koltuklar %v", sp["reveal_order"], seated)
	}
	parts := sp["participants"].([]map[string]any)
	for i, p := range parts {
		if p["user_id"] != seated[i] || p["seat"] != i || !strings.HasPrefix(p["nickname"].(string), "Oyuncu") {
			t.Fatalf("participants[%d] %+v", i, p)
		}
	}
	if sc := sp["scenario"].(Scenario); sc.Title == "" || len(sc.Bunker) != 3 {
		t.Fatalf("senaryo %+v", sc)
	}
	for i, e := range evs[1:7] {
		p := players[i]
		if e.Type != EventYourCards || e.To != p.user || e.Version != 1 {
			t.Fatalf("your_cards[%d]: %+v", i, e)
		}
		var texts []string
		for j, c := range e.Payload.(map[string]any)["cards"].([]map[string]any) {
			if c["category"] != string(engine.Categories[j]) || c["revealed"] != false {
				t.Fatalf("kart %+v", c)
			}
			texts = append(texts, c["text"].(string))
		}
		if !slices.Equal(slices.Sorted(slices.Values(texts)), slices.Sorted(slices.Values(p.texts))) {
			t.Fatalf("%s kartları DB ile uyuşmuyor", p.user)
		}
	}
	pc := evs[7]
	pp := pc.Payload.(map[string]any)
	if pc.Type != engine.EventPhaseChanged || pc.To != "" || pp["phase"] != "SCENARIO_INTRO" || pp["round"] != 0 ||
		pp["turn_user_id"] != nil || !pp["phase_deadline_at"].(time.Time).Equal(deadline) || pp["server_time"] == nil {
		t.Fatalf("phase_changed: %+v", pc)
	}

	// Gizlilik: odaya giden olaylarda hiçbir kart metni veya denge etiketi yok.
	for _, e := range evs {
		if e.To != "" {
			continue
		}
		raw, _ := json.Marshal(e.Payload)
		for _, p := range players {
			for _, text := range p.texts {
				if strings.Contains(string(raw), text) {
					t.Fatalf("%s olayında açılmamış kart metni: %q", e.Type, text)
				}
			}
		}
		if strings.Contains(string(raw), "balance") || strings.Contains(string(raw), "card_id") {
			t.Fatalf("%s olayında gizli alan: %s", e.Type, raw)
		}
	}

	// Oda görünümü in_game; ikinci başlatma INVALID_PHASE; yabancı katılamaz.
	g := a.do("GET", "/v1/rooms/"+l.roomID, l.tokens[1], "")
	if g.body["status"] != "in_game" || g.body["current_game_id"] != gameID {
		t.Fatalf("görünüm: %s", g.raw)
	}
	if r := a.start(l, l.tokens[0]); r.code != 409 || errCode(r) != "INVALID_PHASE" {
		t.Fatalf("ikinci başlatma: %d %s", r.code, r.raw)
	}
}

func TestIntegrationStartConditions(t *testing.T) {
	a := newAPI(t, true)
	few := a.lobby(3, 2, true)
	if r := a.start(few, few.tokens[0]); r.code != 409 || errCode(r) != "START_CONDITIONS_NOT_MET" ||
		r.body["error"].(map[string]any)["details"].(map[string]any)["reason"] != "too_few_players" {
		t.Fatalf("az oyuncu: %d %s", r.code, r.raw)
	}
	l := a.lobby(5, 1, false)
	for i, tok := range l.tokens[:4] { // biri hazır değil
		_ = i
		a.do("POST", "/v1/rooms/"+l.roomID+"/ready", tok, `{"ready":true}`)
	}
	if r := a.start(l, l.tokens[0]); r.code != 409 ||
		r.body["error"].(map[string]any)["details"].(map[string]any)["reason"] != "not_all_ready" {
		t.Fatalf("hazır değil: %d %s", r.code, r.raw)
	}
	a.do("POST", "/v1/rooms/"+l.roomID+"/ready", l.tokens[4], `{"ready":true}`)
	if r := a.start(l, l.tokens[1]); r.code != 403 || errCode(r) != "FORBIDDEN" {
		t.Fatalf("sahip olmayan: %d %s", r.code, r.raw)
	}
	outsider, _ := a.guest("Yabanci")
	if r := a.start(l, outsider); r.code != 404 {
		t.Fatalf("üye olmayan: %d %s", r.code, r.raw)
	}
	if r := a.start(l, ""); r.code != 401 {
		t.Fatalf("oturumsuz: %d", r.code)
	}
	if n := a.scalar(`select count(*) from games`).(int64); n != 0 {
		t.Fatalf("reddedilen başlatmalar %d oyun yazdı", n)
	}
	// Kapasite 1: tur = oyuncu − 1.
	r := a.start(l, l.tokens[0])
	if r.code != 201 || a.scalar(`select total_rounds from games where id = $1`, r.body["game_id"]).(int16) != 4 {
		t.Fatalf("kapasite 1: %d %s", r.code, r.raw)
	}
}

func TestIntegrationStartConcurrent(t *testing.T) {
	a := newAPI(t, true)
	l := a.lobby(10, 2, true)
	var wg sync.WaitGroup
	codes := make([]int, 6)
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i] = a.start(l, l.tokens[0]).code
		}()
	}
	wg.Wait()
	slices.Sort(codes)
	if !slices.Equal(codes, []int{201, 409, 409, 409, 409, 409}) {
		t.Fatalf("kodlar %v", codes)
	}
	if n := a.scalar(`select count(*) from games where room_id = $1`, l.roomID).(int64); n != 1 {
		t.Fatalf("%d oyun", n)
	}
	started := 0
	for _, e := range a.events.all() {
		if e.Type == EventStarted {
			started++
		}
	}
	if started != 1 {
		t.Fatalf("%d game.started", started)
	}
	if n := a.scalar(`select count(*) from player_cards`).(int64); n != 50 {
		t.Fatalf("%d kart", n)
	}
}

func TestIntegrationStartEmptyCatalogRollsBack(t *testing.T) {
	a := newAPI(t, false)
	l := a.lobby(4, 2, true)
	v := a.scalar(`select version from rooms where id = $1`, l.roomID).(int64)
	before := len(a.events.all())
	if r := a.start(l, l.tokens[0]); r.code != 500 || errCode(r) != "INTERNAL" {
		t.Fatalf("boş katalog: %d %s", r.code, r.raw)
	}
	if a.scalar(`select status from rooms where id = $1`, l.roomID) != "lobby" ||
		a.scalar(`select version from rooms where id = $1`, l.roomID).(int64) != v ||
		a.scalar(`select count(*) from games`).(int64) != 0 || len(a.events.all()) != before {
		t.Fatal("başarısız başlatma geri alınmadı")
	}
	// Katalog eklenince aynı lobi başlayabilir.
	seedCatalog(t, a.env)
	if r := a.start(l, l.tokens[0]); r.code != 201 {
		t.Fatalf("katalog sonrası: %d %s", r.code, r.raw)
	}
}
