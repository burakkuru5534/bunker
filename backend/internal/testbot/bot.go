package testbot

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coder/websocket"
)

// Config, botların ortak ayarlarıdır.
type Config struct {
	// API, sunucunun taban adresi (ör. http://localhost:8080).
	API *url.URL
	// AppVersion, X-App-Version başlığı (sunucunun MIN_APP_VERSION'ından küçük olmamalı).
	AppVersion string
	// Delay, her eylemden önce beklenecek süre (insan hızına yakın; 0 olabilir).
	Delay  time.Duration
	HTTP   *http.Client
	Logger *slog.Logger
}

// Bot, tek bir otomatik oyuncudur.
type Bot struct {
	cfg      Config
	Nickname string
	token    string
	UserID   string
	RoomID   string
	brain    *Brain

	// Events, test ve gözlem için her işlenen olayın türünü alır (isteğe bağlı).
	Events chan<- string
}

// New, botu kurar; Register çağrılmadan kullanılamaz.
func New(cfg Config, nickname string) *Bot {
	if cfg.HTTP == nil {
		cfg.HTTP = &http.Client{Timeout: 15 * time.Second}
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &Bot{cfg: cfg, Nickname: nickname}
}

// APIError, sunucunun standart hata yanıtıdır.
type APIError struct {
	Status int
	Code   string
}

func (e *APIError) Error() string { return fmt.Sprintf("HTTP %d %s", e.Status, e.Code) }

// Register, misafir hesap açar.
func (b *Bot) Register(ctx context.Context) error {
	var out struct {
		Token string `json:"token"`
		User  struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := b.call(ctx, http.MethodPost, "/v1/sessions/guest", map[string]any{"nickname": b.Nickname}, &out); err != nil {
		return err
	}
	b.token, b.UserID = out.Token, out.User.ID
	b.brain = NewBrain(b.UserID, nil)
	return nil
}

// Create, oda kurar (bot oda sahibi olur) ve davet kodunu döndürür.
// startAt > 0 ise o kadar oyuncu hazır olunca oyunu kendisi başlatır.
func (b *Bot) Create(ctx context.Context, startAt int) (string, error) {
	var room struct {
		ID   string `json:"id"`
		Code string `json:"invite_code"`
	}
	if err := b.call(ctx, http.MethodPost, "/v1/rooms", map[string]any{}, &room); err != nil {
		return "", err
	}
	b.RoomID = room.ID
	b.brain.StartAt = startAt
	return room.Code, nil
}

// Join, davet koduyla katılır.
func (b *Bot) Join(ctx context.Context, code string) error {
	var room struct {
		ID string `json:"id"`
	}
	if err := b.call(ctx, http.MethodPost, "/v1/rooms/join", map[string]any{"invite_code": code}, &room); err != nil {
		return err
	}
	b.RoomID = room.ID
	return nil
}

// Cleanup, odadan ayrılır ve misafir hesabı siler (veritabanında bot
// hesabı birikmesin). Hataları yalnızca loglar.
func (b *Bot) Cleanup(ctx context.Context) {
	if b.token == "" {
		return
	}
	if b.RoomID != "" {
		if err := b.call(ctx, http.MethodPost, "/v1/rooms/"+b.RoomID+"/leave", nil, nil); err != nil {
			b.cfg.Logger.Warn("ayrılamadı", "bot", b.Nickname, "error", err)
		}
	}
	if err := b.call(ctx, http.MethodDelete, "/v1/me", nil, nil); err != nil {
		b.cfg.Logger.Warn("hesap silinemedi", "bot", b.Nickname, "error", err)
	}
}

// Run, odanın WebSocket akışını dinler ve eylemleri yapar; ctx bitene,
// bot odadan atılana veya bağlantı kalıcı kapanana kadar sürer. Geçici
// kopmalarda yeniden bağlanır.
func (b *Bot) Run(ctx context.Context) error {
	for attempt := 0; ; attempt++ {
		err := b.session(ctx)
		if ctx.Err() != nil {
			return nil
		}
		var ce websocket.CloseError
		if errors.As(err, &ce) {
			switch ce.Code {
			case 4001, 4003, 4026:
				return fmt.Errorf("%s: bağlantı kalıcı kapandı (%d)", b.Nickname, ce.Code)
			}
		}
		wait := time.Duration(min(attempt+1, 10)) * 500 * time.Millisecond
		b.cfg.Logger.Info("yeniden bağlanıyor", "bot", b.Nickname, "error", err, "wait", wait)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(wait):
		}
	}
}

func (b *Bot) session(ctx context.Context) error {
	u := *b.cfg.API
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	u.Path = "/v1/ws"
	u.RawQuery = url.Values{"room_id": {b.RoomID}}.Encode()
	conn, _, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{
		HTTPClient: b.cfg.HTTP,
		HTTPHeader: http.Header{
			"Authorization": {"Bearer " + b.token},
			"X-App-Version": {b.cfg.AppVersion},
		},
	})
	if err != nil {
		return err
	}
	defer conn.CloseNow()
	conn.SetReadLimit(1 << 20)

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		var env struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if json.Unmarshal(data, &env) != nil {
			continue
		}
		b.brain.Handle(env.Type, env.Payload)
		if b.Events != nil {
			select {
			case b.Events <- env.Type:
			default:
			}
		}
		if env.Type == "room.member_kicked" && strings.Contains(string(env.Payload), b.UserID) {
			return errors.New("odadan atıldı")
		}
		for _, a := range b.brain.Next() {
			if err := b.perform(ctx, a); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				b.cfg.Logger.Info("eylem reddedildi", "bot", b.Nickname, "action", a.Kind+" "+a.Type, "error", err)
				var apiErr *APIError
				// Ağ hatası: yeniden denenebilir. Kural reddi: aynı durumda tekrar denenmez.
				if !errors.As(err, &apiErr) {
					b.brain.Retry(a)
				}
				if a.Kind == "start" {
					b.brain.Retry(a)
				}
			}
		}
	}
}

func (b *Bot) perform(ctx context.Context, a Action) error {
	if b.cfg.Delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(b.cfg.Delay):
		}
	}
	switch a.Kind {
	case "ready":
		return b.call(ctx, http.MethodPost, "/v1/rooms/"+b.RoomID+"/ready", map[string]any{"ready": true}, nil)
	case "start":
		err := b.call(ctx, http.MethodPost, "/v1/rooms/"+b.RoomID+"/start", nil, nil)
		if err == nil {
			b.cfg.Logger.Info("oyunu başlattı", "bot", b.Nickname)
		}
		return err
	default:
		err := b.call(ctx, http.MethodPost, "/v1/games/"+b.brain.GameID()+"/actions", map[string]any{
			"action_id": newUUID(), "type": a.Type, "payload": a.Payload,
		}, nil)
		if err == nil {
			b.cfg.Logger.Info("eylem", "bot", b.Nickname, "type", a.Type)
		}
		return err
	}
}

func (b *Bot) call(ctx context.Context, method, path string, body any, out any) error {
	var rd io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, b.cfg.API.JoinPath(path).String(), rd)
	if err != nil {
		return err
	}
	req.Header.Set("X-App-Version", b.cfg.AppVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if b.token != "" {
		req.Header.Set("Authorization", "Bearer "+b.token)
	}
	resp, err := b.cfg.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		var e struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(raw, &e)
		return &APIError{Status: resp.StatusCode, Code: e.Error.Code}
	}
	if out != nil && len(raw) > 0 {
		return json.Unmarshal(raw, out)
	}
	return nil
}

// Token, botun oturum token'ıdır (yalnızca testler; loglanmaz).
func (b *Bot) Token() string { return b.token }

func newUUID() string {
	var u [16]byte
	_, _ = rand.Read(u[:])
	u[6] = u[6]&0x0f | 0x40
	u[8] = u[8]&0x3f | 0x80
	h := hex.EncodeToString(u[:])
	return h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}
