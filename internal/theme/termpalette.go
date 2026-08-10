package theme

// The terminal is the one surface on the site a visitor can re-colour at runtime (the `theme`
// command). That needs the palette as *data* — plain hex strings the client writes into CSS custom
// properties — which the u.Color tokens above cannot provide, since they are compiled into
// generated class names at build time.
//
// This table lives here, next to those tokens, rather than in the client: colours are this
// package's job (DESIGN.md §16), and putting it anywhere else is how a second, drifting palette
// gets born. Aubergine below mirrors the token values above and must be changed with them.
//
// Site-wide theming (aubergine · light · nord across the SSR page too) is a different and much
// larger job — it needs the whole token layer expressed as custom properties — and is tracked
// separately in TODOS §3. This is deliberately only the terminal.

// TermPalette is one terminal colour scheme. Fields are CSS colour strings.
type TermPalette struct {
	// Name is what the visitor types: `theme nord`.
	Name string
	// Blurb is the one-line description `theme` prints when listing.
	Blurb string
	// Chrome.
	Bg     string // terminal panel background
	Bar    string // title bar
	Border string // panel border
	// Text.
	Fg    string // body text
	Dim   string // secondary text
	Faint string // hints, placeholders
	// Signal colours.
	Accent  string // prompt, cursor, headings
	Accent2 string // keys, links
	Green   string // success / status
	Red     string // errors
}

// TermPalettes are the schemes the `theme` command offers. The first entry is the default and
// must stay in sync with the Aubergine tokens above.
var TermPalettes = []TermPalette{
	{
		Name: "aubergine", Blurb: "the site's own palette — Ubuntu orange on deep aubergine",
		Bg: "#121016", Bar: "#26242c", Border: "#38343f",
		Fg: "#f3e9e6", Dim: "#a98ba0", Faint: "#6f5364",
		Accent: "#e95420", Accent2: "#be7be6", Green: "#8ae234", Red: "#ef5350",
	},
	{
		Name: "nord", Blurb: "cool arctic blues",
		Bg: "#2e3440", Bar: "#3b4252", Border: "#4c566a",
		Fg: "#eceff4", Dim: "#a6b6cc", Faint: "#7b88a0",
		Accent: "#88c0d0", Accent2: "#b48ead", Green: "#a3be8c", Red: "#bf616a",
	},
	{
		Name: "paper", Blurb: "light mode, for anyone reading in daylight",
		Bg: "#f6f4fa", Bar: "#e6e2ee", Border: "#cdc6d8",
		Fg: "#241c2b", Dim: "#5c5268", Faint: "#8a809a",
		Accent: "#c2410c", Accent2: "#7c3aed", Green: "#3f7d20", Red: "#b91c1c",
	},
	{
		Name: "matrix", Blurb: "green phosphor, because someone always asks",
		Bg: "#000000", Bar: "#0a1a0a", Border: "#123d12",
		Fg: "#33ff33", Dim: "#1faa1f", Faint: "#137013",
		Accent: "#7fff7f", Accent2: "#00e676", Green: "#33ff33", Red: "#ff5252",
	},
}

// TermPaletteByName returns the palette with the given name and whether it was found.
func TermPaletteByName(name string) (TermPalette, bool) {
	for _, p := range TermPalettes {
		if p.Name == name {
			return p, true
		}
	}
	return TermPalette{}, false
}
