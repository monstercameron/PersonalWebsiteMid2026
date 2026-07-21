//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v4/css"
	. "github.com/monstercameron/GoWebComponents/v4/css/u"
	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/ui"

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
		if inp := doc.Call("getElementById", "term-input"); inp.Truthy() {
			inp.Call("focus", map[string]interface{}{"preventScroll": true})
		}
		return nil
	}, len(scrollback.Get()))

	// Lock the page scroll while the terminal is a fullscreen modal.
	ui.UseEffect(func() func() {
		style := js.Global().Get("document").Get("body").Get("style")
		if expanded.Get() {
			style.Set("overflow", "hidden")
		} else {
			style.Set("overflow", "")
		}
		return func() { style.Set("overflow", "") }
	}, expanded.Get())

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
	// Clicking anywhere in the terminal body focuses the input (don't make the user hit the line).
	onFocus := ui.UseEvent(func() {
		if inp := js.Global().Get("document").Call("getElementById", "term-input"); inp.Truthy() {
			inp.Call("focus", map[string]interface{}{"preventScroll": true})
		}
	})

	prompt := "~"
	if s := sh.Get(); s != nil {
		prompt = s.prompt()
	}
	return windowFrame(expanded.Get(), onExpand, onShrink, onFocus, prompt, scrollback.Get(), input.Get(), onInput, onKey)
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
	case "help", "about", "whoami", "projects", "open", "neofetch", "links", "contact":
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
	return Div(Class(Fg(c), css.Property("white-space", "pre-wrap")), strings.TrimRight(o.text, "\n"))
}

// windowFrame renders the terminal window, inline or as a fullscreen modal when expanded.
func windowFrame(expanded bool, onExpand, onShrink, onFocus ui.Handler, prompt string, sb []ui.Node, inputVal string, onInput, onKey ui.Handler) ui.Node {
	rules := []any{Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusXl),
		css.Property("box-shadow", "0 40px 90px -40px rgba(0,0,0,.8)"), css.Property("overflow", "hidden"),
		css.Property("display", "flex"), css.Property("flex-direction", "column")}
	if expanded {
		// Size purely from the four offsets so width never exceeds the viewport (no width:100%).
		rules = append(rules, css.Property("position", "fixed"), css.Property("top", "24px"),
			css.Property("left", "24px"), css.Property("right", "24px"), css.Property("bottom", "24px"),
			css.Property("z-index", "60"))
	} else {
		rules = append(rules, WFull, MaxWidth(Px(900)))
	}
	frame := Div(Class(rules...),
		titleBar(onExpand, onShrink, expanded),
		termBody(expanded, onFocus, prompt, sb, inputVal, onInput, onKey),
	)
	if expanded {
		scrim := Div(Class(css.Property("position", "fixed"), css.Property("inset", "0"),
			css.Property("background", "rgba(9,3,7,.6)"), css.Property("z-index", "55")))
		return Div(scrim, frame)
	}
	return Div(Class(Flex, JustifyCenter, PadX(Spacing6), PadY(Spacing2)), frame)
}

// titleBar renders the traffic lights (red = shrink, green = expand), title, and status.
func titleBar(onExpand, onShrink ui.Handler, expanded bool) ui.Node {
	return Div(Class(Flex, ItemsCenter, Gap(Spacing2), PadX(Spacing4), PadY(Spacing3),
		css.Property("border-bottom", "1px solid #3a1b2e")),
		lightBtn("#ff5f56", "term-shrink", onShrink), light("#ffbd2e"), lightBtn("#27c93f", "term-expand", onExpand),
		Span(Class(css.Property("flex", "1"), css.Property("text-align", "center"), Fg(theme.Dim), FontSize(Rem(0.82))),
			"cameron — zsh — 80×24"),
		Span(Class(Fg(theme.Dim), FontSize(Rem(0.75))),
			Span(Class(Fg(theme.Green)), "● "), pick(expanded, "click red to shrink", "live · interactive")),
	)
}

// light renders a static traffic-light dot.
func light(hex string) ui.Node {
	return Span(Class(css.Property("width", "12px"), css.Property("height", "12px"),
		css.Property("border-radius", "50%"), css.Property("display", "inline-block"), css.Property("background", hex)))
}

// lightBtn renders a clickable traffic-light dot with an id (for testability).
func lightBtn(hex, id string, on ui.Handler) ui.Node {
	cls := css.New(css.Property("width", "12px"), css.Property("height", "12px"), css.Property("border-radius", "50%"),
		css.Property("display", "inline-block"), css.Property("cursor", "pointer"), css.Property("background", hex))
	return Span(Props{ID: id, Class: string(cls), OnClick: on})
}

// termBody renders the scrollback and the live input line.
func termBody(expanded bool, onFocus ui.Handler, prompt string, sb []ui.Node, inputVal string, onInput, onKey ui.Handler) ui.Node {
	rules := []any{Pad(Spacing5), Flex, FlexCol, Gap(Spacing2), FontSize(Rem(0.95)), LineHeight(Num(1.6))}
	if expanded {
		rules = append(rules, css.Property("flex", "1"), css.Property("overflow-y", "auto"))
	} else {
		rules = append(rules, css.Property("max-height", "460px"), css.Property("overflow-y", "auto"))
	}
	return Div(Class(rules...), FromProps(Props{ID: "term-body", OnClick: onFocus}), sb, inputLine(prompt, inputVal, onInput, onKey))
}

// inputLine renders the prompt plus the controlled text input.
func inputLine(prompt, value string, onInput, onKey ui.Handler) ui.Node {
	inputCls := css.New(css.Property("background", "transparent"), css.Property("border", "none"),
		css.Property("outline", "none"), css.Property("color", "#f3e9e6"), css.Property("flex", "1"),
		css.Property("font", "inherit"), css.Property("margin-left", "8px"), css.Property("caret-color", "#e95420"))
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
			"Welcome. Type ", key("help"), " — try ", key("projects"), ", ", key("ls"), ", ", key("cat notes/todo.txt"), ". Tab completes."),
	}
}

// bootLine renders one "OK label ........ value" boot entry.
func bootLine(label, val string) ui.Node {
	return Div(Class(Flex, ItemsCenter, Gap(Spacing3)),
		Span(Class(Fg(theme.Green), FontSize(Rem(0.7)), css.Property("border", "1px solid #3a1b2e"),
			css.Property("border-radius", "4px"), css.Property("padding", "1px 6px"), css.Property("letter-spacing", "0.1em")), "OK"),
		Span(Class(FontSemibold, Fg(theme.Fg)), label),
		Span(Class(css.Property("flex", "1"), css.Property("border-bottom", "1px dotted #6f5364"),
			css.Property("height", "0"), css.Property("transform", "translateY(-3px)"))),
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
func gap() ui.Node { return Div(Class(css.Property("height", "0.5rem")), " ") }

// pick returns a when cond is true, else b.
func pick(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
