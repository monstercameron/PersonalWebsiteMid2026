package rss

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/monstercameron/earlcameron/internal/store"
)

// TestDailyPromptRotation verifies DailyPrompt is deterministic per calendar day and rotates by
// day-of-year, plus its zero-length guard.
func TestDailyPromptRotation(t *testing.T) {
	prompts := []string{"a", "b", "c"}
	cases := []struct {
		name string
		now  time.Time
		want string
	}{
		{"jan1", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), prompts[1%3]},
		{"jan2", time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), prompts[2%3]},
		{"jan3", time.Date(2026, 1, 3, 15, 30, 0, 0, time.UTC), prompts[3%3]},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DailyPrompt(prompts, tc.now)
			if got != tc.want {
				t.Errorf("DailyPrompt(%v) = %q, want %q", tc.now, got, tc.want)
			}
		})
	}

	// Same day, different times-of-day, must return the same prompt.
	a := DailyPrompt(prompts, time.Date(2026, 3, 5, 1, 0, 0, 0, time.UTC))
	b := DailyPrompt(prompts, time.Date(2026, 3, 5, 23, 59, 0, 0, time.UTC))
	if a != b {
		t.Errorf("DailyPrompt not stable within a day: %q vs %q", a, b)
	}

	if got := DailyPrompt(nil, time.Now()); got != "" {
		t.Errorf("DailyPrompt(nil) = %q, want empty", got)
	}
}

// TestSeedPrompts verifies SeedPrompts only inserts DefaultPrompts once.
func TestSeedPrompts(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	now := time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC)

	n, err := SeedPrompts(ctx, s, now)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if n != len(DefaultPrompts) {
		t.Errorf("first seed inserted %d, want %d", n, len(DefaultPrompts))
	}

	n, err = SeedPrompts(ctx, s, now)
	if err != nil {
		t.Fatalf("reseed: %v", err)
	}
	if n != 0 {
		t.Errorf("second seed inserted %d, want 0 (already populated)", n)
	}

	count, err := s.CountPrompts(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != len(DefaultPrompts) {
		t.Errorf("stored count = %d, want %d", count, len(DefaultPrompts))
	}
}
