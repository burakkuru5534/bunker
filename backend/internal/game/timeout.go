package game

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
	"github.com/burakkuru5534/bunker/backend/internal/room"
)

// ActionPhaseTimeout, süre dolumu sistem eylemidir (game_actions.type;
// actor_id boş).
const ActionPhaseTimeout = "phase_timeout"

// maxCatchUp, tek seferde işlenecek en fazla ardışık süre dolumu. Motor yeni
// aşamayı "şimdi"den başlattığı için normalde tek adım yeter; sınır yalnızca
// beklenmedik bir döngüye karşı koruma.
const maxCatchUp = 20

// clock, veritabanı hassasiyetinde (mikrosaniye) UTC saattir.
func (s *Service) clock() time.Time { return s.now().UTC().Truncate(time.Microsecond) }

// overdue: oyun bitmemiş ve aşamanın süresi dolmuş.
func overdue(st engine.State, now time.Time) bool {
	return st.Phase != engine.PhaseFinished && !st.Deadline.IsZero() && !now.Before(st.Deadline)
}

// applyDue, süresi dolmuş aşamaları sırayla işler: her adım ayrı sürüm,
// fark yazımı ve phase_timeout kaydıdır. Olaylar her adımın sürümüyle döner.
func (s *Service) applyDue(ctx context.Context, tx pgx.Tx, g gameRow, st engine.State, now time.Time) (engine.State, []room.Event, error) {
	var out []room.Event
	for i := 0; overdue(st, now); i++ {
		if i == maxCatchUp {
			return st, nil, fmt.Errorf("game: %s için %d ardışık süre dolumu", g.ID, maxCatchUp)
		}
		next, evs, err := s.engine.Timeout(st, st.Version, now, s.rng)
		if err != nil {
			return st, nil, err
		}
		if err := saveState(ctx, tx, g.ID, st, next, now); err != nil {
			return st, nil, err
		}
		if _, err := tx.Exec(ctx, `insert into game_actions (game_id, actor_id, action_id, type, payload, state_version)
			values ($1, null, $2, $3, $4, $5)`, g.ID, newUUID(), ActionPhaseTimeout,
			map[string]any{"phase": string(st.Phase), "round": st.Round}, next.Version); err != nil {
			return st, nil, err
		}
		wired, err := s.wireEvents(ctx, tx, g, evs, next.Version, now)
		if err != nil {
			return st, nil, err
		}
		out = append(out, wired...)
		st = next
	}
	return st, out, nil
}

// TimeoutResult, zamanlayıcıya dönen sonuçtur: oyunun güncel sürümü ve
// bitiş zamanı (yeniden planlama için).
type TimeoutResult struct {
	Advanced bool
	Finished bool
	Version  int64
	Deadline time.Time
}

// Timeout, zamanlayıcının "aşama süresi doldu" eylemidir (ADR-0005). Oyun
// satırı kilitlenir; sürüm beklenenden farklıysa (aşama zaten ilerledi) veya
// süre henüz dolmadıysa hiçbir şey yazılmaz ve güncel sürüm/bitiş döner.
func (s *Service) Timeout(ctx context.Context, gameID string, expected int64) (TimeoutResult, error) {
	var (
		res    TimeoutResult
		events []room.Event
	)
	err := s.db.Tx(ctx, func(tx pgx.Tx) error {
		old, g, err := loadState(ctx, tx, gameID, true)
		if err != nil {
			return err
		}
		res = TimeoutResult{Finished: old.Phase == engine.PhaseFinished, Version: old.Version, Deadline: old.Deadline}
		now := s.clock()
		if old.Version != expected || !overdue(old, now) {
			return nil
		}
		next, evs, err := s.applyDue(ctx, tx, g, old, now)
		if err != nil {
			return err
		}
		events = evs
		res = TimeoutResult{Advanced: true, Finished: next.Phase == engine.PhaseFinished, Version: next.Version, Deadline: next.Deadline}
		return nil
	})
	if err != nil {
		return TimeoutResult{}, err
	}
	for _, e := range events {
		s.publisher.Publish(e)
	}
	return res, nil
}

// Deadline, aktif bir oyunun planlanacak süre dolumudur.
type Deadline struct {
	GameID  string
	Version int64
	At      time.Time
}

// ActiveDeadlines, bitmemiş tüm oyunların bitiş zamanlarıdır (açılışta
// kurtarma; games_active_deadline_idx).
func (s *Service) ActiveDeadlines(ctx context.Context) ([]Deadline, error) {
	var out []Deadline
	err := s.db.Read(ctx, func(q persistence.Querier) error {
		rows, err := q.Query(ctx, `select id::text, state_version, phase_deadline_at from games
			where phase <> 'finished' order by phase_deadline_at`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var d Deadline
			if err := rows.Scan(&d.GameID, &d.Version, &d.At); err != nil {
				return err
			}
			out = append(out, d)
		}
		return rows.Err()
	})
	return out, err
}

// newUUID, rastgele (v4) UUID'dir.
func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("game: crypto/rand: %v", err))
	}
	b[6], b[8] = b[6]&0x0f|0x40, b[8]&0x3f|0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// errNotFound, zamanlayıcının oyunu bırakması gerektiğini söyler.
func isGameNotFound(err error) bool { return errors.Is(err, errGameNotFound) }
