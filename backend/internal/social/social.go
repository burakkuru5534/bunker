// Package social, arkadaşlık (istek + onay) ve lobiye davettir
// (2026-11-08 kullanıcı kararı; docs/api.md "Arkadaşlar ve davetler").
//
// Kurallar:
//   - Arkadaş kullanıcı adıyla (küresel benzersiz, auth.NicknameKey) bulunur.
//   - İki taraftan biri diğerini engellediyse istek gönderilemez ve kullanıcı
//     "bulunamadı" görünür (engel sızdırılmaz); engellemek arkadaşlığı,
//     istekleri ve davetleri siler (moderation).
//   - Karşılıklı istek varsa yeni istek doğrudan arkadaşlığa döner.
//   - Davet yalnızca arkadaşa, lobideki odadan ve odanın aktif üyesince
//     gönderilir; 10 dakika geçerlidir. Bildirim (push) yok: istemci ana
//     ekranda GET /v1/inbox ile yoklar.
package social

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/httpapi"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
)

// Sınırlar (kötüye kullanım ve veri büyümesi için).
const (
	MaxFriends         = 100
	MaxOutgoing        = 20
	MaxInvitesPerRoom  = 20
	InviteTTL          = 10 * time.Minute
	friendLimitMessage = "arkadaş sınırı"
)

// Service, arkadaşlık ve davet işlemleridir.
type Service struct {
	db     *persistence.DB
	now    func() time.Time
	logger *slog.Logger
}

// NewService, servisi kurar.
func NewService(db *persistence.DB, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{db: db, now: time.Now, logger: logger}
}

// Person, listelerdeki kullanıcıdır.
type Person struct {
	UserID   string    `json:"user_id"`
	Nickname string    `json:"nickname"`
	Since    time.Time `json:"since"`
}

// Friends, GET /v1/friends yanıtıdır.
type Friends struct {
	Friends  []Person `json:"friends"`
	Incoming []Person `json:"incoming"`
	Outgoing []Person `json:"outgoing"`
}

var errNotFound = apperr.New(apperr.NotFound, "kullanıcı bulunamadı")

// List, arkadaşlar ve bekleyen istekler (takma ada göre).
func (s *Service) List(ctx context.Context, p auth.Principal) (Friends, error) {
	out := Friends{Friends: []Person{}, Incoming: []Person{}, Outgoing: []Person{}}
	err := s.db.Read(ctx, func(q persistence.Querier) error {
		for _, x := range []struct {
			sql string
			dst *[]Person
		}{
			{`select u.id::text, u.nickname, f.created_at from friendships f join users u on u.id = f.friend_id
			   where f.user_id = $1 and u.deleted_at is null order by lower(u.nickname), u.id`, &out.Friends},
			{`select u.id::text, u.nickname, r.created_at from friend_requests r join users u on u.id = r.from_user_id
			   where r.to_user_id = $1 and u.deleted_at is null order by r.created_at desc`, &out.Incoming},
			{`select u.id::text, u.nickname, r.created_at from friend_requests r join users u on u.id = r.to_user_id
			   where r.from_user_id = $1 and u.deleted_at is null order by r.created_at desc`, &out.Outgoing},
		} {
			rows, err := q.Query(ctx, x.sql, p.UserID)
			if err != nil {
				return err
			}
			list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Person, error) {
				var v Person
				err := row.Scan(&v.UserID, &v.Nickname, &v.Since)
				return v, err
			})
			if err != nil {
				return err
			}
			*x.dst = append(*x.dst, list...)
		}
		return nil
	})
	return out, err
}

// RequestResult, istek sonucudur: "requested" ya da karşılıklı istekte
// "friends".
type RequestResult struct {
	Status string `json:"status"`
	User   Person `json:"user"`
}

// Request, kullanıcı adıyla arkadaşlık isteği gönderir.
func (s *Service) Request(ctx context.Context, p auth.Principal, nickname string) (RequestResult, error) {
	key := auth.NicknameKey(nickname)
	if key == "" {
		return RequestResult{}, apperr.Field("nickname", apperr.FieldRequired)
	}
	var res RequestResult
	err := s.db.Tx(ctx, func(tx pgx.Tx) error {
		var target Person
		err := tx.QueryRow(ctx, `select id::text, nickname from users
			  where nickname_key = $1 and deleted_at is null`, key).Scan(&target.UserID, &target.Nickname)
		if errors.Is(err, pgx.ErrNoRows) {
			return errNotFound
		}
		if err != nil {
			return err
		}
		if target.UserID == p.UserID {
			return apperr.Field("nickname", apperr.FieldNotAllowed)
		}
		if blocked, err := blockedEither(ctx, tx, p.UserID, target.UserID); err != nil || blocked {
			if err != nil {
				return err
			}
			return errNotFound
		}
		if friends, err := areFriends(ctx, tx, p.UserID, target.UserID); err != nil || friends {
			if err != nil {
				return err
			}
			return apperr.New(apperr.AlreadyFriends, "zaten arkadaşsınız")
		}
		res.User = target
		var reverse bool
		if err := tx.QueryRow(ctx, `select exists (select 1 from friend_requests
			  where from_user_id = $1 and to_user_id = $2)`, target.UserID, p.UserID).Scan(&reverse); err != nil {
			return err
		}
		if reverse {
			res.Status = "friends"
			return s.befriend(ctx, tx, p.UserID, target.UserID)
		}
		var out int
		if err := tx.QueryRow(ctx, `select count(*) from friend_requests where from_user_id = $1`, p.UserID).Scan(&out); err != nil {
			return err
		}
		if out >= MaxOutgoing {
			return apperr.New(apperr.FriendLimit, friendLimitMessage)
		}
		if err := friendLimit(ctx, tx, p.UserID); err != nil {
			return err
		}
		res.Status = "requested"
		_, err = tx.Exec(ctx, `insert into friend_requests (from_user_id, to_user_id) values ($1, $2)
			on conflict do nothing`, p.UserID, target.UserID)
		return err
	})
	if err != nil {
		return RequestResult{}, err
	}
	res.User.Since = s.now().UTC()
	return res, nil
}

// Accept, gelen isteği kabul eder.
func (s *Service) Accept(ctx context.Context, p auth.Principal, fromUserID string) error {
	if !validUUID(fromUserID) {
		return apperr.New(apperr.NotFound, "istek bulunamadı")
	}
	return s.db.Tx(ctx, func(tx pgx.Tx) error {
		var ok bool
		if err := tx.QueryRow(ctx, `select exists (select 1 from friend_requests r join users u on u.id = r.from_user_id
			  where r.from_user_id = $1 and r.to_user_id = $2 and u.deleted_at is null)`, fromUserID, p.UserID).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return apperr.New(apperr.NotFound, "istek bulunamadı")
		}
		return s.befriend(ctx, tx, p.UserID, fromUserID)
	})
}

// befriend, iki kullanıcıyı arkadaş yapar ve aralarındaki istekleri siler.
func (s *Service) befriend(ctx context.Context, tx pgx.Tx, a, b string) error {
	if err := friendLimit(ctx, tx, a); err != nil {
		return err
	}
	if err := friendLimit(ctx, tx, b); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `insert into friendships (user_id, friend_id) values ($1, $2), ($2, $1)
		on conflict do nothing`, a, b); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `delete from friend_requests where (from_user_id = $1 and to_user_id = $2)
		or (from_user_id = $2 and to_user_id = $1)`, a, b)
	return err
}

// DeleteRequest, gelen isteği reddeder ya da gönderdiğini geri çeker
// (iki yönde; yoksa değişiklik yok).
func (s *Service) DeleteRequest(ctx context.Context, p auth.Principal, other string) error {
	if !validUUID(other) {
		return nil
	}
	return s.db.Tx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `delete from friend_requests where (from_user_id = $1 and to_user_id = $2)
			or (from_user_id = $2 and to_user_id = $1)`, p.UserID, other)
		return err
	})
}

// Remove, arkadaşlıktan çıkarır; aralarındaki davetler de silinir.
func (s *Service) Remove(ctx context.Context, p auth.Principal, friendID string) error {
	if !validUUID(friendID) {
		return nil
	}
	return s.db.Tx(ctx, func(tx pgx.Tx) error {
		return Unlink(ctx, tx, p.UserID, friendID)
	})
}

// Unlink, iki kullanıcı arasındaki arkadaşlığı, istekleri ve davetleri
// siler (arkadaşlıktan çıkarma ve engelleme kullanır).
func Unlink(ctx context.Context, tx pgx.Tx, a, b string) error {
	for _, q := range []string{
		`delete from friendships where (user_id = $1 and friend_id = $2) or (user_id = $2 and friend_id = $1)`,
		`delete from friend_requests where (from_user_id = $1 and to_user_id = $2) or (from_user_id = $2 and to_user_id = $1)`,
		`delete from room_invites where (from_user_id = $1 and to_user_id = $2) or (from_user_id = $2 and to_user_id = $1)`,
	} {
		if _, err := tx.Exec(ctx, q, a, b); err != nil {
			return err
		}
	}
	return nil
}

// Invite, arkadaşı lobideki odaya davet eder (aynı davet süresini yeniler).
func (s *Service) Invite(ctx context.Context, p auth.Principal, roomID, toUserID string) error {
	if !validUUID(roomID) {
		return apperr.New(apperr.NotFound, "oda bulunamadı")
	}
	if !validUUID(toUserID) {
		return apperr.Field("user_id", apperr.FieldInvalid)
	}
	return s.db.Tx(ctx, func(tx pgx.Tx) error {
		var status string
		err := tx.QueryRow(ctx, `select r.status from rooms r join room_members m on m.room_id = r.id
			  where r.id = $1 and m.user_id = $2 and m.left_at is null`, roomID, p.UserID).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.New(apperr.NotFound, "oda bulunamadı")
		}
		if err != nil {
			return err
		}
		if status != "lobby" {
			return apperr.New(apperr.InvalidPhase, "davet yalnızca lobide")
		}
		friends, err := areFriends(ctx, tx, p.UserID, toUserID)
		if err != nil {
			return err
		}
		if !friends {
			return apperr.New(apperr.Forbidden, "yalnızca arkadaşlar davet edilebilir")
		}
		var member bool
		if err := tx.QueryRow(ctx, `select exists (select 1 from room_members
			  where room_id = $1 and user_id = $2 and left_at is null)`, roomID, toUserID).Scan(&member); err != nil {
			return err
		}
		if member {
			return nil // zaten odada
		}
		var n int
		if err := tx.QueryRow(ctx, `select count(*) from room_invites where room_id = $1 and expires_at > now()`, roomID).Scan(&n); err != nil {
			return err
		}
		if n >= MaxInvitesPerRoom {
			return apperr.New(apperr.FriendLimit, "davet sınırı")
		}
		_, err = tx.Exec(ctx, `insert into room_invites (room_id, to_user_id, from_user_id, expires_at)
			values ($1, $2, $3, $4)
			on conflict (room_id, to_user_id) do update set from_user_id = excluded.from_user_id,
			  created_at = now(), expires_at = excluded.expires_at`,
			roomID, toUserID, p.UserID, s.now().Add(InviteTTL))
		return err
	})
}

// InviteView, gelen davettir.
type InviteView struct {
	RoomID     string    `json:"room_id"`
	InviteCode string    `json:"invite_code"`
	From       Person    `json:"from"`
	Players    int       `json:"players"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// Inbox, GET /v1/inbox yanıtıdır: geçerli davetler ve gelen istek sayısı.
type Inbox struct {
	Invites          []InviteView `json:"invites"`
	IncomingRequests int          `json:"incoming_requests"`
}

// Inbox, ana ekranın yokladığı özet. Lobide olmayan, süresi geçmiş,
// gönderenle arkadaşlığı bitmiş ya da zaten üyesi olduğum odaların davetleri
// listelenmez.
func (s *Service) Inbox(ctx context.Context, p auth.Principal) (Inbox, error) {
	out := Inbox{Invites: []InviteView{}}
	err := s.db.Read(ctx, func(q persistence.Querier) error {
		rows, err := q.Query(ctx, `
			select i.room_id::text, r.invite_code, u.id::text, u.nickname, i.created_at, i.expires_at,
			       (select count(*) from room_members m where m.room_id = r.id and m.left_at is null and m.role = 'player')
			  from room_invites i
			  join rooms r on r.id = i.room_id
			  join users u on u.id = i.from_user_id
			  join friendships f on f.user_id = i.to_user_id and f.friend_id = i.from_user_id
			 where i.to_user_id = $1 and i.expires_at > now() and r.status = 'lobby' and u.deleted_at is null
			   and not exists (select 1 from room_members m where m.room_id = r.id and m.user_id = $1 and m.left_at is null)
			 order by i.created_at desc
			 limit 20`, p.UserID)
		if err != nil {
			return err
		}
		out.Invites, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (InviteView, error) {
			var v InviteView
			err := row.Scan(&v.RoomID, &v.InviteCode, &v.From.UserID, &v.From.Nickname, &v.From.Since, &v.ExpiresAt, &v.Players)
			return v, err
		})
		if err != nil {
			return err
		}
		return q.QueryRow(ctx, `select count(*) from friend_requests r join users u on u.id = r.from_user_id
			  where r.to_user_id = $1 and u.deleted_at is null`, p.UserID).Scan(&out.IncomingRequests)
	})
	if out.Invites == nil {
		out.Invites = []InviteView{}
	}
	return out, err
}

// Decline, daveti reddeder (yoksa değişiklik yok).
func (s *Service) Decline(ctx context.Context, p auth.Principal, roomID string) error {
	if !validUUID(roomID) {
		return nil
	}
	return s.db.Tx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `delete from room_invites where room_id = $1 and to_user_id = $2`, roomID, p.UserID)
		return err
	})
}

func areFriends(ctx context.Context, tx pgx.Tx, a, b string) (bool, error) {
	var ok bool
	err := tx.QueryRow(ctx, `select exists (select 1 from friendships where user_id = $1 and friend_id = $2)`, a, b).Scan(&ok)
	return ok, err
}

func blockedEither(ctx context.Context, tx pgx.Tx, a, b string) (bool, error) {
	var ok bool
	err := tx.QueryRow(ctx, `select exists (select 1 from blocks where (blocker_id = $1 and blocked_id = $2)
		or (blocker_id = $2 and blocked_id = $1))`, a, b).Scan(&ok)
	return ok, err
}

func friendLimit(ctx context.Context, tx pgx.Tx, userID string) error {
	var n int
	if err := tx.QueryRow(ctx, `select count(*) from friendships where user_id = $1`, userID).Scan(&n); err != nil {
		return err
	}
	if n >= MaxFriends {
		return apperr.New(apperr.FriendLimit, friendLimitMessage)
	}
	return nil
}

func validUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < 36; i++ {
		c := s[i]
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
				return false
			}
		}
	}
	return true
}

// HTTP, arkadaşlık ve davet uç noktalarıdır.
type HTTP struct {
	svc     *Service
	require func(http.Handler) http.Handler
}

// NewHTTP, uç noktaları oturum ara katmanıyla kurar.
func NewHTTP(svc *Service, require func(http.Handler) http.Handler) *HTTP {
	return &HTTP{svc: svc, require: require}
}

// Register, rotaları ekler.
func (h *HTTP) Register(mux *http.ServeMux) {
	for pattern, fn := range map[string]httpapi.HandlerFunc{
		"GET /v1/friends":                           h.list,
		"POST /v1/friends/requests":                 h.request,
		"POST /v1/friends/requests/{userId}/accept": h.accept,
		"DELETE /v1/friends/requests/{userId}":      h.deleteRequest,
		"DELETE /v1/friends/{userId}":               h.remove,
		"POST /v1/rooms/{roomId}/invites":           h.invite,
		"GET /v1/inbox":                             h.inbox,
		"DELETE /v1/invites/{roomId}":               h.decline,
	} {
		mux.Handle(pattern, h.require(httpapi.Handle(fn)))
	}
}

func principal(r *http.Request) auth.Principal {
	p, _ := auth.PrincipalFrom(r.Context())
	return p
}

func noContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *HTTP) list(w http.ResponseWriter, r *http.Request) error {
	f, err := h.svc.List(r.Context(), principal(r))
	if err != nil {
		return err
	}
	w.Header().Set("Cache-Control", "no-store")
	httpapi.WriteJSON(w, http.StatusOK, f)
	return nil
}

// POST /v1/friends/requests {"nickname"} → 201 {"status": "requested"|"friends", "user"}
func (h *HTTP) request(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Nickname string `json:"nickname"`
	}
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		return err
	}
	res, err := h.svc.Request(r.Context(), principal(r), req.Nickname)
	if err != nil {
		return err
	}
	httpapi.WriteJSON(w, http.StatusCreated, res)
	return nil
}

func (h *HTTP) accept(w http.ResponseWriter, r *http.Request) error {
	if err := h.svc.Accept(r.Context(), principal(r), r.PathValue("userId")); err != nil {
		return err
	}
	return noContent(w)
}

func (h *HTTP) deleteRequest(w http.ResponseWriter, r *http.Request) error {
	if err := h.svc.DeleteRequest(r.Context(), principal(r), r.PathValue("userId")); err != nil {
		return err
	}
	return noContent(w)
}

func (h *HTTP) remove(w http.ResponseWriter, r *http.Request) error {
	if err := h.svc.Remove(r.Context(), principal(r), r.PathValue("userId")); err != nil {
		return err
	}
	return noContent(w)
}

// POST /v1/rooms/{roomId}/invites {"user_id"} → 204
func (h *HTTP) invite(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		return err
	}
	if err := h.svc.Invite(r.Context(), principal(r), r.PathValue("roomId"), req.UserID); err != nil {
		return err
	}
	return noContent(w)
}

func (h *HTTP) inbox(w http.ResponseWriter, r *http.Request) error {
	in, err := h.svc.Inbox(r.Context(), principal(r))
	if err != nil {
		return err
	}
	w.Header().Set("Cache-Control", "no-store")
	httpapi.WriteJSON(w, http.StatusOK, in)
	return nil
}

func (h *HTTP) decline(w http.ResponseWriter, r *http.Request) error {
	if err := h.svc.Decline(r.Context(), principal(r), r.PathValue("roomId")); err != nil {
		return err
	}
	return noContent(w)
}
