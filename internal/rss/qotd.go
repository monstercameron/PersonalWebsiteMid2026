// Package rss builds spec-compliant RSS 2.0 feeds plus the QOTD (question-of-the-day) prompt
// rotation, anime-news fetch, and Slack posting helpers used to drive the anime discussion loop.
// It supersedes the RSS logic in internal/anime, which callers should migrate to this package.
package rss

import (
	"context"
	"time"

	"github.com/monstercameron/earlcameron/internal/store"
)

// DefaultPrompts seeds the qotd_prompts table the first time it is empty. It merges the prompt
// set already live in internal/anime.animePrompts with the original DAILY_ANIME_QUESTIONS list
// from the legacy Node site (api/src/app.impure.js), so nothing that was already rotating is
// lost when the store-backed version takes over.
var DefaultPrompts = []string{
	// Ported from internal/anime.animePrompts.
	"Which anime got the ending it deserved — and which one didn't?",
	"Name an anime that's better on a rewatch than the first time through.",
	"What's the most underrated anime of the last five years?",
	"Sub or dub — and does it depend on the show?",
	"Which anime has the best opening sequence, no skips?",
	"What anime world would you actually want to live in?",
	"Which side character deserved to be the main character?",
	"Best fight scene, purely on animation?",
	"An anime you'd recommend to someone who 'doesn't watch anime'?",
	"Which studio is on the best run right now?",
	"What's a trope you're tired of, and one you'll never get tired of?",
	"Best redemption arc — did it earn it?",
	"Which manga adaptation actually improved on the source?",
	"An anime soundtrack you listen to outside the show?",
	"Which anime aged the best visually?",
	"Best 'monster of the week' show that's secretly deep?",
	"Which anime made you cry and you're not ashamed to admit it?",
	"What's the strongest first episode you've ever watched?",
	"Which power system is the most internally consistent?",
	"An anime you dropped that you should probably give another shot?",
	// Ported from the legacy DAILY_ANIME_QUESTIONS list.
	"What currently airing anime has the strongest world-building this season?",
	"Which anime character had the best growth arc this year?",
	"What opening theme are you replaying most this week?",
	"Which anime deserves a sequel right now?",
	"What anime episode had the best pacing recently?",
	"Which studio has the strongest lineup this season?",
	"What under-watched anime should more people pick up?",
	"Which fight scene had the best choreography lately?",
	"What anime side character stole the show for you?",
	"Which genre twist in anime surprised you most recently?",
	"What anime has the strongest emotional payoff right now?",
	"Which anime adaptation improved on its source material?",
	"What anime ending song has been your favorite this month?",
	"Which anime has the most interesting power system currently?",
	"What anime recommendation would you give a non-anime watcher?",
	"Which anime handled tension and suspense the best lately?",
	"What anime character would make the best mentor and why?",
	"Which anime episode had the most rewatch value this season?",
	"What anime soundtrack has been the most memorable recently?",
	"Which anime has the best balance of action and story right now?",
	"What anime romance subplot actually works for you?",
	"Which anime protagonist feels the most grounded and believable?",
	"What anime has the strongest art direction this season?",
	"Which anime reveal or twist landed perfectly for you?",
	"What anime world would you most want to explore for a day?",
	"Which anime had the cleanest first episode hook recently?",
	"What anime deserves more discussion in your community?",
	"Which anime antagonist is the most compelling this year?",
	"What anime scene made you pause and think the longest?",
	"Which anime has the best long-term story potential right now?",
}

// SeedPrompts inserts DefaultPrompts into the store, but only if no prompts exist yet (so it is
// safe to call on every boot). It returns how many prompts were inserted; 0 means the table was
// already populated and nothing was touched.
func SeedPrompts(ctx context.Context, s *store.Store, now time.Time) (int, error) {
	n, err := s.CountPrompts(ctx)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		return 0, nil
	}
	ts := now.Unix()
	for _, p := range DefaultPrompts {
		if _, err := s.AddPrompt(ctx, p, ts); err != nil {
			return 0, err
		}
	}
	return len(DefaultPrompts), nil
}

// DailyPrompt deterministically picks one prompt out of prompts for the given day: it indexes by
// now's day-of-year modulo len(prompts), so the same calendar day always yields the same prompt
// for a fixed prompt list, and consecutive days walk the list in order (wrapping around). It
// returns "" for an empty prompts slice.
func DailyPrompt(prompts []string, now time.Time) string {
	if len(prompts) == 0 {
		return ""
	}
	return prompts[now.UTC().YearDay()%len(prompts)]
}
