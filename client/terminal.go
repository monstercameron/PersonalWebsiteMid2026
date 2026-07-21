//go:build js && wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v4/css"
	. "github.com/monstercameron/GoWebComponents/v4/css/u"
	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/ui"

	"github.com/monstercameron/earlcameron/internal/theme"
)

// Terminal renders the macOS-style terminal window: chrome (traffic lights + title), a boot
// log, a neofetch identity splash, and a prompt. Static for now (Stage A — the visual); the
// interactive engine and gRPC programs land next.
func Terminal() ui.Node {
	return Div(Class(Flex, JustifyCenter, PadX(Spacing6), PadY(Spacing8)),
		Div(Class(WFull, MaxWidth(Px(900)), Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusXl),
			css.Property("box-shadow", "0 40px 90px -40px rgba(0,0,0,.8)"), css.Property("overflow", "hidden")),
			titleBar(),
			termBody(),
		),
	)
}

// titleBar renders the traffic-light window controls and the session title.
func titleBar() ui.Node {
	return Div(Class(Flex, ItemsCenter, Gap(Spacing2), PadX(Spacing4), PadY(Spacing3),
		css.Property("border-bottom", "1px solid #3a1b2e")),
		light("#ff5f56"), light("#ffbd2e"), light("#27c93f"),
		Span(Class(css.Property("flex", "1"), css.Property("text-align", "center"), Fg(theme.Dim), FontSize(Rem(0.82))),
			"cameron — zsh — 80×24"),
		Span(Class(Fg(theme.Dim), FontSize(Rem(0.75))),
			Span(Class(Fg(theme.Green)), "● "), "live · interactive"),
	)
}

// light renders one traffic-light dot of the given color.
func light(hex string) ui.Node {
	return Span(Class(css.Property("width", "12px"), css.Property("height", "12px"),
		css.Property("border-radius", "50%"), css.Property("display", "inline-block"),
		css.Property("background", hex)))
}

// termBody renders the scrollback: boot log, neofetch, and the prompt line.
func termBody() ui.Node {
	return Div(Class(Pad(Spacing5), Flex, FlexCol, Gap(Spacing2), FontSize(Rem(0.95)), LineHeight(Num(1.6))),
		bootLine("loading runtime", "wasm · 4.2 mb"),
		bootLine("dialing /socket", "gRPC-over-ws"),
		bootLine("tunnel established", "14 ms"),
		bootLine("session ready", "ok"),
		gap(),
		neofetch(),
		gap(),
		promptLine(),
		Div(Class(Fg(theme.Faint), FontSize(Rem(0.85))), "type `help` to explore — interactive terminal coming online"),
	)
}

// bootLine renders one "[ok] label · value" boot entry.
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

// promptLine renders the shell prompt with a blinking-style cursor block.
func promptLine() ui.Node {
	return Div(Class(Flex, ItemsCenter, Gap(Spacing1)),
		Span(Class(Fg(theme.Green)), "cameron@portfolio"),
		Span(Class(Fg(theme.Dim)), ":"),
		Span(Class(Fg(theme.Accent2)), "~"),
		Span(Class(Fg(theme.Accent)), "$"),
		Span(Class(css.Property("width", "9px"), css.Property("height", "1.05em"),
			css.Property("display", "inline-block"), css.Property("margin-left", "6px"),
			css.Property("background", "#e95420"))),
	)
}

// gap renders a blank scrollback line.
func gap() ui.Node { return Div(Class(css.Property("height", "0.5rem")), " ") }
