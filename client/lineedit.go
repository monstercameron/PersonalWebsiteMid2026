//go:build js && wasm

package main

import "strings"

// This file holds the readline layer: history recall and the kill/move bindings a real shell
// gives you. It is deliberately free of DOM access so it can be unit-tested directly — the
// terminal component supplies the caret position and applies the result.

// histNav walks a command history the way a shell does: Up moves toward older entries, Down back
// toward newer, and the partially-typed line you had before you started browsing is preserved as
// the "draft" you land on when you come back past the newest entry.
//
// idx is an index into entries, except for the sentinel len(entries), which means "not browsing —
// showing the draft".
type histNav struct {
	entries []string
	idx     int
	draft   string
}

// record appends a command to the history and stops any in-progress browsing, which is what a
// shell does the moment you hit Enter. Consecutive duplicates are collapsed: re-running the same
// command three times should not mean three Up presses to get past it.
func (h *histNav) record(line string) {
	if line != "" && (len(h.entries) == 0 || h.entries[len(h.entries)-1] != line) {
		h.entries = append(h.entries, line)
	}
	h.reset()
}

// reset abandons browsing and forgets the draft.
func (h *histNav) reset() {
	h.idx = len(h.entries)
	h.draft = ""
}

// up moves one entry toward the oldest command and returns the line to show. ok is false when
// there is nothing older (including an empty history), so the caller leaves the input untouched.
// current is the text on screen now; it becomes the draft on the first Up.
func (h *histNav) up(current string) (string, bool) {
	if len(h.entries) == 0 {
		return "", false
	}
	if h.idx > len(h.entries) {
		h.idx = len(h.entries)
	}
	if h.idx == len(h.entries) {
		h.draft = current
	}
	if h.idx == 0 {
		return "", false
	}
	h.idx--
	return h.entries[h.idx], true
}

// down moves one entry toward the newest command, and past the newest back to the draft. ok is
// false when already at the draft.
func (h *histNav) down() (string, bool) {
	if h.idx >= len(h.entries) {
		return "", false
	}
	h.idx++
	if h.idx == len(h.entries) {
		return h.draft, true
	}
	return h.entries[h.idx], true
}

// clampCaret bounds a caret position reported by the DOM to the line, so a stale or missing
// selectionStart can never slice out of range.
func clampCaret(line string, caret int) int {
	if caret < 0 || caret > len(line) {
		return len(line)
	}
	return caret
}

// killToLineStart implements Ctrl+U: delete from the caret back to the start of the line. It
// returns the new line and the new caret, which is always 0 — the surviving text shifts left.
func killToLineStart(line string, caret int) (string, int) {
	return line[clampCaret(line, caret):], 0
}

// killPrevWord implements Ctrl+W: delete the word before the caret, along with any whitespace
// between the caret and that word. The caret lands where the deleted word began.
func killPrevWord(line string, caret int) (string, int) {
	c := clampCaret(line, caret)
	i := c
	for i > 0 && line[i-1] == ' ' {
		i--
	}
	for i > 0 && line[i-1] != ' ' {
		i--
	}
	return line[:i] + line[c:], i
}

// levenshtein returns the edit distance between a and b. It backs the "did you mean" hint on an
// unknown command, so a typo becomes a suggestion instead of a dead end.
func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if a == "" {
		return len(b)
	}
	if b == "" {
		return len(a)
	}
	// Two rolling rows rather than the full matrix — the inputs are command names, but there is
	// no reason to allocate len(a)*len(b) for a distance we only compare against a threshold.
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = minInt(minInt(cur[j-1]+1, prev[j]+1), prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

// didYouMean returns the closest known command to name, or "" when nothing is close enough.
// The threshold scales with the word length so "lss" suggests "ls" but "xyzzy" suggests nothing —
// a wrong suggestion is worse than none, because it sends the visitor down a second dead end.
func didYouMean(name string) string {
	if name == "" {
		return ""
	}
	limit := 2
	if len(name) <= 3 {
		limit = 1
	}
	best, bestDist := "", limit+1
	for _, c := range allCommands {
		// Prefix matches are a stronger signal than raw edit distance for a half-typed command.
		d := levenshtein(strings.ToLower(name), c)
		if strings.HasPrefix(c, strings.ToLower(name)) && len(name) >= 2 {
			d = 1
		}
		if d < bestDist {
			best, bestDist = c, d
		}
	}
	if bestDist > limit {
		return ""
	}
	return best
}
