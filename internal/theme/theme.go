// Package theme holds the project's reusable design tokens — the single source of truth for
// colors across ALL UI (the SSR standard site, the terminal, admin). Always style through these
// tokens; never scatter ad-hoc u.Hex() values, so the look stays consistent and a palette change
// is one edit. Quick-reference table lives in documents/DESIGN.md.
//
// Spacing, radii, and font-size tokens come from css/u's defaults (Spacing2..10, RadiusLg/RadiusXl,
// TextSm, …) — only these project-specific colors live here.
package theme

import "github.com/monstercameron/GoWebComponents/v5/css/u"

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

// Plate is the light surface the Flux brand artwork sits on.
//
// The brand posters and logo lockups are drawn for a near-white ground with a dark navy wordmark,
// so dropping them straight onto the aubergine page would erase half of each lockup. They get an
// explicit light plate instead, which also states what they are: marketing artwork, framed and
// labelled, rather than the page's own skin.
var Plate = u.Hex("#f6f4fa")

// Brand is one Flux product's colour identity, sampled from its own poster.
//
// The site palette stays Aubergine everywhere (DESIGN.md §6) — a project page must still read as
// earlcameron.com. What changes per page is a single accent and the ambient glow behind it, which
// is enough to make ArticleFlux blue and CashFlux green legible as different products before a
// visitor has read a word, without forking the design language five ways.
type Brand struct {
	// Accent is the product's signature hue, brightened from the poster sample until it clears
	// 4.5:1 against Bg — the poster values are tuned for white and go muddy on aubergine.
	Accent u.Color
	// Tint is the lighter partner used for secondary marks and hairlines.
	Tint u.Color
	// Glow is an rgba() string for the page's ambient background wash. Kept as a string because it
	// carries an alpha channel that u.Hex does not model.
	Glow string
}

// Brands maps a project slug to its colour identity. Slugs match internal/content's project IDs
// and the /projects/<slug> route, so a page, a card and a palette can never drift apart.
var Brands = map[string]Brand{
	"articleflux": {Accent: u.Hex("#4d84ff"), Tint: u.Hex("#9ab6ff"), Glow: "rgba(77,132,255,.20)"},
	"cashflux":    {Accent: u.Hex("#3fbe86"), Tint: u.Hex("#8bdcba"), Glow: "rgba(63,190,134,.18)"},
	"codeflux":    {Accent: u.Hex("#7d6bff"), Tint: u.Hex("#b3a8ff"), Glow: "rgba(125,107,255,.20)"},
	"pixelflux":   {Accent: u.Hex("#a86bf0"), Tint: u.Hex("#e56fc8"), Glow: "rgba(168,107,240,.20)"},
	"schemaflux":  {Accent: u.Hex("#3fc9c0"), Tint: u.Hex("#8ee2dc"), Glow: "rgba(63,201,192,.18)"},
}
