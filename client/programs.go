//go:build js && wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v4/css"
	. "github.com/monstercameron/GoWebComponents/v4/css/u"
	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/ui"

	"github.com/monstercameron/earlcameron/internal/theme"
)

// termProjects is the faux client-side project list the terminal programs read from.
var termProjects = []struct{ id, name, status, blurb string }{
	{"gwc", "GoWebComponents", "v4.3.0", "React-style UI framework in Go→WASM. This site runs on it."},
	{"cashflux", "CashFlux", "shipping", "Local-first budgeting suite — all Go/WASM, no JS framework."},
	{"wasibrowser", "WASIBrowser", "prototype", "A no-JavaScript browser that renders WebAssembly apps."},
	{"semanticscript", "SemanticScript", "research", "An agent-first programming language for LLMs."},
	{"semanticassembly", "SemanticAssembly", "research", "Agent-native RISC-V-first assembly layer."},
	{"whispertome", "WhisperToMe", "shipping", "Desktop dictation agent running Whisper on the NPU."},
	{"pathtracer", "Vulkan Path Tracer", "demo", "Real-time GPU path tracer with a live browser demo."},
	{"semanticportrait", "SemanticPortrait", "prototype", "Journaling app that builds a self-portrait graph."},
	{"grpcbridge", "GoGRPCBridge", "v0.0.19", "gRPC over WebSockets for the browser — no proxy."},
}

// programOutput dispatches a command name to its faux program output.
func programOutput(name string, args []string) []ui.Node {
	switch name {
	case "help":
		return helpOut()
	case "about", "whoami":
		return aboutOut()
	case "projects":
		return projectsOut()
	case "open":
		return openOut(args)
	case "neofetch":
		return []ui.Node{neofetch()}
	case "links":
		return linksOut()
	case "contact":
		return contactOut()
	default:
		return []ui.Node{Div(Class(Fg(theme.Red)), name+": command not found — try "), key("help")}
	}
}

// helpOut lists the available programs.
func helpOut() []ui.Node {
	rows := [][2]string{
		{"about", "who I am and how I work"},
		{"projects", "browse featured work"},
		{"open <id>", "a project in detail"},
		{"neofetch", "the identity splash"},
		{"links", "github · site · email"},
		{"contact", "how to reach me"},
		{"ls", "list files"},
		{"clear", "clear the screen"},
	}
	out := []ui.Node{Div(Class(Fg(theme.Dim)), "available programs")}
	for _, r := range rows {
		out = append(out, Div(Class(Flex, Gap(Spacing3)),
			Span(Class(Fg(theme.Accent2), css.Property("min-width", "96px")), r[0]),
			Span(Class(Fg(theme.Dim)), r[1]),
		))
	}
	out = append(out, Div(Class(Fg(theme.Faint), css.Property("white-space", "pre-wrap")),
		"…and a faux Unix shell over a virtual filesystem: ls, cat, mkdir, echo>file, grep, find,\nsort, uniq, wc, du · pipes | and && · Tab completes. Try: cat notes/todo.txt | grep TODO"))
	return out
}

// aboutOut prints the positioning copy.
func aboutOut() []ui.Node {
	return []ui.Node{
		Div("I'm an AI-first engineer. Not a 10x savant — I know what to build, and I use"),
		Div("every tool I have, LLMs included, to build it well and fast."),
		Div(Class(Fg(theme.Dim)), "Deep systems work paired with AI leverage that turns weeks into days."),
	}
}

// projectsOut lists the featured projects.
func projectsOut() []ui.Node {
	out := []ui.Node{Div(Class(Fg(theme.Dim)), "featured — `open <id>` for detail")}
	for _, p := range termProjects {
		out = append(out, Div(Class(Flex, Gap(Spacing3)),
			Span(Class(Fg(theme.Accent2), css.Property("min-width", "150px")), p.id),
			Span(Class(Fg(theme.Fg), css.Property("min-width", "180px")), p.name),
			Span(Class(Fg(theme.Green)), p.status),
		))
	}
	return out
}

// openOut prints one project's detail.
func openOut(args []string) []ui.Node {
	if len(args) == 0 {
		return []ui.Node{Div(Class(Fg(theme.Red)), "usage: open <id>  (try `projects`)")}
	}
	for _, p := range termProjects {
		if p.id == args[0] {
			return []ui.Node{
				Div(Class(FontSemibold, Fg(theme.Accent)), p.name+"  ["+p.status+"]"),
				Div(Class(Fg(theme.Dim)), p.blurb),
			}
		}
	}
	return []ui.Node{Div(Class(Fg(theme.Dim)), "opening "+args[0]+"… (nothing to preview — try `projects` or `open gwc`)")}
}

// linksOut prints the contact links.
func linksOut() []ui.Node {
	row := func(k, v string) ui.Node {
		return Div(Class(Flex, Gap(Spacing3)),
			Span(Class(Fg(theme.Accent2), css.Property("min-width", "56px")), k),
			Span(Class(Fg(theme.Dim)), v))
	}
	return []ui.Node{
		row("github", "github.com/monstercameron"),
		row("site", "earlcameron.com"),
		row("email", "mr.e.cameron@gmail.com"),
	}
}

// contactOut prints how to reach Cam.
func contactOut() []ui.Node {
	return []ui.Node{
		Div("Reach me at ", Span(Class(Fg(theme.Accent)), "mr.e.cameron@gmail.com")),
		Div(Class(Fg(theme.Dim)), "or scroll down for the contact section."),
	}
}
