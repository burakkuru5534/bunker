package realtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/game"
	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
	"github.com/burakkuru5534/bunker/backend/internal/room"
	"github.com/burakkuru5534/bunker/backend/internal/testdb"
)

type env struct {
	t    *testing.T
	db   *testdb.Env
	hub  *Hub
	disc *game.Disconnects
	game *game.Service
	room *room.Service
	srv  *httptest.Server
	ip   atomic.Int64
}

func newEnv(t *testing.T, mutate func(*Options)) *env {
	t.Helper()
	db := testdb.New(t)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	hub := NewHub(logger)
	authSvc := auth.NewService(db.DB, nil, logger)
	authHTTP := auth.NewHTTP(authSvc, 1)
	gameSvc := game.NewService(db.DB, hub, logger).WithPresence(hub)
	roomSvc := room.NewService(db.DB, hub, hub, logger).WithGames(game.RoomHooks{Starter: game.NewStarter(logger), Service: gameSvc})
	opts := DefaultOptions(httpapi.Version{Minor: 1})
	if mutate != nil {
		mutate(&opts)
	}
	ws := NewHandler(hub, authSvc, RoomSnapshotter{Rooms: roomSvc, Games: gameSvc}, opts, logger)
	disc := game.NewDisconnects(gameSvc, roomSvc, hub, logger)
	disc.SetGrace(300 * time.Millisecond)
	hub.SetListener(disc)
	t.Cleanup(func() { _ = disc.Stop(context.Background()) })
	h := httpapi.NewRouter(httpapi.Options{
		Logger: logger, MinAppVersion: httpapi.Version{Minor: 1}, RequestTimeout: 5 * time.Second,
		Register: func(mux *http.ServeMux) {
			authHTTP.Register(mux)
			room.NewHTTP(roomSvc, authHTTP.Require).Register(mux)
			game.NewHTTP(gameSvc, authHTTP.Require).Register(mux)
			ws.Register(mux)
		},
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	t.Cleanup(hub.Shutdown)
	return &env{t: t, db: db, hub: hub, srv: srv, disc: disc, game: gameSvc, room: roomSvc}
}

func (e *env) rest(method, path, token, body string) map[string]any {
	e.t.Helper()
	req, _ := http.NewRequest(method, e.srv.URL+path, strings.NewReader(body))
	req.Header.Set(httpapi.AppVersionHeader, "0.1.0")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", fmt.Sprintf("203.0.113.%d", e.ip.Add(1)%250+1))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	if res.StatusCode >= 300 {
		e.t.Fatalf("%s %s: %d %v", method, path, res.StatusCode, out)
	}
	return out
}

func (e *env) guest(nick string) (token, userID string) {
	r := e.rest("POST", "/v1/sessions/guest", "", `{"nickname":"`+nick+`"}`)
	return r["token"].(string), r["user"].(map[string]any)["id"].(string)
}

type dialOpts struct {
	token, version, roomID string
	noRoom                 bool
}

func (e *env) dial(o dialOpts) *websocket.Conn {
	e.t.Helper()
	h := http.Header{}
	if o.version != "-" {
		v := o.version
		if v == "" {
			v = "0.1.0"
		}
		h.Set(httpapi.AppVersionHeader, v)
	}
	if o.token != "" {
		h.Set("Authorization", "Bearer "+o.token)
	}
	url := "ws" + strings.TrimPrefix(e.srv.URL, "http") + "/v1/ws"
	if !o.noRoom {
		url += "?room_id=" + o.roomID
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPHeader: h})
	if err != nil {
		e.t.Fatalf("dial: %v", err)
	}
	e.t.Cleanup(func() { _ = c.CloseNow() })
	return c
}

// read, sıradaki mesajı okur (5 sn: paralel CI yükünde veritabanı yavaşlayabilir).
func read(t *testing.T, c *websocket.Conn) (Envelope, map[string]any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, data, err := c.Read(ctx)
	if err != nil {
		t.Fatalf("okuma: %v", err)
	}
	var env Envelope
	var raw struct {
		Payload map[string]any `json:"payload"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(data, &raw)
	return env, raw.Payload
}

// readUntil, verilen türdeki mesaja kadar okur.
func readUntil(t *testing.T, c *websocket.Conn, typ string) (Envelope, map[string]any) {
	t.Helper()
	for range 50 {
		env, p := read(t, c)
		if env.Type == typ {
			return env, p
		}
	}
	t.Fatalf("%s gelmedi", typ)
	return Envelope{}, nil
}

// closeCode, bağlantının kapanış kodunu bekler.
func closeCode(t *testing.T, c *websocket.Conn) websocket.StatusCode {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for {
		_, _, err := c.Read(ctx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("kapanış gelmedi")
			}
			return websocket.CloseStatus(err)
		}
	}
}

// roomWith, sahibi + n-1 üyeli oda kurar.
func (e *env) roomWith(n int) (roomID, code string, tokens, users []string) {
	for i := range n {
		tok, uid := e.guest(fmt.Sprintf("Oyuncu%d", i))
		tokens, users = append(tokens, tok), append(users, uid)
	}
	r := e.rest("POST", "/v1/rooms", tokens[0], "")
	roomID, code = r["id"].(string), r["invite_code"].(string)
	for _, tok := range tokens[1:] {
		e.rest("POST", "/v1/rooms/join", tok, `{"invite_code":"`+code+`"}`)
	}
	return
}

func TestWSRejections(t *testing.T) {
	e := newEnv(t, nil)
	roomID, _, tokens, _ := e.roomWith(1)
	outsider, _ := e.guest("Yabanci")
	cases := []struct {
		name string
		o    dialOpts
		want websocket.StatusCode
	}{
		{"sürüm yok", dialOpts{token: tokens[0], version: "-", roomID: roomID}, 4026},
		{"eski sürüm", dialOpts{token: tokens[0], version: "0.0.9", roomID: roomID}, 4026},
		{"token yok", dialOpts{roomID: roomID}, 4001},
		{"bozuk token", dialOpts{token: "x", roomID: roomID}, 4001},
		{"üye değil", dialOpts{token: outsider, roomID: roomID}, 4003},
		{"bozuk oda", dialOpts{token: tokens[0], roomID: "bozuk"}, 4003},
		{"oda yok", dialOpts{token: tokens[0], noRoom: true}, 4003},
	}
	for _, c := range cases {
		if got := closeCode(t, e.dial(c.o)); got != c.want {
			t.Errorf("%s: %d, beklenen %d", c.name, got, c.want)
		}
	}
	if e.hub.Count() != 0 {
		t.Fatalf("reddedilenler hub'da: %d", e.hub.Count())
	}
	// WebSocket olmayan istek: 426 (Accept).
	res, err := http.Get(e.srv.URL + "/v1/ws?room_id=" + roomID)
	if err != nil || res.StatusCode != http.StatusUpgradeRequired {
		t.Fatalf("düz HTTP: %v %v", res.StatusCode, err)
	}
}

func TestWSSnapshotEventsAndPresence(t *testing.T) {
	e := newEnv(t, nil)
	roomID, code, tokens, users := e.roomWith(1)
	c := e.dial(dialOpts{token: tokens[0], roomID: roomID})
	env, p := read(t, c)
	if env.Type != TypeSnapshot || env.V != 1 || env.RoomID != roomID {
		t.Fatalf("ilk mesaj %+v", env)
	}
	me := p["me"].(map[string]any)
	members := p["room"].(map[string]any)["members"].([]any)
	if me["user_id"] != users[0] || me["role"] != "player" || p["game"] != nil || len(members) != 1 ||
		members[0].(map[string]any)["connected"] != true {
		t.Fatalf("anlık görüntü %+v", p)
	}
	// Başka biri REST ile katılır → olay zarfla gelir.
	tok2, uid2 := e.guest("Ayse")
	e.rest("POST", "/v1/rooms/join", tok2, `{"invite_code":"`+code+`"}`)
	env, p = read(t, c)
	if env.Type != room.EventMemberJoined || env.StateVersion != 1 || p["member"].(map[string]any)["user_id"] != uid2 {
		t.Fatalf("member_joined %+v %v", env, p)
	}
	// Bağlı olmayanın connected=false; bağlanınca true, kapanınca false.
	view := e.rest("GET", "/v1/rooms/"+roomID, tokens[0], "")
	if connected(view, uid2) {
		t.Fatal("bağlı değilken connected")
	}
	c2 := e.dial(dialOpts{token: tok2, roomID: roomID})
	read(t, c2)
	if !connected(e.rest("GET", "/v1/rooms/"+roomID, tokens[0], ""), uid2) {
		t.Fatal("bağlıyken connected false")
	}
	_ = c2.Close(websocket.StatusNormalClosure, "")
	waitFor(t, func() bool { return !connected(e.rest("GET", "/v1/rooms/"+roomID, tokens[0], ""), uid2) })

	// sync → yeni anlık görüntü; geçersiz mesajlar yok sayılır.
	for _, m := range []string{`{"v":1,"type":"vote"}`, `bozuk`, `{"v":2,"type":"sync"}`, `{"v":1,"type":"sync"}`} {
		if err := c.Write(context.Background(), websocket.MessageText, []byte(m)); err != nil {
			t.Fatal(err)
		}
	}
	if env, _ := readUntil(t, c, TypeSnapshot); env.Type != TypeSnapshot {
		t.Fatal("sync yanıtı yok")
	}
}

func connected(view map[string]any, uid string) bool {
	for _, m := range view["members"].([]any) {
		mm := m.(map[string]any)
		if mm["user_id"] == uid {
			return mm["connected"] == true
		}
	}
	return false
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("koşul sağlanmadı")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Özel olaylar yalnızca sahibine: game.your_cards herkese kendi kartları.
func TestWSPrivateEvents(t *testing.T) {
	e := newEnv(t, nil)
	seedCatalog(t, e.db)
	roomID, _, tokens, users := e.roomWith(4)
	var conns []*websocket.Conn
	for _, tok := range tokens {
		c := e.dial(dialOpts{token: tok, roomID: roomID})
		read(t, c)
		conns = append(conns, c)
	}
	for _, tok := range tokens {
		e.rest("POST", "/v1/rooms/"+roomID+"/ready", tok, `{"ready":true}`)
	}
	e.rest("POST", "/v1/rooms/"+roomID+"/start", tokens[0], "")
	for i, c := range conns {
		env, _ := readUntil(t, c, game.EventStarted)
		if env.StateVersion != 1 {
			t.Fatalf("game.started sürümü %d", env.StateVersion)
		}
		env, p := read(t, c)
		if env.Type != game.EventYourCards || len(p["cards"].([]any)) != 5 {
			t.Fatalf("%d: %+v", i, env)
		}
		// Kartlar bu kullanıcınınki: DB ile karşılaştır.
		var want []string
		rows, _ := e.db.DB.Pool().Query(context.Background(), `select d.text from player_cards pc
			join card_definitions d on d.id = pc.card_id where pc.user_id = $1 order by d.text`, users[i])
		for rows.Next() {
			var s string
			_ = rows.Scan(&s)
			want = append(want, s)
		}
		rows.Close()
		var got []string
		for _, card := range p["cards"].([]any) {
			got = append(got, card.(map[string]any)["text"].(string))
		}
		if fmt.Sprint(sorted(got)) != fmt.Sprint(want) {
			t.Fatalf("%d başkasının kartlarını aldı", i)
		}
		// Sıradaki mesaj phase_changed: başka your_cards gelmez.
		if env, _ := read(t, c); env.Type != "game.phase_changed" {
			t.Fatalf("%d: beklenmeyen %s", i, env.Type)
		}
	}
}

func sorted(s []string) []string {
	out := append([]string(nil), s...)
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func TestWSKickLeaveAndLimit(t *testing.T) {
	e := newEnv(t, nil)
	roomID, _, tokens, users := e.roomWith(3)
	host := e.dial(dialOpts{token: tokens[0], roomID: roomID})
	read(t, host)
	kicked := e.dial(dialOpts{token: tokens[1], roomID: roomID})
	read(t, kicked)
	e.rest("POST", "/v1/rooms/"+roomID+"/kick", tokens[0], `{"user_id":"`+users[1]+`"}`)
	if env, _ := read(t, kicked); env.Type != room.EventMemberKicked {
		t.Fatalf("atılana olay: %s", env.Type)
	}
	if got := closeCode(t, kicked); got != 4003 {
		t.Fatalf("atılan kapanışı %d", got)
	}
	leaver := e.dial(dialOpts{token: tokens[2], roomID: roomID})
	read(t, leaver)
	e.rest("POST", "/v1/rooms/"+roomID+"/leave", tokens[2], "")
	if got := closeCode(t, leaver); got != websocket.StatusNormalClosure {
		t.Fatalf("ayrılan kapanışı %d", got)
	}
	// Oda sahibi iki olayı da aldı ve bağlı kaldı.
	readUntil(t, host, room.EventMemberKicked)
	readUntil(t, host, room.EventMemberLeft)

	// Kullanıcı başına 2 bağlantı: üçüncüde en eskisi 4008.
	second := e.dial(dialOpts{token: tokens[0], roomID: roomID})
	read(t, second)
	third := e.dial(dialOpts{token: tokens[0], roomID: roomID})
	read(t, third)
	if got := closeCode(t, host); got != 4008 {
		t.Fatalf("en eski bağlantı %d", got)
	}
	waitFor(t, func() bool { return e.hub.Count() == 2 })
}

func TestWSPingTimeoutAndShutdown(t *testing.T) {
	e := newEnv(t, func(o *Options) { o.PingEvery, o.PingTimeout = 50*time.Millisecond, 100*time.Millisecond })
	roomID, _, tokens, _ := e.roomWith(1)
	// Okumayan istemci pong göndermez → bağlantı kapanır.
	silent := e.dial(dialOpts{token: tokens[0], roomID: roomID})
	waitFor(t, func() bool { return e.hub.Count() == 0 })
	_ = silent
	// Okuyan istemci ping'e yanıt verir ve açık kalır.
	alive := e.dial(dialOpts{token: tokens[0], roomID: roomID})
	read(t, alive)
	readErr := make(chan error, 1)
	go func() { _, _, err := alive.Read(context.Background()); readErr <- err }()
	time.Sleep(400 * time.Millisecond)
	if e.hub.Count() != 1 {
		select { // hata ayrıntısı için okumayı bekle, ama sonsuza kadar değil
		case err := <-readErr:
			t.Fatalf("yanıt veren istemci kapandı: %v", err)
		case <-time.After(time.Second):
			t.Fatalf("bağlantı sayısı %d (1 beklenirdi)", e.hub.Count())
		}
	}
	alive.CloseNow()
	waitFor(t, func() bool { return e.hub.Count() == 0 })

	c := e.dial(dialOpts{token: tokens[0], roomID: roomID})
	read(t, c)
	e.hub.Shutdown()
	if got := closeCode(t, c); got != 1012 {
		t.Fatalf("kapanış %d", got)
	}
	if got := closeCode(t, e.dial(dialOpts{token: tokens[0], roomID: roomID})); got != 1012 {
		t.Fatalf("kapanıştan sonra bağlantı %d", got)
	}
}

// seedCatalog: 1 senaryo; kategori başına 12 kart.
func seedCatalog(t *testing.T, db *testdb.Env) {
	t.Helper()
	ctx := context.Background()
	if _, err := db.Migrate.Exec(ctx, `insert into scenarios (id, title, disaster, duration_label, shelter_features, outside_world)
		values ('meteor', 'Göktaşı', 'Uzun bir felaket metni burada yer alır, en az yirmi karakter.', '2 yıl',
		array['a1','b2','c3'], 'Dış dünya hakkında yeterince uzun bir açıklama metni.')`); err != nil {
		t.Fatal(err)
	}
	for ci, c := range []string{"profession", "health", "hobby", "item", "secret"} {
		for i := range 12 {
			if _, err := db.Migrate.Exec(ctx, `insert into card_definitions (id, category, text, balance) values ($1, $2, $3, $4)`,
				(ci+1)*1000+i, c, fmt.Sprintf("%s kartı (%02d)", c, i), []int{-1, 0, 1}[i%3]); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// Oyun sürerken bağlanan (veya yeniden bağlanan) oyuncu, anlık görüntüde
// kendine göre süzülmüş oyunu alır (ENG-013).
func TestWSSnapshotIncludesGame(t *testing.T) {
	e := newEnv(t, nil)
	seedCatalog(t, e.db)
	roomID, _, tokens, users := e.roomWith(4)
	for _, tok := range tokens {
		e.rest("POST", "/v1/rooms/"+roomID+"/ready", tok, `{"ready":true}`)
	}
	gameID := e.rest("POST", "/v1/rooms/"+roomID+"/start", tokens[0], "")["game_id"].(string)
	c := e.dial(dialOpts{token: tokens[2], roomID: roomID})
	env, p := read(t, c)
	g, _ := p["game"].(map[string]any)
	if env.Type != TypeSnapshot || g == nil || g["id"] != gameID || g["phase"] != "SCENARIO_INTRO" || g["state_version"] != float64(1) {
		t.Fatalf("oyun görünümü yok: %v", p["game"])
	}
	mine := g["my_cards"].([]any)
	if len(mine) != 5 {
		t.Fatalf("my_cards %d", len(mine))
	}
	// Kendi kartları DB ile aynı; başka kimsenin kapalı kartı yok.
	raw, _ := json.Marshal(p)
	rows, _ := e.db.DB.Pool().Query(context.Background(), `select pc.user_id::text, d.text from player_cards pc
		join card_definitions d on d.id = pc.card_id where pc.game_id = $1`, gameID)
	defer rows.Close()
	for rows.Next() {
		var uid, text string
		_ = rows.Scan(&uid, &text)
		if in := strings.Contains(string(raw), text); in != (uid == users[2]) {
			t.Fatalf("%s kartı %q görünürlüğü yanlış (görünüyor=%v)", uid, text, in)
		}
	}
	// Bağlı olan connected=true.
	for _, part := range g["participants"].([]any) {
		pp := part.(map[string]any)
		if pp["connected"] != (pp["user_id"] == users[2]) {
			t.Fatalf("connected %+v", pp)
		}
	}
}

// chat.message odaya gider; göndereni engelleyen almaz; bağlanınca son
// mesajlar (engellenenler hariç) anlık görüntüdedir.
func TestWSChat(t *testing.T) {
	e := newEnv(t, nil)
	roomID, _, tokens, users := e.roomWith(3)
	if _, err := e.db.Migrate.Exec(context.Background(), `insert into blocks (blocker_id, blocked_id) values ($1, $2)`, users[2], users[1]); err != nil {
		t.Fatal(err)
	}
	var conns []*websocket.Conn
	for _, tok := range tokens {
		c := e.dial(dialOpts{token: tok, roomID: roomID})
		read(t, c)
		conns = append(conns, c)
	}
	e.rest("POST", "/v1/rooms/"+roomID+"/messages", tokens[1], `{"client_msg_id":"00000000-0000-4000-8000-000000000001","text":"selam"}`)
	e.rest("POST", "/v1/rooms/"+roomID+"/messages", tokens[0], `{"client_msg_id":"00000000-0000-4000-8000-000000000002","text":"herkese"}`)
	for i, c := range conns[:2] {
		if _, p := readUntil(t, c, "chat.message"); p["text"] != "selam" || p["user_id"] != users[1] || p["nickname"] != "Oyuncu1" {
			t.Fatalf("%d: %v", i, p)
		}
	}
	// Engelleyen (u2) ilk mesajı almaz: ilk chat.message "herkese".
	if _, p := readUntil(t, conns[2], "chat.message"); p["text"] != "herkese" {
		t.Fatalf("engelleyen engellenenin mesajını aldı: %v", p)
	}
	// Yeni bağlanan geçmişi görür; engelleyenin geçmişinde engellenen yok.
	for i, want := range map[int]int{0: 2, 2: 1} {
		c := e.dial(dialOpts{token: tokens[i], roomID: roomID})
		_, p := read(t, c)
		if chat := p["chat"].([]any); len(chat) != want {
			t.Fatalf("%d geçmişi %d mesaj, beklenen %d", i, len(chat), want)
		}
	}
}
