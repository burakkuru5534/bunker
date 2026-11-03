package game

import (
	"context"
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
)

// norm, karşılaştırma için durumu kanonik yapar: saatler UTC, boş dilimler nil.
func norm(s engine.State) engine.State {
	s.PhaseStartedAt, s.Deadline, s.FinishedAt = s.PhaseStartedAt.UTC(), s.Deadline.UTC(), s.FinishedAt.UTC()
	s.Players = slices.Clone(s.Players)
	nilIfEmpty := func(v []engine.Vote) []engine.Vote {
		if len(v) == 0 {
			return nil
		}
		return v
	}
	s.Votes, s.TieVotes = nilIfEmpty(s.Votes), nilIfEmpty(s.TieVotes)
	if len(s.TieCandidates) == 0 {
		s.TieCandidates = nil
	}
	if len(s.PendingTally) == 0 {
		s.PendingTally = nil
	}
	var results []engine.RoundResult
	for _, r := range s.Results {
		if len(r.Tally) == 0 {
			r.Tally = nil
		}
		if len(r.LotteryCandidates) == 0 {
			r.LotteryCandidates = nil
		}
		results = append(results, r)
	}
	s.Results = results
	return s
}

// load, oyunu kendi transaction'ında okur.
func (a *api) load(gameID string) engine.State {
	a.t.Helper()
	var s engine.State
	if err := a.env.DB.Read(context.Background(), func(q persistence.Querier) error {
		var err error
		s, _, err = loadState(context.Background(), q, gameID, false)
		return err
	}); err != nil {
		a.t.Fatal(err)
	}
	return s
}

// advance, aşamanın süresini doldurur (zamanlayıcı ENG-007 gelene kadar
// testlerde elle): kilitli yükle → Timeout → farkı yaz.
func (a *api) advance(gameID string, rng engine.Rand) engine.State {
	a.t.Helper()
	var next engine.State
	err := a.env.DB.Tx(context.Background(), func(tx pgx.Tx) error {
		old, _, err := loadState(context.Background(), tx, gameID, true)
		if err != nil {
			return err
		}
		var evs []engine.Event
		next, evs, err = engine.New(engine.DefaultDurations).Timeout(old, old.Version, old.Deadline, rng)
		if err != nil {
			return err
		}
		_ = evs
		return saveState(context.Background(), tx, gameID, old, next, old.Deadline)
	})
	if err != nil {
		a.t.Fatal(err)
	}
	return next
}

// startGame, n kişilik hazır lobi kurup oyunu başlatır.
func (a *api) startGame(n, capacity int) (lobby, string) {
	a.t.Helper()
	l := a.lobby(n, capacity, true)
	r := a.start(l, l.tokens[0])
	if r.code != 201 {
		a.t.Fatalf("başlat: %d %s", r.code, r.raw)
	}
	return l, r.body["game_id"].(string)
}

// Tam oyunlar boyunca her geçişte: yazılan fark yeniden yüklenince motorun
// durumunun birebir aynısı olmalı (oylar, eşitlik, kura, eleme, bitiş).
func TestIntegrationStateRoundTrip(t *testing.T) {
	a := newAPI(t, true)
	for _, tc := range []struct{ n, capacity int }{{4, 1}, {6, 2}, {7, 1}, {10, 2}} {
		t.Run(fmt.Sprintf("%d_%d", tc.n, tc.capacity), func(t *testing.T) {
			a.t = t
			_, gameID := a.startGame(tc.n, tc.capacity)
			rng := rand.New(rand.NewPCG(uint64(tc.n), 7))
			s := a.load(gameID)
			if s.Phase != engine.PhaseScenarioIntro || s.Version != 1 || len(s.Players) != tc.n {
				t.Fatalf("ilk durum: %s v%d", s.Phase, s.Version)
			}
			ctx := context.Background()
			tieBreaks := 0
			for steps := 0; s.Phase != engine.PhaseFinished; steps++ {
				if steps > 300 {
					t.Fatal("oyun bitmedi")
				}
				// Oylar eylemle gelir (ENG-008): burada doğrudan yazılır.
				var votes []engine.Vote
				kind := ""
				switch s.Phase {
				case engine.PhaseVoting:
					kind = voteRegular
					for _, p := range s.Players {
						if p.Alive() && rng.IntN(4) > 0 {
							target := s.Players[rng.IntN(tc.n)]
							if target.Alive() && target.UserID != p.UserID {
								votes = append(votes, engine.Vote{Voter: p.UserID, Target: target.UserID})
							}
						}
					}
				case engine.PhaseTieBreak:
					tieBreaks++
					kind = voteTiebreak
					for _, sp := range s.Spectators() {
						if rng.IntN(3) > 0 {
							votes = append(votes, engine.Vote{Voter: sp, Target: s.TieCandidates[rng.IntN(len(s.TieCandidates))]})
						}
					}
				}
				for _, v := range votes {
					if err := a.env.DB.Tx(ctx, func(tx pgx.Tx) error { return insertVote(ctx, tx, gameID, s.Round, kind, v) }); err != nil {
						t.Fatal(err)
					}
				}
				if kind == voteRegular {
					s.Votes = append(s.Votes, votes...)
				} else if kind == voteTiebreak {
					s.TieVotes = append(s.TieVotes, votes...)
				}
				if got := a.load(gameID); !reflect.DeepEqual(norm(got), norm(s)) {
					t.Fatalf("oylar sonrası yüklenen durum farklı\nyüklenen: %+v\nbeklenen: %+v", norm(got), norm(s))
				}
				// Aynı rng ile motor sonucu ve kaydedilen sonuç karşılaştırılır.
				seed := rng.Uint64()
				want, _, err := engine.New(engine.DefaultDurations).Timeout(s, s.Version, s.Deadline, rand.New(rand.NewPCG(seed, 1)))
				if err != nil {
					t.Fatal(err)
				}
				a.advance(gameID, rand.New(rand.NewPCG(seed, 1)))
				got := a.load(gameID)
				if !reflect.DeepEqual(norm(got), norm(want)) {
					t.Fatalf("%s → %s geçişinden sonra yüklenen durum farklı\nyüklenen: %+v\nbeklenen: %+v",
						s.Phase, want.Phase, norm(got), norm(want))
				}
				s = got
			}
			// Kazananlar ve tur sonuçları satır düzeyinde.
			winners := a.scalar(`select count(*) from game_participants where game_id = $1 and is_winner`, gameID).(int64)
			rows := a.scalar(`select count(*) from round_results where game_id = $1`, gameID).(int64)
			if int(winners) != tc.capacity || int(rows) != tc.n-tc.capacity {
				t.Fatalf("kazanan %d, sonuç satırı %d", winners, rows)
			}
			if c := a.scalar(`select count(*) from player_cards pc join game_participants p using (game_id, user_id)
				where pc.game_id = $1 and p.status <> 'alive' and pc.revealed_round is null`, gameID).(int64); c != 0 {
				t.Fatalf("elenmişlerin %d kartı kapalı", c)
			}
			t.Logf("%d eşitlik bozma aşaması", tieBreaks)
		})
	}
}

// saveState, kilitlenmeden ilerlemiş bir sürümün üstüne yazmayı reddeder.
func TestIntegrationSaveStateVersionGuard(t *testing.T) {
	a := newAPI(t, true)
	_, gameID := a.startGame(4, 1)
	old := a.load(gameID)
	a.advance(gameID, rand.New(rand.NewPCG(1, 1)))
	next, _, err := engine.New(engine.DefaultDurations).Timeout(old, old.Version, old.Deadline, rand.New(rand.NewPCG(1, 1)))
	if err != nil {
		t.Fatal(err)
	}
	err = a.env.DB.Tx(context.Background(), func(tx pgx.Tx) error {
		return saveState(context.Background(), tx, gameID, old, next, time.Now())
	})
	if err == nil {
		t.Fatal("eski sürüm üzerine yazım kabul edildi")
	}
}
