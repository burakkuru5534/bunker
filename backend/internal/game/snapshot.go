package game

import (
	"context"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
	"github.com/burakkuru5534/bunker/backend/internal/room"
)

// GameView, session.snapshot.game alanıdır (api.md §5): oyunun belirli bir
// kullanıcıya göre süzülmüş hali (ENG-013, ENG-015). Görünürlük
// (game-design §6 "Gizlilik"):
//   - Kartın sahibi kendi 5 kartını her zaman görür (MyCards).
//   - Diğer oyuncuların yalnızca açılmış kartları görünür; elenen/ayrılanın
//     kartları elenince açıldığı için izleyiciler onları da görür.
//   - Oyun bitince tüm kartlar görünür.
//   - Oyların hedefleri hiç görünmez; yalnızca sayaçlar.
type GameView struct {
	ID              string            `json:"id"`
	StateVersion    int64             `json:"state_version"`
	Phase           string            `json:"phase"`
	Round           int               `json:"round"`
	TotalRounds     int               `json:"total_rounds"`
	Capacity        int               `json:"capacity"`
	PhaseDeadlineAt *time.Time        `json:"phase_deadline_at"`
	Scenario        Scenario          `json:"scenario"`
	TurnUserID      *string           `json:"turn_user_id"`
	Participants    []ViewParticipant `json:"participants"`
	MyCards         []MyCard          `json:"my_cards"`
	MyActionState   MyActionState     `json:"my_action_state"`
	// DiscussionReadyCount: bu turda "Hazırım" diyen hayattakiler (toplam =
	// hayattaki katılımcı sayısı).
	DiscussionReadyCount int            `json:"discussion_ready_count"`
	VoteProgress         Progress       `json:"vote_progress"`
	Tiebreak             *TiebreakView  `json:"tiebreak"`
	LastRoundResult      map[string]any `json:"last_round_result"`
}

// ViewParticipant, katılımcının herkese açık hali.
type ViewParticipant struct {
	UserID        string       `json:"user_id"`
	Nickname      string       `json:"nickname"`
	Seat          int          `json:"seat"`
	Status        string       `json:"status"`
	Connected     bool         `json:"connected"`
	RevealedCards []ResultCard `json:"revealed_cards"`
}

// MyCard, izleyicinin kendi kartı.
type MyCard struct {
	Category string `json:"category"`
	Text     string `json:"text"`
	Revealed bool   `json:"revealed"`
}

// MyActionState, izleyicinin bu aşamada yaptıkları.
type MyActionState struct {
	Voted           bool `json:"voted"`
	DiscussionReady bool `json:"discussion_ready"`
	TiebreakVoted   bool `json:"tiebreak_voted"`
}

// Progress, "x/y oy verildi" sayacı.
type Progress struct {
	Cast  int `json:"cast"`
	Total int `json:"total"`
}

// TiebreakView, eşitlik bozma aşamasındaki adaylar ve ilerleme.
type TiebreakView struct {
	Candidates []string `json:"candidates"`
	Progress   Progress `json:"progress"`
}

// WithPresence, katılımcıların "bağlı mı" bilgisini bağlar (WebSocket hub).
func (s *Service) WithPresence(p room.Presence) *Service {
	s.presence = p
	return s
}

// GameSnapshot, oyunun userID'ye göre süzülmüş görünümüdür
// (realtime.GameSnapshotter). Yetki çağıranda (oda üyeliği) denetlenir.
func (s *Service) GameSnapshot(ctx context.Context, userID, gameID string) (any, error) {
	var view GameView
	err := s.db.Read(ctx, func(q persistence.Querier) error {
		st, g, err := loadState(ctx, q, gameID, false)
		if err != nil {
			return err
		}
		view, err = s.buildView(ctx, q, st, g, userID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return view, nil
}

func (s *Service) buildView(ctx context.Context, q persistence.Querier, st engine.State, g gameRow, viewer string) (GameView, error) {
	scenario, err := loadScenario(ctx, q, g.ScenarioID)
	if err != nil {
		return GameView{}, err
	}
	nicks, err := loadNicknames(ctx, q, g.ID)
	if err != nil {
		return GameView{}, err
	}
	var ids []int32
	for _, p := range st.Players {
		for _, c := range engine.Categories {
			ids = append(ids, int32(p.Cards[c].ID))
		}
	}
	texts, err := loadCardTexts(ctx, q, ids)
	if err != nil {
		return GameView{}, err
	}

	finished := st.Phase == engine.PhaseFinished
	alive := st.AliveCount()
	v := GameView{
		ID: g.ID, StateVersion: st.Version, Phase: WirePhase(st.Phase), Round: st.Round,
		TotalRounds: st.TotalRounds, Capacity: st.Capacity, Scenario: scenario,
		Participants: []ViewParticipant{}, MyCards: []MyCard{},
		DiscussionReadyCount: st.ReadyCount(),
		VoteProgress:         Progress{Cast: st.VotesCast(), Total: alive},
	}
	if !finished {
		d := st.Deadline.UTC()
		v.PhaseDeadlineAt = &d
	}
	if st.Phase == engine.PhaseReveal && st.TurnSeat >= 0 {
		turn := st.Players[st.TurnSeat].UserID
		v.TurnUserID = &turn
	}
	for _, p := range st.Players {
		vp := ViewParticipant{UserID: p.UserID, Nickname: nicks[p.UserID], Seat: p.Seat, Status: string(p.Status),
			RevealedCards: []ResultCard{}}
		if s.presence != nil {
			vp.Connected = s.presence.Connected(p.UserID)
		}
		for _, c := range engine.Categories {
			card := p.Cards[c]
			// Tek görünürlük kuralı: açılmış kart veya bitmiş oyun.
			if card.Revealed() || finished {
				vp.RevealedCards = append(vp.RevealedCards, ResultCard{Category: string(c), Text: texts[card.ID]})
			}
			if p.UserID == viewer {
				v.MyCards = append(v.MyCards, MyCard{Category: string(c), Text: texts[card.ID], Revealed: card.Revealed()})
			}
		}
		v.Participants = append(v.Participants, vp)
	}
	if me := st.Player(viewer); me != nil {
		v.MyActionState.DiscussionReady = st.Phase == engine.PhaseDiscussion && me.ReadyRound == st.Round
	}
	for _, vote := range st.Votes {
		if vote.Voter == viewer {
			v.MyActionState.Voted = true
		}
	}
	for _, vote := range st.TieVotes {
		if vote.Voter == viewer {
			v.MyActionState.TiebreakVoted = true
		}
	}
	if st.Phase == engine.PhaseTieBreak {
		v.Tiebreak = &TiebreakView{Candidates: append([]string{}, st.TieCandidates...),
			Progress: Progress{Cast: st.TieVotesCast(), Total: len(st.Spectators())}}
	}
	if n := len(st.Results); n > 0 {
		r := st.Results[n-1]
		cards := []map[string]any{}
		if p := st.Player(r.EliminatedID); p != nil {
			for _, c := range engine.Categories {
				cards = append(cards, map[string]any{"category": string(c), "text": texts[p.Cards[c].ID]})
			}
		}
		tally := r.Tally
		if tally == nil {
			tally = []engine.TallyEntry{}
		}
		v.LastRoundResult = map[string]any{"round": r.Round, "user_id": r.EliminatedID, "method": string(r.Method),
			"tally": tally, "lottery_candidates": r.LotteryCandidates, "cards": cards}
	}
	return v, nil
}

// loadNicknames, katılımcıların güncel takma adları (silinmiş hesap
// "Silinmiş oyuncu").
func loadNicknames(ctx context.Context, q persistence.Querier, gameID string) (map[string]string, error) {
	rows, err := q.Query(ctx, `select p.user_id::text, u.nickname from game_participants p
		join users u on u.id = p.user_id where p.game_id = $1`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, nick string
		if err := rows.Scan(&id, &nick); err != nil {
			return nil, err
		}
		out[id] = nick
	}
	return out, rows.Err()
}
