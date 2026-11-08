package engine

import "time"

// Durations, aşama süreleridir (game-design §8; MVP'de sabit).
type Durations struct {
	ScenarioIntro     time.Duration
	RevealPerPlayer   time.Duration
	DiscussionSmall   time.Duration // hayatta ≤ 6
	DiscussionLarge   time.Duration // hayatta ≥ 7
	Voting            time.Duration
	TieBreak          time.Duration
	Elimination       time.Duration
	LargeTableAtLeast int
}

// DefaultDurations, game-design §8 tablosu.
var DefaultDurations = Durations{
	ScenarioIntro:     30 * time.Second, // anlatımlı tanıtım animasyonu (docs/ui/scenario-intro-assets.md)
	RevealPerPlayer:   20 * time.Second,
	DiscussionSmall:   90 * time.Second,
	DiscussionLarge:   120 * time.Second,
	Voting:            30 * time.Second,
	TieBreak:          20 * time.Second,
	Elimination:       8 * time.Second,
	LargeTableAtLeast: 7,
}

func (d Durations) discussion(alive int) time.Duration {
	if alive >= d.LargeTableAtLeast {
		return d.DiscussionLarge
	}
	return d.DiscussionSmall
}
