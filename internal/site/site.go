// Package site holds the standard-site UI as GoWebComponents components, styled with GWC's
// typed CSS (css/u). The same components render server-side (SSR failsafe / SEO, via
// RenderHTML) and will hydrate in the browser later. No raw CSS, no JS.
package site

import (
	"strconv"
	"strings"

	"github.com/monstercameron/GoWebComponents/v5/css"
	. "github.com/monstercameron/GoWebComponents/v5/css/u"
	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/ui"

	"github.com/monstercameron/earlcameron/internal/theme"
	"github.com/monstercameron/earlcameron/proto/sitepb"
)

// Colors come from the shared token package (internal/theme); see documents/DESIGN.md.

// --- motion -------------------------------------------------------------------------------
//
// DESIGN.md §7: restraint is the strategy — a few earned moments, never confetti, and everything
// collapses to instant under prefers-reduced-motion. So the whole site uses ONE gesture, `rise`: a
// short lift into place with a fade. It plays twice, in two different ways:
//
//   - riseOnLoad, staggered, for the hero — the single orchestrated entrance.
//   - riseOnScroll, for each section below it — driven by `animation-timeline: view()`, which is
//     native CSS scroll-driven animation and needs no JavaScript and no observer. Browsers that
//     do not support it ignore the timeline and simply run the animation once at load, ending at
//     the same final state; nothing is ever left invisible.
//
// One gesture reused is what makes a page feel composed. Six different gestures is what makes it
// feel generated.

// riseDistance is how far the gesture travels. Small on purpose: enough to read as movement,
// not enough to feel like the page is assembling itself.
const riseDistance = 10

// riseKeyframes is the shared entrance gesture.
func riseKeyframes() css.Rule {
	return css.Keyframes("rise",
		css.At("from", css.Opacity(0), css.Transform(css.TranslateY(css.Px(riseDistance)))),
		css.At("to", css.Opacity(1), css.Transform(css.TranslateY(css.Px(0)))),
	)
}

// stillRules collapse the gesture for visitors who asked for reduced motion: no animation at all,
// final state applied directly. This is the accommodation, not a shortened version of the effect.
func stillRules() []css.Rule {
	return ReducedMotion(
		css.Raw("animation-name", "none"),
		css.Raw("animation-delay", "0ms"),
		css.Opacity(1),
		css.Raw("transform", "none"),
	)
}

// riseOnLoad plays the gesture once at load, after delayMs. animation-fill-mode:both holds the
// element at the "from" frame during its delay, which is what makes a stagger read as a sequence
// instead of a flicker.
func riseOnLoad(delayMs int) []any {
	return []any{
		riseKeyframes(),
		css.Animation(css.Ms(460), css.EaseOut),
		css.Raw("animation-delay", strconv.Itoa(delayMs)+"ms"),
		css.Raw("animation-fill-mode", "both"),
		stillRules(),
	}
}

// riseOnScroll ties the gesture to the element's own progress through the viewport. The range ends
// well before the element is centred, so a section has finished arriving by the time it is being
// read — motion that is still resolving under the reader's eye is a distraction, not a flourish.
func riseOnScroll() []any {
	return []any{
		riseKeyframes(),
		css.Animation(css.Ms(460), css.EaseOut),
		css.Raw("animation-fill-mode", "both"),
		css.Raw("animation-timeline", "view()"),
		css.Raw("animation-range", "entry 5% cover 22%"),
		stillRules(),
	}
}

// sectionRules is the standard section wrapper: column layout, vertical rhythm, and the scroll
// reveal. Every section below the hero uses it, so they all arrive the same way.
func sectionRules(gap css.Rule) []any {
	return append([]any{Flex, FlexCol, gap, PadY(Spacing6)}, riseOnScroll()...)
}

// lift is the hover response shared by every card-like surface: a 2px rise and a deeper shadow,
// so the thing under the pointer reads as pickable. Paired with the existing border-accent hover.
func lift() []any {
	return []any{
		Transition(PropAll, Ms(180), EaseOut),
		Hover(css.Transform(css.TranslateY(css.Px(-2))),
			css.Raw("box-shadow", "0 18px 40px -24px rgba(0,0,0,.85)")),
		ReducedMotion(css.Raw("transition-duration", "0ms")),
	}
}

// focusRing is the keyboard-only affordance: an accent outline on :focus-visible, which appears for
// tab navigation and not for mouse clicks. Part of the a11y floor in DESIGN.md, not decoration.
func focusRing() []css.Rule {
	return css.FocusVisible(
		css.Raw("outline", "2px solid "+theme.Accent.String()),
		css.Raw("outline-offset", "3px"),
	)
}

// featuredCount is how many of content.featured are billed as case studies in the hero's proof
// strip and the "Selected work" grid. The remainder falls through to the quieter Labs shelf. The
// split is positional because sitepb.Project has no tier field — see internal/content.featured,
// which is ordered for exactly this.
const featuredCount = 4

// Page renders the full standard site for the given content. baseURL is the public origin
// (config.BaseURL) and is used to print absolute feed URLs — a relative "/anime.xml" is useless in
// a feed reader, which is the one place those URLs are meant to be pasted.
func Page(_ *sitepb.About, projects []*sitepb.Project, baseURL string) ui.Node {
	featured, labs := splitTiers(projects)
	return Div(Class(Fg(theme.Fg)),
		center(
			topNav(),
			hero(featured),
			termMount(),
			work(featured),
			velocityStrip(),
			how(),
			labsShelf(labs),
			elsewhere(),
			animeRadar(baseURL),
			contact(),
			footer(),
		),
	)
}

// splitTiers divides the project list into the featured case studies and the Labs remainder,
// tolerating a list shorter than featuredCount so a trimmed content set cannot panic the page.
func splitTiers(projects []*sitepb.Project) (featured, labs []*sitepb.Project) {
	if len(projects) <= featuredCount {
		return projects, nil
	}
	return projects[:featuredCount], projects[featuredCount:]
}

// RenderHTML renders the standard site to a complete, self-contained HTML document (markup plus
// the harvested typed-CSS <style> block). Intended to run once at startup over static content.
func RenderHTML(about *sitepb.About, projects []*sitepb.Project, baseURL string) (string, error) {
	markup, err := ui.RenderToString(Page(about, projects, baseURL))
	if err != nil {
		return "", err
	}
	head := `<!doctype html><html lang="en"><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width,initial-scale=1">` +
		// These four strings ARE the first impression: the tab, the search result, and the preview
		// card a recruiter sees when the link is pasted into Slack or LinkedIn — read long before
		// anyone scrolls. They must track the hero's positioning. They did not on 2026-07-29: the
		// hero had moved to "Senior software engineer…" while the title still said "AI-native
		// systems engineer", so every shared link and every crawler still reported the old, narrower
		// framing. If the H1 changes again, change these in the same commit.
		`<title>Earl Cameron — Senior Software Engineer · AI-native products & developer platforms</title>` +
		`<meta name="description" content="Senior software engineer building AI-native products, developer platforms, and unconventional systems — Go, WebAssembly, local AI. Creator of CashFlux, ArticleFlux and GoWebComponents; this site runs on the framework and gRPC bridge I wrote.">` +
		`<meta property="og:title" content="Earl Cameron — Senior Software Engineer · AI-native products & developer platforms">` +
		`<meta property="og:description" content="Go, WebAssembly, local AI, developer platforms. Creator of CashFlux, ArticleFlux, GoWebComponents and WASIBrowser — and this site is the proof of work.">` +
		`<meta property="og:type" content="website">` +
		`<meta property="og:url" content="https://www.earlcameron.com/">` +
		`<meta name="twitter:card" content="summary">` +
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
// min-width:0 overrides the flex-item min-width:auto floor so a wide child (the terminal's
// nowrap rows) can never inflate the column past the viewport on small screens.
func center(children ...ui.Node) ui.Node {
	args := []any{Class(WFull, MaxWidth(Px(1000)), PadX(Spacing6), Flex, FlexCol, Gap(Spacing5),
		css.Raw("min-width", "0"))}
	for _, c := range children {
		args = append(args, c)
	}
	return Div(Class(Flex, JustifyCenter), Div(args...))
}

// sansFont is Cam's voice (proportional sans) for prose, contrasted with the mono machine voice.
// font-family isn't in css/u, so use the typed css.Property escape hatch.
var sansFont = css.Raw("font-family", `-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif`)

// monoFont restates the body's machine voice for the places that need it back *inside* a sansFont
// block — command chips, figures — since font-family inherits.
var monoFont = css.Raw("font-family", `ui-monospace,"SF Mono",SFMono-Regular,Menlo,"Cascadia Code","JetBrains Mono",monospace`)

// articleFluxURL is the live ArticleFlux instance. It points at /home, not the host root: the root
// resolves to the reader, which requires an account, so a visitor with no login lands on a sign-in
// form. /home is the public front door that explains the project and links its demo.
const articleFluxURL = "https://feed.earlcameron.com/home"

// topNav renders the site navigation so the single home page indexes every visitor-facing page:
// the on-page sections (work, anime, contact) plus the standalone pages (résumé, CashFlux,
// ArticleFlux). The admin console is owner-only, so its entry lives in the footer, not here.
func topNav() ui.Node {
	link := func(href, text string) ui.Node {
		return A(Class(Fg(theme.Dim), Hover(Fg(theme.Accent)), TextSize(TextSm),
			Transition(PropColors, Ms(150), EaseOut), focusRing(),
			ReducedMotion(css.Raw("transition-duration", "0ms"))), Props{Href: href}, text)
	}
	return Div(Class(Flex, ItemsCenter, JustifyBetween, Gap(Spacing3), PadY(Spacing4), css.Raw("flex-wrap", "wrap")),
		A(Class(FontSemibold, Fg(theme.Accent)), Props{Href: "/"}, "~/earlcameron"),
		Div(Class(Flex, Gap(Spacing5), ItemsCenter, css.Raw("flex-wrap", "wrap")),
			link("#work", "work"),
			link("/resume", "résumé"),
			// CashFlux and ArticleFlux are separate full apps — open them in their own tab so the
			// portfolio stays put (matches the elsewhere() cards).
			A(Class(Fg(theme.Dim), Hover(Fg(theme.Accent)), TextSize(TextSm)),
				Props{Href: "/budget/", Target: "_blank", Rel: "noopener"}, "cashflux"),
			A(Class(Fg(theme.Dim), Hover(Fg(theme.Accent)), TextSize(TextSm)),
				Props{Href: articleFluxURL, Target: "_blank", Rel: "noopener"}, "articleflux"),
			link("#anime", "anime"),
			link("#contact", "contact"),
		),
	)
}

// hero renders the identity block: eyebrow, headline (with accent), thesis (sans), social links,
// and the launch CTA — matching the mockup. The terminal mounts immediately below (see Page).
func hero(featured []*sitepb.Project) ui.Node {
	return Div(Class(Flex, FlexCol, Gap(Spacing5), PadY(Spacing6)),
		Div(Class(append([]any{Flex, ItemsCenter, Gap(Spacing3), TextSize(TextSm), Fg(theme.Accent), Tracking(Ems(0.18))},
			riseOnLoad(0)...)...),
			Span(Class(css.Raw("width", "26px"), css.Raw("height", "1px"),
				css.Raw("background", "#e95420"), css.Raw("display", "inline-block"))),
			// No city, by Cam's instruction (2026-07-29): the eyebrow states availability, not a
			// location. Keep it that way — a street-level identifier here is not worth the reach.
			Span("EARL CAMERON · REMOTE"),
		),
		// "AI-native systems engineer" was the old headline. "Systems engineer" reads narrowly to a
		// recruiter — OS, networking, embedded, infra administration — and undersells product and
		// platform work. Lead with the broad, accurate label; "systems" survives in the tail, where
		// it describes the unconventional projects rather than defining the whole career.
		// Smaller on mobile than the desktop step would suggest: at 1.9rem this headline ran six
		// lines on a 390px screen and pushed "View the work" past the fold, which defeats the point
		// of a first screen a recruiter can act on.
		H1(Class(append([]any{FontSize(Rem(1.6)), Md(FontSize(Rem(2.6))), FontBold, LineHeight(Num(1.1))},
			riseOnLoad(70)...)...),
			Span("Senior software engineer building "),
			Span(Class(Fg(theme.Accent)), "AI-native products"),
			Span(", developer platforms, and unconventional systems."),
		),
		P(Class(append([]any{sansFont, FontSize(Rem(1.05)), Md(FontSize(Rem(1.2))), Fg(theme.Dim), MaxWidth(Px(640)), LineHeight(Num(1.5))},
			riseOnLoad(140)...)...),
			"I work across Go, WebAssembly, local AI, distributed systems, and product engineering. ",
			Span(Class(Fg(theme.Fg), FontSemibold), "Currently building agentic systems at UKG"),
			". ",
			Span(Class(Fg(theme.Fg), FontSemibold), "This site is the proof"),
			" — rendered by a UI framework I wrote, over a gRPC bridge I built.",
		),
		proofStrip(featured),
		Div(Class(append([]any{Flex, Gap(Spacing5), TextSize(TextSm), css.Raw("flex-wrap", "wrap")},
			riseOnLoad(280)...)...),
			socialLink("https://github.com/monstercameron", "◆", "github"),
			socialLink("/resume", "⬇", "résumé"),
			socialLink("https://www.linkedin.com/in/earl-cameron/", "in", "linkedin"),
			socialLink("https://www.earlcameron.com", "✦", "earlcameron.com"),
			socialLink("mailto:cam@earlcameron.com", "✉", "email"),
		),
		Div(Class(riseOnLoad(350)...), launchCTA()),
	)
}

// proofStrip names the featured projects directly under the positioning statement, so a visitor
// learns who Cam is without inferring it from a grid of cards further down the page. The names come
// from the featured slice rather than a literal, so this can never drift out of sync with the grid.
func proofStrip(featured []*sitepb.Project) ui.Node {
	row := []any{Class(append([]any{Flex, ItemsCenter, Gap(Spacing2), TextSize(TextSm), css.Raw("flex-wrap", "wrap")},
		riseOnLoad(210)...)...),
		Span(Class(Fg(theme.Faint)), "creator of")}
	for i, p := range featured {
		if i > 0 {
			row = append(row, Span(Class(Fg(theme.Faint)), "·"))
		}
		row = append(row, Span(Class(Fg(theme.Fg), FontSemibold), p.GetName()))
	}
	return Div(row...)
}

// launchCTA renders the primary actions a recruiter came for — the work, the résumé, a way to make
// contact — followed by the terminal invitation.
//
// The terminal used to own the loud orange button here, which framed it as the way into the site:
// a gate. It is not navigation, so it now reads as an aside with its suggested commands shown
// rather than guessed, and the work is the primary action instead.
//
// The copy does NOT say the terminal is something to do "while the wasm loads" — on the public site
// the terminal *is* the wasm (client/main.go mounts only Terminal into #term-root; the rest of this
// page is server-rendered Go), so it cannot exist before the binary is up. Framing it as a loading
// distraction is both false and weaker than the truth: it is the running proof of the stack.
func launchCTA() ui.Node {
	primary := func(href, text string) ui.Node {
		return A(Class(Bg(theme.Accent), Fg(Hex("#ffffff")), FontSemibold, Rounded(RadiusLg), PadX(Spacing5), PadY(Spacing3),
			css.Raw("text-decoration", "none"), Transition(PropAll, Ms(180), EaseOut),
			Hover(css.Transform(css.TranslateY(css.Px(-1)))), focusRing(),
			ReducedMotion(css.Raw("transition-duration", "0ms"))), Props{Href: href}, text)
	}
	secondary := func(href, text string, external bool) ui.Node {
		p := Props{Href: href}
		if external {
			p.Target, p.Rel = "_blank", "noopener"
		}
		return A(Class(Border(theme.Border), Fg(theme.Fg), FontSemibold, Rounded(RadiusLg), PadX(Spacing5), PadY(Spacing3),
			Hover(Border(theme.Accent), css.Transform(css.TranslateY(css.Px(-1)))),
			Transition(PropAll, Ms(180), EaseOut), focusRing(),
			ReducedMotion(css.Raw("transition-duration", "0ms")),
			// No sitewide <a> reset exists, so kill the default underline — these read as buttons.
			css.Raw("text-decoration", "none")), p, text)
	}
	return Div(Class(Flex, FlexCol, Gap(Spacing4)),
		Div(Class(Flex, ItemsCenter, Gap(Spacing3), css.Raw("flex-wrap", "wrap")),
			primary("#work", "View the work"),
			secondary("/resume", "Read the résumé", true),
			secondary("#contact", "Contact", false),
		),
		// #term-launch is bound by the wasm terminal on mount (client/terminal.go): clicking it
		// expands the terminal to fullscreen and focuses its input. The ID is the contract between
		// this server-rendered control and the component below it — renaming it silently breaks the
		// button. A <button> rather than a <span> so it is keyboard-reachable and announces itself;
		// font/border/background are reset because browser defaults do not inherit the mono face.
		Div(Class(sansFont, Fg(theme.Dim), TextSize(TextSm), MaxWidth(Px(620)), LineHeight(Num(1.6))),
			"Everything above is server-rendered Go. The terminal below ",
			Span(Class(Fg(theme.Fg), FontSemibold), "is"),
			" the WebAssembly — a real shell over gRPC, not a screenshot. Try ",
			termHint("projects"), " ", termHint("open cashflux"), " ", termHint("neofetch"), " or ",
			Button(Class(Fg(theme.Accent), FontSemibold, css.Raw("cursor", "pointer"), css.Raw("font", "inherit"),
				css.Raw("border", "none"), css.Raw("background", "none"), css.Raw("padding", "0"),
				css.Raw("text-decoration", "underline"), Hover(Fg(theme.Accent2)), focusRing()),
				Props{ID: "term-launch"},
				"open it fullscreen"),
			".",
		),
	)
}

// termHint renders one suggested command as a keycap-ish chip, so the commands are visible instead
// of something a visitor has to guess at (or type `help` to discover).
func termHint(cmd string) ui.Node {
	return Span(Class(Fg(theme.Accent2), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing2),
		monoFont, css.Raw("white-space", "nowrap")), cmd)
}

// socialLink renders a glyphed external link.
func socialLink(href, glyph, text string) ui.Node {
	return A(Class(Fg(theme.Dim), Flex, ItemsCenter, Gap(Spacing2), Hover(Fg(theme.Fg)),
		Transition(PropColors, Ms(150), EaseOut), focusRing(),
		ReducedMotion(css.Raw("transition-duration", "0ms"))),
		Props{Href: href, Target: "_blank", Rel: "noopener"},
		Span(Class(Fg(theme.Accent2)), glyph), Span(text))
}

// termMount is the element the wasm terminal mounts into, placed right after the hero (the
// mockup's unified hero). No-JS visitors simply see the standard site without it.
func termMount() ui.Node {
	return Div(FromProps(Props{ID: "term-root"}))
}

// work renders the featured case studies — the four projects billed as the professional story.
// Labs (labsShelf) carries everything else, deliberately quieter, so a visitor can tell
// production-grade work from experiments without opening a repository.
func work(projects []*sitepb.Project) ui.Node {
	// Wider columns than the old nine-up grid: at the 1000px content width these land two-up, which
	// is what makes them read as case studies rather than tiles.
	//
	// min(420px,100%) rather than a bare 420px: with auto-fill, a fixed track minimum wider than the
	// container does not collapse — the grid stays 420px and pushes the page into horizontal scroll.
	// Measured as 54px of overflow at a 390px viewport before this guard.
	cards := []any{Class(Grid, Gap(Spacing4), css.Raw("grid-template-columns", "repeat(auto-fill,minmax(min(420px,100%),1fr))"))}
	for _, p := range projects {
		cards = append(cards, card(p))
	}
	return Div(FromProps(Props{ID: "work"}), Class(sectionRules(Gap(Spacing5))...),
		// The eyebrow carries the section's source link: shell path on the left, the GitHub profile
		// every card resolves to on the right. It sits on the "where am I" line rather than under
		// the grid, so a visitor scanning the cards for repos finds the whole set without scrolling
		// past them. Wraps to a second line on narrow viewports.
		Div(Class(Flex, ItemsCenter, JustifyBetween, Gap(Spacing3), css.Raw("flex-wrap", "wrap")),
			label("~/projects · featured"),
			A(Class(Fg(theme.Accent2), TextSize(TextSm)),
				Props{Href: "https://github.com/monstercameron", Target: "_blank", Rel: "noopener"},
				"github.com/monstercameron ↗"),
		),
		headed("Selected work.",
			"Four projects I'd want to be judged on — a product, an AI pipeline, a framework, and a "+
				"systems bet. The smaller experiments are further down, in Labs."),
		Div(cards...),
	)
}

// evidenceLinks are per-project receipts for a claim the card makes in words.
//
// GoWebComponents says it was "benchmarked head-to-head with React — faster on overall geomean",
// which is the single least verifiable sentence on the page and the first thing a sceptical
// engineer would want to check. The numbers exist (docs/benchmarks in the GWC repo: the microbench
// report, latest.json, reference.json); they were just not reachable from here.
//
// This is a map rather than a field on sitepb.Project because adding one would mean regenerating
// the protos, and protoc is not available in this environment (TODOS §2). If a second project ever
// earns a third link, promote it to the proto rather than growing this.
var evidenceLinks = map[string]struct{ label, href string }{
	"gwc": {"benchmarks ↗", "https://github.com/monstercameron/GoWebComponents/tree/main/docs/benchmarks"},
}

// card renders one project.
func card(p *sitepb.Project) ui.Node {
	tags := []any{Class(Flex, Gap(Spacing2))}
	for _, t := range p.GetTags() {
		tags = append(tags, Span(Class(TextSize(TextSm), Fg(theme.Dim), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing2)), t))
	}
	links := []any{Class(Flex, Gap(Spacing4), TextSize(TextSm), css.Raw("flex-wrap", "wrap")),
		A(Class(Fg(theme.Accent2)), Props{Href: p.GetRepo(), Target: "_blank", Rel: "noopener"}, "code ↗")}
	if p.GetDemo() != "" {
		links = append(links, A(Class(Fg(theme.Accent2)), Props{Href: p.GetDemo(), Target: "_blank", Rel: "noopener"}, "demo ↗"))
	}
	if e, ok := evidenceLinks[p.GetId()]; ok {
		links = append(links, A(Class(Fg(theme.Accent2)), Props{Href: e.href, Target: "_blank", Rel: "noopener"}, e.label))
	}
	return Div(Class(append([]any{Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusXl), Pad(Spacing5), Flex, FlexCol, Gap(Spacing3),
		Hover(Border(theme.Accent))}, lift()...)...),
		Div(Class(Flex, JustifyBetween, ItemsCenter, Gap(Spacing3)),
			Span(Class(FontSemibold, Fg(theme.Accent), FontSize(Rem(1.1))), p.GetGlyph()+"  "+p.GetName()),
			Span(Class(TextSize(TextSm), Fg(theme.Green)), p.GetStatus()),
		),
		P(Class(Fg(theme.Fg), TextSize(TextSm)), p.GetBlurb()),
		// A featured card gets the long description too — the blurb says what it is, this says what
		// was actually built. It is the closest thing to a case study until the case-study pages
		// land (TODOS §14.F), and it is the difference between a tile and an argument.
		P(Class(sansFont, Fg(theme.Dim), TextSize(TextSm), LineHeight(Num(1.55))), p.GetLong()),
		Div(tags...),
		Div(links...),
	)
}

// labsShelf renders the non-featured projects as a compact list rather than cards. The tier
// difference is carried by *form*, not just size: cards read as billed work, a ruled list reads as
// a shelf of smaller bets, and a list also absorbs any number of entries without the ragged final
// row a grid produces.
func labsShelf(projects []*sitepb.Project) ui.Node {
	if len(projects) == 0 {
		return Div()
	}
	rows := []any{Class(Flex, FlexCol)}
	for _, p := range projects {
		links := []any{Class(Flex, Gap(Spacing3), TextSize(TextSm)),
			A(Class(Fg(theme.Accent2)), Props{Href: p.GetRepo(), Target: "_blank", Rel: "noopener"}, "code ↗")}
		if p.GetDemo() != "" {
			links = append(links, A(Class(Fg(theme.Accent2)), Props{Href: p.GetDemo(), Target: "_blank", Rel: "noopener"}, "demo ↗"))
		}
		rows = append(rows, Div(Class(Flex, FlexCol, Gap(Spacing2), Md(FlexRow, ItemsCenter, Gap(Spacing4)),
			PadY(Spacing3), PadX(Spacing2), Rounded(RadiusLg), css.BorderTop(css.Px(1), theme.Border),
			Transition(PropColors, Ms(180), EaseOut), Hover(Bg(theme.BgRaised)),
			ReducedMotion(css.Raw("transition-duration", "0ms"))),
			Span(Class(Fg(theme.Accent), FontSemibold, css.Raw("min-width", "210px")), p.GetGlyph()+"  "+p.GetName()),
			Span(Class(TextSize(TextSm), Fg(theme.Faint), css.Raw("min-width", "80px")), p.GetStatus()),
			Span(Class(sansFont, TextSize(TextSm), Fg(theme.Dim), css.Raw("flex", "1")), p.GetBlurb()),
			Div(links...),
		))
	}
	return Div(FromProps(Props{ID: "labs"}), Class(sectionRules(Gap(Spacing4))...),
		label("~/labs"),
		headed("Labs.",
			"Infrastructure, experiments, and language research. Smaller bets — still built, still running."),
		Div(rows...),
	)
}

// velocityStrip is the evidence layer: CashFlux measured rather than described.
//
// Numbers are counted from the CashFlux repository's tracked files on 2026-07-29 — re-measure with
// `git ls-files` before changing them, and do not swap in a `find` count, which picks up untracked
// build output and overstated the test figure by ~6x on the first pass. Deliberately no
// lines-of-code figure: LOC rewards duplication and verbosity, and readers know it.
func velocityStrip() ui.Node {
	stat := func(n, label string) ui.Node {
		return Div(Class(append([]any{Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusLg), Pad(Spacing4), Flex, FlexCol, Gap(Spacing1)}, lift()...)...),
			Span(Class(FontSize(Rem(1.6)), FontBold, Fg(theme.Fg)), n),
			Span(Class(sansFont, TextSize(TextSm), Fg(theme.Dim)), label),
		)
	}
	return Div(Class(sectionRules(Gap(Spacing4))...),
		label("~/evidence · cashflux"),
		headed("Built at high velocity.",
			"CashFlux went from first commit to a working local-first financial platform in six weeks — "+
				"transactions, budgets, goals, reports, household ownership, a to-do system and an "+
				"on-device assistant. These are counted from the repository, not estimated."),
		Div(Class(Grid, Gap(Spacing3), css.Raw("grid-template-columns", "repeat(auto-fill,minmax(160px,1fr))")),
			stat("26", "documented releases"),
			stat("221", "internal packages"),
			stat("50", "application routes"),
			stat("2,998", "tests"),
			stat("6", "weeks, first commit to platform"),
		),
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
	return Div(Class(sectionRules(Gap(Spacing5))...),
		label("uname -a · how this site works"),
		sectionH2("The medium is the résumé."),
		Div(Class(Flex, FlexCol, Bg(theme.BgRaised), Rounded(RadiusXl)),
			row("01", "GoWebComponents", "the frontend is Go compiled to WebAssembly, a framework I wrote. Zero npm."),
			row("02", "gRPC bridge", "the browser speaks gRPC-over-WebSocket through GoGRPCBridge. One same-origin socket."),
			row("03", "Go backend", "a real gRPC server streams content, stores messages, powers live status."),
		),
	)
}

// elsewhere renders the links section: résumé, the two live apps, LinkedIn, YouTube, GitHub.
func elsewhere() ui.Node {
	card := func(href, glyph, title, sub string) ui.Node {
		return A(Class(append([]any{Flex, ItemsCenter, Gap(Spacing3), Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusLg),
			Pad(Spacing4), Hover(Border(theme.Accent)), css.Raw("text-decoration", "none")}, append(lift(), focusRing())...)...),
			Props{Href: href, Target: "_blank", Rel: "noopener"},
			Span(Class(Fg(theme.Accent2), FontSize(Rem(1.25)), css.Raw("min-width", "24px")), glyph),
			Div(Class(Flex, FlexCol),
				Span(Class(FontSemibold, Fg(theme.Fg)), title),
				Span(Class(sansFont, Fg(theme.Dim), TextSize(TextSm)), sub),
			),
		)
	}
	grid := []any{Class(Grid, Gap(Spacing3), css.Raw("grid-template-columns", "repeat(auto-fill,minmax(230px,1fr))"))}
	grid = append(grid,
		card("/resume", "⬇", "Résumé", "read it, save as PDF"),
		card("/budget/", "◱", "CashFlux", "my budgeting app — try it live"),
		card(articleFluxURL, "◨", "ArticleFlux", "my feed reader — running live"),
		card("https://github.com/monstercameron", "◆", "GitHub", "open-source work"),
		card("https://www.linkedin.com/in/earl-cameron/", "in", "LinkedIn", "experience & network"),
		card("https://www.youtube.com/@EarlCameron007", "▶", "YouTube", "builds & demos"),
	)
	return Div(Class(sectionRules(Gap(Spacing5))...),
		label("~/elsewhere"),
		sectionH2("Find me around."),
		Div(grid...),
	)
}

// animeRadar renders the public anime-release RSS feeds (a personal feature): the Release Radar
// (episodes I'm tracking) and the daily Question-of-the-Day prompt feed. These are document-plane
// RSS (HTTP GET), safe to link publicly — the config that drives them stays owner-gated at /admin.
func animeRadar(baseURL string) ui.Node {
	// A feed card is no longer one big <a>: it now holds a link, a readable URL, and a copy button,
	// and nesting controls inside an anchor is invalid. The title carries the link instead.
	feed := func(path, glyph, title, sub string) ui.Node {
		url := strings.TrimRight(baseURL, "/") + path
		return Div(Class(append([]any{Flex, FlexCol, Gap(Spacing3), Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusLg),
			Pad(Spacing4), Hover(Border(theme.Accent))}, lift()...)...),
			Div(Class(Flex, ItemsCenter, Gap(Spacing3)),
				Span(Class(Fg(theme.Accent2), FontSize(Rem(1.25)), css.Raw("min-width", "24px")), glyph),
				Div(Class(Flex, FlexCol),
					A(Class(FontSemibold, Fg(theme.Fg), Hover(Fg(theme.Accent)), css.Raw("text-decoration", "none")),
						Props{Href: path}, title),
					Span(Class(sansFont, Fg(theme.Dim), TextSize(TextSm)), sub),
				),
			),
			feedURLRow(url),
		)
	}
	grid := []any{Class(Grid, Gap(Spacing3), css.Raw("grid-template-columns", "repeat(auto-fill,minmax(min(320px,100%),1fr))"))}
	grid = append(grid,
		feed("/anime.xml", "◈", "Release Radar", "RSS · new episodes I'm tracking"),
		feed("/anime/qotd.xml", "✎", "Question of the Day", "RSS · a daily anime prompt"),
	)
	return Div(FromProps(Props{ID: "anime"}), Class(sectionRules(Gap(Spacing5))...),
		label("~/anime · release radar"),
		headed("Anime, on RSS.",
			"Two feeds, open to anyone. Copy either address and paste it into your feed reader."),
		Div(grid...),
	)
}

// feedURLRow renders the absolute feed URL in a selectable read-only field beside a copy button.
//
// Two affordances on purpose. The button is the fast path, but it depends on the wasm client having
// booted and on the Clipboard API being available (it is not in an insecure context), so the field
// carries the URL in full either way — a visitor can always select and copy it by hand, which is the
// behaviour a no-JS or failed-boot visitor falls back to rather than being left with nothing.
//
// data-copy-url is the contract with client/clipboard.go, which binds the button after the wasm
// loads. The field is readonly rather than disabled so its text stays selectable and focusable.
func feedURLRow(url string) ui.Node {
	return Div(Class(Flex, ItemsCenter, Gap(Spacing2)),
		Input(Class(css.Raw("flex", "1"), css.Raw("min-width", "0"), monoFont, TextSize(TextSm),
			Fg(theme.Dim), Bg(theme.Bg), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing2), PadY(Spacing2),
			css.Raw("text-overflow", "ellipsis")),
			Props{Value: url, ReadOnly: true, Type: "text", Aria: map[string]string{"label": "RSS feed URL"}}),
		Button(Class(Fg(theme.Accent2), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing3), PadY(Spacing2),
			TextSize(TextSm), Hover(Border(theme.Accent), Fg(theme.Accent)), css.Raw("cursor", "pointer"),
			css.Raw("font", "inherit"), css.Raw("background", "none"), focusRing(),
			Transition(PropColors, Ms(180), EaseOut), css.Raw("white-space", "nowrap"),
			ReducedMotion(css.Raw("transition-duration", "0ms"))),
			Props{Type: "button", Data: map[string]string{"copy-url": url}},
			"copy"),
	)
}

// contact renders the contact section (mailto for the no-JS path; the terminal has a live form).
func contact() ui.Node {
	return Div(FromProps(Props{ID: "contact"}), Class(sectionRules(Gap(Spacing4))...),
		label("./contact"),
		sectionH2("Let's build something."),
		P(Class(Fg(theme.Dim)),
			"Email me at ",
			A(Class(Fg(theme.Accent)), Props{Href: "mailto:cam@earlcameron.com"}, "cam@earlcameron.com"),
			". The terminal has a live contact form over gRPC."),
		P(Class(sansFont, Fg(theme.Dim), TextSize(TextSm)),
			"Best fit: senior systems or AI-tooling roles — Go, WebAssembly, on-device ML, agent infrastructure."),
	)
}

// footer renders the site footer, including a discreet owner entry to the password-gated admin.
//
// Design notes (2026-07-29): this was a full 1px box — Border() sets four sides — floating at the
// bottom of a page that uses boxes for cards and nothing else, so it read as a stray rectangle
// rather than an ending. It is now a single hairline rule above the content, which is what a
// footer divider is, and the credit line was rewritten to close the page's own argument: the two
// pieces of the stack that are Cam's own work are links, and "zero npm" is the one bright note,
// deliberately the only accent colour down here. `admin` drops to Faint — its doc comment says
// discreet, and at Dim it was competing with the copyright beside it.
func footer() ui.Node {
	stackLink := func(href, text string) ui.Node {
		return A(Class(Fg(theme.Accent2), Hover(Fg(theme.Accent))), Props{Href: href, Target: "_blank", Rel: "noopener"}, text)
	}
	dot := func() ui.Node { return Span(Class(Fg(theme.Faint)), "·") }
	return Footer(Class(Flex, FlexCol, Gap(Spacing3), Md(FlexRow, JustifyBetween, ItemsCenter),
		PadY(Spacing6), css.BorderTop(css.Px(1), theme.Border), Fg(theme.Dim), TextSize(TextSm)),
		Div(Class(Flex, ItemsCenter, Gap(Spacing2), css.Raw("flex-wrap", "wrap")),
			Span(Class(Fg(theme.Faint)), "built with"),
			stackLink("https://github.com/monstercameron/GoWebComponents", "GoWebComponents"),
			dot(),
			stackLink("https://github.com/monstercameron/GoGRPCBridge", "GoGRPCBridge"),
			// No separator before "zero npm": on a narrow viewport the line wraps here, and a
			// leading "·" at the start of the second line reads as a typo. The accent colour is
			// separation enough.
			Span(Class(Fg(theme.Accent)), "zero npm"),
		),
		Div(Class(Flex, Gap(Spacing4), ItemsCenter),
			Span("© 2026 Earl Cameron"),
			A(Class(Fg(theme.Faint), Hover(Fg(theme.Accent2)), css.Raw("text-decoration", "none")),
				Props{Href: "/admin"}, "admin"),
		),
	)
}

// headed pairs a section heading with its standfirst, tightly. The sections use Gap(Spacing5)
// between their blocks, which is right between a heading and its content but far too airy between a
// heading and the sentence explaining it — those two are one unit and should read as one.
func headed(title, sub string) ui.Node {
	return Div(Class(Flex, FlexCol, Gap(Spacing2)),
		sectionH2(title),
		P(Class(sansFont, Fg(theme.Dim), TextSize(TextSm), MaxWidth(Px(640)), LineHeight(Num(1.55))), sub),
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
