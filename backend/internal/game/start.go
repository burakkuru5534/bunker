// Package game, oyun modülüdür: saf motoru (engine) veritabanı ve oda
// katmanına bağlar.
package game

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
	"github.com/burakkuru5534/bunker/backend/internal/room"
)

// Olay türleri (api.md §5); motorun ürettikleri engine'dedir.
const (
	EventStarted   = "game.started"
	EventYourCards = "game.your_cards"
)

// Starter, room.GameStarter uygulamasıdır (ENG-003).
type Starter struct {
	engine engine.Engine
	rng    engine.Rand
	now    func() time.Time
	logger *slog.Logger
}

// NewStarter, gerçek süreler, kriptografik rastgelelik ve sistem saatiyle kurar.
func NewStarter(logger *slog.Logger) *Starter {
	if logger == nil {
		logger = slog.Default()
	}
	return &Starter{engine: engine.New(engine.DefaultDurations), rng: engine.CryptoRand{}, now: time.Now, logger: logger}
}

// StartGame, odanın transaction'ında: katalogu okur, dağıtır (koltuk,
// senaryo, kartlar), oyunu DEALING'den SCENARIO_INTRO'ya geçirir ve yazar.
// Katalog yetersizse içerik hatasıdır: loglanır, istemciye INTERNAL gider.
func (s *Starter) StartGame(ctx context.Context, tx pgx.Tx, in room.GameStart) (room.GameStarted, error) {
	cat, err := loadCatalog(ctx, tx)
	if err != nil {
		return room.GameStarted{}, err
	}
	ids := make([]string, len(in.Players))
	nick := map[string]string{}
	for i, m := range in.Players {
		ids[i], nick[m.UserID] = m.UserID, m.Nickname
	}
	deal, err := engine.DealGame(ids, cat, s.rng)
	if errors.Is(err, engine.ErrCatalog) {
		s.logger.Error("kart kataloğu yetersiz; oyun başlatılamadı", "room_id", in.RoomID, "players", len(ids), "error", err)
	}
	if err != nil {
		return room.GameStarted{}, err
	}
	st, err := s.engine.NewGame(deal.Players, in.Capacity)
	if err != nil {
		return room.GameStarted{}, err
	}
	now := s.now().UTC().Truncate(time.Microsecond) // PostgreSQL hassasiyeti
	st, engineEvents, err := s.engine.Begin(st, now)
	if err != nil {
		return room.GameStarted{}, err
	}
	gameID, err := insertGame(ctx, tx, in.RoomID, deal.ScenarioID, st)
	if err != nil {
		return room.GameStarted{}, err
	}
	scenario, err := loadScenario(ctx, tx, deal.ScenarioID)
	if err != nil {
		return room.GameStarted{}, err
	}
	var cardIDs []int32
	for _, p := range st.Players {
		for _, c := range engine.Categories {
			cardIDs = append(cardIDs, int32(p.Cards[c].ID))
		}
	}
	texts, err := loadCardTexts(ctx, tx, cardIDs)
	if err != nil {
		return room.GameStarted{}, err
	}

	// Olaylar: herkese game.started, her oyuncuya yalnızca kendi kartları,
	// sonra motorun aşama olayı. Hepsi oyunun state_version'ını taşır.
	v := st.Version
	var participants []map[string]any
	var order []string
	for _, p := range st.Players {
		participants = append(participants, map[string]any{"user_id": p.UserID, "nickname": nick[p.UserID], "seat": p.Seat})
	}
	first := st
	first.Round = 1 // tur 1 açıklama sırası (RevealOrder turdan hesaplar)
	for _, seat := range first.RevealOrder() {
		order = append(order, st.Players[seat].UserID)
	}
	events := []room.Event{{Type: EventStarted, Version: v, Payload: map[string]any{
		"game_id": gameID, "scenario": scenario, "capacity": st.Capacity, "total_rounds": st.TotalRounds,
		"participants": participants, "reveal_order": order,
	}}}
	for _, p := range st.Players {
		var cards []map[string]any
		for _, c := range engine.Categories {
			text, ok := texts[p.Cards[c].ID]
			if !ok {
				return room.GameStarted{}, fmt.Errorf("game: kart %d metni yok", p.Cards[c].ID)
			}
			cards = append(cards, map[string]any{"category": string(c), "text": text, "revealed": false})
		}
		events = append(events, room.Event{Type: EventYourCards, Version: v, To: p.UserID, Payload: map[string]any{"cards": cards}})
	}
	for _, e := range engineEvents {
		events = append(events, wireEvent(e, v, now))
	}
	for i := range events {
		events[i].GameID = gameID
	}
	return room.GameStarted{GameID: gameID, Events: events}, nil
}

// wireEvent, motor olayını zarf olayına çevirir: aşama adları istemcide
// büyük harflidir (api.md: "REVEAL"), phase_changed'e server_time eklenir.
func wireEvent(e engine.Event, version int64, now time.Time) room.Event {
	payload := map[string]any{}
	for k, v := range e.Payload {
		payload[k] = v
	}
	if e.Type == engine.EventPhaseChanged {
		payload["phase"] = WirePhase(engine.Phase(payload["phase"].(string)))
		payload["server_time"] = now
	}
	return room.Event{Type: e.Type, Version: version, To: e.To, Payload: payload}
}

// WirePhase, aşamanın istemci adıdır (ör. scenario_intro → SCENARIO_INTRO).
func WirePhase(p engine.Phase) string { return strings.ToUpper(string(p)) }
