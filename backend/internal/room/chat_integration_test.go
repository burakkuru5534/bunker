package room

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/ratelimit"
)

func uuidN(i int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", i) }

func (a *api) say(tok, roomID, id, text string) resp {
	return a.do("POST", "/v1/rooms/"+roomID+"/messages", tok, fmt.Sprintf(`{"client_msg_id":%q,"text":%q}`, id, text))
}

type maskFilter struct{}

func (maskFilter) Mask(s string) string { return strings.ReplaceAll(s, "kötü", "***") }

func TestIntegrationChatLobby(t *testing.T) {
	a := newAPI(t)
	a.svc.WithTextFilter(maskFilter{})
	hostTok, roomID, code := a.room("Burak")
	tok, uid := a.guest("Ayse")
	a.join(tok, code)
	before := len(a.events.all())

	r := a.say(tok, roomID, uuidN(1), "  merhaba\nkötü   dünya ")
	if r.code != 201 || r.body["id"] == nil || r.body["created_at"] == nil {
		t.Fatalf("gönder: %d %s", r.code, r.raw)
	}
	evs := a.events.all()[before:]
	if len(evs) != 1 || evs[0].Type != EventChatMessage || evs[0].Version != 0 {
		t.Fatalf("olay %+v", evs)
	}
	m := evs[0].Payload.(ChatMessage)
	if m.Text != "merhaba *** dünya" || m.UserID != uid || m.Nickname != "Ayse" || m.ID != int64(r.body["id"].(float64)) {
		t.Fatalf("mesaj %+v", m)
	}
	// Ağ tekrarı: aynı client_msg_id → aynı mesaj, yeni olay yok, sıklık sınırına takılmaz.
	r2 := a.say(tok, roomID, uuidN(1), "farklı metin")
	if r2.code != 201 || r2.body["id"] != r.body["id"] || len(a.events.all()) != before+1 {
		t.Fatalf("tekrar: %d %s", r2.code, r2.raw)
	}
	// Saniyede 1: hemen ikinci yeni mesaj 429 + Retry-After.
	r3 := a.say(tok, roomID, uuidN(2), "ikinci")
	if r3.code != 429 || errCode(r3) != "RATE_LIMITED" {
		t.Fatalf("sıklık: %d %s", r3.code, r3.raw)
	}
	// Doğrulama.
	for name, c := range map[string]struct {
		body string
		code int
		want string
	}{
		"id yok":     {`{"text":"x"}`, 400, "VALIDATION_FAILED"},
		"id bozuk":   {`{"client_msg_id":"x","text":"x"}`, 400, "VALIDATION_FAILED"},
		"metin yok":  {`{"client_msg_id":"` + uuidN(3) + `"}`, 400, "VALIDATION_FAILED"},
		"boş":        {`{"client_msg_id":"` + uuidN(3) + `","text":"   "}`, 400, "MESSAGE_INVALID"},
		"uzun":       {`{"client_msg_id":"` + uuidN(3) + `","text":"` + strings.Repeat("a", 201) + `"}`, 400, "MESSAGE_INVALID"},
		"fazla alan": {`{"client_msg_id":"` + uuidN(3) + `","text":"x","x":1}`, 400, "VALIDATION_FAILED"},
	} {
		if r := a.do("POST", "/v1/rooms/"+roomID+"/messages", hostTok, c.body); r.code != c.code || errCode(r) != c.want {
			t.Errorf("%s: %d %s", name, r.code, r.raw)
		}
	}
	outsider, _ := a.guest("Mehmet")
	if r := a.say(outsider, roomID, uuidN(4), "sızma"); r.code != 404 {
		t.Fatalf("üye değil: %d", r.code)
	}
	var n int64
	_ = a.env.DB.Pool().QueryRow(context.Background(), `select count(*) from chat_messages where room_id = $1`, roomID).Scan(&n)
	if n != 1 {
		t.Fatalf("%d mesaj kaydı", n)
	}
	// Oyun bitmiş odada (lobiye dönmeden) da yazılabilir.
	a.exec("update rooms set status = 'finished' where id = $1", roomID)
	if r := a.say(hostTok, roomID, uuidN(5), "iyi oyundu"); r.code != 201 {
		t.Fatalf("bitmiş oda: %d %s", r.code, r.raw)
	}
}

func TestIntegrationChatTenSecondLimit(t *testing.T) {
	a := newAPI(t)
	a.svc.chat = chatLimits{perSecond: ratelimit.New(100, time.Second, 10), perTen: ratelimit.New(5, 10*time.Second, 10)}
	tok, roomID, _ := a.room("Burak")
	for i := range 5 {
		if r := a.say(tok, roomID, uuidN(10+i), "m"); r.code != 201 {
			t.Fatalf("%d: %d", i, r.code)
		}
	}
	if r := a.say(tok, roomID, uuidN(20), "altıncı"); r.code != 429 {
		t.Fatalf("10 sn'de 6.: %d", r.code)
	}
}

func TestIntegrationChatBlocksAndHistory(t *testing.T) {
	a := newAPI(t)
	a.svc.chat = chatLimits{perSecond: ratelimit.New(100, time.Second, 10), perTen: ratelimit.New(100, time.Second, 10)}
	hostTok, roomID, code := a.room("Burak")
	tok, uid := a.guest("Ayse")
	a.join(tok, code)
	host := a.scalarRoom(`select host_user_id::text from rooms where id = $1`, roomID)
	// Oda sahibi Ayşe'yi engelledi.
	a.exec("insert into blocks (blocker_id, blocked_id) values ($1, $2)", host, uid)
	before := len(a.events.all())
	a.say(tok, roomID, uuidN(1), "beni görmeyecek")
	a.say(hostTok, roomID, uuidN(2), "herkese")
	evs := a.events.all()[before:]
	if len(evs) != 2 || fmt.Sprint(evs[0].Exclude) != "["+host+"]" || len(evs[1].Exclude) != 0 {
		t.Fatalf("hariç tutma: %+v / %+v", evs[0].Exclude, evs[1].Exclude)
	}
	// 55 mesaj doğrudan yazılır (HTTP'den göndermek genel REST sınırına —
	// kullanıcı başına 10 sn'de 30 istek — takılır).
	a.exec(`insert into chat_messages (room_id, user_id, client_msg_id, body, created_at)
		select $1, $2, gen_random_uuid(), 'mesaj ' || lpad(i::text, 2, '0'), now() - (54 - i) * interval '1 millisecond' - interval '1 second'
		from generate_series(0, 54) i`, roomID, host)
	hist, err := a.svc.RecentMessages(context.Background(), host, roomID)
	if err != nil {
		t.Fatal(err)
	}
	// Oda sahibinin gördüğü 56 mesaj (55 eski + "herkese"; Ayşe'ninki
	// engelli): son 50 → "mesaj 06" … "mesaj 54", "herkese"; eskiden yeniye.
	if len(hist) != ChatHistory || hist[len(hist)-1].Text != "herkese" || hist[0].Text != "mesaj 06" {
		t.Fatalf("geçmiş: %d, ilk %q, son %q", len(hist), hist[0].Text, hist[len(hist)-1].Text)
	}
	// Engelleyen için engellenenin mesajı geçmişte yok; Ayşe kendi mesajını görür.
	for _, m := range hist {
		if m.UserID == uid {
			t.Fatal("engellenen mesaj geçmişte")
		}
	}
	// Ayşe'nin geçmişinde kendi mesajı var (55+1 mesajdan son 50'si: onunki
	// en eskilerden; bu yüzden ayrıca son 50'ye girecek kadar yeni bir mesaj).
	a.say(tok, roomID, uuidN(500), "kendi mesajım")
	mine, _ := a.svc.RecentMessages(context.Background(), uid, roomID)
	if mine[len(mine)-1].Text != "kendi mesajım" || mine[len(mine)-1].UserID != uid {
		t.Fatalf("kendi mesajı görünmüyor: %+v", mine[len(mine)-1])
	}
	hostView, _ := a.svc.RecentMessages(context.Background(), host, roomID)
	if hostView[len(hostView)-1].UserID == uid {
		t.Fatal("engelleyen yeni mesajı görüyor")
	}
}

func (a *api) scalarRoom(sql string, args ...any) string {
	a.t.Helper()
	var s string
	if err := a.env.DB.Pool().QueryRow(context.Background(), sql, args...).Scan(&s); err != nil {
		a.t.Fatal(err)
	}
	return s
}
