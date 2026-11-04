// Package testbot, geliştirme ve testler için otomatik oyunculardır: eksik
// oyuncuların yerine odaya katılır, hazır olur, kart açar, oy verir.
// Yalnızca herkese açık API'yi (REST + WebSocket) kullanır; sunucuya ayrıcalıklı
// erişimi yoktur. Üretimde çalıştırılmaz (cmd/testbot yerel/test adreslerine
// sınırlıdır).
package testbot

import (
	"encoding/json"
	"math/rand/v2"
	"slices"
)

// Kategoriler açılma sırasında (api.md; 1. turda yalnızca meslek).
var categories = []string{"profession", "health", "hobby", "item", "secret"}

// Action, botun yapmak istediği bir istektir.
type Action struct {
	Kind    string // ready | start | game
	Type    string // oyun eylemi türü (Kind == game)
	Payload map[string]any
	key     string // aynı eylemi tekrar üretmemek için
}

// Brain, sunucu olaylarından botun bilmesi gerekenleri tutar ve sıradaki
// eylemleri seçer. Saf ve eşzamanlılıksızdır (Bot tek goroutine'den çağırır).
// Kural denetimi sunucudadır; Brain yalnızca makul eylem üretir, reddedilen
// eylem botu bozmaz.
type Brain struct {
	Me      string
	IsHost  bool // AutoStart için: odanın sahibi miyim
	StartAt int  // 0: başlatmaz; ≥4: bu kadar hazır oyuncu olunca başlatır

	rng *rand.Rand

	synced  bool
	status  string // lobby | in_game | finished
	members map[string]member
	order   []string // katılma sırası

	gameID  string
	phase   string
	round   int
	turn    string
	alive   map[string]bool
	seats   []string // koltuk sırası
	myCards map[string]bool
	tbCands []string

	done map[string]bool
}

type member struct {
	Role  string `json:"role"`
	Ready bool   `json:"is_ready"`
}

// NewBrain, kendi kullanıcı kimliğiyle kurar; rng nil ise rastgele.
func NewBrain(me string, rng *rand.Rand) *Brain {
	if rng == nil {
		rng = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	}
	b := &Brain{Me: me, rng: rng}
	b.resetGame()
	b.members = map[string]member{}
	return b
}

func (b *Brain) resetGame() {
	b.gameID, b.phase, b.round, b.turn = "", "", 0, ""
	b.alive = map[string]bool{}
	b.seats = nil
	b.myCards = map[string]bool{}
	b.tbCands = nil
	b.done = map[string]bool{}
}

// GameID, sürmekte olan (veya biten) oyunun kimliğidir.
func (b *Brain) GameID() string { return b.gameID }

// Phase, oyunun aşamasıdır (oyun yoksa boş).
func (b *Brain) Phase() string { return b.phase }

// Status, odanın durumudur.
func (b *Brain) Status() string { return b.status }

// Handle, bir sunucu olayını işler. Snapshot'tan önceki olaylar yok sayılır.
func (b *Brain) Handle(typ string, payload json.RawMessage) {
	if typ == "session.snapshot" {
		b.snapshot(payload)
		return
	}
	if !b.synced {
		return
	}
	var p map[string]json.RawMessage
	if json.Unmarshal(payload, &p) != nil {
		return
	}
	str := func(k string) string {
		var s string
		_ = json.Unmarshal(p[k], &s)
		return s
	}
	switch typ {
	case "room.member_joined":
		var m struct {
			Member struct {
				UserID string `json:"user_id"`
				member
			} `json:"member"`
		}
		if json.Unmarshal(payload, &m) == nil && m.Member.UserID != "" {
			if _, ok := b.members[m.Member.UserID]; !ok {
				b.order = append(b.order, m.Member.UserID)
			}
			b.members[m.Member.UserID] = m.Member.member
		}
	case "room.member_left", "room.member_kicked":
		id := str("user_id")
		delete(b.members, id)
		b.order = slices.DeleteFunc(b.order, func(x string) bool { return x == id })
	case "room.ready_changed":
		var r struct {
			UserID string `json:"user_id"`
			Ready  bool   `json:"is_ready"`
		}
		if json.Unmarshal(payload, &r) == nil {
			if m, ok := b.members[r.UserID]; ok {
				m.Ready = r.Ready
				b.members[r.UserID] = m
			}
		}
	case "room.host_changed":
		b.IsHost = str("host_user_id") == b.Me
	case "room.returned_to_lobby":
		var r struct {
			Room json.RawMessage `json:"room"`
		}
		if json.Unmarshal(payload, &r) == nil {
			b.room(r.Room)
		}
		b.resetGame()
	case "game.started":
		var g struct {
			GameID       string `json:"game_id"`
			Participants []struct {
				UserID string `json:"user_id"`
				Seat   int    `json:"seat"`
			} `json:"participants"`
		}
		if json.Unmarshal(payload, &g) != nil {
			return
		}
		b.resetGame()
		b.status = "in_game"
		b.gameID = g.GameID
		slices.SortFunc(g.Participants, func(a, c struct {
			UserID string `json:"user_id"`
			Seat   int    `json:"seat"`
		}) int {
			return a.Seat - c.Seat
		})
		for _, x := range g.Participants {
			b.seats = append(b.seats, x.UserID)
			b.alive[x.UserID] = true
		}
	case "game.your_cards":
		var c struct {
			Cards []struct {
				Category string `json:"category"`
				Revealed bool   `json:"revealed"`
			} `json:"cards"`
		}
		if json.Unmarshal(payload, &c) == nil {
			for _, x := range c.Cards {
				b.myCards[x.Category] = x.Revealed
			}
		}
	case "game.phase_changed":
		var ph struct {
			Phase string `json:"phase"`
			Round int    `json:"round"`
			Turn  string `json:"turn_user_id"`
		}
		if json.Unmarshal(payload, &ph) == nil {
			b.phase, b.round, b.turn = ph.Phase, ph.Round, ph.Turn
		}
	case "game.card_revealed":
		if str("user_id") == b.Me {
			b.myCards[str("category")] = true
		}
	case "game.tiebreak_started":
		var t struct {
			Candidates []string `json:"candidates"`
		}
		if json.Unmarshal(payload, &t) == nil {
			b.tbCands = t.Candidates
		}
	case "game.player_eliminated":
		b.alive[str("user_id")] = false
	case "game.player_connection":
		if str("state") == "left" {
			b.alive[str("user_id")] = false
		}
	case "game.finished":
		b.phase = "FINISHED"
		b.status = "finished"
	}
}

func (b *Brain) snapshot(payload json.RawMessage) {
	var s struct {
		Room json.RawMessage `json:"room"`
		Game *struct {
			ID           string `json:"id"`
			Phase        string `json:"phase"`
			Round        int    `json:"round"`
			Turn         string `json:"turn_user_id"`
			Participants []struct {
				UserID string `json:"user_id"`
				Seat   int    `json:"seat"`
				Status string `json:"status"`
			} `json:"participants"`
			MyCards []struct {
				Category string `json:"category"`
				Revealed bool   `json:"revealed"`
			} `json:"my_cards"`
			MyAction struct {
				Voted           bool `json:"voted"`
				DiscussionReady bool `json:"discussion_ready"`
				TiebreakVoted   bool `json:"tiebreak_voted"`
			} `json:"my_action_state"`
			Tiebreak *struct {
				Candidates []string `json:"candidates"`
			} `json:"tiebreak"`
		} `json:"game"`
	}
	if json.Unmarshal(payload, &s) != nil {
		return
	}
	b.synced = true
	b.room(s.Room)
	b.resetGame()
	if g := s.Game; g != nil {
		b.gameID, b.phase, b.round, b.turn = g.ID, g.Phase, g.Round, g.Turn
		for _, x := range g.Participants {
			b.seats = append(b.seats, x.UserID)
			b.alive[x.UserID] = x.Status == "alive"
		}
		for _, c := range g.MyCards {
			b.myCards[c.Category] = c.Revealed
		}
		if g.Tiebreak != nil {
			b.tbCands = g.Tiebreak.Candidates
		}
		// Sunucunun bildiği eylemlerim tekrar gönderilmez.
		if g.MyAction.Voted {
			b.done[b.key("vote")] = true
		}
		if g.MyAction.DiscussionReady {
			b.done[b.key("discussion_ready")] = true
		}
		if g.MyAction.TiebreakVoted {
			b.done[b.key("tiebreak_vote")] = true
		}
	}
}

func (b *Brain) room(raw json.RawMessage) {
	var r struct {
		Status  string `json:"status"`
		Host    string `json:"host_user_id"`
		Members []struct {
			UserID string `json:"user_id"`
			member
		} `json:"members"`
	}
	if json.Unmarshal(raw, &r) != nil {
		return
	}
	b.status = r.Status
	b.IsHost = r.Host == b.Me
	b.members = map[string]member{}
	b.order = nil
	for _, m := range r.Members {
		b.members[m.UserID] = m.member
		b.order = append(b.order, m.UserID)
	}
}

func (b *Brain) key(kind string) string {
	return b.gameID + ":" + b.phase + ":" + itoa(b.round) + ":" + b.turn + ":" + kind
}

// Next, şimdi yapılacak eylemleri döndürür; her eylem bir kez üretilir
// (başarısız olursa Retry ile yeniden üretilebilir hale gelir).
func (b *Brain) Next() []Action {
	if !b.synced {
		return nil
	}
	var out []Action
	add := func(a Action) {
		if !b.done[a.key] {
			b.done[a.key] = true
			out = append(out, a)
		}
	}
	switch b.status {
	case "lobby":
		me, ok := b.members[b.Me]
		if ok && me.Role == "player" && !me.Ready {
			add(Action{Kind: "ready", key: "lobby:ready"})
		}
		if b.IsHost && b.StartAt > 0 && b.readyPlayers() >= b.StartAt && b.allReady() {
			add(Action{Kind: "start", key: "lobby:start:" + itoa(len(b.order))})
		}
	case "in_game":
		b.gameActions(add)
	}
	return out
}

// Retry, başarısız eylemi yeniden denenebilir yapar.
func (b *Brain) Retry(a Action) { delete(b.done, a.key) }

func (b *Brain) gameActions(add func(Action)) {
	alive := b.alive[b.Me]
	game := func(typ string, payload map[string]any) Action {
		return Action{Kind: "game", Type: typ, Payload: payload, key: b.key(typ)}
	}
	switch b.phase {
	case "REVEAL":
		if alive && b.turn == b.Me {
			if c := b.pickCard(); c != "" {
				add(game("reveal_card", map[string]any{"category": c}))
			}
		}
	case "DISCUSSION":
		if alive {
			add(game("discussion_ready", map[string]any{}))
		}
	case "VOTING":
		if alive {
			var targets []string
			for _, id := range b.seats {
				if id != b.Me && b.alive[id] {
					targets = append(targets, id)
				}
			}
			if len(targets) > 0 {
				add(game("vote", map[string]any{"target_user_id": targets[b.rng.IntN(len(targets))]}))
			}
		}
	case "TIE_BREAK":
		if _, participant := b.alive[b.Me]; participant && !alive {
			var targets []string
			for _, id := range b.tbCands {
				if b.alive[id] {
					targets = append(targets, id)
				}
			}
			if len(targets) > 0 {
				add(game("tiebreak_vote", map[string]any{"target_user_id": targets[b.rng.IntN(len(targets))]}))
			}
		}
	}
}

// pickCard: 1. turda meslek; sonra kapalı kartlardan rastgele.
func (b *Brain) pickCard() string {
	if b.round == 1 {
		if revealed, ok := b.myCards["profession"]; ok && !revealed {
			return "profession"
		}
		return ""
	}
	var closed []string
	for _, c := range categories {
		if revealed, ok := b.myCards[c]; ok && !revealed {
			closed = append(closed, c)
		}
	}
	if len(closed) == 0 {
		return ""
	}
	return closed[b.rng.IntN(len(closed))]
}

func (b *Brain) readyPlayers() int {
	n := 0
	for _, m := range b.members {
		if m.Role == "player" && m.Ready {
			n++
		}
	}
	return n
}

func (b *Brain) allReady() bool {
	for _, m := range b.members {
		if m.Role == "player" && !m.Ready {
			return false
		}
	}
	return true
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
