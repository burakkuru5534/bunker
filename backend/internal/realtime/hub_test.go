package realtime

import (
	"context"
	"testing"

	"github.com/burakkuru5534/bunker/backend/internal/room"
)

// Yavaş istemci: kuyruk dolunca yayın bloklanmaz, istemci 4009 ile kapanır.
func TestSlowClientDropped(t *testing.T) {
	h := NewHub(nil)
	c := newClient(context.Background(), nil, "u", "r", 2)
	if !h.register(c) {
		t.Fatal("kayıt")
	}
	for range 3 {
		h.Publish(room.Event{RoomID: "r", Type: "x", Payload: map[string]any{}})
	}
	if c.ctx.Err() == nil {
		t.Fatal("kuyruk taştı ama istemci kapanmadı")
	}
	if code, _ := c.closeStatus(); code != 4009 {
		t.Fatalf("kod %d", code)
	}
	// Başka odaya ve başka kullanıcıya özel olay gitmez.
	other := newClient(context.Background(), nil, "v", "r", 4)
	h.register(other)
	h.Publish(room.Event{RoomID: "r", Type: "x", To: "u", Payload: map[string]any{}})
	h.Publish(room.Event{RoomID: "baska", Type: "x", Payload: map[string]any{}})
	if len(other.send) != 0 {
		t.Fatalf("%d yanlış mesaj", len(other.send))
	}
}
