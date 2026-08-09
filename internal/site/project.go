package site

// The Flux project pages — one dedicated case-study page per branded product, at /projects/<slug>.
//
// # The design problem, and the decision
//
// The five Flux products have their own commissioned artwork, and that artwork is bright, white
// and unmistakably SaaS-marketing. This site is dark aubergine and deliberately reads as a
// terminal (DESIGN.md §6). Dropping a white poster onto that page would either shatter the site's
// identity or bleach the artwork into the background, and re-skinning the site per product would
// mean a recruiter clicking a project card no longer knows whose portfolio they are on.
//
// So the page does neither. It stays Aubergine, and it presents the poster as what it actually
// is — the product's marketing sheet — mounted inside a macOS terminal window, on its own light
// plate, captioned. That is the signature element of these pages, and it is the site's own
// argument stated in one image: the polished product artwork and the shell it was built in, in
// the same frame. It reuses the terminal chrome the site already owns rather than inventing a
// second visual vocabulary for the same idea.
//
// # What changes per product, and what does not
//
// Changes: one accent colour and the ambient background glow, both sampled from that product's
// poster (theme.Brands). That is enough for ArticleFlux-blue and CashFlux-green to be legible as
// different products before a word is read.
//
// Does not change: the ground, the surfaces, the borders, the two type voices, and the single
// `rise` motion gesture. Five products, one design language — which is the difference between a
// system and five one-off pages.
//
// # Structure
//
// The section order answers TODOS §14.G in order: what it is → what it does → what it is built
// on → what was hard → the evidence → how honest the whole thing is. The eyebrows are shell paths
// (`~/projects/cashflux · built on`) because this site has a real `projects` program and those
// paths are true, not decoration. There is deliberately no 01/02/03 numbering: the sections are
// facets of one argument, not a sequence, and numbering them would claim an order the content
// does not have.

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/v5/css"
	. "github.com/monstercameron/GoWebComponents/v5/css/u"
	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/ui"

	"github.com/monstercameron/earlcameron/internal/content"
	"github.com/monstercameron/earlcameron/internal/theme"
)

// ProjectPage renders one Flux project page. brand carries the per-product accent; it is looked
// up by the caller so an unknown slug cannot reach here with a zero-value palette.
func ProjectPage(p content.FluxProject, brand theme.Brand, siblings []content.FluxProject) ui.Node {
	return Div(Class(Fg(theme.Fg)),
		center(
			projectNav(p.Name),
			projectHero(p, brand),
			brandPlate(p, brand),
			capabilities(p, brand),
			builtOn(p, brand),
			hardParts(p, brand),
			evidence(p, brand),
			honesty(p, brand),
			siblingNav(p, siblings),
			footer(),
		),
	)
}

// RenderProjectHTML renders a project page to a complete HTML document.
//
// The ambient glow in the bootstrap <style> is the product's, not the site's: it is the one place
// the brand colour touches the page ground, and it is what makes the page feel like a different
// product without being a different site. Everything else in this head block matches
// RenderHTML — same reset, same mono stack, same fixed attachment.
func RenderProjectHTML(p content.FluxProject, brand theme.Brand, siblings []content.FluxProject, baseURL string) (string, error) {
	markup, err := ui.RenderToString(ProjectPage(p, brand, siblings))
	if err != nil {
		return "", err
	}
	base := strings.TrimRight(baseURL, "/")
	url := base + "/projects/" + p.Slug
	// The description is the lede, which is already written to be the one jargon-free sentence a
	// non-specialist can repeat — exactly the job of a search snippet and an unfurl card.
	title := p.Name + " — " + p.Tagline + " · Earl Cameron"
	head := `<!doctype html><html lang="en"><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<title>` + esc(title) + `</title>` +
		`<meta name="description" content="` + esc(p.Lede) + `">` +
		`<meta property="og:title" content="` + esc(title) + `">` +
		`<meta property="og:description" content="` + esc(p.Lede) + `">` +
		`<meta property="og:type" content="article">` +
		`<meta property="og:url" content="` + esc(url) + `">` +
		`<meta property="og:image" content="` + esc(base+"/static/brand/"+p.Slug+"-og.jpg") + `">` +
		`<meta name="twitter:card" content="summary_large_image">` +
		`<link rel="canonical" href="` + esc(url) + `">` +
		`<style>*{box-sizing:border-box}html,body{margin:0}body{color:#f3e9e6;` +
		`font-family:ui-monospace,"SF Mono",SFMono-Regular,Menlo,"Cascadia Code","JetBrains Mono",monospace;` +
		`background:radial-gradient(62vw 52vw at 10% -10%,` + brand.Glow + `,transparent 62%),` +
		`radial-gradient(55vw 55vw at 105% 115%,rgba(233,84,32,.10),transparent 55%),#17040f;` +
		`background-attachment:fixed}</style>` +
		css.StyleBlock() + `</head><body>`
	return head + markup + `</body></html>`, nil
}

// esc escapes the few characters that can break out of an HTML attribute or text node. The page
// content is authored in this repository rather than submitted by anyone, so this is defence in
// depth against a future edit, not a filter on hostile input.
func esc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

// brandLabel is the section eyebrow, in the product's colour rather than the site's orange.
//
// The home page's label() prints its eyebrows in Ubuntu orange, which is right there — orange is
// the site's voice. On a product page it is wrong: the eyebrows are the loudest repeated element,
// and leaving them orange put the one colour on the page that had nothing to do with the product
// in the position of most emphasis, fighting the green on CashFlux and the blue on ArticleFlux.
// The orange survives where it belongs — the `~/earlcameron` mark in the nav, which is the site
// signing the page.
func brandLabel(text string, brand theme.Brand) ui.Node {
	return Div(Class(TextSize(TextSm), Fg(brand.Accent), Tracking(Ems(0.15))), text)
}

// projectNav is the page's own top bar: the way back to the portfolio, plus the one link a
// visitor who arrived here from a search result needs. It is deliberately thinner than the home
// page's topNav — a project page has one parent and does not need the full site index.
func projectNav(name string) ui.Node {
	return Div(Class(Flex, ItemsCenter, JustifyBetween, Gap(Spacing3), PadY(Spacing4), css.Raw("flex-wrap", "wrap")),
		A(Class(FontSemibold, Fg(theme.Accent), css.Raw("text-decoration", "none"), focusRing()),
			Props{Href: "/"}, "~/earlcameron"),
		Div(Class(Flex, Gap(Spacing4), ItemsCenter, TextSize(TextSm), css.Raw("flex-wrap", "wrap")),
			A(Class(Fg(theme.Dim), Hover(Fg(theme.Accent)), Transition(PropColors, Ms(150), EaseOut),
				ReducedMotion(css.Raw("transition-duration", "0ms")), focusRing()),
				Props{Href: "/#work"}, "← all work"),
			A(Class(Fg(theme.Dim), Hover(Fg(theme.Accent)), focusRing()), Props{Href: "/resume"}, "résumé"),
			A(Class(Fg(theme.Dim), Hover(Fg(theme.Accent)), focusRing()), Props{Href: "/#contact"}, "contact"),
		),
		Span(Class(css.Raw("position", "absolute"), css.Raw("left", "-9999px")), name),
	)
}

// projectHero is the identity block: the product's mark, its wordmark, its own tagline, the
// one-sentence lede, and the two links that matter.
//
// The wordmark is set in the site's mono face and split two-tone at the same break the poster
// uses — "Cash" in the page's text colour, "Flux" in the product's accent. That reproduces the
// brand lockup's structure without importing its typeface, which would put a fifth font on a site
// whose whole type argument is two system voices (DESIGN.md §6).
func projectHero(p content.FluxProject, brand theme.Brand) ui.Node {
	return Div(Class(Flex, FlexCol, Gap(Spacing5), PadY(Spacing5)),
		Div(Class(append([]any{Flex, ItemsCenter, Gap(Spacing4), css.Raw("flex-wrap", "wrap")}, riseOnLoad(0)...)...),
			Img(Class(css.Raw("width", "64px"), css.Raw("height", "auto"), css.Raw("flex", "0 0 auto")),
				Props{Src: "/static/brand/" + p.Slug + "-mark.webp", Alt: p.Name + " mark",
					Width: "64", Loading: "eager"}),
			H1(Class(FontSize(Rem(2)), Md(FontSize(Rem(3))), FontBold, LineHeight(Num(1)),
				Tracking(Ems(-0.02)), css.Raw("margin", "0")),
				Span(p.Head), Span(Class(Fg(brand.Accent)), p.Tail),
			),
		),
		Div(Class(append([]any{Flex, ItemsCenter, Gap(Spacing3), css.Raw("flex-wrap", "wrap")}, riseOnLoad(70)...)...),
			P(Class(sansFont, FontSize(Rem(1.15)), Md(FontSize(Rem(1.35))), Fg(theme.Fg), css.Raw("margin", "0")),
				p.Tagline),
			statusChip(p.Status, brand),
		),
		P(Class(append([]any{sansFont, FontSize(Rem(1.02)), Fg(theme.Dim), MaxWidth(Px(660)), LineHeight(Num(1.55)),
			css.Raw("margin", "0")}, riseOnLoad(140)...)...), p.Lede),
		P(Class(append([]any{sansFont, TextSize(TextSm), Fg(theme.Dim), MaxWidth(Px(660)), LineHeight(Num(1.65)),
			css.Raw("margin", "0")}, riseOnLoad(200)...)...), p.Pitch),
		Div(append([]any{Class(append([]any{Flex, Gap(Spacing3), ItemsCenter, css.Raw("flex-wrap", "wrap")},
			riseOnLoad(260)...)...)}, ctaButtons(p, brand)...)...),
	)
}

// ctaButtons returns the hero's action row. A project with a live demo gets both buttons; one
// without gets only the repository button, because rendering the same destination twice side by
// side reads as a broken template rather than emphasis.
func ctaButtons(p content.FluxProject, brand theme.Brand) []any {
	if p.Demo == "" {
		return []any{ctaPrimary(p, brand)}
	}
	return []any{ctaPrimary(p, brand), ctaSecondary(p.Repo, "Read the code ↗")}
}

// ctaPrimary is the live demo where one exists, and the repository where one does not. A page
// that offers "Live demo" and then 404s is worse than a page that never claimed to have one, so
// the label always names what is actually on the other end.
func ctaPrimary(p content.FluxProject, brand theme.Brand) ui.Node {
	href, label := p.Demo, p.DemoLabel
	if href == "" {
		href, label = p.Repo, "Read the code ↗"
		// The secondary button would then duplicate this one; hardParts and the stack rows still
		// link the repo, so the row simply carries one button.
	}
	return A(Class(Bg(brand.Accent), Fg(Hex("#0d0209")), FontSemibold, Rounded(RadiusLg),
		PadX(Spacing5), PadY(Spacing3), css.Raw("text-decoration", "none"),
		Transition(PropAll, Ms(180), EaseOut), Hover(css.Transform(css.TranslateY(css.Px(-1)))),
		ReducedMotion(css.Raw("transition-duration", "0ms")), focusRing()),
		Props{Href: href, Target: "_blank", Rel: "noopener"}, label)
}

// ctaSecondary is the outlined companion button. It renders nothing when its href duplicates the
// primary — see ctaPrimary.
func ctaSecondary(href, label string) ui.Node {
	return A(Class(Border(theme.Border), Fg(theme.Fg), FontSemibold, Rounded(RadiusLg),
		PadX(Spacing5), PadY(Spacing3), css.Raw("text-decoration", "none"),
		Hover(Border(theme.Accent), css.Transform(css.TranslateY(css.Px(-1)))),
		Transition(PropAll, Ms(180), EaseOut), ReducedMotion(css.Raw("transition-duration", "0ms")),
		focusRing()),
		Props{Href: href, Target: "_blank", Rel: "noopener"}, label)
}

// statusChip prints the maturity label beside the tagline rather than in a footnote. Maturity is
// the first thing that would undermine every other claim on the page if a reader found it
// somewhere else, so it sits in the hero.
func statusChip(status string, brand theme.Brand) ui.Node {
	return Span(Class(monoFont, TextSize(TextSm), Fg(brand.Tint), Border(brand.Accent),
		Rounded(RadiusFull), PadX(Spacing3), PadY(Spacing1), css.Raw("white-space", "nowrap")),
		status)
}

// brandPlate is the signature element: the product's marketing poster, on the light ground it was
// drawn for, inside the macOS terminal chrome this site already uses for its shell.
//
// The frame is doing argument, not decoration. The whole portfolio's thesis is that the polished
// thing and the machine underneath it are the same work; this puts both in one image. The title
// bar carries the repository path, so the frame also labels what the artwork is — a brand sheet
// for a real repository — which keeps the page honest about the fact that a marketing poster is
// not itself evidence of anything.
func brandPlate(p content.FluxProject, brand theme.Brand) ui.Node {
	dot := func(hex string) ui.Node {
		return Span(Class(css.Raw("width", "11px"), css.Raw("height", "11px"),
			css.Raw("border-radius", "50%"), css.Raw("background", hex),
			css.Raw("display", "inline-block"), css.Raw("flex", "0 0 auto")))
	}
	return Div(Class(sectionRules(Gap(Spacing3))...),
		Div(Class(Rounded(RadiusXl), css.Raw("overflow", "hidden"), Border(theme.TermBorder),
			css.Raw("box-shadow", "0 30px 80px -40px rgba(0,0,0,.9)")),
			Div(Class(Flex, ItemsCenter, Gap(Spacing2), Bg(theme.TermBar), PadX(Spacing3), PadY(Spacing2)),
				dot("#ff5f57"), dot("#febc2e"), dot("#28c840"),
				Span(Class(monoFont, TextSize(TextSm), Fg(Hex("#9c98a6")), PadX(Spacing3),
					css.Raw("overflow", "hidden"), css.Raw("text-overflow", "ellipsis"),
					css.Raw("white-space", "nowrap")),
					"monstercameron/"+p.Name+" — brand sheet"),
			),
			Div(Class(Bg(theme.Plate)),
				Img(Class(css.Raw("display", "block"), css.Raw("width", "100%"), css.Raw("height", "auto")),
					Props{Src: "/static/brand/" + p.Slug + "-poster.webp",
						Alt:     p.Name + " brand poster — " + p.Tagline,
						Width:   "1200", Height: "900",
						Loading: "lazy"}),
			),
		),
		P(Class(sansFont, TextSize(TextSm), Fg(theme.Faint), LineHeight(Num(1.6)), css.Raw("margin", "0")),
			"Brand artwork for ", Span(Class(Fg(brand.Tint)), p.Name),
			". The product is real and the code is linked above; the poster is a design exercise, not a claim."),
	)
}

// capabilities answers "what does it do" in the reader's vocabulary — what a person controls and
// recognizes, never how the system is built. The build details get their own section below.
func capabilities(p content.FluxProject, brand theme.Brand) ui.Node {
	// Two-up at the 1000px content width, not three: four capabilities in a three-column grid
	// leave a ragged single card on the second row, and the wider column is what lets each card
	// carry a real sentence instead of a fragment.
	cards := []any{Class(Grid, Gap(Spacing3),
		css.Raw("grid-template-columns", "repeat(auto-fit,minmax(min(420px,100%),1fr))"))}
	for _, c := range p.Capabilities {
		cards = append(cards, Div(Class(append([]any{Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusXl),
			Pad(Spacing5), Flex, FlexCol, Gap(Spacing2), Hover(Border(brand.Accent))}, lift()...)...),
			H3(Class(FontSemibold, Fg(theme.Fg), FontSize(Rem(1.02)), css.Raw("margin", "0")), c.Title),
			P(Class(sansFont, TextSize(TextSm), Fg(theme.Dim), LineHeight(Num(1.6)), css.Raw("margin", "0")), c.Body),
		))
	}
	return Div(Class(sectionRules(Gap(Spacing5))...),
		brandLabel("~/projects/"+p.Slug+" · what it does", brand),
		sectionH2("What you get."),
		Div(cards...),
	)
}

// builtOn is the section Cam asked for by name: the underlying libraries, said out loud.
//
// The pieces he wrote are marked rather than merely listed. On a portfolio the difference between
// "uses gRPC" and "wrote the gRPC-over-WebSocket tunnel this depends on" is the entire point, and
// a reader skimming a technology list will not infer it.
func builtOn(p content.FluxProject, brand theme.Brand) ui.Node {
	rows := []any{Class(Flex, FlexCol, Bg(theme.BgRaised), Rounded(RadiusXl), css.Raw("overflow", "hidden"))}
	for i, s := range p.Stack {
		name := ui.Node(Span(Class(FontSemibold, Fg(theme.Fg)), s.Name))
		if s.Href != "" {
			name = A(Class(FontSemibold, Fg(brand.Tint), Hover(Fg(brand.Accent)),
				css.Raw("text-decoration", "none"), Transition(PropColors, Ms(150), EaseOut),
				ReducedMotion(css.Raw("transition-duration", "0ms")), focusRing()),
				Props{Href: s.Href, Target: "_blank", Rel: "noopener"}, s.Name+" ↗")
		}
		head := []any{Class(Flex, ItemsCenter, Gap(Spacing2), css.Raw("flex-wrap", "wrap"),
			css.Raw("min-width", "0"), Md(css.Raw("flex", "0 0 240px"))), name}
		if s.Mine {
			head = append(head, Span(Class(monoFont, TextSize(TextSm), Fg(theme.Accent),
				Border(theme.Accent), Rounded(RadiusFull), PadX(Spacing2),
				css.Raw("white-space", "nowrap")), "mine"))
		}
		// The hairline goes in the class list, not the child list. Appending a css.Rule to the
		// element's children renders it as literal text — it is styling, and a Div's variadic
		// arguments treat anything that is not a Node or a PropOption as content.
		cls := []any{Flex, FlexCol, Gap(Spacing2), Md(FlexRow, ItemsStart, Gap(Spacing4)), Pad(Spacing4)}
		if i > 0 {
			cls = append(cls, css.BorderTop(css.Px(1), theme.Border))
		}
		rows = append(rows, Div(Class(cls...), Div(head...),
			Span(Class(sansFont, TextSize(TextSm), Fg(theme.Dim), LineHeight(Num(1.6)), css.Raw("flex", "1")), s.Why)))
	}
	return Div(Class(sectionRules(Gap(Spacing5))...),
		brandLabel("~/projects/"+p.Slug+" · built on", brand),
		headed("What it runs on.",
			"Including the two libraries underneath it that are my own work — the framework the interface "+
				"is written in, and the transport it talks over."),
		Div(rows...),
	)
}

// hardParts is the section a hiring manager actually reads: two or three real problems, with the
// measurement or the decision that resolved each. Not a technology list — those are above.
func hardParts(p content.FluxProject, brand theme.Brand) ui.Node {
	items := []any{Class(Flex, FlexCol, Gap(Spacing4))}
	for _, h := range p.Hard {
		items = append(items, Div(Class(Flex, FlexCol, Gap(Spacing2), Pad(Spacing5),
			Bg(theme.BgRaised), Rounded(RadiusXl),
			css.BorderLeft(css.Px(3), brand.Accent)),
			H3(Class(FontSemibold, Fg(theme.Fg), FontSize(Rem(1.02)), LineHeight(Num(1.4)), css.Raw("margin", "0")), h.Title),
			P(Class(sansFont, TextSize(TextSm), Fg(theme.Dim), LineHeight(Num(1.65)), css.Raw("margin", "0")), h.Body),
		))
	}
	return Div(Class(sectionRules(Gap(Spacing5))...),
		brandLabel("~/projects/"+p.Slug+" · hard parts", brand),
		headed("What was actually difficult.",
			"The problems worth describing, and what the measurement said. Not the technology list — that is above."),
		Div(items...),
	)
}

// evidence prints the countable facts. Every figure here is counted from the repository; none is
// estimated, and lines of code is deliberately absent because it rewards verbosity.
func evidence(p content.FluxProject, brand theme.Brand) ui.Node {
	cells := []any{Class(Grid, Gap(Spacing3),
		css.Raw("grid-template-columns", "repeat(auto-fit,minmax(min(160px,100%),1fr))"))}
	for _, f := range p.Evidence {
		cells = append(cells, Div(Class(append([]any{Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusLg),
			Pad(Spacing4), Flex, FlexCol, Gap(Spacing1)}, lift()...)...),
			Span(Class(FontSize(Rem(1.45)), FontBold, Fg(brand.Tint), LineHeight(Num(1.1))), f.Figure),
			Span(Class(sansFont, TextSize(TextSm), Fg(theme.Dim), LineHeight(Num(1.4))), f.Label),
		))
	}
	return Div(Class(sectionRules(Gap(Spacing5))...),
		brandLabel("~/projects/"+p.Slug+" · evidence", brand),
		headed("Counted, not estimated.",
			"Figures taken from the repository itself. No line-of-code count — it rewards duplication and every reader knows it."),
		Div(cells...),
	)
}

// honesty is the disclosure section: spare-time, AI-assisted, and this project's real maturity.
//
// It is a full section rather than a footnote on purpose. A recruiter who works out on their own
// that a "platform" is an alpha discounts everything else on the page; a recruiter who is told
// plainly reads the rest as credible. Framed as the claim it actually is — this much shipped
// surface area, by one person, in spare time — it is the strongest section here, not the weakest.
func honesty(p content.FluxProject, brand theme.Brand) ui.Node {
	return Div(Class(sectionRules(Gap(Spacing4))...),
		brandLabel("~/projects/"+p.Slug+" · honest status", brand),
		sectionH2("Where this really is."),
		Div(Class(Flex, FlexCol, Gap(Spacing4), Pad(Spacing5), Bg(theme.BgRaised), Rounded(RadiusXl),
			Border(theme.Border)),
			Div(Class(Flex, ItemsCenter, Gap(Spacing3), css.Raw("flex-wrap", "wrap")),
				Span(Class(monoFont, TextSize(TextSm), Fg(theme.Faint)), p.Name+" today:"),
				statusChip(p.Status, brand),
			),
			P(Class(sansFont, TextSize(TextSm), Fg(theme.Fg), LineHeight(Num(1.65)), css.Raw("margin", "0")), p.Maturity),
			P(Class(sansFont, TextSize(TextSm), Fg(theme.Dim), LineHeight(Num(1.65)), css.Raw("margin", "0")),
				content.Disclosure()),
		),
	)
}

// siblingNav offers the other four products at the end of the page, so a reader who finished one
// case study has somewhere to go that is not the back button.
func siblingNav(p content.FluxProject, siblings []content.FluxProject) ui.Node {
	cards := []any{Class(Grid, Gap(Spacing3),
		css.Raw("grid-template-columns", "repeat(auto-fit,minmax(min(220px,100%),1fr))"))}
	for _, s := range siblings {
		if s.Slug == p.Slug {
			continue
		}
		b := theme.Brands[s.Slug]
		cards = append(cards, A(Class(append([]any{Flex, ItemsCenter, Gap(Spacing3), Bg(theme.BgRaised),
			Border(theme.Border), Rounded(RadiusLg), Pad(Spacing4), css.Raw("text-decoration", "none"),
			Hover(Border(b.Accent))}, append(lift(), focusRing())...)...),
			Props{Href: "/projects/" + s.Slug},
			Img(Class(css.Raw("width", "34px"), css.Raw("height", "auto"), css.Raw("flex", "0 0 auto")),
				Props{Src: "/static/brand/" + s.Slug + "-mark.webp", Alt: "", Width: "34", Loading: "lazy"}),
			Div(Class(Flex, FlexCol),
				Span(Class(FontSemibold, Fg(theme.Fg)), Span(s.Head), Span(Class(Fg(b.Accent)), s.Tail)),
				Span(Class(sansFont, TextSize(TextSm), Fg(theme.Dim)), s.Status),
			),
		))
	}
	return Div(Class(sectionRules(Gap(Spacing5))...),
		label("~/projects · the rest of the family"),
		sectionH2("Four more."),
		Div(cards...),
	)
}
