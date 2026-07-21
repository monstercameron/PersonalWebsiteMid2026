// Package site holds the standard-site UI as GoWebComponents components, styled with GWC's
// typed CSS (css/u). The same components render server-side (SSR failsafe / SEO, via
// RenderHTML) and will hydrate in the browser later. No raw CSS, no JS.
package site

import (
	"github.com/monstercameron/GoWebComponents/v4/css"
	. "github.com/monstercameron/GoWebComponents/v4/css/u"
	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/ui"

	"github.com/monstercameron/earlcameron/proto/sitepb"
)

// Aubergine palette (Ubuntu-souled): deep aubergine ground, warm off-white text, Ubuntu-orange
// accent, purple secondary. See documents/DESIGN.md.
var (
	cBg      = Hex("#17040f")
	cBg2     = Hex("#210a19")
	cFg      = Hex("#f3e9e6")
	cDim     = Hex("#a98ba0")
	cBorder  = Hex("#3a1b2e")
	cAccent  = Hex("#e95420")
	cAccent2 = Hex("#be7be6")
	cGreen   = Hex("#8ae234")
)

// Page renders the full standard site for the given content.
func Page(about *sitepb.About, projects []*sitepb.Project) ui.Node {
	return Div(Class(Bg(cBg), Fg(cFg)),
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
		`<title>Earl Cameron — AI-native systems engineer</title>` + css.StyleBlock() + `</head><body>`
	return head + markup + `</body></html>`, nil
}

// center wraps children in a max-width column, horizontally centered by a flex parent.
func center(children ...ui.Node) ui.Node {
	args := []any{Class(WFull, MaxWidth(Px(1000)), PadX(Spacing6), Flex, FlexCol, Gap(Spacing8))}
	for _, c := range children {
		args = append(args, c)
	}
	return Div(Class(Flex, JustifyCenter), Div(args...))
}

// hero renders the identity block: eyebrow, headline, thesis, and social links.
func hero(about *sitepb.About) ui.Node {
	return Div(Class(Flex, FlexCol, Gap(Spacing4), PadY(Spacing10)),
		Div(Class(TextSize(TextSm), Fg(cAccent), Tracking(Ems(0.2))), "EARL CAMERON · LAUDERHILL, FL"),
		H1(Class(FontSize(Rem(2)), Md(FontSize(Rem(2.8))), FontBold, LineHeight(Num(1.08))), about.GetHeadline()),
		P(Class(FontSize(Rem(1.05)), Md(FontSize(Rem(1.15))), Fg(cDim), MaxWidth(Px(640))), about.GetBody()),
		Div(Class(Flex, Gap(Spacing5), TextSize(TextSm)),
			extLink("https://github.com/monstercameron", "github"),
			extLink("https://www.earlcameron.com", "earlcameron.com"),
			A(Class(Fg(cAccent2)), Props{Href: "mailto:mr.e.cameron@gmail.com"}, "email"),
		),
		Div(Class(TextSize(TextSm), Fg(cDim)),
			"Prefer a shell? This site has a live terminal — enable JavaScript and type ",
			Span(Class(Fg(cAccent)), "help"), "."),
	)
}

// work renders the featured-projects section.
func work(projects []*sitepb.Project) ui.Node {
	cards := []any{Class(Flex, FlexCol, Gap(Spacing4))}
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
		tags = append(tags, Span(Class(TextSize(TextSm), Fg(cDim), Border(cBorder), Rounded(RadiusLg), PadX(Spacing2)), t))
	}
	links := []any{Class(Flex, Gap(Spacing4), TextSize(TextSm)),
		A(Class(Fg(cAccent2)), Props{Href: p.GetRepo(), Target: "_blank", Rel: "noopener"}, "code ↗")}
	if p.GetDemo() != "" {
		links = append(links, A(Class(Fg(cAccent2)), Props{Href: p.GetDemo(), Target: "_blank", Rel: "noopener"}, "demo ↗"))
	}
	return Div(Class(Bg(cBg2), Border(cBorder), Rounded(RadiusXl), Pad(Spacing5), Flex, FlexCol, Gap(Spacing3),
		Transition(PropAll, Ms(160), Ease), Hover(Border(cAccent))),
		Div(Class(Flex, JustifyBetween, ItemsCenter),
			Span(Class(FontSemibold, Fg(cAccent)), p.GetGlyph()+"  "+p.GetName()),
			Span(Class(TextSize(TextSm), Fg(cGreen)), p.GetStatus()),
		),
		P(Class(Fg(cDim), TextSize(TextSm)), p.GetBlurb()),
		Div(tags...),
		Div(links...),
	)
}

// how renders the "how this site works" architecture section.
func how() ui.Node {
	row := func(n, title, body string) ui.Node {
		return Div(Class(Flex, FlexCol, Gap(Spacing2), Md(FlexRow, Gap(Spacing3)), Pad(Spacing4), Border(cBorder)),
			Span(Class(Fg(cAccent)), n),
			Span(Class(FontSemibold), title),
			Span(Class(Fg(cDim), TextSize(TextSm)), body),
		)
	}
	return Div(Class(Flex, FlexCol, Gap(Spacing5), PadY(Spacing6)),
		label("uname -a · how this site works"),
		sectionH2("The medium is the résumé."),
		Div(Class(Flex, FlexCol, Bg(cBg2), Rounded(RadiusXl)),
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
		P(Class(Fg(cDim)),
			"Email me at ",
			A(Class(Fg(cAccent)), Props{Href: "mailto:mr.e.cameron@gmail.com"}, "mr.e.cameron@gmail.com"),
			". The terminal has a live contact form over gRPC."),
	)
}

// footer renders the site footer.
func footer() ui.Node {
	return Div(Class(Flex, FlexCol, Gap(Spacing2), Md(FlexRow, JustifyBetween), PadY(Spacing8), Border(cBorder), Fg(cDim), TextSize(TextSm)),
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
	return Div(Class(TextSize(TextSm), Fg(cAccent), Tracking(Ems(0.15))), text)
}

// extLink renders an external link that opens in a new tab.
func extLink(href, text string) ui.Node {
	return A(Class(Fg(cAccent2)), Props{Href: href, Target: "_blank", Rel: "noopener"}, text)
}
