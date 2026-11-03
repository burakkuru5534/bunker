package game

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/apperr"
	"github.com/burakkuru5534/bunker/backend/internal/auth"
	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
)

// Result, biten oyunun herkese açık sonucudur (api.md GET /result;
// game.finished olayının payload'ı). Oyun bittiği için tüm kartlar açıktır.
type Result struct {
	GameID     string         `json:"game_id"`
	FinishedAt time.Time      `json:"finished_at"`
	EndReason  string         `json:"end_reason"`
	Scenario   ResultScenario `json:"scenario"`
	Capacity   int            `json:"capacity"`
	Players    []ResultPlayer `json:"players"`
	Rounds     []ResultRound  `json:"rounds"`
}

// ResultScenario, sonuçtaki kısa senaryo bilgisidir.
type ResultScenario struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// ResultPlayer, katılımcının sonucu (koltuk sırasıyla).
type ResultPlayer struct {
	UserID          string       `json:"user_id"`
	Nickname        string       `json:"nickname"`
	IsWinner        bool         `json:"is_winner"`
	Status          string       `json:"status"`
	EliminatedRound *int         `json:"eliminated_round"`
	Cards           []ResultCard `json:"cards"`
}

// ResultCard, kategori ve metin (kimlik ve denge etiketi yok).
type ResultCard struct {
	Category string `json:"category"`
	Text     string `json:"text"`
}

// ResultRound, turun sonucu.
type ResultRound struct {
	Round            int                 `json:"round"`
	EliminatedUserID *string             `json:"eliminated_user_id"`
	Method           string              `json:"method"`
	Tally            []engine.TallyEntry `json:"tally"`
}

var errNotFinished = errors.New("game: oyun bitmedi")

// loadResult, sonucu okur. Takma adlar users'tan gelir (hesabı silinenin
// adı "Silinmiş oyuncu" olarak görünür, SAFE-009).
func loadResult(ctx context.Context, q persistence.Querier, gameID string) (Result, error) {
	r := Result{GameID: gameID, Players: []ResultPlayer{}, Rounds: []ResultRound{}}
	var (
		finished *time.Time
		reason   *string
		capacity int16
	)
	err := q.QueryRow(ctx, `select g.finished_at, g.end_reason, g.capacity, s.id, s.title
		from games g join scenarios s on s.id = g.scenario_id where g.id = $1`, gameID).
		Scan(&finished, &reason, &capacity, &r.Scenario.ID, &r.Scenario.Title)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, errGameNotFound
	}
	if err != nil {
		return r, err
	}
	if finished == nil {
		return r, errNotFinished
	}
	r.FinishedAt, r.EndReason, r.Capacity = finished.UTC(), *reason, int(capacity)

	rows, err := q.Query(ctx, `select p.user_id::text, u.nickname, p.is_winner, p.status, p.eliminated_round
		from game_participants p join users u on u.id = p.user_id
		where p.game_id = $1 order by p.seat`, gameID)
	if err != nil {
		return r, err
	}
	index := map[string]int{}
	for rows.Next() {
		var (
			p    = ResultPlayer{Cards: []ResultCard{}}
			elim *int16
		)
		if err := rows.Scan(&p.UserID, &p.Nickname, &p.IsWinner, &p.Status, &elim); err != nil {
			rows.Close()
			return r, err
		}
		if elim != nil {
			v := int(*elim)
			p.EliminatedRound = &v
		}
		index[p.UserID] = len(r.Players)
		r.Players = append(r.Players, p)
	}
	if err := rows.Err(); err != nil {
		return r, err
	}

	// Kartlar kategori sırasıyla (meslek → gizli sır).
	rows, err = q.Query(ctx, `select pc.user_id::text, pc.category, d.text
		from player_cards pc join card_definitions d on d.id = pc.card_id
		where pc.game_id = $1
		order by array_position(array['profession','health','hobby','item','secret'], pc.category)`, gameID)
	if err != nil {
		return r, err
	}
	for rows.Next() {
		var (
			user string
			c    ResultCard
		)
		if err := rows.Scan(&user, &c.Category, &c.Text); err != nil {
			rows.Close()
			return r, err
		}
		p := &r.Players[index[user]]
		p.Cards = append(p.Cards, c)
	}
	if err := rows.Err(); err != nil {
		return r, err
	}

	rows, err = q.Query(ctx, `select round, eliminated_id::text, method, tally from round_results
		where game_id = $1 order by round`, gameID)
	if err != nil {
		return r, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			rr    ResultRound
			round int16
		)
		if err := rows.Scan(&round, &rr.EliminatedUserID, &rr.Method, &rr.Tally); err != nil {
			return r, err
		}
		rr.Round = int(round)
		if rr.Tally == nil {
			rr.Tally = []engine.TallyEntry{}
		}
		r.Rounds = append(r.Rounds, rr)
	}
	return r, rows.Err()
}

// Result, biten oyunun sonucunu döndürür. Yetki: oyunun katılımcısı (değilse
// — oyun yoksa veya kimlik bozuksa da — NOT_FOUND). Oyun sürüyorsa
// INVALID_PHASE (kapalı kartlar sızmasın).
func (s *Service) Result(ctx context.Context, p auth.Principal, gameID string) (Result, error) {
	notFound := apperr.New(apperr.NotFound, "oyun bulunamadı")
	if !validUUID(gameID) {
		return Result{}, notFound
	}
	var r Result
	err := s.db.Read(ctx, func(q persistence.Querier) error {
		var member bool
		if err := q.QueryRow(ctx, `select exists (select 1 from game_participants where game_id = $1 and user_id = $2)`,
			gameID, p.UserID).Scan(&member); err != nil {
			return err
		}
		if !member {
			return notFound
		}
		var err error
		r, err = loadResult(ctx, q, gameID)
		switch {
		case errors.Is(err, errGameNotFound):
			return notFound
		case errors.Is(err, errNotFinished):
			return apperr.New(apperr.InvalidPhase, "oyun henüz bitmedi")
		}
		return err
	})
	return r, err
}
