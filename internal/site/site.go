// Package site holds the standard-site UI as GoWebComponents components, styled with GWC's
// typed CSS (css/u). The same components render server-side (SSR failsafe / SEO, via
// RenderHTML) and will hydrate in the browser later. No raw CSS, no JS.
package site

import (
	"github.com/monstercameron/GoWebComponents/v4/css"
	. "github.com/monstercameron/GoWebComponents/v4/css/u"
	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/ui"

	"github.com/monstercameron/earlcameron/internal/theme"
	"github.com/monstercameron/earlcameron/proto/sitepb"
)

// Colors come from the shared token package (internal/theme); see documents/DESIGN.md.

// Page renders the full standard site for the given content.
func Page(_ *sitepb.About, projects []*sitepb.Project) ui.Node {
	return Div(Class(Fg(theme.Fg)),
		center(
			hero(),
			termMount(),
			work(projects),
			how(),
			contact(),
			footer(),
		),
	)
}

// RenderHTML renders the standard site to a complete, self-contained HTML document (markup plus
// the harvested typed-CSS <style> block). Intended to run once at startup over static content.
func RenderHTML(about *sitepb.About, projects []*sitepb.Project) (string, error) {
	markup, err := ui.RenderToString(Page(about, projects))
	if err != nil {
		return "", err
	}
	head := `<!doctype html><html lang="en"><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<title>Earl Cameron — AI-native systems engineer</title>` +
		// Minimal bootstrap glue (permitted): reset, base color, mono font-family, and the ambient
		// glow background. Everything else is typed css/u.
		`<style>*{box-sizing:border-box}html,body{margin:0}body{color:#f3e9e6;` +
		`font-family:ui-monospace,"SF Mono",SFMono-Regular,Menlo,"Cascadia Code","JetBrains Mono",monospace;` +
		`background:radial-gradient(60vw 50vw at 12% -8%,rgba(190,123,230,.16),transparent 60%),` +
		`radial-gradient(55vw 55vw at 105% 115%,rgba(233,84,32,.14),transparent 55%),#17040f;` +
		`background-attachment:fixed}` +
		`#term-body{scrollbar-width:thin;scrollbar-color:#4a4652 transparent}` +
		`#term-body::-webkit-scrollbar{width:9px}` +
		`#term-body::-webkit-scrollbar-track{background:transparent}` +
		`#term-body::-webkit-scrollbar-thumb{background:#4a4652;border-radius:8px}` +
		`#term-body::-webkit-scrollbar-thumb:hover{background:#5c5866}</style>` +
		css.StyleBlock() + `</head><body>`
	// The wasm terminal mounts into #term-root (rendered inside the hero). The two <script> lines
	// are the only JavaScript in the project: the wasm bootstrap glue.
	boot := `<script src="/static/wasm_exec.js"></script>` +
		`<script>const go=new Go();` +
		`WebAssembly.instantiateStreaming(fetch("/static/app.wasm"),go.importObject)` +
		`.then(function(r){go.run(r.instance);})` +
		`.catch(function(e){console.error("wasm boot failed",e);});</script>`
	return head + markup + boot + `</body></html>`, nil
}

// center wraps children in a max-width column, horizontally centered by a flex parent.
func center(children ...ui.Node) ui.Node {
	args := []any{Class(WFull, MaxWidth(Px(1000)), PadX(Spacing6), Flex, FlexCol, Gap(Spacing5))}
	for _, c := range children {
		args = append(args, c)
	}
	return Div(Class(Flex, JustifyCenter), Div(args...))
}

// sansFont is Cam's voice (proportional sans) for prose, contrasted with the mono machine voice.
// font-family isn't in css/u, so use the typed css.Property escape hatch.
var sansFont = css.Raw("font-family", `-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif`)

// hero renders the identity block: eyebrow, headline (with accent), thesis (sans), social links,
// and the launch CTA — matching the mockup. The terminal mounts immediately below (see Page).
func hero() ui.Node {
	return Div(Class(Flex, FlexCol, Gap(Spacing5), PadY(Spacing6)),
		Div(Class(Flex, ItemsCenter, Gap(Spacing3), TextSize(TextSm), Fg(theme.Accent), Tracking(Ems(0.18))),
			Span(Class(css.Raw("width", "26px"), css.Raw("height", "1px"),
				css.Raw("background", "#e95420"), css.Raw("display", "inline-block"))),
			Span("EARL CAMERON · LAUDERHILL, FL"),
		),
		H1(Class(FontSize(Rem(2)), Md(FontSize(Rem(2.9))), FontBold, LineHeight(Num(1.06))),
			Span("AI-native systems engineer. "),
			Span(Class(Fg(theme.Accent)), "I ship "),
			Span("ambitious things, fast."),
		),
		P(Class(sansFont, FontSize(Rem(1.05)), Md(FontSize(Rem(1.2))), Fg(theme.Dim), MaxWidth(Px(620)), LineHeight(Num(1.5))),
			"I pair real systems judgment — Go, WebAssembly, on-device ML, low-level performance — with ",
			Span(Class(Fg(theme.Fg), FontSemibold), "LLMs in the loop"),
			". The result: I design, build, and ship at a pace that used to take a team. ",
			Span(Class(Fg(theme.Fg), FontSemibold), "This whole site is the proof"),
			" — rendered by a UI framework I wrote, talking to a Go backend over a gRPC bridge I built.",
		),
		Div(Class(Flex, Gap(Spacing5), TextSize(TextSm)),
			socialLink("https://github.com/monstercameron", "◆", "github"),
			socialLink("https://www.earlcameron.com", "✦", "earlcameron.com"),
			socialLink("mailto:mr.e.cameron@gmail.com", "✉", "email"),
		),
		launchCTA(),
	)
}

// launchCTA renders the orange call-to-action + pitch that invites using the terminal below.
func launchCTA() ui.Node {
	return Div(Class(Flex, ItemsCenter, Gap(Spacing4), css.Raw("flex-wrap", "wrap")),
		Div(Class(Bg(theme.Accent), Fg(Hex("#ffffff")), FontSemibold, Rounded(RadiusLg), PadX(Spacing5), PadY(Spacing3),
			css.Raw("cursor", "pointer")),
			"▶ Launch the live terminal"),
		Div(Class(sansFont, Fg(theme.Dim), TextSize(TextSm), MaxWidth(Px(380))),
			"Not a screenshot — a real shell wired to a Go backend over gRPC. Every command runs. ",
			Span(Class(Fg(theme.Accent)), "▸ type below to start"),
		),
	)
}

// socialLink renders a glyphed external link.
func socialLink(href, glyph, text string) ui.Node {
	return A(Class(Fg(theme.Dim), Flex, ItemsCenter, Gap(Spacing2)),
		Props{Href: href, Target: "_blank", Rel: "noopener"},
		Span(Class(Fg(theme.Accent2)), glyph), Span(text))
}

// termMount is the element the wasm terminal mounts into, placed right after the hero (the
// mockup's unified hero). No-JS visitors simply see the standard site without it.
func termMount() ui.Node {
	return Div(FromProps(Props{ID: "term-root"}))
}

// work renders the featured-projects section.
func work(projects []*sitepb.Project) ui.Node {
	// Responsive auto-fill grid: as many ~260px columns as fit, collapsing to one on mobile.
	// grid-template-columns isn't in css/u, so use the typed css.Prop escape hatch (not raw CSS).
	cards := []any{Class(Grid, Gap(Spacing4), css.Raw("grid-template-columns", "repeat(auto-fill,minmax(260px,1fr))"))}
	for _, p := range projects {
		cards = append(cards, card(p))
	}
	return Div(Class(Flex, FlexCol, Gap(Spacing5), PadY(Spacing6)),
		label("~/projects · featured"),
		sectionH2("Selected work."),
		Div(cards...),
	)
}

// card renders one project.
func card(p *sitepb.Project) ui.Node {
	tags := []any{Class(Flex, Gap(Spacing2))}
	for _, t := range p.GetTags() {
		tags = append(tags, Span(Class(TextSize(TextSm), Fg(theme.Dim), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing2)), t))
	}
	links := []any{Class(Flex, Gap(Spacing4), TextSize(TextSm)),
		A(Class(Fg(theme.Accent2)), Props{Href: p.GetRepo(), Target: "_blank", Rel: "noopener"}, "code ↗")}
	if p.GetDemo() != "" {
		links = append(links, A(Class(Fg(theme.Accent2)), Props{Href: p.GetDemo(), Target: "_blank", Rel: "noopener"}, "demo ↗"))
	}
	return Div(Class(Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusXl), Pad(Spacing5), Flex, FlexCol, Gap(Spacing3),
		Transition(PropAll, Ms(160), Ease), Hover(Border(theme.Accent))),
		Div(Class(Flex, JustifyBetween, ItemsCenter),
			Span(Class(FontSemibold, Fg(theme.Accent)), p.GetGlyph()+"  "+p.GetName()),
			Span(Class(TextSize(TextSm), Fg(theme.Green)), p.GetStatus()),
		),
		P(Class(Fg(theme.Dim), TextSize(TextSm)), p.GetBlurb()),
		Div(tags...),
		Div(links...),
	)
}

// how renders the "how this site works" architecture section.
func how() ui.Node {
	row := func(n, title, body string) ui.Node {
		return Div(Class(Flex, FlexCol, Gap(Spacing2), Md(FlexRow, Gap(Spacing3)), Pad(Spacing4), Border(theme.Border)),
			Span(Class(Fg(theme.Accent)), n),
			Span(Class(FontSemibold), title),
			Span(Class(Fg(theme.Dim), TextSize(TextSm)), body),
		)
	}
	return Div(Class(Flex, FlexCol, Gap(Spacing5), PadY(Spacing6)),
		label("uname -a · how this site works"),
		sectionH2("The medium is the résumé."),
		Div(Class(Flex, FlexCol, Bg(theme.BgRaised), Rounded(RadiusXl)),
			row("01", "GoWebComponents", "the frontend is Go compiled to WebAssembly, a framework I wrote. Zero npm."),
			row("02", "gRPC bridge", "the browser speaks gRPC-over-WebSocket through GoGRPCBridge. One same-origin socket."),
			row("03", "Go backend", "a real gRPC server streams content, stores messages, powers live status."),
		),
	)
}

// contact renders the contact section (mailto for the no-JS path; the terminal has a live form).
func contact() ui.Node {
	return Div(Class(Flex, FlexCol, Gap(Spacing4), PadY(Spacing6)),
		label("./contact"),
		sectionH2("Let's build something."),
		P(Class(Fg(theme.Dim)),
			"Email me at ",
			A(Class(Fg(theme.Accent)), Props{Href: "mailto:mr.e.cameron@gmail.com"}, "mr.e.cameron@gmail.com"),
			". The terminal has a live contact form over gRPC."),
	)
}

// footer renders the site footer.
func footer() ui.Node {
	return Div(Class(Flex, FlexCol, Gap(Spacing2), Md(FlexRow, JustifyBetween), PadY(Spacing8), Border(theme.Border), Fg(theme.Dim), TextSize(TextSm)),
		Span("built with GoWebComponents · Go · gRPC"),
		Span("© 2026 Earl Cameron"),
	)
}

// sectionH2 renders a responsive section heading (smaller on mobile, larger at the Md breakpoint).
func sectionH2(text string) ui.Node {
	return H2(Class(FontSize(Rem(1.35)), Md(FontSize(Rem(1.6))), FontSemibold), text)
}

// label renders a mono-style section eyebrow.
func label(text string) ui.Node {
	return Div(Class(TextSize(TextSm), Fg(theme.Accent), Tracking(Ems(0.15))), text)
}
