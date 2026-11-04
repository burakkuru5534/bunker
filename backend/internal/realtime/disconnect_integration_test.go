package realtime

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/burakkuru5534/bunker/backend/internal/game"
)

// startedGame: n kişilik oyun başlatır; herkes bağlı (ilk mesajlar okunmuş).
func (e *env) startedGame(n int) (roomID, gameID string, tokens, users []string, conns []*websocket.Conn) {
	e.t.Helper()
	seedCatalog(e.t, e.db)
	roomID, _, tokens, users = e.roomWith(n)
	for _, tok := range tokens {
		e.rest("POST", "/v1/rooms/"+roomID+"/ready", tok, `{"ready":true}`)
	}
	for _, tok := range tokens {
		c := e.dial(dialOpts{token: tok, roomID: roomID})
		read(e.t, c)
		conns = append(conns, c)
	}
	gameID = e.rest("POST", "/v1/rooms/"+roomID+"/start", tokens[0], "")["game_id"].(string)
	return
}

func (e *env) val(sql string, args ...any) any {
	e.t.Helper()
	var v any
	if err := e.db.DB.Pool().QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		e.t.Fatalf("%s: %v", sql, err)
	}
	return v
}

// connectionEvent, bağlantı olayını bekler (15 sn: süreye dayalı akış ve
// paralel CI yükünde test veritabanı havuzu yavaşlayabilir).
func connectionEvent(t *testing.T, c *websocket.Conn, user, state string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		_, data, err := c.Read(ctx)
		cancel()
		if err != nil {
			t.Fatalf("%s %s beklenirken: %v", user, state, err)
		}
		var env Envelope
		var raw struct {
			Payload map[string]any `json:"payload"`
		}
		_ = json.Unmarshal(data, &env)
		_ = json.Unmarshal(data, &raw)
		p := raw.Payload
		if env.Type != game.EventPlayerConnection {
			continue
		}
		if p["user_id"] == user && p["state"] == state {
			if env.StateVersion != 0 {
				t.Fatalf("bağlantı olayı sürüm taşımamalı: %d", env.StateVersion)
			}
			return
		}
	}
	t.Fatalf("%s %s gelmedi", user, state)
}

func TestDisconnectAndReconnectWithinGrace(t *testing.T) {
	e := newEnv(t, nil)
	roomID, gameID, tokens, users, conns := e.startedGame(4)
	_ = conns[1].Close(websocket.StatusNormalClosure, "")
	connectionEvent(t, conns[0], users[1], "disconnected")
	if e.val(`select disconnected_at is not null from game_participants where game_id = $1 and user_id = $2`, gameID, users[1]) != true {
		t.Fatal("kopma kaydı yok")
	}
	// Süre dolmadan geri döner.
	back := e.dial(dialOpts{token: tokens[1], roomID: roomID})
	if env, p := read(t, back); env.Type != TypeSnapshot || p["game"] == nil {
		t.Fatal("geri dönüşte oyun görünümü yok")
	}
	connectionEvent(t, conns[0], users[1], "reconnected")
	time.Sleep(500 * time.Millisecond) // eski süre geçti
	if e.val(`select status from game_participants where game_id = $1 and user_id = $2`, gameID, users[1]) != "alive" ||
		e.val(`select disconnected_at is null from game_participants where game_id = $1 and user_id = $2`, gameID, users[1]) != true {
		t.Fatal("geri dönen oyuncu ayrılmış sayıldı")
	}
	if e.disc.Pending() != 0 {
		t.Fatal("bekleyen süre kaldı")
	}
}

func TestDisconnectTimeoutLeaves(t *testing.T) {
	e := newEnv(t, nil)
	roomID, gameID, tokens, users, conns := e.startedGame(5)
	_ = conns[2].CloseNow() // ağ koptu
	connectionEvent(t, conns[0], users[2], "disconnected")
	connectionEvent(t, conns[0], users[2], "left")
	if e.val(`select status from game_participants where game_id = $1 and user_id = $2`, gameID, users[2]) != "left" {
		t.Fatal("oyunda ayrılmış değil")
	}
	if n := e.val(`select count(*) from game_actions where game_id = $1 and actor_id = $2 and type = 'disconnect_timeout'`, gameID, users[2]).(int64); n != 1 {
		t.Fatalf("%d disconnect_timeout kaydı", n)
	}
	if e.val(`select left_at is not null from room_members where room_id = $1 and user_id = $2`, roomID, users[2]) != true {
		t.Fatal("odadan ayrılmış değil")
	}
	// Eski bağlantıyla dönmeye çalışırsa artık üye değil (4003); kodla izleyici olarak döner.
	if got := closeCode(t, e.dial(dialOpts{token: tokens[2], roomID: roomID})); got != 4003 {
		t.Fatalf("üye olmayan bağlantı %d", got)
	}
}

func TestDisconnectInLobbyIgnored(t *testing.T) {
	e := newEnv(t, nil)
	roomID, _, tokens, _ := e.roomWith(2)
	c := e.dial(dialOpts{token: tokens[1], roomID: roomID})
	read(t, c)
	_ = c.CloseNow()
	time.Sleep(500 * time.Millisecond)
	if e.disc.Pending() != 0 || e.val(`select count(*) from room_members where room_id = $1 and left_at is null`, roomID).(int64) != 2 {
		t.Fatal("lobide kopma işlem gördü")
	}
}

// Açılış kurtarması: sunucu yeni açıldı. Bağlanan oyuncular kalır; hiç
// dönmeyen süre sonunda ayrılır; önceden kopmuşun kalan süresi korunur.
func TestDisconnectRecoveryOnStart(t *testing.T) {
	e := newEnv(t, nil)
	_, gameID, _, users, conns := e.startedGame(6)
	// Yeni açılmış sunucuyu taklit et: u5 hiç dönmedi; u4 kapanıştan önce
	// 250 ms önce kopmuştu (50 ms kaldı).
	_ = conns[5].CloseNow()
	_ = conns[4].CloseNow()
	time.Sleep(100 * time.Millisecond)
	_ = e.disc.Stop(context.Background()) // eski süreler bellekte gitti
	if _, err := e.db.Migrate.Exec(context.Background(), `update game_participants set disconnected_at =
		case when user_id = $2 then now() - interval '250 milliseconds' else null end where game_id = $1`, gameID, users[4]); err != nil {
		t.Fatal(err)
	}
	fresh := game.NewDisconnects(e.game, e.room, e.hub, nil)
	fresh.SetGrace(300 * time.Millisecond)
	e.hub.SetListener(fresh)
	defer fresh.Stop(context.Background())
	start := time.Now()
	fresh.Start()
	// İki dönmeyen ayrılır; dört bağlı oyuncunun kaydı silinir.
	waitFor(t, func() bool {
		return e.val(`select count(*) from game_participants where game_id = $1 and status = 'left'`, gameID).(int64) == 2 &&
			e.val(`select count(*) from game_participants where game_id = $1 and status = 'alive' and disconnected_at is null`, gameID).(int64) == 4
	})
	if time.Since(start) > 2*time.Second {
		t.Fatal("kurtarma çok yavaş")
	}
	for i, u := range users[:4] {
		st := e.val(`select status from game_participants where game_id = $1 and user_id = $2`, gameID, u)
		cleared := e.val(`select disconnected_at is null from game_participants where game_id = $1 and user_id = $2`, gameID, u)
		if st != "alive" || cleared != true {
			t.Fatalf("bağlı oyuncu %d: %v %v", i, st, cleared)
		}
	}
}

// Bayat "bağlandı" bildirimi: ilk bağlantının bildirimi gecikip kopmadan
// SONRA işlenirse (yükte goroutine sırası garanti değil) kopma kaydını
// silmemeli; aksi halde oyuncu hiç "ayrıldı" sayılmaz.
func TestDisconnectStaleConnectedNotification(t *testing.T) {
	e := newEnv(t, nil)
	roomID, gameID, _, users, conns := e.startedGame(4)
	_ = conns[1].CloseNow()
	connectionEvent(t, conns[0], users[1], "disconnected")
	e.disc.UserConnected(users[1], roomID) // gecikmiş, bayat bildirim
	connectionEvent(t, conns[0], users[1], "left")
	if e.val(`select status from game_participants where game_id = $1 and user_id = $2`, gameID, users[1]) != "left" {
		t.Fatal("bayat bildirim kopma kaydını sildi")
	}
}
