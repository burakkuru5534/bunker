package game

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
	"github.com/burakkuru5534/bunker/backend/internal/room"
)

// Eylem türleri (api.md "Oyun" eylem tablosu; game_actions.type).
const (
	ActionRevealCard      = "reveal_card"
	ActionDiscussionReady = "discussion_ready"
	ActionVote            = "vote"
	ActionTiebreakVote    = "tiebreak_vote"
)

// Service, oyun eylemleridir (ENG-005…).
type Service struct {
	db        *persistence.DB
	engine    engine.Engine
	publisher room.Publisher
	rng       engine.Rand
	now       func() time.Time
	logger    *slog.Logger
}

// NewService, servisi kurar. publisher nil olabilir (RT öncesi).
func NewService(db *persistence.DB, publisher room.Publisher, logger *slog.Logger) *Service {
	if publisher == nil {
		publisher = room.NoPublisher{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{db: db, engine: engine.New(engine.DefaultDurations), publisher: publisher,
		rng: engine.CryptoRand{}, now: time.Now, logger: logger}
}

// ActionRequest, POST /v1/games/{id}/actions gövdesidir.
type ActionRequest struct {
	ActionID string          `json:"action_id"`
	Type     string          `json:"type"`
	Payload  json.RawMessage `json:"payload"`
}

// ActionResult, eylem yanıtıdır.
type ActionResult struct {
	StateVersion int64 `json:"state_version"`
	Duplicate    bool  `json:"duplicate"`
}

// action, doğrulanmış ve motora uygulanmaya hazır eylemdir.
type action struct {
	typ     string
	payload map[string]any // game_actions.payload (denetim kaydı)
	apply   func(e engine.Engine, s engine.State, userID string, now time.Time, rng engine.Rand) (engine.State, []engine.Event, error)
	// persist, durum farkının dışında yazılması gerekeni yazar (oylar; bkz. saveState).
	persist func(ctx context.Context, tx pgx.Tx, gameID string, old engine.State, userID string) error
}

// parseAction, gövdeyi doğrular (transaction'dan önce; geçersiz istek
// kilit almaz).
func parseAction(req ActionRequest) (action, error) {
	var fe apperr.FieldErrors
	if req.ActionID == "" {
		fe.Add("action_id", apperr.FieldRequired)
	} else if !validUUID(req.ActionID) {
		fe.Add("action_id", apperr.FieldInvalid)
	}
	switch req.Type {
	case "":
		fe.Add("type", apperr.FieldRequired)
	case ActionRevealCard, ActionDiscussionReady, ActionVote:
	case ActionTiebreakVote:
		// ENG-009'da eklenecek.
		fe.Add("type", apperr.FieldNotAllowed)
	default:
		fe.Add("type", apperr.FieldNotAllowed)
	}
	if err := fe.Err(); err != nil {
		return action{}, err
	}
	switch req.Type {
	case ActionRevealCard:
		var p struct {
			Category *string `json:"category"`
		}
		if err := strictUnmarshal(req.Payload, &p); err != nil {
			return action{}, err
		}
		if p.Category == nil {
			return action{}, apperr.Field("payload.category", apperr.FieldRequired)
		}
		c := engine.Category(*p.Category)
		return action{typ: req.Type, payload: map[string]any{"category": *p.Category},
			apply: func(e engine.Engine, s engine.State, userID string, now time.Time, _ engine.Rand) (engine.State, []engine.Event, error) {
				return e.Reveal(s, userID, c, now)
			}}, nil
	case ActionDiscussionReady:
		var p struct{}
		if err := strictUnmarshal(req.Payload, &p); err != nil {
			return action{}, err
		}
		return action{typ: req.Type, payload: map[string]any{},
			apply: func(e engine.Engine, s engine.State, userID string, now time.Time, _ engine.Rand) (engine.State, []engine.Event, error) {
				return e.DiscussionReady(s, userID, now)
			}}, nil
	default: // ActionVote
		var p struct {
			Target *string `json:"target_user_id"`
		}
		if err := strictUnmarshal(req.Payload, &p); err != nil {
			return action{}, err
		}
		switch {
		case p.Target == nil:
			return action{}, apperr.Field("payload.target_user_id", apperr.FieldRequired)
		case !validUUID(*p.Target):
			return action{}, apperr.Field("payload.target_user_id", apperr.FieldInvalid)
		}
		target := strings.ToLower(*p.Target)
		// Hedef denetim kaydına yazılmaz: oylar yalnızca votes tablosunda
		// (gizli; api.md "Gizlilik kuralları").
		return action{typ: req.Type, payload: map[string]any{},
			apply: func(e engine.Engine, s engine.State, userID string, now time.Time, rng engine.Rand) (engine.State, []engine.Event, error) {
				return e.Vote(s, userID, target, now, rng)
			},
			persist: func(ctx context.Context, tx pgx.Tx, gameID string, old engine.State, userID string) error {
				return insertVote(ctx, tx, gameID, old.Round, voteRegular, engine.Vote{Voter: userID, Target: target})
			}}, nil
	}
}

// strictUnmarshal, eylem yükünü çözer: nesne olmalı, bilinmeyen alan yok.
func strictUnmarshal(raw json.RawMessage, dst any) error {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		raw = json.RawMessage(`{}`)
	}
	if bytes.TrimSpace(raw)[0] != '{' {
		return apperr.Field("payload", apperr.FieldInvalidType)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) {
			return apperr.Field("payload."+typeErr.Field, apperr.FieldInvalidType)
		}
		return apperr.Field("payload", apperr.FieldUnknown)
	}
	return nil
}

// Act, bir oyuncu eylemini işler (ADR-0004): oyun satırı kilitlenir; çağıran
// katılımcı değilse NOT_FOUND (varlık sızmaz); aynı action_id daha önce
// işlendiyse kaydedilen sürüm duplicate=true ile döner; değilse motor
// uygulanır, fark ve eylem kaydı yazılır, olaylar commit sonrası yayınlanır.
// Reddedilen eylemler kaydedilmez (aynı action_id ile yeniden denenebilir).
//
// Aşamanın süresi dolmuşsa (zamanlayıcı henüz işlemediyse; ör. instance
// uyuyordu) önce süre dolumu kendi transaction'ında işlenip commit edilir,
// eylem sonra güncel duruma uygulanır (ADR-0005 "ek güvence"). Böylece eylem
// reddedilse de süre dolumu kaybolmaz.
func (s *Service) Act(ctx context.Context, p auth.Principal, gameID string, req ActionRequest) (ActionResult, error) {
	notFound := apperr.New(apperr.NotFound, "oyun bulunamadı")
	if !validUUID(gameID) {
		return ActionResult{}, notFound
	}
	act, err := parseAction(req)
	if err != nil {
		return ActionResult{}, err
	}
	for attempt := 0; ; attempt++ {
		var (
			res      ActionResult
			events   []room.Event
			timedOut bool
		)
		err = s.db.Tx(ctx, func(tx pgx.Tx) error {
			old, g, err := loadState(ctx, tx, gameID, true)
			if errors.Is(err, errGameNotFound) {
				return notFound
			}
			if err != nil {
				return err
			}
			if old.Player(p.UserID) == nil {
				return notFound
			}
			var prev int64
			err = tx.QueryRow(ctx, `select state_version from game_actions where game_id = $1 and actor_id = $2 and action_id = $3`,
				gameID, p.UserID, req.ActionID).Scan(&prev)
			if err == nil {
				res = ActionResult{StateVersion: prev, Duplicate: true}
				return nil
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			now := s.clock()
			if attempt == 0 && overdue(old, now) {
				_, events, err = s.applyDue(ctx, tx, g, old, now)
				timedOut = true
				return err
			}
			next, engineEvents, err := act.apply(s.engine, old, p.UserID, now, s.rng)
			if errors.Is(err, engine.ErrNoChange) {
				// Geçerli ama etkisiz (ör. ikinci "Hazırım"): sürüm artmaz, olay
				// yok; tekrar gönderim duplicate dönsün diye kaydedilir.
				res = ActionResult{StateVersion: old.Version}
				_, err = tx.Exec(ctx, `insert into game_actions (game_id, actor_id, action_id, type, payload, state_version)
					values ($1, $2, $3, $4, $5, $6)`, gameID, p.UserID, req.ActionID, act.typ, act.payload, old.Version)
				return err
			}
			if err != nil {
				return err
			}
			if err := saveState(ctx, tx, gameID, old, next, now); err != nil {
				return err
			}
			if act.persist != nil {
				if err := act.persist(ctx, tx, gameID, old, p.UserID); err != nil {
					return err
				}
			}
			if _, err := tx.Exec(ctx, `insert into game_actions (game_id, actor_id, action_id, type, payload, state_version)
				values ($1, $2, $3, $4, $5, $6)`, gameID, p.UserID, req.ActionID, act.typ, act.payload, next.Version); err != nil {
				return err
			}
			events, err = s.wireEvents(ctx, tx, g, engineEvents, next.Version, now)
			res = ActionResult{StateVersion: next.Version}
			return err
		})
		if err != nil {
			return ActionResult{}, err
		}
		for _, e := range events {
			s.publisher.Publish(e)
		}
		if !timedOut {
			return res, nil
		}
	}
}

// wireEvents, motor olaylarını istemci biçimine çevirir: kart kimliği
// yerine metin (yalnızca açılan kart), aşama adı büyük harf.
func (s *Service) wireEvents(ctx context.Context, tx pgx.Tx, g gameRow, evs []engine.Event, version int64, now time.Time) ([]room.Event, error) {
	var ids []int32
	for _, e := range evs {
		switch e.Type {
		case engine.EventCardRevealed:
			ids = append(ids, int32(e.Payload["card_id"].(int)))
		case engine.EventPlayerEliminated:
			for _, c := range e.Payload["cards"].([]map[string]any) {
				ids = append(ids, int32(c["card_id"].(int)))
			}
		}
	}
	var texts map[int]string
	if len(ids) > 0 {
		var err error
		if texts, err = loadCardTexts(ctx, tx, ids); err != nil {
			return nil, err
		}
	}
	roomID := ""
	if g.RoomID != nil {
		roomID = *g.RoomID
	}
	out := make([]room.Event, 0, len(evs))
	for _, e := range evs {
		w := wireEvent(e, version, now)
		w.RoomID, w.GameID = roomID, g.ID
		payload := w.Payload.(map[string]any)
		switch e.Type {
		case engine.EventCardRevealed:
			payload["text"] = texts[e.Payload["card_id"].(int)]
			delete(payload, "card_id")
		case engine.EventPlayerEliminated:
			// Elenenin tüm kartları metinle (kimlik ve denge etiketi yok).
			var cards []map[string]any
			for _, c := range e.Payload["cards"].([]map[string]any) {
				cards = append(cards, map[string]any{"category": c["category"], "text": texts[c["card_id"].(int)]})
			}
			payload["cards"] = cards
		}
		out = append(out, w)
	}
	return out, nil
}

// validUUID, kanonik 36 karakterlik UUID biçimidir (veritabanına geçersiz
// kimlik gönderilmez; room paketindekiyle aynı kural).
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
