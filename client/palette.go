//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v5/css"

	"github.com/monstercameron/earlcameron/internal/theme"
)

// The terminal renders its colours through CSS custom properties rather than the compiled colour
// classes the rest of the site uses, because `theme` has to change them at runtime and a generated
// class name has its hex baked in. Every helper below carries the Aubergine value as the var()
// fallback, so the terminal is correctly coloured before applyPalette runs and stays correct if a
// browser ignores the properties entirely.
//
// The custom properties are set with element.style.setProperty rather than through a GWC Style
// map: the typed prop path does not carry `--*` names through to the DOM.

// themeKey is the localStorage key holding the visitor's chosen terminal palette.
const themeKey = "earlcameron.term.theme"

// colour helpers. Each returns a typed css rule referencing one custom property.
func cFg() css.Rule     { return css.Raw("color", "var(--t-fg, "+theme.TermPalettes[0].Fg+")") }
func cDim() css.Rule    { return css.Raw("color", "var(--t-dim, "+theme.TermPalettes[0].Dim+")") }
func cFaint() css.Rule  { return css.Raw("color", "var(--t-faint, "+theme.TermPalettes[0].Faint+")") }
func cAccent() css.Rule { return css.Raw("color", "var(--t-accent, "+theme.TermPalettes[0].Accent+")") }
func cAccent2() css.Rule {
	return css.Raw("color", "var(--t-accent2, "+theme.TermPalettes[0].Accent2+")")
}
func cGreen() css.Rule { return css.Raw("color", "var(--t-green, "+theme.TermPalettes[0].Green+")") }
func cRed() css.Rule   { return css.Raw("color", "var(--t-red, "+theme.TermPalettes[0].Red+")") }
func cBg() css.Rule    { return css.Raw("background", "var(--t-bg, "+theme.TermPalettes[0].Bg+")") }
func cBarBg() css.Rule { return css.Raw("background", "var(--t-bar, "+theme.TermPalettes[0].Bar+")") }

// cBorder returns a 1px border in the palette's border colour.
func cBorder() css.Rule {
	return css.Raw("border", "1px solid var(--t-border, "+theme.TermPalettes[0].Border+")")
}

// cBorderBottom returns the title bar's dividing rule.
func cBorderBottom() css.Rule {
	return css.Raw("border-bottom", "1px solid var(--t-border, "+theme.TermPalettes[0].Border+")")
}

// cCaret colours the text cursor in the input.
func cCaret() css.Rule {
	return css.Raw("caret-color", "var(--t-accent, "+theme.TermPalettes[0].Accent+")")
}

// applyPalette writes a palette into the custom properties the terminal reads. They go on
// documentElement rather than the terminal frame so the fullscreen scrim and any portalled node
// inherit them too.
func applyPalette(p theme.TermPalette) {
	style := js.Global().Get("document").Get("documentElement").Get("style")
	for name, val := range map[string]string{
		"--t-bg": p.Bg, "--t-bar": p.Bar, "--t-border": p.Border,
		"--t-fg": p.Fg, "--t-dim": p.Dim, "--t-faint": p.Faint,
		"--t-accent": p.Accent, "--t-accent2": p.Accent2,
		"--t-green": p.Green, "--t-red": p.Red,
	} {
		style.Call("setProperty", name, val)
	}
}

// savedPaletteName returns the visitor's stored palette choice, or the default.
func savedPaletteName() string {
	v := js.Global().Get("localStorage").Call("getItem", themeKey)
	if v.Truthy() {
		if _, ok := theme.TermPaletteByName(v.String()); ok {
			return v.String()
		}
	}
	return theme.TermPalettes[0].Name
}

// runTheme implements the `theme` command: with no argument it lists the palettes and marks the
// active one; with a name it switches and persists the choice.
func runTheme(args []string, t *termCtl) {
	if len(args) == 0 {
		active := savedPaletteName()
		rows := []progRow{dim("terminal palettes — `theme <name>` to switch")}
		for _, p := range theme.TermPalettes {
			mark := "  "
			if p.Name == active {
				mark = "● "
			}
			rows = append(rows, kv(mark+p.Name, 150, p.Blurb))
		}
		t.emitRows(rows)
		t.emit(gap())
		return
	}
	name := strings.ToLower(args[0])
	p, ok := theme.TermPaletteByName(name)
	if !ok {
		var names []string
		for _, q := range theme.TermPalettes {
			names = append(names, q.Name)
		}
		t.emitRows([]progRow{{text: "theme: no palette " + name + " — try " + strings.Join(names, ", "), style: rowErr}})
		t.emit(gap())
		return
	}
	applyPalette(p)
	js.Global().Get("localStorage").Call("setItem", themeKey, p.Name)
	t.emitRows([]progRow{row("palette → " + p.Name), dim(p.Blurb)})
	t.emit(gap())
}
