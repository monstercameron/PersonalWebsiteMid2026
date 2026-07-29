//go:build js && wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v5/css"
	. "github.com/monstercameron/GoWebComponents/v5/css/u"
	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/ui"

	"github.com/monstercameron/earlcameron/internal/theme"
)

// termProjects is the faux client-side project list the terminal programs read from.
// Order mirrors internal/content.featured: the first four are the billed case studies, the rest
// render in the site's Labs shelf.
var termProjects = []struct{ id, name, status, blurb string }{
	{"cashflux", "CashFlux", "shipping", "Local-first budgeting suite — all Go/WASM, no JS framework."},
	{"articleflux", "ArticleFlux", "shipping", "Self-hosted feed reader, Go all the way down. Real gRPC in the browser."},
	{"gwc", "GoWebComponents", "v5.0.1", "React-style UI framework in Go→WASM. This site runs on it."},
	{"wasibrowser", "WASIBrowser", "prototype", "A no-JavaScript browser that renders WebAssembly apps."},
	{"grpcbridge", "GoGRPCBridge", "v1.1.1", "gRPC over WebSockets for the browser — no proxy."},
	{"pathtracer", "WebGL Path Tracer", "demo", "Path tracing live in a browser tab — materials, physics, benchmarks."},
	{"semanticscript", "SemanticScript", "research", "An agent-first programming language for LLMs."},
	{"semanticassembly", "SemanticAssembly", "research", "Agent-native RISC-V-first assembly layer."},
	{"semanticportrait", "SemanticPortrait", "prototype", "Journaling app that builds a self-portrait graph."},
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
	case "resume":
		return resumeOut()
	case "anime":
		return animeOut()
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
		{"links", "github · linkedin · youtube · email"},
		{"resume", "summary + PDF download"},
		{"anime", "release radar + RSS feeds"},
		{"contact", "how to reach me"},
		{"ls", "list files"},
		{"clear", "clear the screen"},
	}
	out := []ui.Node{Div(Class(Fg(theme.Dim)), "available programs")}
	for _, r := range rows {
		out = append(out, Div(Class(Flex, Gap(Spacing3)),
			Span(Class(Fg(theme.Accent2), css.Raw("min-width", "96px")), r[0]),
			Span(Class(Fg(theme.Dim)), r[1]),
		))
	}
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
			Span(Class(Fg(theme.Accent2), css.Raw("min-width", "150px")), p.id),
			Span(Class(Fg(theme.Fg), css.Raw("min-width", "180px")), p.name),
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
			Span(Class(Fg(theme.Accent2), css.Raw("min-width", "56px")), k),
			Span(Class(Fg(theme.Dim)), v))
	}
	return []ui.Node{
		row("github", "github.com/monstercameron"),
		row("linkedin", "linkedin.com/in/earl-cameron"),
		row("youtube", "youtube.com/@EarlCameron007"),
		row("site", "earlcameron.com"),
		row("email", "cam@earlcameron.com"),
	}
}

// resumeOut prints a résumé summary + download link.
func resumeOut() []ui.Node {
	return []ui.Node{
		Div(Class(FontSemibold, Fg(theme.Accent)), "Earl Cameron — Senior Software Engineer"),
		Div(Class(Fg(theme.Dim)), "UKG (2020–present) · Go · C# · React · agents & AI infra · on-device LLMs"),
		Div("full breakdown: ", Span(Class(Fg(theme.Accent2)), "cat notes/experience.md")),
		Div(Class(Fg(theme.Dim)), "read + save as PDF: ", Span(Class(Fg(theme.Accent2)), "/resume")),
	}
}

// animeOut prints the anime release-radar RSS feeds (a personal feature; config is owner-gated).
func animeOut() []ui.Node {
	return []ui.Node{
		Div(Class(FontSemibold, Fg(theme.Accent)), "anime — release radar"),
		Div(Class(Fg(theme.Dim)), "I track shows and publish two RSS feeds anyone can subscribe to:"),
		Div(Class(Flex, Gap(Spacing3)),
			Span(Class(Fg(theme.Accent2), css.Raw("min-width", "150px")), "/anime.xml"),
			Span(Class(Fg(theme.Dim)), "Release Radar — new episodes I'm tracking"),
		),
		Div(Class(Flex, Gap(Spacing3)),
			Span(Class(Fg(theme.Accent2), css.Raw("min-width", "150px")), "/anime/qotd.xml"),
			Span(Class(Fg(theme.Dim)), "Question of the Day — a daily anime prompt"),
		),
	}
}

// contactOut prints how to reach Cam.
func contactOut() []ui.Node {
	return []ui.Node{
		Div("Reach me at ", Span(Class(Fg(theme.Accent)), "cam@earlcameron.com")),
		Div(Class(Fg(theme.Dim)), "or scroll down for the contact section."),
	}
}
