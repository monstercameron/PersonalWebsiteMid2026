//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v5/css"
	. "github.com/monstercameron/GoWebComponents/v5/css/u"
	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/ui"

	"github.com/monstercameron/earlcameron/internal/theme"
)

// Terminal is the interactive macOS-style terminal: it boots, runs a faux shell (portfolio
// programs + ~30 bash commands over a localStorage-cached virtual filesystem), and expands to a
// fullscreen modal.
func Terminal() ui.Node {
	scrollback := ui.UseState([]ui.Node{})
	input := ui.UseState("")
	expanded := ui.UseState(false)
	booted := ui.UseRef(false)
	sh := ui.UseRef[*shell](nil)

	// Seed the shell + boot log once, after mount.
	ui.UseEffect(func() func() {
		if !booted.Get() {
			booted.Set(true)
			sh.Set(newShell())
			scrollback.Set(bootScrollback())
		}
		return nil
	}, true)

	// After the scrollback grows: auto-scroll to the newest line and keep the input focused.
	ui.UseEffect(func() func() {
		doc := js.Global().Get("document")
		if el := doc.Call("getElementById", "term-body"); el.Truthy() {
			el.Set("scrollTop", el.Get("scrollHeight"))
		}
		focusInput()
		return nil
	}, len(scrollback.Get()))

	// Lock the page scroll while the terminal is a fullscreen modal, and put the caret in the input
	// on the way in — expanding without focus leaves a visitor looking at a shell that ignores
	// typing. Runs after the render that applied `expanded`, so the input element is in place.
	ui.UseEffect(func() func() {
		style := js.Global().Get("document").Get("body").Get("style")
		if expanded.Get() {
			style.Set("overflow", "hidden")
			focusInput()
		} else {
			style.Set("overflow", "")
		}
		return func() { style.Set("overflow", "") }
	}, expanded.Get())

	// Escape shrinks the fullscreen modal. The listener is on the document rather than the input so
	// it still works after the visitor has clicked into the scrollback to select text — losing input
	// focus should not trap them in a fullscreen overlay. Bound only while expanded, so Escape means
	// nothing to the inline terminal.
	ui.UseEffect(func() func() {
		if !expanded.Get() {
			return nil
		}
		doc := js.Global().Get("document")
		cb := js.FuncOf(func(_ js.Value, args []js.Value) any {
			if len(args) > 0 && args[0].Get("key").String() == "Escape" {
				expanded.Set(false)
			}
			return nil
		})
		doc.Call("addEventListener", "keydown", cb)
		return func() {
			doc.Call("removeEventListener", "keydown", cb)
			cb.Release()
		}
	}, expanded.Get())

	// Bind the hero's "▶ Launch the live terminal" CTA. It is server-rendered above this component
	// (internal/site.launchCTA), so it sits outside the wasm tree and cannot be given a handler at
	// render time — it has to be reached through the DOM once, on mount. Without this the button is
	// decoration: it has cursor:pointer and does nothing.
	ui.UseEffect(func() func() {
		btn := js.Global().Get("document").Call("getElementById", "term-launch")
		if !btn.Truthy() {
			return nil
		}
		cb := js.FuncOf(func(js.Value, []js.Value) any {
			// Focus here as well as in the effect above: if the terminal is already expanded the
			// state does not change, so that effect will not re-run, and a second click on the CTA
			// should still hand the visitor a live caret.
			expanded.Set(true)
			focusInput()
			return nil
		})
		btn.Call("addEventListener", "click", cb)
		return func() {
			btn.Call("removeEventListener", "click", cb)
			cb.Release()
		}
	}, true)

	onInput := ui.UseEvent(func(e ui.Event) { input.Set(e.GetValue()) })
	onKey := ui.UseEvent(func(e ui.Event) {
		switch e.GetKey() {
		case "Enter":
			e.PreventDefault()
			v := input.Get()
			input.Set("")
			runCommand(v, scrollback, sh.Get())
		case "Tab":
			e.PreventDefault()
			input.Set(complete(input.Get(), sh.Get()))
		}
	})
	onExpand := ui.UseEvent(func() { expanded.Set(true) })
	onShrink := ui.UseEvent(func() { expanded.Set(false) })
	// Clicking anywhere in the terminal body focuses the input — but NOT while the user is
	// selecting text (focusing the input would clear the selection).
	onFocus := ui.UseEvent(func() {
		if sel := js.Global().Get("window").Call("getSelection"); sel.Truthy() && sel.Call("toString").String() != "" {
			return
		}
		focusInput()
	})

	prompt := "~"
	if s := sh.Get(); s != nil {
		prompt = s.prompt()
	}
	return windowFrame(expanded.Get(), onExpand, onShrink, onFocus, prompt, scrollback.Get(), input.Get(), onInput, onKey)
}

// focusInput puts the caret in the terminal input without scrolling the page to it — preventScroll
// matters because the input can be below the fold when the CTA above it is clicked.
func focusInput() {
	if inp := js.Global().Get("document").Call("getElementById", "term-input"); inp.Truthy() {
		inp.Call("focus", map[string]interface{}{"preventScroll": true})
	}
}

// runCommand echoes the command and appends its output. Portfolio commands (help/projects/…)
// render as styled nodes; everything else runs through the shell. `clear` wipes the scrollback.
func runCommand(raw string, scrollback ui.State[[]ui.Node], sh *shell) {
	cmd := strings.TrimSpace(raw)
	if cmd == "" {
		return
	}
	promptStr := "~"
	if sh != nil {
		promptStr = sh.prompt()
	}
	if cmd == "clear" {
		scrollback.Set(nil)
		return
	}
	fields := strings.Fields(cmd)
	var out []ui.Node
	if isPortfolio(fields[0]) && !strings.ContainsAny(cmd, "|>&") {
		out = programOutput(fields[0], fields[1:])
	} else if sh != nil {
		for _, o := range sh.run(cmd) {
			out = append(out, shellLine(o))
		}
	}
	scrollback.Update(func(prev []ui.Node) []ui.Node {
		next := append([]ui.Node{}, prev...)
		next = append(next, echoLine(promptStr, cmd))
		next = append(next, out...)
		next = append(next, gap())
		return next
	})
}

// isPortfolio reports whether a command is a styled portfolio program (vs a shell command).
func isPortfolio(name string) bool {
	switch name {
	case "help", "about", "whoami", "projects", "open", "neofetch", "links", "resume", "anime", "contact":
		return true
	}
	return false
}

// shellLine renders one block of shell output (pre-wrap; red on error).
func shellLine(o outLine) ui.Node {
	c := theme.Fg
	if o.isErr {
		c = theme.Red
	}
	return Div(Class(Fg(c), css.Raw("white-space", "pre-wrap")), strings.TrimRight(o.text, "\n"))
}

// windowFrame renders the terminal window, inline or as a fullscreen modal when expanded.
func windowFrame(expanded bool, onExpand, onShrink, onFocus ui.Handler, prompt string, sb []ui.Node, inputVal string, onInput, onKey ui.Handler) ui.Node {
	rules := []any{Bg(theme.TermBg), Border(theme.TermBorder), Rounded(RadiusXl),
		css.Raw("box-shadow", "0 30px 70px -18px rgba(0,0,0,.9)"), css.Raw("overflow", "hidden"),
		css.Raw("display", "flex"), css.Raw("flex-direction", "column")}
	if expanded {
		// Size purely from the four offsets so width never exceeds the viewport (no width:100%).
		rules = append(rules, css.Raw("position", "fixed"), css.Raw("top", "24px"),
			css.Raw("left", "24px"), css.Raw("right", "24px"), css.Raw("bottom", "24px"),
			css.Raw("z-index", "60"))
		// DESIGN.md §7 calls expand-to-fullscreen the signature transition. A true FLIP from the
		// inline frame to the modal would need measurement before paint; this is the honest cheap
		// version — the modal arrives from slightly under its final size, so the change of state
		// reads as one movement instead of a cut. Collapses to nothing under reduced motion.
		rules = append(rules,
			css.Keyframes("term-open",
				css.At("from", css.Opacity(0), css.Transform(css.Scale(0.985), css.TranslateY(css.Px(6)))),
				css.At("to", css.Opacity(1), css.Transform(css.Scale(1), css.TranslateY(css.Px(0)))),
			),
			css.Animation(css.Ms(220), css.EaseOut),
			ReducedMotion(css.Raw("animation-name", "none")))
	} else {
		// min-width:0 lets the frame shrink below its content's min-content width (nowrap prompt
		// rows + the input) so it never widens the page column on narrow viewports.
		rules = append(rules, WFull, MaxWidth(Px(900)), css.Raw("min-width", "0"))
	}
	frame := Div(Class(rules...),
		titleBar(onExpand, onShrink, expanded),
		termBody(expanded, onFocus, prompt, sb, inputVal, onInput, onKey),
	)
	if expanded {
		scrim := Div(Class(css.Raw("position", "fixed"), css.Raw("inset", "0"),
			css.Raw("background", "rgba(9,3,7,.6)"), css.Raw("z-index", "55"),
			css.Keyframes("scrim-in", css.At("from", css.Opacity(0)), css.At("to", css.Opacity(1))),
			css.Animation(css.Ms(220), css.EaseOut),
			ReducedMotion(css.Raw("animation-name", "none"))))
		return Div(scrim, frame)
	}
	return Div(Class(Flex, JustifyCenter, PadX(Spacing6), PadY(Spacing2)), frame)
}

// titleBar renders the traffic lights (red = shrink, green = expand), title, and status.
func titleBar(onExpand, onShrink ui.Handler, expanded bool) ui.Node {
	return Div(Class(Flex, ItemsCenter, Gap(Spacing2), PadX(Spacing4), PadY(Spacing3), Bg(theme.TermBar),
		css.Raw("border-bottom", "1px solid #38343f")),
		lightGroup(onExpand, onShrink),
		Span(Class(css.Raw("flex", "1"), css.Raw("text-align", "center"), Fg(theme.Dim), FontSize(Rem(0.82))),
			"cameron — zsh — 80×24"),
		Span(Class(Fg(theme.Dim), FontSize(Rem(0.75))),
			Span(Class(Fg(theme.Green)), "● "), pick(expanded, "esc or red to shrink", "live · interactive")),
	)
}

// lightGroup renders the three traffic lights as a hover group: the glyphs stay hidden until the
// pointer is anywhere over the cluster, which is how macOS behaves — the dots read as colour at
// rest and as controls the moment you go for them.
//
// The wrapper carries the literal class "group" because that is the marker u.GroupHover compiles
// against (".group:hover &"); it is concatenated with the generated class rather than set through
// Class() so both survive.
//
// The glyphs describe what these buttons actually do here, which is not the macOS mapping: red
// shrinks the fullscreen modal back down (so it is "−", not "×" — nothing closes) and green expands
// it. Yellow is inert decoration and deliberately gets no glyph, so it never looks clickable.
func lightGroup(onExpand, onShrink ui.Handler) ui.Node {
	wrap := css.New(css.Raw("display", "flex"), css.Raw("align-items", "center"), css.Raw("gap", "8px"))
	return Div(Props{Class: "group " + string(wrap)},
		lightBtn("#ff5f56", "term-shrink", "−", onShrink),
		light("#ffbd2e"),
		lightBtn("#27c93f", "term-expand", "⤢", onExpand),
	)
}

// dotRules are the shared traffic-light dot geometry: a 12px circle that centres its glyph.
func dotRules(hex string) []any {
	return []any{css.Raw("width", "12px"), css.Raw("height", "12px"), css.Raw("border-radius", "50%"),
		css.Raw("display", "flex"), css.Raw("align-items", "center"), css.Raw("justify-content", "center"),
		css.Raw("background", hex)}
}

// light renders a static traffic-light dot.
func light(hex string) ui.Node {
	return Span(Class(dotRules(hex)...))
}

// lightBtn renders a clickable traffic-light dot with an id (for testability) and a glyph that
// fades in while the cluster is hovered.
func lightBtn(hex, id, glyph string, on ui.Handler) ui.Node {
	return Span(Class(append(dotRules(hex), css.Raw("cursor", "pointer"))...),
		Props{ID: id, OnClick: on},
		// Dark ink on the bright dot, sized to sit inside 12px. Hidden at rest; revealed by the
		// parent's hover, not its own, so all the glyphs appear together.
		Span(Class(css.Raw("font-size", "9px"), css.Raw("line-height", "1"), css.Raw("font-weight", "700"),
			css.Raw("color", "rgba(0,0,0,.62)"), css.Raw("opacity", "0"),
			css.Raw("transition", "opacity .12s ease"),
			css.Raw("pointer-events", "none"), css.Raw("user-select", "none"),
			GroupHover(css.Raw("opacity", "1"))),
			glyph),
	)
}

// termBody renders the scrollback and the live input line.
func termBody(expanded bool, onFocus ui.Handler, prompt string, sb []ui.Node, inputVal string, onInput, onKey ui.Handler) ui.Node {
	// overflow-x:auto — lines wider than a narrow viewport scroll inside the terminal (real
	// terminal behavior) instead of being clipped by the frame's overflow:hidden.
	rules := []any{Pad(Spacing5), Flex, FlexCol, Gap(Spacing2), FontSize(Rem(0.95)), LineHeight(Num(1.6)),
		css.Raw("overflow-x", "auto"),
		css.Raw("user-select", "text"), css.Raw("-webkit-user-select", "text")}
	if expanded {
		rules = append(rules, css.Raw("flex", "1"), css.Raw("overflow-y", "auto"))
	} else {
		rules = append(rules, css.Raw("max-height", "460px"), css.Raw("overflow-y", "auto"))
	}
	return Div(Class(rules...), FromProps(Props{ID: "term-body", OnClick: onFocus}), sb, inputLine(prompt, inputVal, onInput, onKey))
}

// inputLine renders the prompt plus the controlled text input.
func inputLine(prompt, value string, onInput, onKey ui.Handler) ui.Node {
	inputCls := css.New(css.Raw("background", "transparent"), css.Raw("border", "none"),
		css.Raw("outline", "none"), css.Raw("color", "#f3e9e6"), css.Raw("flex", "1"),
		css.Raw("min-width", "0"),
		css.Raw("font", "inherit"), css.Raw("margin-left", "8px"), css.Raw("caret-color", "#e95420"))
	return Div(Class(Flex, ItemsCenter),
		promptSigil(prompt),
		Input(Props{ID: "term-input", Class: string(inputCls), Value: value, Placeholder: "type a command…",
			OnInput: onInput, OnKeyDown: onKey}),
	)
}

// promptSigil renders "cameron@portfolio:<cwd>$".
func promptSigil(prompt string) ui.Node {
	return Span(Class(Flex, Gap(Spacing1)),
		Span(Class(Fg(theme.Green)), "cameron@portfolio"),
		Span(Class(Fg(theme.Dim)), ":"),
		Span(Class(Fg(theme.Accent2)), prompt),
		Span(Class(Fg(theme.Accent)), "$"),
	)
}

// echoLine renders the echoed command after the prompt.
func echoLine(prompt, c string) ui.Node {
	return Div(Class(Flex, ItemsCenter, Gap(Spacing1)), promptSigil(prompt), Span(Class(Fg(theme.Fg)), " "+c))
}

// bootScrollback is the initial scrollback: boot log, neofetch, and a welcome line.
func bootScrollback() []ui.Node {
	return []ui.Node{
		bootLine("loading runtime", "wasm · 4.2 mb"),
		bootLine("dialing /socket", "gRPC-over-ws"),
		bootLine("tunnel established", "14 ms"),
		bootLine("mounting /home/cam", "vfs · localStorage"),
		bootLine("session ready", "ok"),
		gap(),
		neofetch(),
		gap(),
		Div(Class(Fg(theme.Faint), FontSize(Rem(0.85))),
			"Welcome. Type ", key("help"), " — recruiters, ", key("cat notes/about.md"), ". Tab completes."),
	}
}

// bootLine renders one "OK label ........ value" boot entry.
func bootLine(label, val string) ui.Node {
	return Div(Class(Flex, ItemsCenter, Gap(Spacing3)),
		Span(Class(Fg(theme.Green), FontSize(Rem(0.7)), css.Raw("border", "1px solid #38343f"),
			css.Raw("border-radius", "4px"), css.Raw("padding", "1px 6px"), css.Raw("letter-spacing", "0.1em")), "OK"),
		Span(Class(FontSemibold, Fg(theme.Fg)), label),
		Span(Class(css.Raw("flex", "1"), css.Raw("border-bottom", "1px dotted #6f5364"),
			css.Raw("height", "0"), css.Raw("transform", "translateY(-3px)"))),
		Span(Class(Fg(theme.Dim)), val),
	)
}

// neofetch renders the identity splash as key/value lines.
func neofetch() ui.Node {
	kv := func(k, v string) ui.Node {
		return Div(Class(Flex, Gap(Spacing3)),
			Span(Class(Fg(theme.Accent2)), k),
			Span(Class(Fg(theme.Fg)), v),
		)
	}
	return Div(Class(Flex, FlexCol, Gap(Spacing1)),
		Span(Class(Fg(theme.Fg), FontSemibold), "earl cameron"),
		Span(Class(Fg(theme.Accent)), "──────────────────────────────"),
		kv("role ", "AI-native systems engineer"),
		kv("stack", "Go · WASM · gRPC · on-device ML"),
		kv("this ", "rendered by GoWebComponents (my framework)"),
		kv("wire ", "gRPC-over-WebSocket · Go backend"),
		kv("mode ", "human judgment × LLM leverage"),
	)
}

// key renders a highlighted command name (visual hint).
func key(s string) ui.Node { return Span(Class(Fg(theme.Accent2)), s) }

// gap renders a blank scrollback line.
func gap() ui.Node { return Div(Class(css.Raw("height", "0.5rem")), " ") }

// pick returns a when cond is true, else b.
func pick(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
