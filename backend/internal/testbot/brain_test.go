package testbot

import (
	"encoding/json"
	"math/rand/v2"
	"testing"
)

func raw(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func memberJSON(id string, ready bool) map[string]any {
	return map[string]any{"user_id": id, "nickname": id, "role": "player", "is_ready": ready, "connected": true}
}

func lobby(host string, members ...map[string]any) json.RawMessage {
	return raw(map[string]any{
		"room": map[string]any{"status": "lobby", "host_user_id": host, "members": members},
		"game": nil,
	})
}

func kinds(as []Action) []string {
	var out []string
	for _, a := range as {
		if a.Kind == "game" {
			out = append(out, a.Type)
		} else {
			out = append(out, a.Kind)
		}
	}
	return out
}

func newBrain(me string) *Brain { return NewBrain(me, rand.New(rand.NewPCG(1, 2))) }

func TestBrainIgnoresEventsBeforeSnapshot(t *testing.T) {
	b := newBrain("me")
	b.Handle("room.member_joined", raw(map[string]any{"member": memberJSON("x", false)}))
	if got := b.Next(); len(got) != 0 {
		t.Fatalf("snapshot öncesi eylem: %v", kinds(got))
	}
}

func TestBrainLobbyReadyOnceAndHostStarts(t *testing.T) {
	b := newBrain("me")
	b.StartAt = 4
	b.Handle("session.snapshot", lobby("me", memberJSON("me", false), memberJSON("a", true), memberJSON("b", true)))
	if got := kinds(b.Next()); len(got) != 1 || got[0] != "ready" {
		t.Fatalf("ilk eylem hazır olmalı: %v", got)
	}
	if got := b.Next(); len(got) != 0 {
		t.Fatalf("hazır tekrar üretilmemeli: %v", kinds(got))
	}
	b.Handle("room.ready_changed", raw(map[string]any{"user_id": "me", "is_ready": true}))
	if got := b.Next(); len(got) != 0 {
		t.Fatalf("3 oyuncuyla başlatılmamalı: %v", kinds(got))
	}
	b.Handle("room.member_joined", raw(map[string]any{"member": memberJSON("c", false)}))
	if got := b.Next(); len(got) != 0 {
		t.Fatalf("herkes hazır değilken başlatılmamalı: %v", kinds(got))
	}
	b.Handle("room.ready_changed", raw(map[string]any{"user_id": "c", "is_ready": true}))
	if got := kinds(b.Next()); len(got) != 1 || got[0] != "start" {
		t.Fatalf("4 hazır oyuncuyla başlatmalı: %v", got)
	}
}

func TestBrainHostWaitsForEveryone(t *testing.T) {
	b := newBrain("me")
	b.StartAt = 4
	// 4 hazır ama beşinci hazır değil: sunucu reddederdi, bot denemez.
	b.Handle("session.snapshot", lobby("me", memberJSON("me", true), memberJSON("a", true),
		memberJSON("b", true), memberJSON("c", true), memberJSON("d", false)))
	if got := b.Next(); len(got) != 0 {
		t.Fatalf("herkes hazır değilken başlatılmamalı: %v", kinds(got))
	}
}

func TestBrainNonHostNeverStarts(t *testing.T) {
	b := newBrain("me")
	b.StartAt = 4
	b.Handle("session.snapshot", lobby("a", memberJSON("a", true), memberJSON("me", true), memberJSON("b", true), memberJSON("c", true)))
	if got := b.Next(); len(got) != 0 {
		t.Fatalf("sahip olmayan başlatmamalı: %v", kinds(got))
	}
	b.Handle("room.host_changed", raw(map[string]any{"host_user_id": "me"}))
	if got := kinds(b.Next()); len(got) != 1 || got[0] != "start" {
		t.Fatalf("sahip olunca başlatmalı: %v", got)
	}
}

// started: me, a, b koltuk sırasıyla; kartlarım kapalı.
func started(b *Brain) {
	b.Handle("session.snapshot", lobby("a", memberJSON("a", true), memberJSON("me", true), memberJSON("b", true)))
	b.Handle("game.started", raw(map[string]any{"game_id": "g1", "participants": []map[string]any{
		{"user_id": "b", "seat": 2}, {"user_id": "me", "seat": 0}, {"user_id": "a", "seat": 1},
	}}))
	cards := []map[string]any{}
	for _, c := range categories {
		cards = append(cards, map[string]any{"category": c, "text": c, "revealed": false})
	}
	b.Handle("game.your_cards", raw(map[string]any{"cards": cards}))
}

func phase(b *Brain, p string, round int, turn string) {
	b.Handle("game.phase_changed", raw(map[string]any{"phase": p, "round": round, "turn_user_id": turn}))
}

func TestBrainRevealProfessionFirstThenOthers(t *testing.T) {
	b := newBrain("me")
	started(b)
	phase(b, "REVEAL", 1, "a")
	if got := b.Next(); len(got) != 0 {
		t.Fatalf("sıra başkasında: %v", kinds(got))
	}
	phase(b, "REVEAL", 1, "me")
	got := b.Next()
	if len(got) != 1 || got[0].Payload["category"] != "profession" {
		t.Fatalf("1. turda meslek: %+v", got)
	}
	b.Handle("game.card_revealed", raw(map[string]any{"user_id": "me", "category": "profession", "text": "x"}))
	phase(b, "REVEAL", 2, "me")
	got = b.Next()
	if len(got) != 1 || got[0].Payload["category"] == "profession" || got[0].Payload["category"] == nil {
		t.Fatalf("2. turda açık olmayan kart: %+v", got)
	}
}

func TestBrainDiscussionAndVoteOncePerRound(t *testing.T) {
	b := newBrain("me")
	started(b)
	phase(b, "DISCUSSION", 1, "")
	if got := kinds(b.Next()); len(got) != 1 || got[0] != "discussion_ready" {
		t.Fatalf("hazırım: %v", got)
	}
	phase(b, "VOTING", 1, "")
	got := b.Next()
	if len(got) != 1 || got[0].Type != "vote" {
		t.Fatalf("oy: %v", kinds(got))
	}
	if tgt := got[0].Payload["target_user_id"]; tgt == "me" || tgt == nil {
		t.Fatalf("kendine oy: %v", tgt)
	}
	if again := b.Next(); len(again) != 0 {
		t.Fatalf("aynı turda ikinci oy: %v", kinds(again))
	}
	// Ölü oyuncuya oy verilmez.
	b.Handle("game.player_eliminated", raw(map[string]any{"user_id": "a", "method": "vote"}))
	phase(b, "VOTING", 2, "")
	for range 20 {
		got := b.Next()
		if len(got) == 1 && got[0].Payload["target_user_id"] != "b" {
			t.Fatalf("elenmişe oy: %v", got[0].Payload)
		}
		b.Retry(got[0])
	}
}

func TestBrainEliminatedOnlyTiebreak(t *testing.T) {
	b := newBrain("me")
	started(b)
	b.Handle("game.player_eliminated", raw(map[string]any{"user_id": "me", "method": "vote"}))
	phase(b, "DISCUSSION", 2, "")
	phase(b, "VOTING", 2, "")
	if got := b.Next(); len(got) != 0 {
		t.Fatalf("elenmiş oy vermez: %v", kinds(got))
	}
	b.Handle("game.tiebreak_started", raw(map[string]any{"candidates": []string{"a", "b"}}))
	phase(b, "TIE_BREAK", 2, "")
	got := b.Next()
	if len(got) != 1 || got[0].Type != "tiebreak_vote" {
		t.Fatalf("eşitlik oyu: %v", kinds(got))
	}
}

func TestBrainSnapshotRestoresDoneActions(t *testing.T) {
	b := newBrain("me")
	b.Handle("session.snapshot", raw(map[string]any{
		"room": map[string]any{"status": "in_game", "host_user_id": "a", "members": []any{memberJSON("me", true)}},
		"game": map[string]any{
			"id": "g1", "phase": "VOTING", "round": 1, "turn_user_id": nil,
			"participants": []map[string]any{
				{"user_id": "me", "seat": 0, "status": "alive"},
				{"user_id": "a", "seat": 1, "status": "alive"},
			},
			"my_cards":        []any{},
			"my_action_state": map[string]any{"voted": true},
		},
	}))
	if got := b.Next(); len(got) != 0 {
		t.Fatalf("verilmiş oy tekrarlanmamalı: %v", kinds(got))
	}
	if b.GameID() != "g1" {
		t.Fatalf("oyun kimliği: %q", b.GameID())
	}
}

func TestBrainReturnToLobbyReadiesAgain(t *testing.T) {
	b := newBrain("me")
	started(b)
	b.Handle("game.finished", raw(map[string]any{}))
	if b.Status() != "finished" {
		t.Fatal(b.Status())
	}
	b.Handle("room.returned_to_lobby", raw(map[string]any{"room": map[string]any{
		"status": "lobby", "host_user_id": "a", "members": []any{memberJSON("me", false), memberJSON("a", false)},
	}}))
	if got := kinds(b.Next()); len(got) != 1 || got[0] != "ready" {
		t.Fatalf("lobide yeniden hazır: %v", got)
	}
}
