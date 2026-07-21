//go:build js && wasm

// Command client is the GoWebComponents (Go→WASM) frontend. It mounts the interactive terminal
// over the server-rendered standard site and (next) dials the backend via gRPC-over-WebSocket.
// The only JavaScript in the project is the wasm bootstrap that loads this binary. Build:
// GOOS=js GOARCH=wasm.
package main

import (
	. "github.com/monstercameron/GoWebComponents/v4/css/u"
	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/ui"

	"github.com/monstercameron/earlcameron/internal/theme"
)

// main mounts the terminal into the #term-root element placed by the SSR shell.
func main() {
	ui.Run("#term-root", root)
}

// root is the top-level terminal component. For now it proves the wasm + typed-CSS pipeline is
// live; the interactive engine and gRPC programs land next.
func root() ui.Node {
	return Div(Class(Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusLg), Pad(Spacing4),
		Fg(theme.Fg)),
		Span(Class(Fg(theme.Green)), "cameron@portfolio"),
		Span(Class(Fg(theme.Dim)), ":"),
		Span(Class(Fg(theme.Accent2)), "~"),
		Span(Class(Fg(theme.Accent)), "$ "),
		Span("wasm terminal online — gRPC programs next"),
	)
}
