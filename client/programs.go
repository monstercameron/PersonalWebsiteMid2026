//go:build js && wasm

package main

import (
	"strconv"
	"strings"

	"github.com/monstercameron/GoWebComponents/v5/css"
	. "github.com/monstercameron/GoWebComponents/v5/css/u"
	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// Portfolio programs are described as data (progRow) rather than as ui.Nodes, so the same
// definition renders two ways: styled nodes when the program is run on its own, and plain text
// when it appears in a pipeline. Before this the two halves of the terminal were separate modes —
// `projects | grep Go` silently did nothing — and the alternative, a second hand-written text
// renderer per program, would have been copy-paste that drifts.

// rowStyle selects the visual treatment of a progRow's text.
type rowStyle int

const (
	rowPlain rowStyle = iota // default foreground
	rowHead                  // semibold accent — a program's title line
	rowDim                   // muted — captions and hints
	rowErr                   // red — usage errors
)

// progRow is one line of portfolio-program output.
type progRow struct {
	// key is an optional left column (accent-coloured, padded to keyWidth px) for tabular output.
	key      string
	keyWidth int
	// text is the line body. When href is set it renders as a link.
	text string
	href string
	// tail is an optional third column, rendered in the status colour. Padding text into columns
	// with spaces does not survive HTML, which collapses runs of whitespace — a real column needs
	// to be a real element.
	tail  string
	style rowStyle
}

// row builds a plain body-text row.
func row(text string) progRow { return progRow{text: text} }

// head builds a program's title row.
func head(text string) progRow { return progRow{text: text, style: rowHead} }

// dim builds a muted caption row.
func dim(text string) progRow { return progRow{text: text, style: rowDim} }

// kv builds a two-column row with the given key-column width in px.
func kv(key string, width int, text string) progRow {
	return progRow{key: key, keyWidth: width, text: text}
}

// link builds a row whose text is a clickable link to href.
func link(text, href string) progRow { return progRow{text: text, href: href} }

// termProjects is the client-side project list the terminal programs read from. Order mirrors
// internal/content.featured: the first four are the billed case studies, the rest render in the
// site's Labs shelf.
//
// ⚠️ Hand-synced with internal/content.featured and client/vfs.go projectsMD (TODOS §0). Wiring
// these to the real ContentService over gRPC is tracked in TODOS §12 and is the actual fix.
var termProjects = []struct{ id, name, status, blurb, repo string }{
	{"cashflux", "CashFlux", "shipping", "Local-first budgeting suite — all Go/WASM, no JS framework.", "https://github.com/monstercameron/CashFlux"},
	{"articleflux", "ArticleFlux", "shipping", "Self-hosted feed reader, Go all the way down. Real gRPC in the browser.", "https://github.com/monstercameron/ArticleFlux"},
	{"animefeedflux", "AnimeFeedFlux", "v0.2.0", "AI-written anime feeds — trivia, ranked news, roundups — published as RSS on a schedule.", "https://github.com/monstercameron/AnimeFeedFlux"},
	{"gwc", "GoWebComponents", "v5.0.1", "React-style UI framework in Go→WASM. This site runs on it.", "https://github.com/monstercameron/GoWebComponents"},
	{"wasibrowser", "WASIBrowser", "prototype", "A no-JavaScript browser that renders WebAssembly apps.", "https://github.com/monstercameron/WASIBrowser"},
	{"grpcbridge", "GoGRPCBridge", "v1.1.2", "gRPC over WebSockets for the browser — no proxy.", "https://github.com/monstercameron/GoGRPCBridge"},
	{"pathtracer", "WebGL Path Tracer", "demo", "Path tracing live in a browser tab — materials, physics, benchmarks.", "https://github.com/monstercameron/pathtracer"},
	{"semanticscript", "SemanticScript", "research", "An agent-first programming language for LLMs.", "https://github.com/monstercameron/SemanticScript"},
	{"semanticassembly", "SemanticAssembly", "research", "Agent-native RISC-V-first assembly layer.", "https://github.com/monstercameron/SemanticAssembly"},
	{"semanticportrait", "SemanticPortrait", "prototype", "Journaling app that builds a self-portrait graph.", "https://github.com/monstercameron/SemanticPortrait"},
}

// caseStudySlugs are the projects with a written case-study page at /projects/<slug>. Only these
// get a "read the case study" link — pointing the others at a 404 is exactly the dead-end link
// TODOS §13.B is about.
var caseStudySlugs = map[string]bool{"cashflux": true, "articleflux": true, "animefeedflux": true}

// portfolioPrograms is the set of command names that render as styled portfolio programs.
var portfolioPrograms = map[string]bool{
	"help": true, "about": true, "whoami": true, "projects": true, "open": true,
	"neofetch": true, "links": true, "resume": true, "anime": true, "contact": true,
}

// isPortfolio reports whether a command is a styled portfolio program (vs a shell command).
func isPortfolio(name string) bool { return portfolioPrograms[name] }

// programOutput dispatches a command name to its program output rows.
func programOutput(name string, args []string) []progRow {
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
		return neofetchOut()
	case "links":
		return linksOut()
	case "resume":
		return resumeOut()
	case "anime":
		return animeOut()
	case "contact":
		return contactOut()
	default:
		return []progRow{{text: name + ": command not found — try `help`", style: rowErr}}
	}
}

// --- rendering: the same rows, two ways ---

// renderRows renders program rows as styled terminal nodes.
func renderRows(rows []progRow) []ui.Node {
	out := make([]ui.Node, 0, len(rows))
	for _, r := range rows {
		out = append(out, renderRow(r))
	}
	return out
}

// renderRow renders one program row.
func renderRow(r progRow) ui.Node {
	body := rowBody(r)
	if r.key == "" {
		return Div(body)
	}
	// The key column holds its width and never shrinks; the body takes the rest and wraps inside
	// its own column. Letting the row wrap instead put a lone key on one line and its value on the
	// next, which on a phone read as a rendering fault rather than as a table.
	keyCls := []any{cAccent2(), css.Raw("flex-shrink", "0")}
	if r.keyWidth > 0 {
		keyCls = append(keyCls, css.Raw("min-width", strconv.Itoa(r.keyWidth)+"px"))
	}
	// With no third column the body takes the rest of the line. With one, it takes a fixed column
	// instead — letting it grow would shove the status to the far edge of a wide terminal, where
	// it reads as an unrelated word rather than as this row's status.
	bodyCls := []any{css.Raw("flex", "1"), css.Raw("min-width", "0"), css.Raw("overflow-wrap", "anywhere")}
	if r.tail != "" {
		bodyCls = []any{css.Raw("min-width", "190px"), css.Raw("overflow-wrap", "anywhere")}
	}
	cells := []any{Class(Flex, Gap(Spacing3), css.Raw("align-items", "baseline")),
		Span(Class(keyCls...), r.key),
		Span(Class(bodyCls...), body),
	}
	if r.tail != "" {
		cells = append(cells, Span(Class(cGreen(), css.Raw("flex-shrink", "0")), r.tail))
	}
	return Div(cells...)
}

// rowBody renders a row's text span in its style, as a link when href is set.
func rowBody(r progRow) ui.Node {
	if r.href != "" {
		// Internal links (/resume, /projects/…) stay in the tab; external ones open in a new one so
		// a visitor mid-session does not lose the terminal state they built up.
		p := Props{Href: r.href,
			Class: string(css.New(cAccent2(), css.Raw("text-decoration", "underline"),
				css.Raw("text-underline-offset", "3px")))}
		if strings.HasPrefix(r.href, "http") {
			p.Target, p.Rel = "_blank", "noopener noreferrer"
		}
		return A(p, r.text)
	}
	switch r.style {
	case rowHead:
		return Span(Class(FontSemibold, cAccent()), r.text)
	case rowDim:
		return Span(Class(cDim()), r.text)
	case rowErr:
		return Span(Class(cRed()), r.text)
	default:
		return Span(Class(cFg()), r.text)
	}
}

// rowsText renders program rows as plain text, which is what a pipeline consumes. The key column
// is padded so `projects | awk '{print $1}'` and friends see the columns a reader sees.
func rowsText(rows []progRow) string {
	var b strings.Builder
	for _, r := range rows {
		if r.key != "" {
			b.WriteString(padRight(r.key, 18))
			b.WriteString(" ")
		}
		b.WriteString(r.text)
		if r.tail != "" {
			b.WriteString("  " + r.tail)
		}
		if r.href != "" {
			b.WriteString("  " + r.href)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// padRight pads s with spaces to at least n columns.
func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

// --- the programs ---

// helpOut lists the available programs. The easter eggs are deliberately missing: finding them is
// the reward for poking around, and listing them would turn a discovery into a menu item.
func helpOut() []progRow {
	rows := []progRow{dim("available programs")}
	for _, r := range [][2]string{
		{"about", "who I am and how I work"},
		{"projects", "browse featured work"},
		{"open <id>", "a project in detail"},
		{"resume", "summary + the full résumé"},
		{"contact", "send me a message from here"},
		{"links", "github · linkedin · youtube · email"},
		{"tour", "a guided demo — no typing required"},
		{"stats", "live numbers from the server"},
		{"bench", "benchmark this wasm runtime, for real"},
		{"neofetch", "the identity splash"},
		{"anime", "release radar + RSS feeds"},
		{"theme", "switch the terminal palette"},
		{"share", "copy a link that reruns your session"},
		{"ls", "list files — plus ~80 other shell commands"},
		{"ping", "time a real round trip to the server"},
		{"tree", "the filesystem, drawn"},
		{"reset", "restore the filesystem"},
		{"clear", "clear the screen"},
	} {
		rows = append(rows, kv(r[0], 112, r[1]))
	}
	rows = append(rows, dim("↑/↓ history · Tab completes · `man <cmd>` for detail"))
	return rows
}

// aboutOut prints the positioning copy.
func aboutOut() []progRow {
	return []progRow{
		row("I'm an AI-first engineer. Not a 10x savant — I know what to build, and I use"),
		row("every tool I have, LLMs included, to build it well and fast."),
		dim("Deep systems work paired with AI leverage that turns weeks into days."),
	}
}

// projectsOut lists the featured projects.
func projectsOut() []progRow {
	rows := []progRow{dim("featured — `open <id>` for detail")}
	for _, p := range termProjects {
		rows = append(rows, progRow{key: p.id, keyWidth: 150, text: p.name, tail: p.status})
	}
	return rows
}

// openOut prints one project's detail, with real links out to the repo and — where one exists —
// the written case study.
func openOut(args []string) []progRow {
	if len(args) == 0 {
		return []progRow{{text: "usage: open <id>  (try `projects`)", style: rowErr}}
	}
	id := args[0]
	for _, p := range termProjects {
		if p.id != id {
			continue
		}
		rows := []progRow{
			head(p.name + "  [" + p.status + "]"),
			dim(p.blurb),
			link("repo · "+strings.TrimPrefix(p.repo, "https://"), p.repo),
		}
		if caseStudySlugs[p.id] {
			rows = append(rows, link("case study · /projects/"+p.id, "/projects/"+p.id))
		}
		return rows
	}
	suggestion := ""
	for _, p := range termProjects {
		if levenshtein(id, p.id) <= 2 {
			suggestion = "  did you mean `open " + p.id + "`?"
			break
		}
	}
	return []progRow{{text: "open: no project " + id + suggestion, style: rowErr},
		dim("run `projects` for the list")}
}

// linksOut prints the contact links.
func linksOut() []progRow {
	return []progRow{
		kv("github", 72, "github.com/monstercameron"),
		kv("linkedin", 72, "linkedin.com/in/earl-cameron"),
		kv("youtube", 72, "youtube.com/@EarlCameron007"),
		kv("site", 72, "earlcameron.com"),
		kv("email", 72, "cam@earlcameron.com"),
	}
}

// resumeOut prints a résumé summary. Running `resume` on its own also opens the résumé itself —
// see openResume; the rows are what a pipeline and the summary line see.
func resumeOut() []progRow {
	return []progRow{
		head("Earl Cameron — Senior Software Engineer"),
		dim("UKG (2019–present) · Go · C# · React · agents & AI infra · on-device LLMs"),
		link("open the full résumé · /resume", "/resume"),
		dim("or read it here: cat notes/experience.md"),
	}
}

// animeOut prints the anime release-radar RSS feeds (a personal feature; config is owner-gated).
func animeOut() []progRow {
	return []progRow{
		head("anime — release radar"),
		dim("I track shows and publish two RSS feeds anyone can subscribe to:"),
		kv("/anime.xml", 150, "Release Radar — new episodes I'm tracking"),
		kv("/anime/qotd.xml", 150, "Question of the Day — a daily anime prompt"),
	}
}

// contactOut prints the static contact details. The bare `contact` command runs the interactive
// form instead (see startContact); these rows are the pipeline and fallback view.
func contactOut() []progRow {
	return []progRow{
		row("Reach me at cam@earlcameron.com"),
		dim("run `contact` on its own to send a message without leaving the terminal."),
	}
}

// neofetchOut renders the identity splash as rows.
func neofetchOut() []progRow {
	return []progRow{
		{text: "earl cameron", style: rowHead},
		{text: "──────────────────────────────", style: rowDim},
		kv("role", 56, "AI-native systems engineer"),
		kv("stack", 56, "Go · WASM · gRPC · on-device ML"),
		kv("this", 56, "rendered by GoWebComponents (my framework)"),
		kv("wire", 56, "gRPC-over-WebSocket · Go backend"),
		kv("mode", 56, "human judgment × LLM leverage"),
	}
}
