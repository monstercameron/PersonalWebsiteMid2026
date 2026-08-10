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

// idleHintMs is how long the terminal waits, with the visitor having typed nothing, before it
// offers one suggestion in the placeholder. Once — an attract loop is an annoyance, not a hint.
const idleHintMs = 20000

// chipCommands are the one-tap commands under the input. They exist mostly for phones, where a
// bare text field and a "type a command…" placeholder is a bounce: without something to tap, the
// terminal asks for a hardware keyboard the visitor does not have.
var chipCommands = []string{"help", "tour", "projects", "resume", "contact", "stats"}

// Terminal is the interactive macOS-style terminal: it boots, runs a faux shell (portfolio
// programs + ~30 bash commands over a localStorage-cached virtual filesystem), and expands to a
// fullscreen modal.
func Terminal() ui.Node {
	scrollback := ui.UseState([]ui.Node{})
	input := ui.UseState("")
	expanded := ui.UseState(false)
	placeholder := ui.UseState("type a command…")
	promptOverride := ui.UseState("")
	booted := ui.UseRef(false)
	sh := ui.UseRef[*shell](nil)
	capture := ui.UseRef[func(*termCtl, string)](nil)
	gen := ui.UseRef(0)
	bootLive := ui.UseRef(true)
	// pendingCaret carries a caret position that must be applied *after* the render that changed
	// the input's value — assigning value moves the caret to the end, so Ctrl+U and Alt+Backspace
	// would otherwise land the cursor in the wrong place.
	pendingCaret := ui.UseRef(-1)
	typed := ui.UseRef(false)
	// wasExpanded distinguishes "collapsed because the visitor closed the modal" from "collapsed
	// because the page just loaded" — only the first should move focus.
	wasExpanded := ui.UseRef(false)

	t := &termCtl{scrollback: scrollback, input: input, sh: sh.Get(),
		capture: capture, promptOverride: promptOverride, gen: gen, bootLive: bootLive}

	// Seed the shell + boot log once, after mount.
	ui.UseEffect(func() func() {
		if booted.Get() {
			return nil
		}
		booted.Set(true)
		if p, ok := theme.TermPaletteByName(savedPaletteName()); ok {
			applyPalette(p)
		}
		s := newShell()
		sh.Set(s)
		t.sh = s
		scrollback.Set(bootScrollback())
		// The boot log's numbers are measured, not written — see measureBoot. They arrive a moment
		// after the lines they belong to, which is also what a real boot looks like.
		measureBoot(t)
		runDeepLink(t, expanded)
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

	// Apply a caret position requested by a line-editing key, now that the new value is in the DOM.
	ui.UseEffect(func() func() {
		if c := pendingCaret.Get(); c >= 0 {
			setCaret(c)
			pendingCaret.Set(-1)
		}
		return nil
	}, input.Get())

	// Lock the page scroll while the terminal is a fullscreen modal, and put the caret in the input
	// on the way in — expanding without focus leaves a visitor looking at a shell that ignores
	// typing. Runs after the render that applied `expanded`, so the input element is in place.
	// On the way out focus returns to the control that opened it, so a keyboard visitor is not
	// dropped at the top of the document.
	ui.UseEffect(func() func() {
		style := js.Global().Get("document").Get("body").Get("style")
		if expanded.Get() {
			style.Set("overflow", "hidden")
			wasExpanded.Set(true)
			focusInput()
		} else {
			style.Set("overflow", "")
			// Only on the way *back* from fullscreen. This effect also runs on mount, and moving
			// focus because a page finished loading is its own accessibility bug.
			if wasExpanded.Get() {
				wasExpanded.Set(false)
				restoreLaunchFocus()
			}
		}
		return func() { style.Set("overflow", "") }
	}, expanded.Get())

	// Escape shrinks the fullscreen modal, and Tab is trapped inside it. The listener is on the
	// document rather than the input so it still works after the visitor has clicked into the
	// scrollback to select text — losing input focus should not trap them in a fullscreen overlay,
	// and a modal that lets Tab walk out into the page behind it is not a modal.
	ui.UseEffect(func() func() {
		if !expanded.Get() {
			return nil
		}
		doc := js.Global().Get("document")
		cb := js.FuncOf(func(_ js.Value, args []js.Value) any {
			if len(args) == 0 {
				return nil
			}
			switch args[0].Get("key").String() {
			case "Escape":
				expanded.Set(false)
			case "Tab":
				trapTab(args[0])
			}
			return nil
		})
		doc.Call("addEventListener", "keydown", cb)
		return func() {
			doc.Call("removeEventListener", "keydown", cb)
			cb.Release()
		}
	}, expanded.Get())

	// One idle suggestion, if the visitor has not typed and the terminal is actually on screen.
	ui.UseEffect(func() func() {
		after(idleHintMs, func() {
			if typed.Get() || !terminalInView() {
				return
			}
			placeholder.Set("try: tour — a guided demo, no typing")
		})
		return nil
	}, true)

	// Bind the hero's "open it fullscreen" CTA. It is server-rendered above this component
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

	onInput := ui.UseEvent(func(e ui.Event) {
		typed.Set(true)
		input.Set(e.GetValue())
	})
	onKey := ui.UseEvent(func(e ui.Event) {
		handleKey(e, t, scrollback, input, pendingCaret, typed)
	})
	onExpand := ui.UseEvent(func() { expanded.Set(true) })
	onShrink := ui.UseEvent(func() { expanded.Set(false) })
	onChip := ui.UseEvent(func(e ui.Event) {
		cmd := e.JSValue().Get("currentTarget").Get("dataset").Get("cmd")
		if !cmd.Truthy() {
			return
		}
		typed.Set(true)
		input.Set("")
		runCommand(cmd.String(), t)
		focusInput()
	})
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
	return windowFrame(frameArgs{
		expanded: expanded.Get(), onExpand: onExpand, onShrink: onShrink, onFocus: onFocus,
		onChip: onChip, prompt: prompt, override: promptOverride.Get(), sb: scrollback.Get(),
		inputVal: input.Get(), placeholder: placeholder.Get(), onInput: onInput, onKey: onKey,
	})
}

// handleKey implements the input line's key bindings: Enter runs, Tab completes, Up/Down walk the
// history, and the Ctrl chords are the readline edits a shell user reaches for without thinking.
//
// Ctrl+W (kill previous word) is deliberately NOT bound: in Chrome it closes the tab and cannot be
// prevented, so binding it would cost a visitor the whole page. Alt+Backspace — the other standard
// binding for the same edit — is bound instead, and Ctrl+Backspace already does it natively.
func handleKey(e ui.Event, t *termCtl, scrollback ui.State[[]ui.Node], input ui.State[string],
	pendingCaret ui.Ref[int], typed ui.Ref[bool]) {
	raw := e.JSValue()
	key := e.GetKey()
	ctrl := raw.Get("ctrlKey").Bool() || raw.Get("metaKey").Bool()
	alt := raw.Get("altKey").Bool()
	line := input.Get()

	if alt && key == "Backspace" {
		e.PreventDefault()
		next, caret := killPrevWord(line, caretPos(raw))
		pendingCaret.Set(caret)
		input.Set(next)
		return
	}

	if ctrl {
		switch strings.ToLower(key) {
		case "l":
			e.PreventDefault()
			scrollback.Set(nil)
		case "c":
			// Only when nothing is selected — hijacking Ctrl+C over a selection would break copy,
			// and a terminal a visitor cannot copy out of is worse than one without ^C.
			if sel := js.Global().Get("window").Call("getSelection"); sel.Truthy() && sel.Call("toString").String() != "" {
				return
			}
			e.PreventDefault()
			t.gen.Set(t.gen.Get() + 1) // cancel anything still printing
			t.emit(echoLine(livePrompt(t), line+"^C"))
			t.endCapture()
			input.Set("")
		case "a":
			e.PreventDefault()
			setCaret(0)
		case "e":
			e.PreventDefault()
			setCaret(len(line))
		case "u":
			e.PreventDefault()
			next, caret := killToLineStart(line, caretPos(raw))
			pendingCaret.Set(caret)
			input.Set(next)
		}
		return
	}

	switch key {
	case "Enter":
		e.PreventDefault()
		typed.Set(true)
		input.Set("")
		runCommand(line, t)
	case "Tab":
		e.PreventDefault()
		input.Set(complete(line, t.sh))
	case "ArrowUp":
		if t.sh == nil {
			return
		}
		e.PreventDefault()
		if next, ok := t.sh.nav.up(line); ok {
			input.Set(next)
		}
	case "ArrowDown":
		if t.sh == nil {
			return
		}
		e.PreventDefault()
		if next, ok := t.sh.nav.down(); ok {
			input.Set(next)
		}
	}
}

// livePrompt returns whatever the input line is currently prefixed with — the capture label during
// an interactive form, the shell sigil otherwise.
func livePrompt(t *termCtl) string {
	if o := t.promptOverride.Get(); o != "" {
		return o
	}
	if t.sh != nil {
		return t.sh.prompt()
	}
	return "~"
}

// caretPos reads selectionStart off a keyboard event's target, falling back to -1 (end of line).
func caretPos(rawEvent js.Value) int {
	target := rawEvent.Get("target")
	if !target.Truthy() {
		return -1
	}
	s := target.Get("selectionStart")
	if s.Type() != js.TypeNumber {
		return -1
	}
	return s.Int()
}

// setCaret places the caret in the terminal input.
func setCaret(pos int) {
	if inp := js.Global().Get("document").Call("getElementById", "term-input"); inp.Truthy() {
		inp.Call("setSelectionRange", pos, pos)
	}
}

// focusInput puts the caret in the terminal input without scrolling the page to it — preventScroll
// matters because the input can be below the fold when the CTA above it is clicked.
func focusInput() {
	if inp := js.Global().Get("document").Call("getElementById", "term-input"); inp.Truthy() {
		inp.Call("focus", map[string]interface{}{"preventScroll": true})
	}
}

// restoreLaunchFocus returns focus to the hero's launch button after the fullscreen modal closes.
//
// It restores when focus is inside the terminal or has been dropped entirely — collapsing the
// modal unmounts whatever was focused, so the common case is that the browser has fallen back to
// <body> and a keyboard visitor is stranded at the top of the document with no idea where they
// are. It does NOT restore when focus is on some other real element, because then the visitor
// moved it deliberately and yanking it back would be the bug rather than the fix.
func restoreLaunchFocus() {
	doc := js.Global().Get("document")
	active := doc.Get("activeElement")
	frame := doc.Call("getElementById", "term-frame")

	focusDropped := !active.Truthy() || active.Equal(doc.Get("body"))
	insideTerminal := active.Truthy() && frame.Truthy() && frame.Call("contains", active).Bool()
	if !focusDropped && !insideTerminal {
		return
	}
	if btn := doc.Call("getElementById", "term-launch"); btn.Truthy() {
		btn.Call("focus", map[string]interface{}{"preventScroll": true})
	}
}

// trapTab keeps Tab inside the fullscreen modal by wrapping focus at its ends.
func trapTab(rawEvent js.Value) {
	doc := js.Global().Get("document")
	frame := doc.Call("getElementById", "term-frame")
	if !frame.Truthy() {
		return
	}
	nodes := frame.Call("querySelectorAll", "a[href], button, input, [tabindex]:not([tabindex='-1'])")
	n := nodes.Get("length").Int()
	if n == 0 {
		return
	}
	firstEl, lastEl := nodes.Index(0), nodes.Index(n-1)
	active := doc.Get("activeElement")
	back := rawEvent.Get("shiftKey").Bool()
	switch {
	case back && active.Equal(firstEl):
		rawEvent.Call("preventDefault")
		lastEl.Call("focus")
	case !back && active.Equal(lastEl):
		rawEvent.Call("preventDefault")
		firstEl.Call("focus")
	}
}

// terminalInView reports whether the terminal is at least partly on screen, so the idle hint does
// not fire for someone who scrolled past it long ago.
func terminalInView() bool {
	el := js.Global().Get("document").Call("getElementById", "term-frame")
	if !el.Truthy() {
		return false
	}
	r := el.Call("getBoundingClientRect")
	h := js.Global().Get("innerHeight").Float()
	return r.Get("bottom").Float() > 0 && r.Get("top").Float() < h
}

// shellLine renders one block of shell output (pre-wrap; red on error).
func shellLine(o outLine) ui.Node {
	c := cFg()
	if o.isErr {
		c = cRed()
	}
	return Div(Class(c, css.Raw("white-space", "pre-wrap")), strings.TrimRight(o.text, "\n"))
}

// frameArgs bundles what windowFrame needs. It is a struct rather than a parameter list because
// the list had grown past the point where the call site said anything about which argument was
// which.
type frameArgs struct {
	expanded                    bool
	onExpand, onShrink, onFocus ui.Handler
	onChip, onInput, onKey      ui.Handler
	prompt, override, inputVal  string
	placeholder                 string
	sb                          []ui.Node
}

// windowFrame renders the terminal window, inline or as a fullscreen modal when expanded.
func windowFrame(a frameArgs) ui.Node {
	rules := []any{cBg(), cBorder(), Rounded(RadiusXl),
		css.Raw("box-shadow", "0 30px 70px -18px rgba(0,0,0,.9)"), css.Raw("overflow", "hidden"),
		css.Raw("display", "flex"), css.Raw("flex-direction", "column")}
	if a.expanded {
		// Size purely from the four offsets so width never exceeds the viewport (no width:100%).
		rules = append(rules, css.Raw("position", "fixed"), css.Raw("top", "24px"),
			css.Raw("left", "24px"), css.Raw("right", "24px"), css.Raw("bottom", "24px"),
			css.Raw("z-index", "60"))
		// On a phone the 24px inset wastes a quarter of the usable width; the modal goes nearly
		// edge-to-edge instead, which is the difference between a readable line and a wrapped one.
		rules = append(rules, css.Raw("top", "8px"), css.Raw("left", "8px"),
			css.Raw("right", "8px"), css.Raw("bottom", "8px"),
			Md(css.Raw("top", "24px"), css.Raw("left", "24px"),
				css.Raw("right", "24px"), css.Raw("bottom", "24px")))
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
	// role/aria-modal announce the fullscreen state to a screen reader, which otherwise has no way
	// to know the rest of the page went inert.
	frameProps := Props{ID: "term-frame", Aria: map[string]string{"label": "Interactive terminal"}}
	if a.expanded {
		frameProps.Role = "dialog"
		frameProps.Aria["modal"] = "true"
	}
	frame := Div(Class(rules...), FromProps(frameProps),
		titleBar(a.onExpand, a.onShrink, a.expanded),
		termBody(a),
	)
	if a.expanded {
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
	return Div(Class(Flex, ItemsCenter, Gap(Spacing2), PadX(Spacing4), PadY(Spacing3), cBarBg(),
		cBorderBottom()),
		lightGroup(onExpand, onShrink),
		Span(Class(css.Raw("flex", "1"), css.Raw("text-align", "center"), cDim(), FontSize(Rem(0.82)),
			css.Raw("white-space", "nowrap"), css.Raw("overflow", "hidden"),
			css.Raw("text-overflow", "ellipsis")),
			"cameron — zsh — 80×24"),
		// The status tail is noise on a narrow screen, where the title alone already fills the bar.
		Span(Class(cDim(), FontSize(Rem(0.75)), css.Raw("display", "none"), Md(css.Raw("display", "block"))),
			Span(Class(cGreen()), "● "), pick(expanded, "esc or red to shrink", "live · interactive")),
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
		lightBtn("#ff5f56", "term-shrink", "−", "Shrink the terminal", onShrink),
		light("#ffbd2e"),
		lightBtn("#27c93f", "term-expand", "⤢", "Expand the terminal to fullscreen", onExpand),
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
	return Span(Class(dotRules(hex)...), FromProps(Props{Aria: map[string]string{"hidden": "true"}}))
}

// lightBtn renders a clickable traffic-light dot with an id (for testability) and a glyph that
// fades in while the cluster is hovered. It is a real <button> with a label: a coloured <span> is
// invisible to a screen reader and unreachable by keyboard, and these are the window's only
// controls.
func lightBtn(hex, id, glyph, label string, on ui.Handler) ui.Node {
	return Button(Class(append(dotRules(hex), css.Raw("cursor", "pointer"), css.Raw("padding", "0"),
		css.Raw("border", "none"), focusRingTerm())...),
		Props{ID: id, OnClick: on, Aria: map[string]string{"label": label}},
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

// focusRingTerm is the terminal's visible keyboard-focus indicator, drawn only for :focus-visible
// so it never appears on a mouse click. Returned as a slice because css.FocusVisible produces one
// rule per declaration.
func focusRingTerm() []css.Rule {
	return css.FocusVisible(
		css.Raw("outline", "2px solid var(--t-accent, #e95420)"),
		css.Raw("outline-offset", "3px"),
	)
}

// termBody renders the scrollback, the live input line, and the command chips.
func termBody(a frameArgs) ui.Node {
	// overflow-x:auto — lines wider than a narrow viewport scroll inside the terminal (real
	// terminal behavior) instead of being clipped by the frame's overflow:hidden.
	rules := []any{Pad(Spacing4), Md(Pad(Spacing5)), Flex, FlexCol, Gap(Spacing2),
		FontSize(Rem(0.82)), Md(FontSize(Rem(0.95))), LineHeight(Num(1.6)),
		css.Raw("overflow-x", "auto"),
		css.Raw("user-select", "text"), css.Raw("-webkit-user-select", "text")}
	if a.expanded {
		rules = append(rules, css.Raw("flex", "1"), css.Raw("overflow-y", "auto"))
	} else {
		rules = append(rules, css.Raw("max-height", "460px"), css.Raw("overflow-y", "auto"))
	}
	// role=log + aria-live: the scrollback is where every answer appears, and without this a screen
	// reader is told nothing at all when a command produces output. "polite" so it queues behind
	// whatever the visitor is already hearing rather than interrupting them mid-word.
	body := Div(Class(rules...), FromProps(Props{ID: "term-body", OnClick: a.onFocus,
		Role: "log", Aria: map[string]string{"live": "polite", "label": "Terminal output"}}),
		a.sb, inputLine(a))
	return Div(Class(Flex, FlexCol, css.Raw("min-height", "0"), css.Raw("flex", "1")),
		body, chipBar(a.onChip))
}

// chipBar renders the one-tap command chips beneath the input.
func chipBar(onChip ui.Handler) ui.Node {
	chips := []any{Class(Flex, Gap(Spacing2), PadX(Spacing4), PadY(Spacing3), cBarBg(),
		css.Raw("border-top", "1px solid var(--t-border, #38343f)"),
		css.Raw("flex-wrap", "wrap"), css.Raw("align-items", "center")),
		FromProps(Props{Aria: map[string]string{"label": "Suggested commands"}, Role: "group"}),
	}
	for _, c := range chipCommands {
		chips = append(chips, chip(c, onChip))
	}
	return Div(chips...)
}

// chip renders one tappable command. The command travels on a data attribute rather than in a
// closure per chip, so the handler is allocated once for the whole bar.
func chip(cmd string, onChip ui.Handler) ui.Node {
	return Button(Class(cAccent2(), cBorder(), Rounded(RadiusLg), PadX(Spacing3), PadY(Spacing1),
		css.Raw("background", "transparent"), css.Raw("font", "inherit"),
		css.Raw("font-size", "0.78rem"), css.Raw("cursor", "pointer"),
		css.Raw("white-space", "nowrap"), css.Raw("min-height", "32px"),
		Hover(cAccent()), Transition(PropColors, Ms(150), EaseOut),
		ReducedMotion(css.Raw("transition-duration", "0ms"))),
		Props{OnClick: onChip, Data: map[string]string{"cmd": cmd},
			Aria: map[string]string{"label": "Run " + cmd}},
		cmd)
}

// inputLine renders the prompt plus the controlled text input.
func inputLine(a frameArgs) ui.Node {
	inputCls := css.New(css.Raw("background", "transparent"), css.Raw("border", "none"),
		css.Raw("outline", "none"), cFg(), css.Raw("flex", "1"),
		css.Raw("min-width", "0"),
		css.Raw("font", "inherit"), css.Raw("margin-left", "8px"), cCaret())
	sigil := promptSigil(a.prompt)
	if a.override != "" {
		sigil = capturePrompt(a.override)
	}
	return Div(Class(Flex, ItemsCenter),
		sigil,
		// The label is visually hidden rather than absent: "type a command…" is a placeholder, and
		// a placeholder is not an accessible name — it disappears the moment anything is typed.
		Label(Class(srOnly()...), Props{For: "term-input"}, "Terminal command"),
		Input(Props{ID: "term-input", Class: string(inputCls), Value: a.inputVal,
			Placeholder: a.placeholder, OnInput: a.onInput, OnKeyDown: a.onKey,
			AutoComplete: "off", Aria: map[string]string{"label": "Terminal command"}}),
	)
}

// srOnly hides an element visually while leaving it available to assistive technology.
func srOnly() []any {
	return []any{css.Raw("position", "absolute"), css.Raw("width", "1px"), css.Raw("height", "1px"),
		css.Raw("padding", "0"), css.Raw("margin", "-1px"), css.Raw("overflow", "hidden"),
		css.Raw("clip", "rect(0,0,0,0)"), css.Raw("white-space", "nowrap"), css.Raw("border", "0")}
}

// promptSigil renders "cameron@portfolio:<cwd>$".
func promptSigil(prompt string) ui.Node {
	return Span(Class(Flex, Gap(Spacing1), css.Raw("white-space", "nowrap")),
		Span(Class(cGreen(), css.Raw("display", "none"), Md(css.Raw("display", "inline"))), "cameron@portfolio"),
		Span(Class(cDim(), css.Raw("display", "none"), Md(css.Raw("display", "inline"))), ":"),
		Span(Class(cAccent2()), prompt),
		Span(Class(cAccent()), "$"),
	)
}

// capturePrompt renders the interactive-form prompt that replaces the shell sigil while a program
// is reading a line (`contact`).
func capturePrompt(label string) ui.Node {
	return Span(Class(Flex, Gap(Spacing1), css.Raw("white-space", "nowrap")),
		Span(Class(cAccent()), label),
		Span(Class(cDim()), "›"),
	)
}

// echoLine renders the echoed command after the prompt.
func echoLine(prompt, c string) ui.Node {
	return Div(Class(Flex, ItemsCenter, Gap(Spacing1)), promptSigil(prompt), Span(Class(cFg()), " "+c))
}

// capturedLine renders a line submitted to an interactive program, after its field label.
func capturedLine(label, c string) ui.Node {
	return Div(Class(Flex, ItemsCenter, Gap(Spacing1)), capturePrompt(label), Span(Class(cFg()), " "+c))
}

// bootScrollback is the initial scrollback: boot log, neofetch, and a welcome line. The measured
// values arrive shortly after, from measureBoot.
func bootScrollback() []ui.Node {
	return []ui.Node{
		bootLine("loading runtime", ""),
		bootLine("dialing /socket", ""),
		bootLine("mounting /home/cam", "vfs · localStorage"),
		bootLine("session ready", "ok"),
		gap(),
		Div(Class(Flex, FlexCol, Gap(Spacing1)), renderRows(neofetchOut())),
		gap(),
		Div(Class(cFaint(), FontSize(Rem(0.85))),
			"Welcome. Type ", key("help"), ", tap a chip, or run ", key("tour"), " for the guided version."),
	}
}

// bootLine renders one "OK label ........ value" boot entry.
func bootLine(label, val string) ui.Node {
	return Div(Class(Flex, ItemsCenter, Gap(Spacing3)),
		Span(Class(cGreen(), FontSize(Rem(0.7)), cBorder(),
			css.Raw("border-radius", "4px"), css.Raw("padding", "1px 6px"), css.Raw("letter-spacing", "0.1em")), "OK"),
		Span(Class(FontSemibold, cFg()), label),
		Span(Class(css.Raw("flex", "1"), css.Raw("border-bottom", "1px dotted var(--t-faint, #6f5364)"),
			css.Raw("height", "0"), css.Raw("transform", "translateY(-3px)"))),
		Span(Class(cDim()), val),
	)
}

// key renders a highlighted command name (visual hint).
func key(s string) ui.Node { return Span(Class(cAccent2()), s) }

// gap renders a blank scrollback line.
func gap() ui.Node { return Div(Class(css.Raw("height", "0.5rem")), " ") }

// pick returns a when cond is true, else b.
func pick(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
