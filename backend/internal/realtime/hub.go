// Package realtime, WebSocket katmanıdır (ADR-0003, ADR-0007): istemciler
// REST ile komut gönderir, sunucu olayları WebSocket ile iter. Bağlantılar
// bellekte oda → bağlantılar eşlemesinde tutulur (tek instance). Her
// bağlantının sınırlı gönderim kuyruğu ve kendi yazıcı goroutine'i vardır;
// kuyruğu dolan (yavaş) istemci 4009 ile kesilir, yayın hiç bloklanmaz.
package realtime

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
	"github.com/burakkuru5534/bunker/backend/internal/room"
)

// Sınırlar (api.md §5).
const (
	// MaxConnsPerUser: kullanıcı başına eşzamanlı bağlantı; fazlasında en
	// eskisi 4008 ile kapanır.
	MaxConnsPerUser = 2
	// DefaultQueueSize: bağlantı başına bekleyen mesaj sınırı.
	DefaultQueueSize = 64
)

// Envelope, sunucu → istemci mesaj zarfıdır (api.md §5 "Zarf").
type Envelope struct {
	V            int       `json:"v"`
	Type         string    `json:"type"`
	RoomID       string    `json:"room_id"`
	StateVersion int64     `json:"state_version,omitempty"`
	TS           time.Time `json:"ts"`
	Payload      any       `json:"payload"`
}

// EnvelopeVersion, zarf biçiminin sürümüdür.
const EnvelopeVersion = 1

// Hub, aktif bağlantılardır. room.Publisher (olay yayını) ve room.Presence
// (bağlı mı) arayüzlerini uygular.
type Hub struct {
	logger    *slog.Logger
	queueSize int
	now       func() time.Time

	mu     sync.Mutex
	rooms  map[string]map[*Client]struct{}
	users  map[string][]*Client // kullanıcının bağlantıları, en eskisi başta
	closed bool
}

// NewHub, boş bir hub kurar.
func NewHub(logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{logger: logger, queueSize: DefaultQueueSize, now: time.Now,
		rooms: map[string]map[*Client]struct{}{}, users: map[string][]*Client{}}
}

var (
	_ room.Publisher = (*Hub)(nil)
	_ room.Presence  = (*Hub)(nil)
)

// register, bağlantıyı ekler; kullanıcının bağlantı sayısı sınırı aşarsa en
// eskisini 4008 ile kapatır. Hub kapalıysa false.
func (h *Hub) register(c *Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return false
	}
	// Sınır doluysa en eski bağlantı(lar) kapatılır; yenisi her zaman kabul.
	for conns := h.users[c.userID]; len(conns) >= MaxConnsPerUser; conns = h.users[c.userID] {
		conns[0].close(httpapi.CloseTooMany, "çok fazla bağlantı")
		h.removeLocked(conns[0])
	}
	h.users[c.userID] = append(h.users[c.userID], c)
	if h.rooms[c.roomID] == nil {
		h.rooms[c.roomID] = map[*Client]struct{}{}
	}
	h.rooms[c.roomID][c] = struct{}{}
	return true
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.removeLocked(c)
}

func (h *Hub) removeLocked(c *Client) {
	if set := h.rooms[c.roomID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.rooms, c.roomID)
		}
	}
	conns := h.users[c.userID]
	for i, x := range conns {
		if x == c {
			conns = append(conns[:i:i], conns[i+1:]...)
			break
		}
	}
	if len(conns) == 0 {
		delete(h.users, c.userID)
	} else {
		h.users[c.userID] = conns
	}
}

// Connected, kullanıcının en az bir açık bağlantısı var mı (room.Presence).
// Kullanıcı aynı anda tek aktif odada olabildiği için oda ayrımı gerekmez.
func (h *Hub) Connected(userID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.users[userID]) > 0
}

// Count, açık bağlantı sayısıdır.
func (h *Hub) Count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, conns := range h.users {
		n += len(conns)
	}
	return n
}

// Publish, olayı odadaki bağlantılara (To doluysa yalnızca o kullanıcınınkine)
// iletir (room.Publisher). Commit sonrası çağrılır; hiç bloklanmaz.
//
// Üyeliği biten bağlantılar olaydan sonra kapatılır: atılan 4003 (ana
// ekran), kendi isteğiyle ayrılan 1000.
func (h *Hub) Publish(e room.Event) {
	msg, err := json.Marshal(Envelope{V: EnvelopeVersion, Type: e.Type, RoomID: e.RoomID,
		StateVersion: e.Version, TS: h.now().UTC(), Payload: e.Payload})
	if err != nil {
		h.logger.Error("olay kodlanamadı", "type", e.Type, "room_id", e.RoomID, "error", err)
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.rooms[e.RoomID] {
		if e.To == "" || c.userID == e.To {
			c.enqueue(msg)
		}
	}
	var gone string
	code, reason := httpapi.CloseNormal, "odadan ayrıldı"
	switch e.Type {
	case room.EventMemberKicked:
		gone, code, reason = payloadUser(e), httpapi.CloseNotMember, "odadan atıldı"
	case room.EventMemberLeft:
		gone = payloadUser(e)
	}
	if gone != "" {
		for c := range h.rooms[e.RoomID] {
			if c.userID == gone {
				c.closeAfterQueued(code, reason) // önce olay, sonra kapanış
			}
		}
	}
}

func payloadUser(e room.Event) string {
	p, _ := e.Payload.(map[string]any)
	u, _ := p["user_id"].(string)
	return u
}

// Shutdown, yeni bağlantıları reddeder ve tümünü 1012 (yeniden başlatma) ile
// kapatır; istemciler yeniden bağlanır (ADR-0006).
func (h *Hub) Shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	for _, conns := range h.users {
		for _, c := range conns {
			c.close(httpapi.CloseServiceRestart, "sunucu yeniden başlıyor")
		}
	}
}
