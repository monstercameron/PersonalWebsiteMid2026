// Package theme holds the project's reusable design tokens — the single source of truth for
// colors across ALL UI (the SSR standard site, the terminal, admin). Always style through these
// tokens; never scatter ad-hoc u.Hex() values, so the look stays consistent and a palette change
// is one edit. Quick-reference table lives in documents/DESIGN.md.
//
// Spacing, radii, and font-size tokens come from css/u's defaults (Spacing2..10, RadiusLg/RadiusXl,
// TextSm, …) — only these project-specific colors live here.
package theme

import "github.com/monstercameron/GoWebComponents/v4/css/u"

// Aubergine palette (Ubuntu-souled): deep aubergine ground, warm off-white text, Ubuntu-orange
// accent, purple secondary. See documents/DESIGN.md for the design rationale.
var (
	Bg       = u.Hex("#17040f") // deep near-black aubergine ground
	BgRaised = u.Hex("#210a19") // raised surfaces (cards, panels)
	Fg       = u.Hex("#f3e9e6") // warm off-white text
	Dim      = u.Hex("#a98ba0") // muted mauve — secondary text
	Faint    = u.Hex("#6f5364") // faint — hints, placeholders
	Border   = u.Hex("#3a1b2e") // aubergine-tinted border
	Accent   = u.Hex("#e95420") // Ubuntu orange — prompt, cursor, CTA, active
	Accent2  = u.Hex("#be7be6") // purple — links, secondary highlights
	Green    = u.Hex("#8ae234") // success / status
	Red      = u.Hex("#ef5350") // error
	Yellow   = u.Hex("#f2b840") // warning
	Cyan     = u.Hex("#4dd0e1") // info
)

// Terminal chrome — a near-black macOS-Terminal-style panel that stands out against the
// aubergine page. Keep the hex values in sync with the raw css.Property borders in the client.
var (
	TermBg     = u.Hex("#121016") // near-black terminal background
	TermBar    = u.Hex("#26242c") // title bar (mac-like dark gray)
	TermBorder = u.Hex("#38343f") // terminal border
)
