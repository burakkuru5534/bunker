package realtime

import (
	"context"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// outMsg, kuyruktaki öğedir: veri ya da (data nil ise) sıralı kapanış isteği.
type outMsg struct {
	data   []byte
	code   int
	reason string
}

// Client, tek bir WebSocket bağlantısıdır. Yazma yalnızca writer
// goroutine'inden yapılır; diğerleri kuyruğa ekler veya kapatma ister.
//
// İki bağlam vardır: readCtx bağlantı boyunca yaşar (okuma iptal edilirse
// kütüphane bağlantıyı kapanış kodu göndermeden koparır); ctx kapatma
// isteğiyle iptal edilir ve yazıcıyı durdurur — yazıcı kapanış çerçevesini
// gönderir, okuma da bunun üzerine biter.
type Client struct {
	userID, roomID string
	conn           *websocket.Conn
	send           chan outMsg

	readCtx context.Context
	ctx     context.Context
	cancel  context.CancelFunc

	mu          sync.Mutex
	closeCode   websocket.StatusCode
	closeReason string
}

func newClient(parent context.Context, conn *websocket.Conn, userID, roomID string, queue int) *Client {
	ctx, cancel := context.WithCancel(parent)
	// Kapanış işareti her zaman sığsın diye kuyrukta bir yer ayrılır.
	return &Client{userID: userID, roomID: roomID, conn: conn, send: make(chan outMsg, queue+1),
		readCtx: parent, ctx: ctx, cancel: cancel, closeCode: websocket.StatusNormalClosure}
}

// enqueue, mesajı kuyruğa ekler; kuyruk doluysa istemci yavaştır ve 4009 ile
// hemen kesilir (yayın bloklanmaz). Hub kilidi altında çağrılabilir.
func (c *Client) enqueue(msg []byte) {
	if c.ctx.Err() != nil {
		return
	}
	if len(c.send) >= cap(c.send)-1 { // son yer kapanış işareti için
		c.close(4009, "kuyruk doldu")
		return
	}
	select {
	case c.send <- outMsg{data: msg}:
	default:
		c.close(4009, "kuyruk doldu")
	}
}

// closeAfterQueued, kuyruktaki mesajlar gönderildikten sonra kapatır (ör.
// atılan önce member_kicked olayını alır, sonra 4003). Bloklanmaz.
func (c *Client) closeAfterQueued(code int, reason string) {
	select {
	case c.send <- outMsg{code: code, reason: reason}:
	default:
		c.close(code, reason)
	}
}

// close, bağlantının hemen kapanmasını ister (ilk kod geçerli); asıl
// kapatmayı writer yapar. Bloklanmaz.
func (c *Client) close(code int, reason string) {
	c.mu.Lock()
	if c.ctx.Err() == nil {
		c.closeCode, c.closeReason = websocket.StatusCode(code), reason
	}
	c.mu.Unlock()
	c.cancel()
}

func (c *Client) closeStatus() (websocket.StatusCode, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closeCode, c.closeReason
}

// writeLoop, kuyruğu yazar ve düzenli ping gönderir; ping yanıtı pingTimeout
// içinde gelmezse bağlantı kapanır. Kapatma istenince kapanış çerçevesini
// gönderir.
func (c *Client) writeLoop(pingEvery, pingTimeout, writeTimeout time.Duration) {
	ticker := time.NewTicker(pingEvery)
	defer ticker.Stop()
	pingFailed := make(chan struct{}, 1)
	for {
		select {
		case <-c.ctx.Done():
			code, reason := c.closeStatus()
			if code == websocket.StatusPolicyViolation {
				// Ping'e yanıt yok: karşı taraf ölü; kapanış el sıkışması
				// (5 sn bekleme) yapılmadan hemen kesilir.
				_ = c.conn.CloseNow()
				return
			}
			_ = c.conn.Close(code, reason)
			return
		case <-pingFailed:
			c.close(int(websocket.StatusPolicyViolation), "ping yanıtı yok")
		case m := <-c.send:
			if m.data == nil {
				c.close(m.code, m.reason)
				continue
			}
			ctx, cancel := context.WithTimeout(c.ctx, writeTimeout)
			err := c.conn.Write(ctx, websocket.MessageText, m.data)
			cancel()
			if err != nil {
				c.close(int(websocket.StatusGoingAway), "yazılamadı")
			}
		case <-ticker.C:
			// Ping, okuyucunun pong'u işlemesini bekler; yazmayı bloklamasın
			// diye ayrı goroutine'de.
			go func() {
				ctx, cancel := context.WithTimeout(c.ctx, pingTimeout)
				defer cancel()
				if err := c.conn.Ping(ctx); err != nil && c.ctx.Err() == nil {
					select {
					case pingFailed <- struct{}{}:
					default:
					}
				}
			}()
		}
	}
}
