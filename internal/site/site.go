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
func Page(about *sitepb.About, projects []*sitepb.Project) ui.Node {
	return Div(Class(Bg(theme.Bg), Fg(theme.Fg)),
		center(
			hero(about),
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
		// Minimal bootstrap glue (permitted): reset, base bg/fg to avoid a flash, and the mono
		// font-family the typed CSS system can't express. Everything else is typed css/u.
		`<style>*{box-sizing:border-box}html,body{margin:0}body{background:#17040f;color:#f3e9e6;` +
		`font-family:ui-monospace,"SF Mono",SFMono-Regular,Menlo,"Cascadia Code","JetBrains Mono",monospace}</style>` +
		css.StyleBlock() + `</head><body>`
	// #term-root is where the wasm terminal mounts; the standard-site markup follows as the
	// no-JS failsafe. The two <script> lines are the only JavaScript in the project: the wasm
	// bootstrap glue.
	mount := `<div id="term-root"></div>`
	boot := `<script src="/static/wasm_exec.js"></script>` +
		`<script>const go=new Go();` +
		`WebAssembly.instantiateStreaming(fetch("/static/app.wasm"),go.importObject)` +
		`.then(function(r){go.run(r.instance);})` +
		`.catch(function(e){console.error("wasm boot failed",e);});</script>`
	return head + mount + markup + boot + `</body></html>`, nil
}

// center wraps children in a max-width column, horizontally centered by a flex parent.
func center(children ...ui.Node) ui.Node {
	args := []any{Class(WFull, MaxWidth(Px(1000)), PadX(Spacing6), Flex, FlexCol, Gap(Spacing5))}
	for _, c := range children {
		args = append(args, c)
	}
	return Div(Class(Flex, JustifyCenter), Div(args...))
}

// hero renders the identity block: eyebrow, headline, thesis, and social links.
func hero(about *sitepb.About) ui.Node {
	return Div(Class(Flex, FlexCol, Gap(Spacing4), PadY(Spacing6)),
		Div(Class(TextSize(TextSm), Fg(theme.Accent), Tracking(Ems(0.2))), "EARL CAMERON · LAUDERHILL, FL"),
		H1(Class(FontSize(Rem(2)), Md(FontSize(Rem(2.8))), FontBold, LineHeight(Num(1.08))), about.GetHeadline()),
		P(Class(FontSize(Rem(1.05)), Md(FontSize(Rem(1.15))), Fg(theme.Dim), MaxWidth(Px(640))), about.GetBody()),
		Div(Class(Flex, Gap(Spacing5), TextSize(TextSm)),
			extLink("https://github.com/monstercameron", "github"),
			extLink("https://www.earlcameron.com", "earlcameron.com"),
			A(Class(Fg(theme.Accent2)), Props{Href: "mailto:mr.e.cameron@gmail.com"}, "email"),
		),
		Div(Class(TextSize(TextSm), Fg(theme.Dim)),
			"Prefer a shell? This site has a live terminal — enable JavaScript and type ",
			Span(Class(Fg(theme.Accent)), "help"), "."),
	)
}

// work renders the featured-projects section.
func work(projects []*sitepb.Project) ui.Node {
	// Responsive auto-fill grid: as many ~260px columns as fit, collapsing to one on mobile.
	// grid-template-columns isn't in css/u, so use the typed css.Prop escape hatch (not raw CSS).
	cards := []any{Class(Grid, Gap(Spacing4), css.Property("grid-template-columns", "repeat(auto-fill,minmax(260px,1fr))"))}
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

// extLink renders an external link that opens in a new tab.
func extLink(href, text string) ui.Node {
	return A(Class(Fg(theme.Accent2)), Props{Href: href, Target: "_blank", Rel: "noopener"}, text)
}
