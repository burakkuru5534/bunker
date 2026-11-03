package game

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
)

// Oyun durumu PostgreSQL'dedir (ADR-0004). loadState tam engine.State'i
// kurar; saveState eski ve yeni durum arasındaki farkı yazar. Motor durumu
// dışında tutulan tek şey katılımcı başına kalıcı alanlardır (takma ad
// users'tan, kart metni katalogdan okunur).

var errGameNotFound = errors.New("game: oyun bulunamadı")

// gameRow, durumun dışındaki oyun bilgisidir.
type gameRow struct {
	ID         string
	RoomID     *string
	ScenarioID string
}

// loadState, oyunu okur; lock ise games satırı FOR UPDATE ile kilitlenir
// (aynı oyundaki eylemler ve zamanlayıcı sıraya girer).
func loadState(ctx context.Context, q persistence.Querier, gameID string, lock bool) (engine.State, gameRow, error) {
	var (
		s        engine.State
		g        = gameRow{ID: gameID}
		phase    string
		turn     *int16
		deadline *time.Time
		finished *time.Time
		reason   *string
		round    int16
		total    int16
		capacity int16
	)
	sql := `select room_id::text, scenario_id, phase, round, total_rounds, capacity, turn_seat,
		phase_started_at, phase_deadline_at, state_version, finished_at, end_reason
		from games where id = $1`
	if lock {
		sql += ` for update`
	}
	err := q.QueryRow(ctx, sql, gameID).Scan(&g.RoomID, &g.ScenarioID, &phase, &round, &total, &capacity, &turn,
		&s.PhaseStartedAt, &deadline, &s.Version, &finished, &reason)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, g, errGameNotFound
	}
	if err != nil {
		return s, g, err
	}
	s.Phase, s.Round, s.TotalRounds, s.Capacity, s.TurnSeat = engine.Phase(phase), int(round), int(total), int(capacity), -1
	if turn != nil {
		s.TurnSeat = int(*turn)
	}
	if deadline != nil {
		s.Deadline = *deadline
	}
	if finished != nil {
		s.FinishedAt = *finished
	}
	if reason != nil {
		s.EndReason = engine.EndReason(*reason)
	}

	if err := loadPlayers(ctx, q, gameID, &s); err != nil {
		return s, g, err
	}
	if err := loadVotes(ctx, q, gameID, &s); err != nil {
		return s, g, err
	}
	if err := loadResults(ctx, q, gameID, &s); err != nil {
		return s, g, err
	}
	// Eşitlik verisi saklanmaz; turun normal oylarından türetilir
	// (sayım deterministiktir).
	if s.Phase == engine.PhaseTieBreak {
		s.PendingTally = s.Tally()
		for _, t := range s.PendingTally {
			if t.Votes == s.PendingTally[0].Votes {
				s.TieCandidates = append(s.TieCandidates, t.UserID)
			}
		}
	}
	return s, g, nil
}

func loadPlayers(ctx context.Context, q persistence.Querier, gameID string, s *engine.State) error {
	rows, err := q.Query(ctx, `select user_id::text, seat, status, eliminated_round, discussion_ready_round
		from game_participants where game_id = $1 order by seat`, gameID)
	if err != nil {
		return err
	}
	index := map[string]int{}
	for rows.Next() {
		var (
			p           = engine.Player{Cards: map[engine.Category]engine.Card{}}
			seat        int16
			status      string
			elim, ready *int16
		)
		if err := rows.Scan(&p.UserID, &seat, &status, &elim, &ready); err != nil {
			rows.Close()
			return err
		}
		p.Seat, p.Status = int(seat), engine.PlayerStatus(status)
		if elim != nil {
			p.EliminatedRound = int(*elim)
		}
		if ready != nil {
			p.ReadyRound = int(*ready)
		}
		if p.Seat != len(s.Players) {
			rows.Close()
			return fmt.Errorf("game: %s oyununda koltuk boşluğu (%d)", gameID, p.Seat)
		}
		index[p.UserID] = len(s.Players)
		s.Players = append(s.Players, p)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows, err = q.Query(ctx, `select user_id::text, category, card_id, revealed_round, reveal_method
		from player_cards where game_id = $1`, gameID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			user, category string
			c              engine.Card
			revealed       *int16
			method         *string
		)
		if err := rows.Scan(&user, &category, &c.ID, &revealed, &method); err != nil {
			return err
		}
		if revealed != nil {
			c.RevealedRound, c.Method = int(*revealed), engine.RevealMethod(*method)
		}
		s.Players[index[user]].Cards[engine.Category(category)] = c
	}
	return rows.Err()
}

// loadVotes, yalnızca bu turun oylarını okur (önceki turlar round_results'ta).
func loadVotes(ctx context.Context, q persistence.Querier, gameID string, s *engine.State) error {
	if s.Round < 1 {
		return nil
	}
	rows, err := q.Query(ctx, `select kind, voter_id::text, target_id::text from votes
		where game_id = $1 and round = $2 order by created_at, voter_id`, gameID, s.Round)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			kind string
			v    engine.Vote
		)
		if err := rows.Scan(&kind, &v.Voter, &v.Target); err != nil {
			return err
		}
		if kind == voteTiebreak {
			s.TieVotes = append(s.TieVotes, v)
		} else {
			s.Votes = append(s.Votes, v)
		}
	}
	return rows.Err()
}

func loadResults(ctx context.Context, q persistence.Querier, gameID string, s *engine.State) error {
	rows, err := q.Query(ctx, `select round, eliminated_id::text, method, tally, lottery_candidates::text[]
		from round_results where game_id = $1 order by round`, gameID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			r          engine.RoundResult
			round      int16
			eliminated *string
			method     string
		)
		if err := rows.Scan(&round, &eliminated, &method, &r.Tally, &r.LotteryCandidates); err != nil {
			return err
		}
		r.Round, r.Method = int(round), engine.EliminationMethod(method)
		if eliminated != nil {
			r.EliminatedID = *eliminated
		}
		s.Results = append(s.Results, r)
	}
	return rows.Err()
}

const (
	voteRegular  = "regular"
	voteTiebreak = "tiebreak"
)

// saveState, old → next farkını yazar. old, aynı transaction'da kilitli
// okunmuş durumdur; next, motorun döndürdüğüdür. now, kart açılış zamanı.
func saveState(ctx context.Context, tx pgx.Tx, gameID string, old, next engine.State, now time.Time) error {
	if next.Version <= old.Version {
		return fmt.Errorf("game: sürüm ilerlemedi (%d → %d)", old.Version, next.Version)
	}
	var finished, reason any
	if next.Phase == engine.PhaseFinished {
		finished, reason = next.FinishedAt, string(next.EndReason)
	}
	tag, err := tx.Exec(ctx, `update games set phase = $2, round = $3, turn_seat = $4, phase_started_at = $5,
		phase_deadline_at = $6, state_version = $7, finished_at = $8, end_reason = $9
		where id = $1 and state_version = $10`,
		gameID, string(next.Phase), next.Round, turnSeat(next), next.PhaseStartedAt, deadline(next), next.Version,
		finished, reason, old.Version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("game: %s sürüm %d değil (kilit dışı yazım?)", gameID, old.Version)
	}

	var (
		cardUsers, cardCats, cardMethods []string
		cardRounds                       []int16
	)
	for i, p := range next.Players {
		before := old.Players[i]
		winner := next.Phase == engine.PhaseFinished && p.Alive()
		if p.Status != before.Status || p.EliminatedRound != before.EliminatedRound ||
			p.ReadyRound != before.ReadyRound || winner {
			if _, err := tx.Exec(ctx, `update game_participants set status = $3, eliminated_round = $4,
				discussion_ready_round = $5, is_winner = $6 where game_id = $1 and user_id = $2`,
				gameID, p.UserID, string(p.Status), eliminatedRound(p), nullIfZero(p.ReadyRound), winner); err != nil {
				return err
			}
		}
		for _, c := range engine.Categories {
			if card := p.Cards[c]; card.Revealed() && !before.Cards[c].Revealed() {
				cardUsers, cardCats = append(cardUsers, p.UserID), append(cardCats, string(c))
				cardRounds, cardMethods = append(cardRounds, int16(card.RevealedRound)), append(cardMethods, string(card.Method))
			}
		}
	}
	if len(cardUsers) > 0 {
		tag, err := tx.Exec(ctx, `update player_cards pc set revealed_round = t.r, reveal_method = t.m, revealed_at = $2
			from unnest($3::uuid[], $4::text[], $5::smallint[], $6::text[]) as t(u, c, r, m)
			where pc.game_id = $1 and pc.user_id = t.u and pc.category = t.c and pc.revealed_round is null`,
			gameID, now, cardUsers, cardCats, cardRounds, cardMethods)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(cardUsers) {
			return fmt.Errorf("game: %d kart açılacaktı, %d açıldı", len(cardUsers), tag.RowsAffected())
		}
	}

	// Oylar burada yazılmaz: oy eylemi kendi oyunu açıkça ekler (motor bir
	// eşitlik oyunu aynı eylemde sonuçlandırıp listeyi temizleyebilir; farktan
	// yazmak o oyu kaybettirirdi). Bkz. insertVote.

	for _, r := range next.Results[len(old.Results):] {
		var lottery any
		if r.LotteryCandidates != nil {
			lottery = r.LotteryCandidates
		}
		tally := r.Tally
		if tally == nil {
			tally = []engine.TallyEntry{}
		}
		if _, err := tx.Exec(ctx, `insert into round_results (game_id, round, eliminated_id, method, tally, lottery_candidates)
			values ($1, $2, $3, $4, $5, $6::uuid[])`,
			gameID, r.Round, nullIfEmpty(r.EliminatedID), string(r.Method), tally, lottery); err != nil {
			return err
		}
	}
	return nil
}

// insertVote, tek bir oyu yazar (oy eylemleri, ENG-008/009). Birincil
// anahtar (oyun, tur, tür, oy veren) ikinci oyu engeller.
func insertVote(ctx context.Context, tx pgx.Tx, gameID string, round int, kind string, v engine.Vote) error {
	_, err := tx.Exec(ctx, `insert into votes (game_id, round, kind, voter_id, target_id) values ($1, $2, $3, $4, $5)`,
		gameID, round, kind, v.Voter, v.Target)
	return err
}

func eliminatedRound(p engine.Player) any {
	if p.Alive() {
		return nil
	}
	return p.EliminatedRound
}

func nullIfZero(v int) any {
	if v == 0 {
		return nil
	}
	return v
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
