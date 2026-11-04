package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/config"
	"github.com/burakkuru5534/bunker/backend/internal/game"
	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
	"github.com/burakkuru5534/bunker/backend/internal/realtime"
	"github.com/burakkuru5534/bunker/backend/internal/testbot"
	"github.com/burakkuru5534/bunker/backend/internal/testdb"
)

// Uçtan uca: gerçek sunucu (tüm modüller, gerçek PostgreSQL, gerçek katalog
// içeriği), gerçek HTTP + WebSocket istemcileri olarak test botları. Bir bot
// oda kurar, üçü katılır; oyun başlar ve sunucunun kurallarıyla sonuna kadar
// oynanır. Süreler kısaltılmıştır; zaman aşımı yolları da işler.
func TestIntegrationEndToEndFullGame(t *testing.T) {
	env := testdb.NewWithContent(t)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	cfg, err := config.Load(func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	fast := engine.Durations{
		ScenarioIntro: 100 * time.Millisecond, RevealPerPlayer: 3 * time.Second,
		DiscussionSmall: 3 * time.Second, DiscussionLarge: 3 * time.Second,
		Voting: 3 * time.Second, TieBreak: 2 * time.Second, Elimination: 100 * time.Millisecond,
		LargeTableAtLeast: 7,
	}
	d := deps{DB: env.DB, Timers: game.NewTimers(logger), Hub: realtime.NewHub(logger), Life: &lifecycle{}, Durations: &fast}
	srv := httptest.NewServer(newHandler(cfg, logger, d))
	d.Timers.Start()
	for _, start := range d.Life.start {
		start()
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.CloseClientConnections()
		srv.Close()
		d.Hub.Shutdown()
		for _, stop := range d.Life.stop {
			_ = stop(ctx)
		}
		_ = d.Timers.Stop(ctx)
	})

	api, _ := url.Parse(srv.URL)
	bcfg := testbot.Config{API: api, AppVersion: "0.1.0", Logger: logger}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	finished := make(chan string, 16)
	var bots []*testbot.Bot
	var code string
	for i := range 4 {
		b := testbot.New(bcfg, "Bot "+itoa(i+1))
		if err := b.Register(ctx); err != nil {
			t.Fatalf("hesap: %v", err)
		}
		if i == 0 {
			if code, err = b.Create(ctx, 4); err != nil {
				t.Fatal(err)
			}
		} else if err := b.Join(ctx, code); err != nil {
			t.Fatal(err)
		}
		events := make(chan string, 256)
		b.Events = events
		go func(name string) {
			for typ := range events {
				if typ == "game.finished" {
					finished <- name
				}
			}
		}(b.Nickname)
		bots = append(bots, b)
	}

	runCtx, stopBots := context.WithCancel(ctx)
	var wg sync.WaitGroup
	for _, b := range bots {
		wg.Go(func() {
			if err := b.Run(runCtx); err != nil {
				t.Errorf("%s: %v", b.Nickname, err)
			}
		})
	}

	seen := map[string]bool{}
	for len(seen) < len(bots) {
		select {
		case name := <-finished:
			seen[name] = true
		case <-ctx.Done():
			t.Fatalf("oyun bitmedi; bitişi gören botlar: %v", seen)
		}
	}
	stopBots()
	wg.Wait()

	// Sonuç: 4 oyuncu, kapasite 2 → 2 kazanan, 2 tur, her turda bir eleme.
	var gameID string
	if err := env.DB.Pool().QueryRow(ctx, `select id::text from games`).Scan(&gameID); err != nil {
		t.Fatal(err)
	}
	var result struct {
		EndReason string `json:"end_reason"`
		Players   []struct {
			IsWinner bool `json:"is_winner"`
			Cards    []any
		} `json:"players"`
		Rounds []struct {
			Eliminated *string `json:"eliminated_user_id"`
		} `json:"rounds"`
	}
	getJSON(t, ctx, srv.URL+"/v1/games/"+gameID+"/result", bots[0], &result)
	winners := 0
	for _, p := range result.Players {
		if p.IsWinner {
			winners++
		}
		if len(p.Cards) != 5 {
			t.Fatalf("sonuçta 5 kart olmalı: %d", len(p.Cards))
		}
	}
	if result.EndReason != "capacity_reached" || winners != 2 || len(result.Rounds) != 2 {
		t.Fatalf("sonuç: %+v (kazanan %d)", result, winners)
	}
	for _, r := range result.Rounds {
		if r.Eliminated == nil {
			t.Fatalf("her turda eleme olmalı: %+v", result.Rounds)
		}
	}

	// Temizlik: botlar ayrılır ve hesaplarını siler.
	for _, b := range bots {
		b.Cleanup(ctx)
	}
	var active int
	if err := env.DB.Pool().QueryRow(ctx, `select count(*) from users where deleted_at is null`).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if active != 0 {
		t.Fatalf("silinmemiş bot hesabı: %d", active)
	}
}

func getJSON(t *testing.T, ctx context.Context, u string, b *testbot.Bot, out any) {
	t.Helper()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("X-App-Version", "0.1.0")
	req.Header.Set("Authorization", "Bearer "+b.Token())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s: %d", u, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatal(err)
	}
}
