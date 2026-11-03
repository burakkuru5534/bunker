package room

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"golang.org/x/text/unicode/norm"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
	"github.com/burakkuru5534/bunker/backend/internal/ratelimit"
)

// Sohbet kuralları (product-requirements §4, api.md).
const (
	EventChatMessage = "chat.message"
	MaxMessageRunes  = 200
	// ChatHistory, anlık görüntüdeki son mesaj sayısı (ADR-0006).
	ChatHistory = 50
)

// TextFilter, sohbet metnindeki küfürleri maskeler (SAFE-013). nil: filtre yok.
type TextFilter interface {
	Mask(text string) string
}

// WithTextFilter, küfür filtresini bağlar.
func (s *Service) WithTextFilter(f TextFilter) *Service {
	s.textFilter = f
	return s
}

// chatLimits: kullanıcı başına saniyede 1 ve 10 sn'de 5 mesaj (sunucu tarafı).
type chatLimits struct{ perSecond, perTen *ratelimit.Limiter }

func newChatLimits() chatLimits {
	return chatLimits{perSecond: ratelimit.New(1, time.Second, 100_000), perTen: ratelimit.New(5, 10*time.Second, 100_000)}
}

func (l chatLimits) allow(userID string) error {
	if ok, wait := l.perSecond.Allow(userID); !ok {
		return apperr.RateLimit(wait)
	}
	if ok, wait := l.perTen.Allow(userID); !ok {
		return apperr.RateLimit(wait)
	}
	return nil
}

// CleanMessage, mesaj metnini normalleştirir: NFC; satır sonu ve sekme →
// boşluk; diğer kontrol ve biçim karakterleri (yön değiştiren Unicode
// karakterleri dahil — metni ters göstererek yanıltmaya karşı) atılır;
// ardışık boşluklar teke iner, kırpılır. Boş veya 200 karakterden uzunsa
// MESSAGE_INVALID.
func CleanMessage(raw string) (string, error) {
	if !utf8.ValidString(raw) {
		return "", messageInvalid("geçersiz karakter")
	}
	var b strings.Builder
	space := false
	for _, r := range norm.NFC.String(raw) {
		switch {
		case r == '\n' || r == '\r' || r == '\t' || unicode.IsSpace(r):
			space = b.Len() > 0
			continue
		case unicode.IsControl(r) || unicode.Is(unicode.Cf, r):
			continue
		}
		if space {
			b.WriteByte(' ')
			space = false
		}
		b.WriteRune(r)
	}
	text := b.String()
	switch n := utf8.RuneCountInString(text); {
	case n == 0:
		return "", messageInvalid("mesaj boş")
	case n > MaxMessageRunes:
		return "", messageInvalid("mesaj en fazla 200 karakter olabilir")
	}
	return text, nil
}

func messageInvalid(msg string) error { return apperr.New(apperr.MessageInvalid, msg) }

// ChatMessage, sohbet mesajının istemci görünümüdür.
type ChatMessage struct {
	ID        int64     `json:"id"`
	UserID    string    `json:"user_id"`
	Nickname  string    `json:"nickname"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// SendMessage, odaya sohbet mesajı yazar. Lobide (ve oyun bitince) her aktif
// üye, oyun sürerken yalnızca hayattaki oyuncular yazabilir (izleyici ve
// elenen okur, yazamaz → FORBIDDEN). Üye değilse NOT_FOUND. Aynı
// client_msg_id tekrar gelirse ilk mesaj döner, yeniden yayınlanmaz.
// Mesaj, gönderen engelleyenlere gönderilmez.
func (s *Service) SendMessage(ctx context.Context, p auth.Principal, roomID, clientMsgID, raw string) (ChatMessage, error) {
	var fe apperr.FieldErrors
	if clientMsgID == "" {
		fe.Add("client_msg_id", apperr.FieldRequired)
	} else if !validUUID(clientMsgID) {
		fe.Add("client_msg_id", apperr.FieldInvalid)
	}
	if err := fe.Err(); err != nil {
		return ChatMessage{}, err
	}
	text, err := CleanMessage(raw)
	if err != nil {
		return ChatMessage{}, err
	}
	if s.textFilter != nil {
		text = s.textFilter.Mask(text)
	}

	var (
		msg      ChatMessage
		fresh    bool
		excluded []string
	)
	err = s.db.Tx(ctx, func(tx pgx.Tx) error {
		m, err := LoadMembership(ctx, tx, roomID, p.UserID)
		if err != nil {
			return err
		}
		if m.RoomStatus == StatusInGame {
			var alive bool
			if err := tx.QueryRow(ctx, `select exists (select 1 from game_participants
				where game_id = $1 and user_id = $2 and status = 'alive')`, m.CurrentGameID, p.UserID).Scan(&alive); err != nil {
				return err
			}
			if !alive {
				return apperr.New(apperr.Forbidden, "izleyiciler sohbete yazamaz")
			}
		}
		// Tekrar: aynı client_msg_id ile kayıtlı mesaj aynen döner (sıklık
		// sınırına da takılmaz: ağ tekrarı yeni mesaj değildir).
		err = tx.QueryRow(ctx, `select c.id, u.nickname, c.body, c.created_at from chat_messages c
			join users u on u.id = c.user_id where c.room_id = $1 and c.user_id = $2 and c.client_msg_id = $3`,
			roomID, p.UserID, clientMsgID).Scan(&msg.ID, &msg.Nickname, &msg.Text, &msg.CreatedAt)
		if err == nil {
			msg.UserID = p.UserID
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if err := s.chat.allow(p.UserID); err != nil {
			return err
		}
		msg = ChatMessage{UserID: p.UserID, Text: text}
		if err := tx.QueryRow(ctx, `insert into chat_messages (room_id, game_id, user_id, client_msg_id, body)
			values ($1, $2, $3, $4, $5) returning id, created_at`,
			roomID, m.CurrentGameID, p.UserID, clientMsgID, text).Scan(&msg.ID, &msg.CreatedAt); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `select nickname from users where id = $1`, p.UserID).Scan(&msg.Nickname); err != nil {
			return err
		}
		// Gönderen engelleyen oda üyeleri bu mesajı almaz.
		rows, err := tx.Query(ctx, `select b.blocker_id::text from blocks b
			join room_members m on m.user_id = b.blocker_id and m.room_id = $1 and m.left_at is null
			where b.blocked_id = $2`, roomID, p.UserID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			excluded = append(excluded, id)
		}
		fresh = true
		return rows.Err()
	})
	if err != nil {
		return ChatMessage{}, err
	}
	msg.CreatedAt = msg.CreatedAt.UTC()
	if fresh {
		s.publisher.Publish(Event{RoomID: roomID, Type: EventChatMessage, Exclude: excluded, Payload: msg})
	}
	return msg, nil
}

// RecentMessages, odanın son mesajlarıdır (eskiden yeniye), izleyicinin
// engellediklerininki hariç. Yetki çağıranda (oda üyeliği) denetlenir.
func (s *Service) RecentMessages(ctx context.Context, viewerID, roomID string) ([]ChatMessage, error) {
	out := []ChatMessage{}
	err := s.db.Read(ctx, func(q persistence.Querier) error {
		rows, err := q.Query(ctx, `select c.id, c.user_id::text, u.nickname, c.body, c.created_at
			from chat_messages c join users u on u.id = c.user_id
			where c.room_id = $1 and not exists (select 1 from blocks b where b.blocker_id = $2 and b.blocked_id = c.user_id)
			order by c.created_at desc, c.id desc limit $3`, roomID, viewerID, ChatHistory)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var m ChatMessage
			if err := rows.Scan(&m.ID, &m.UserID, &m.Nickname, &m.Text, &m.CreatedAt); err != nil {
				return err
			}
			m.CreatedAt = m.CreatedAt.UTC()
			out = append(out, m)
		}
		return rows.Err()
	})
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, err
}

// POST /v1/rooms/{roomId}/messages {"client_msg_id","text"} → 201 {"id","created_at"}
func (h *HTTP) message(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		ClientMsgID string  `json:"client_msg_id"`
		Text        *string `json:"text"`
	}
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		return err
	}
	if req.Text == nil {
		return apperr.Field("text", apperr.FieldRequired)
	}
	p, _ := auth.PrincipalFrom(r.Context())
	msg, err := h.svc.SendMessage(r.Context(), p, r.PathValue("roomId"), req.ClientMsgID, *req.Text)
	if err != nil {
		return err
	}
	httpapi.WriteJSON(w, http.StatusCreated, map[string]any{"id": msg.ID, "created_at": msg.CreatedAt})
	return nil
}
