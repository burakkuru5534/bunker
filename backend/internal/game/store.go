package game

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
)

// insertGame, yeni oyunu, katılımcıları ve dağıtılan kartları yazar
// (üç ifade; satırlar unnest ile toplu). Durum s, Begin sonrası haldir.
func insertGame(ctx context.Context, tx pgx.Tx, roomID, scenarioID string, s engine.State) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `insert into games
		(room_id, scenario_id, capacity, total_rounds, phase, round, turn_seat,
		 phase_started_at, phase_deadline_at, state_version, started_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $8)
		returning id`,
		roomID, scenarioID, s.Capacity, s.TotalRounds, string(s.Phase), s.Round, turnSeat(s),
		s.PhaseStartedAt, deadline(s), s.Version).Scan(&id)
	if err != nil {
		return "", err
	}
	var (
		users      []string
		seats      []int16
		cardUsers  []string
		categories []string
		cardIDs    []int32
	)
	for _, p := range s.Players {
		users, seats = append(users, p.UserID), append(seats, int16(p.Seat))
		for _, c := range engine.Categories {
			cardUsers, categories = append(cardUsers, p.UserID), append(categories, string(c))
			cardIDs = append(cardIDs, int32(p.Cards[c].ID))
		}
	}
	if _, err := tx.Exec(ctx, `insert into game_participants (game_id, user_id, seat)
		select $1, u, s from unnest($2::uuid[], $3::smallint[]) as t(u, s)`, id, users, seats); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `insert into player_cards (game_id, user_id, category, card_id)
		select $1, u, c, k from unnest($2::uuid[], $3::text[], $4::int[]) as t(u, c, k)`,
		id, cardUsers, categories, cardIDs); err != nil {
		return "", err
	}
	return id, nil
}

func turnSeat(s engine.State) any {
	if s.Phase != engine.PhaseReveal {
		return nil
	}
	return s.TurnSeat
}

func deadline(s engine.State) any {
	if s.Phase == engine.PhaseFinished {
		return nil
	}
	return s.Deadline
}
