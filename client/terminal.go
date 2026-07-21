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

// Terminal is the interactive macOS-style terminal: it boots, takes typed commands, runs faux
// programs, and can expand into a fullscreen modal (green light / shrink with the red light).
func Terminal() ui.Node {
	scrollback := ui.UseState([]ui.Node{})
	input := ui.UseState("")
	expanded := ui.UseState(false)
	booted := ui.UseRef(false)

	// Seed the boot log + neofetch once, after mount.
	ui.UseEffect(func() func() {
		if !booted.Get() {
			booted.Set(true)
			scrollback.Set(bootScrollback())
		}
		return nil
	}, true)

	// After the scrollback grows: auto-scroll to the newest line and keep the input focused
	// (re-render can drop focus) — like a real terminal. preventScroll avoids page jumps.
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

	// Lock the page scroll while the terminal is a fullscreen modal (it gets its own scroll).
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
		if e.GetKey() == "Enter" {
			e.PreventDefault()
			v := input.Get()
			input.Set("")
			runCommand(v, scrollback)
		}
	})
	onExpand := ui.UseEvent(func() { expanded.Set(true) })
	onShrink := ui.UseEvent(func() { expanded.Set(false) })

	return windowFrame(expanded.Get(), onExpand, onShrink, scrollback.Get(), input.Get(), onInput, onKey)
}

// runCommand echoes the command and appends its program output to the scrollback. `clear` wipes it.
func runCommand(raw string, scrollback ui.State[[]ui.Node]) {
	cmd := strings.TrimSpace(raw)
	if cmd == "" {
		return
	}
	if cmd == "clear" {
		scrollback.Set([]ui.Node{})
		return
	}
	fields := strings.Fields(cmd)
	out := programOutput(fields[0], fields[1:])
	scrollback.Update(func(prev []ui.Node) []ui.Node {
		next := append([]ui.Node{}, prev...)
		next = append(next, echoLine(cmd))
		next = append(next, out...)
		next = append(next, gap())
		return next
	})
}

// windowFrame renders the terminal window, inline or as a fullscreen modal when expanded.
func windowFrame(expanded bool, onExpand, onShrink ui.Handler, sb []ui.Node, inputVal string, onInput, onKey ui.Handler) ui.Node {
	rules := []any{WFull, Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusXl),
		css.Property("box-shadow", "0 40px 90px -40px rgba(0,0,0,.8)"), css.Property("overflow", "hidden"),
		css.Property("display", "flex"), css.Property("flex-direction", "column")}
	if expanded {
		rules = append(rules, css.Property("position", "fixed"), css.Property("inset", "24px"),
			css.Property("z-index", "60"), css.Property("max-height", "calc(100vh - 48px)"))
	} else {
		rules = append(rules, MaxWidth(Px(900)))
	}
	frame := Div(Class(rules...),
		titleBar(onExpand, onShrink, expanded),
		termBody(expanded, sb, inputVal, onInput, onKey),
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
func termBody(expanded bool, sb []ui.Node, inputVal string, onInput, onKey ui.Handler) ui.Node {
	rules := []any{Pad(Spacing5), Flex, FlexCol, Gap(Spacing2), FontSize(Rem(0.95)), LineHeight(Num(1.6))}
	if expanded {
		rules = append(rules, css.Property("flex", "1"), css.Property("overflow-y", "auto"))
	} else {
		rules = append(rules, css.Property("max-height", "460px"), css.Property("overflow-y", "auto"))
	}
	return Div(Class(rules...), FromProps(Props{ID: "term-body"}), sb, inputLine(inputVal, onInput, onKey))
}

// inputLine renders the prompt plus the controlled text input.
func inputLine(value string, onInput, onKey ui.Handler) ui.Node {
	inputCls := css.New(css.Property("background", "transparent"), css.Property("border", "none"),
		css.Property("outline", "none"), css.Property("color", "#f3e9e6"), css.Property("flex", "1"),
		css.Property("font", "inherit"), css.Property("margin-left", "8px"), css.Property("caret-color", "#e95420"))
	return Div(Class(Flex, ItemsCenter),
		promptSigil(),
		Input(Props{ID: "term-input", Class: string(inputCls), Value: value, Placeholder: "type a command…",
			OnInput: onInput, OnKeyDown: onKey}),
	)
}

// promptSigil renders the "cameron@portfolio:~$" prompt.
func promptSigil() ui.Node {
	return Span(Class(Flex, Gap(Spacing1)),
		Span(Class(Fg(theme.Green)), "cameron@portfolio"),
		Span(Class(Fg(theme.Dim)), ":"),
		Span(Class(Fg(theme.Accent2)), "~"),
		Span(Class(Fg(theme.Accent)), "$"),
	)
}

// echoLine renders the echoed command after the prompt.
func echoLine(c string) ui.Node {
	return Div(Class(Flex, ItemsCenter, Gap(Spacing1)), promptSigil(), Span(Class(Fg(theme.Fg)), " "+c))
}

// bootScrollback is the initial scrollback: boot log, neofetch, and a welcome line.
func bootScrollback() []ui.Node {
	return []ui.Node{
		bootLine("loading runtime", "wasm · 4.2 mb"),
		bootLine("dialing /socket", "gRPC-over-ws"),
		bootLine("tunnel established", "14 ms"),
		bootLine("session ready", "ok"),
		gap(),
		neofetch(),
		gap(),
		Div(Class(Fg(theme.Faint), FontSize(Rem(0.85))),
			"Welcome. Type ", key("help"), " to explore — try ", key("projects"), ", ", key("neofetch"), ", ", key("about"), "."),
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
