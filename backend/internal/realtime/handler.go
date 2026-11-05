package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
	"github.com/burakkuru5534/bunker/backend/internal/observability"
	"github.com/burakkuru5534/bunker/backend/internal/ratelimit"
)

// Authenticator, bearer token'ı doğrular (auth.Service).
type Authenticator interface {
	Authenticate(ctx context.Context, raw string) (auth.Principal, error)
}

// Snapshotter, bağlanınca ve "sync" isteğinde gönderilen tam durumu kurar.
// Üye değilse NOT_FOUND döner (bağlantı 4003 ile kapanır).
type Snapshotter interface {
	Snapshot(ctx context.Context, p auth.Principal, roomID string) (any, error)
}

// Options, işleyici ayarlarıdır (api.md §5).
type Options struct {
	MinAppVersion httpapi.Version
	PingEvery     time.Duration // 25 sn
	PingTimeout   time.Duration // ping'e yanıt süresi; toplam ~60 sn sessizlikte kapanır
	WriteTimeout  time.Duration
	// Kullanıcı başına bağlantı denemesi ve "sync" sıklığı.
	ConnectLimit, SyncLimit         int
	ConnectPer, SyncPer             time.Duration
	SnapshotTimeout, HandshakeLimit time.Duration
}

// DefaultOptions, üretim ayarlarıdır.
func DefaultOptions(min httpapi.Version) Options {
	return Options{
		MinAppVersion: min, PingEvery: 25 * time.Second, PingTimeout: 35 * time.Second, WriteTimeout: 10 * time.Second,
		ConnectLimit: 10, ConnectPer: time.Minute, SyncLimit: 5, SyncPer: 10 * time.Second,
		SnapshotTimeout: 5 * time.Second, HandshakeLimit: 10 * time.Second,
	}
}

// maxClientMessage: istemci yalnızca {"v":1,"type":"sync"} gönderir.
const maxClientMessage = 1024

// Handler, GET /v1/ws işleyicisidir.
type Handler struct {
	hub      *Hub
	auth     Authenticator
	snap     Snapshotter
	opts     Options
	logger   *slog.Logger
	connects *ratelimit.Limiter
}

// NewHandler, işleyiciyi kurar.
func NewHandler(hub *Hub, a Authenticator, snap Snapshotter, opts Options, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{hub: hub, auth: a, snap: snap, opts: opts, logger: logger,
		connects: ratelimit.New(opts.ConnectLimit, opts.ConnectPer, 100_000)}
}

// Register, rotayı ekler.
func (h *Handler) Register(mux *http.ServeMux) { mux.Handle("GET /v1/ws", h) }

// ServeHTTP: yükseltme her zaman yapılır; yetki, üyelik ve sürüm hataları
// HTTP durumu değil kapanış koduyla bildirilir (mobil istemci yükseltme
// öncesi durumu okuyamaz; api.md §5).
//
// Köken (Origin) denetlenmez: token yalnızca Authorization başlığındadır ve
// tarayıcılar WebSocket isteğine bu başlığı ekleyemez; çapraz site WebSocket
// ele geçirme (CSWSH) çerez taşımadığı için mümkün değildir.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return // Accept yanıtı yazdı (ör. WebSocket isteği değil → 426)
	}
	conn.SetReadLimit(maxClientMessage)
	// Sunucu kapanınca bile bağlantı bağlamı istekten bağımsız yaşar; hub
	// kapanışta 1012 ile kapatır.
	ctx := context.WithoutCancel(r.Context())
	reject := func(code int, reason string) { _ = conn.Close(websocket.StatusCode(code), reason) }

	if err := httpapi.CheckAppVersion(r, h.opts.MinAppVersion); err != nil {
		reject(httpapi.VersionCloseCode(err), "uygulama sürümü desteklenmiyor")
		return
	}
	raw, _ := auth.BearerToken(r.Header.Get("Authorization"))
	hctx, cancel := context.WithTimeout(ctx, h.opts.HandshakeLimit)
	defer cancel()
	p, err := h.auth.Authenticate(hctx, raw)
	if err != nil {
		code := httpapi.CloseCodeFor(err)
		h.denied(r.Context(), code, "", "")
		reject(code, "yetkisiz")
		return
	}
	if ok, _ := h.connects.Allow(p.UserID); !ok {
		h.denied(r.Context(), httpapi.CloseTooMany, p.UserID, "")
		reject(httpapi.CloseTooMany, "çok fazla bağlantı denemesi")
		return
	}
	roomID := r.URL.Query().Get("room_id")
	// Üyelik, hub'a kayıttan önce doğrulanır: üye olmayan bir an bile oda
	// olayı almaz. Kayıttan sonra görüntü tazelenir (arada olay kaçmasın).
	snapshot, err := h.snapshot(hctx, p, roomID)
	if err != nil {
		code := httpapi.CloseCodeFor(err)
		if code == httpapi.CloseInternal {
			h.logger.Error("ws anlık görüntüsü kurulamadı", "user_id", p.UserID, "room_id", roomID, "error", err)
		}
		h.denied(r.Context(), code, p.UserID, roomID)
		reject(code, "odaya bağlanılamadı")
		return
	}

	c := newClient(ctx, conn, p.UserID, roomID, h.hub.queueSize)
	// Önce kayıt (bundan sonraki olaylar kuyruğa girer), sonra anlık görüntü
	// doğrudan yazılır, sonra kuyruk: ilk mesaj her zaman session.snapshot.
	// Görüntüden önceki/eş sürümlü olaylar istemcide yok sayılır (api.md §5).
	if !h.hub.register(c) {
		reject(httpapi.CloseServiceRestart, "sunucu yeniden başlıyor")
		return
	}
	defer h.hub.unregister(c)
	if snapshot, err = h.snapshot(hctx, p, roomID); err == nil {
		wctx, wcancel := context.WithTimeout(ctx, h.opts.WriteTimeout)
		err = conn.Write(wctx, websocket.MessageText, snapshot)
		wcancel()
	}
	if err != nil {
		c.close(httpapi.CloseCodeFor(err), "odaya bağlanılamadı")
	}
	cancel()
	h.logger.Info("ws bağlandı", "user_id", p.UserID, "room_id", roomID)

	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		c.writeLoop(h.opts.PingEvery, h.opts.PingTimeout, h.opts.WriteTimeout)
	}()
	h.readLoop(c, p)
	c.close(httpapi.CloseNormal, "")
	<-writerDone
	code, _ := c.closeStatus()
	h.logger.Info("ws kapandı", "user_id", p.UserID, "room_id", roomID, "code", int(code))
}

// deniedEvents, WS kapanış kodlarının güvenlik olaylarıdır (SAFE-011).
var deniedEvents = map[int]observability.SecurityEvent{
	httpapi.CloseUnauthorized: observability.EventAuthFailed,
	httpapi.CloseNotMember:    observability.EventAccessDenied,
	httpapi.CloseTooMany:      observability.EventRateLimited,
}

// denied, reddedilen bağlantıyı güvenlik olayı olarak loglar (kod güvenlik
// olayı değilse — ör. 5xx, sürüm — bir şey yapmaz).
func (h *Handler) denied(ctx context.Context, code int, userID, roomID string) {
	event, ok := deniedEvents[code]
	if !ok {
		return
	}
	attrs := []slog.Attr{slog.String("route", "GET /v1/ws"), slog.Int("close_code", code)}
	if userID != "" {
		attrs = append(attrs, slog.String("user_id", userID))
	}
	if roomID != "" {
		attrs = append(attrs, slog.String("room_id", roomID))
	}
	observability.Security(ctx, h.logger, slog.LevelWarn, event, "ws bağlantısı reddedildi", attrs...)
}

// snapshot, session.snapshot zarfını kodlar.
func (h *Handler) snapshot(ctx context.Context, p auth.Principal, roomID string) ([]byte, error) {
	sctx, cancel := context.WithTimeout(ctx, h.opts.SnapshotTimeout)
	defer cancel()
	payload, err := h.snap.Snapshot(sctx, p, roomID)
	if err != nil {
		return nil, err
	}
	return json.Marshal(Envelope{V: EnvelopeVersion, Type: TypeSnapshot, RoomID: roomID, TS: h.hub.now().UTC(), Payload: payload})
}

// TypeSnapshot, tam durum mesajıdır.
const TypeSnapshot = "session.snapshot"

// readLoop, istemci mesajlarını okur: yalnızca {"v":1,"type":"sync"} (yeni
// anlık görüntü) kabul edilir, diğerleri yok sayılır. Okuma ayrıca ping
// yanıtlarını (pong) işler.
func (h *Handler) readLoop(c *Client, p auth.Principal) {
	syncs := ratelimit.New(h.opts.SyncLimit, h.opts.SyncPer, 1)
	for {
		_, data, err := c.conn.Read(c.readCtx)
		if err != nil {
			return // istemci kapattı, ağ koptu veya yazıcı kapanış gönderdi
		}
		var msg struct {
			V    int    `json:"v"`
			Type string `json:"type"`
		}
		if json.Unmarshal(data, &msg) != nil || msg.V != EnvelopeVersion || msg.Type != "sync" || !allowed(syncs) {
			continue
		}
		snap, err := h.snapshot(c.readCtx, p, c.roomID)
		if err != nil {
			if apperr.CodeOf(err) == apperr.NotFound {
				c.close(httpapi.CloseNotMember, "oda üyesi değil")
				return
			}
			continue
		}
		c.enqueue(snap)
	}
}

func allowed(l *ratelimit.Limiter) bool {
	ok, _ := l.Allow("sync")
	return ok
}
