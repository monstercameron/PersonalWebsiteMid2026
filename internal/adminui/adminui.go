// Package adminui renders the owner-gated admin console as GoWebComponents — typed css/u utilities
// and internal/theme tokens, the same design system as the public site (internal/site). It renders
// server-side per request; because the css registry + sink are process-global, renders are
// serialized with a mutex. This replaces the old hand-written admin HTML + adminCSS blob.
package adminui

import (
	"html"
	"strconv"
	"sync"

	"github.com/monstercameron/GoWebComponents/v4/css"
	. "github.com/monstercameron/GoWebComponents/v4/css/u"
	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/ui"

	"github.com/monstercameron/earlcameron/internal/theme"
)

// renderMu serializes SSR renders (the css sink/registry are process-global).
var renderMu sync.Mutex

// AnimeCard is a view-model for one show in the console (search result or tracked entry).
type AnimeCard struct {
	ID      int
	Title   string
	Meta    string
	Cover   string
	Tracked bool
}

// bootstrapCSS is the minimal glue the typed system can't express: reset, base bg/gradient, mono
// font, placeholder + link colors. Everything else is typed css/u.
const bootstrapCSS = `*{box-sizing:border-box}html,body{margin:0}body{color:#f3e9e6;` +
	`font-family:ui-monospace,"SF Mono",SFMono-Regular,Menlo,"Cascadia Code","JetBrains Mono",monospace;` +
	`background:radial-gradient(60vw 50vw at 12% -8%,rgba(190,123,230,.16),transparent 60%),` +
	`radial-gradient(55vw 55vw at 105% 115%,rgba(233,84,32,.14),transparent 55%),#17040f;` +
	`background-attachment:fixed;min-height:100vh}input::placeholder{color:#a98ba0}a{color:#be7be6;text-decoration:none}`

// render builds the page tree and wraps it in a full admin HTML document with the harvested typed
// CSS. The tree is built inside the lock because css/u emits each class's CSS into the process-wide
// sink at node-construction time; StyleBlock (called after) then serializes what was emitted. The
// sink accumulates a bounded, deduplicated set of classes across requests — no per-request Reset,
// which would wipe classes emitted by a concurrent render.
func render(title string, build func() ui.Node) (string, error) {
	renderMu.Lock()
	defer renderMu.Unlock()
	markup, err := ui.RenderToString(build())
	if err != nil {
		return "", err
	}
	head := `<!doctype html><html lang="en"><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<title>` + html.EscapeString(title) + ` · admin</title>` +
		`<style>` + bootstrapCSS + `</style>` + css.StyleBlock() + `</head><body>`
	return head + markup + `</body></html>`, nil
}

// LoginPage renders the sign-in card. enabled=false shows a disabled notice; errMsg is optional.
func LoginPage(enabled bool, errMsg string) (string, error) {
	return render("sign in", func() ui.Node {
		var inner ui.Node
		if !enabled {
			inner = P(Class(Fg(theme.Dim), TextSize(TextSm)), "Admin is disabled — set the ADMIN_PASSWORD environment variable.")
		} else {
			var errNode ui.Node = Span()
			if errMsg != "" {
				errNode = Div(Class(Fg(theme.Red), TextSize(TextSm)), errMsg)
			}
			inner = Form(Class(Flex, FlexCol, Gap(Spacing3)), Props{Method: "post", Action: "/admin/login"},
				field("username", "text", "username", true),
				field("password", "password", "password", false),
				errNode,
				primaryBtn("Sign in"),
			)
		}
		card := Div(Class(Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusXl), Pad(Spacing6), Flex, FlexCol, Gap(Spacing4),
			fullWidth, MaxWidth(Px(380))),
			eyebrow("~/admin"),
			H2(Class(FontSize(Rem(1.5)), FontSemibold), "Sign in"),
			P(Class(Fg(theme.Dim), TextSize(TextSm)), "Owner access to the anime tracker and résumé tools."),
			inner,
		)
		return Div(Class(Flex, FlexCol, ItemsCenter, JustifyCenter, Gap(Spacing4), minScreen, Pad(Spacing5)),
			card,
			A(Class(Fg(theme.Dim), TextSize(TextSm)), Props{Href: "/"}, "← back to the site"),
		)
	})
}

// AnimeConsole renders the anime tracker: search, results, tracked shows, feeds, and release check.
func AnimeConsole(query string, results, tracked []AnimeCard) (string, error) {
	return render("anime tracker", func() ui.Node {
		sections := []ui.Node{
			header("anime"),
			feedsBar(),
			Form(Class(Flex, Gap(Spacing2)), Props{Method: "get", Action: "/admin/anime"},
				searchField(query),
				primaryBtn("Search"),
			),
		}
		if len(results) > 0 {
			sections = append(sections, sectionLabel("results"), animeGrid(results))
		}
		sections = append(sections, sectionLabel("tracked ("+strconv.Itoa(len(tracked))+")"))
		if len(tracked) == 0 {
			sections = append(sections, P(Class(Fg(theme.Dim), TextSize(TextSm)), "Nothing tracked yet — search above and hit “track”."))
		} else {
			sections = append(sections, animeGrid(tracked))
		}
		return page(sections...)
	})
}

// ResumeTool renders the résumé tailoring tool. enabled=false shows a disabled notice; msg optional.
func ResumeTool(enabled bool, msg string) (string, error) {
	return render("résumé tool", func() ui.Node {
		body := []ui.Node{
			header("résumé"),
			P(Class(Fg(theme.Dim), TextSize(TextSm)),
				"Canonical résumé: ", A(Class(Fg(theme.Accent2)), Props{Href: "/resume", Target: "_blank", Rel: "noopener"}, "/resume"),
				" — open it, then Save as PDF."),
		}
		if msg != "" {
			body = append(body, Div(Class(Fg(theme.Dim), TextSize(TextSm)), msg))
		}
		if !enabled {
			body = append(body, P(Class(Fg(theme.Dim), TextSize(TextSm)), "Set the OPENAI_API_KEY environment variable to enable tailoring."))
		} else {
			body = append(body,
				Form(Class(Flex, Gap(Spacing2)), Props{Method: "post", Action: "/admin/resume/tailor"},
					searchFieldNamed("url", "paste a job-posting URL…"),
					primaryBtn("Tailor résumé"),
				),
				P(Class(Fg(theme.Dim), TextSize(TextSm)),
					"The tool fetches the posting and re-emphasizes real facts to fit it — it never invents experience. Review before sending, then Save as PDF."),
			)
		}
		return page(body...)
	})
}

// --- shared building blocks ---

// page is the standard console content column (max-width, centered, padded).
func page(children ...ui.Node) ui.Node {
	nodes := []any{Class(Flex, FlexCol, Gap(Spacing5), centered, MaxWidth(Px(1000)), PadX(Spacing6), PadY(Spacing8))}
	for _, c := range children {
		nodes = append(nodes, c)
	}
	return Div(nodes...)
}

// header renders the console top bar with brand + tab nav + logout; active names the current tab.
func header(active string) ui.Node {
	tab := func(href, text string) ui.Node {
		c := theme.Dim
		if text == active {
			c = theme.Accent
		}
		return A(Class(Fg(c), Hover(Fg(theme.Accent)), TextSize(TextSm)), Props{Href: href}, text)
	}
	return Div(Class(Flex, ItemsCenter, JustifyBetween, Gap(Spacing3), PadY(Spacing2)),
		A(Class(FontSemibold, Fg(theme.Accent)), Props{Href: "/"}, "~/admin"),
		Div(Class(Flex, Gap(Spacing5), ItemsCenter),
			tab("/admin/anime", "anime"),
			tab("/admin/resume", "résumé"),
			Form(Class(inlineFlex), Props{Method: "post", Action: "/admin/logout"}, ghostBtn("logout")),
		),
	)
}

// feedsBar renders the public RSS feed links + the release-check action.
func feedsBar() ui.Node {
	return Div(Class(Flex, ItemsCenter, Gap(Spacing4), Fg(theme.Dim), TextSize(TextSm), css.Raw("flex-wrap", "wrap")),
		Span("feeds:"),
		A(Class(Fg(theme.Accent2)), Props{Href: "/anime.xml", Target: "_blank", Rel: "noopener"}, "/anime.xml"),
		A(Class(Fg(theme.Accent2)), Props{Href: "/anime/qotd.xml", Target: "_blank", Rel: "noopener"}, "/anime/qotd.xml"),
		Form(Class(inlineFlex), Props{Method: "post", Action: "/admin/anime/check"}, ghostBtn("run release check")),
	)
}

// animeGrid renders a responsive grid of anime cards.
func animeGrid(cards []AnimeCard) ui.Node {
	nodes := []any{Class(Grid, Gap(Spacing3), css.Raw("grid-template-columns", "repeat(auto-fill,minmax(250px,1fr))"))}
	for _, c := range cards {
		nodes = append(nodes, animeCard(c))
	}
	return Div(nodes...)
}

// animeCard renders one show with a cover, metadata, and a track/remove action.
func animeCard(a AnimeCard) ui.Node {
	endpoint := "/admin/anime/track"
	var btn ui.Node = primaryBtn("track")
	if a.Tracked {
		endpoint = "/admin/anime/untrack"
		btn = ghostBtn("remove")
	}
	return Div(Class(Flex, Gap(Spacing3), Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusLg), Pad(Spacing3)),
		Img(Class(css.Raw("width", "56px"), css.Raw("height", "80px"), css.Raw("object-fit", "cover"), Rounded(RadiusLg), Bg(theme.Border)),
			Props{Src: a.Cover, Alt: ""}),
		Div(Class(Flex, FlexCol, Gap(Spacing2)),
			Span(Class(FontSemibold, TextSize(TextSm)), a.Title),
			Span(Class(Fg(theme.Dim), TextSize(TextSm)), a.Meta),
			Form(Class(inlineFlex), Props{Method: "post", Action: endpoint},
				Input(Props{Type: "hidden", Name: "id", Value: strconv.Itoa(a.ID)}),
				btn,
			),
		),
	)
}

// field renders a styled text/password input.
func field(name, typ, placeholder string, autofocus bool) ui.Node {
	return Input(inputClass(fullWidth), Props{
		Name: name, Type: typ, Placeholder: placeholder, AutoFocus: autofocus, AutoComplete: autoComplete(name),
	})
}

// searchField renders the AniList search input (grows to fill the row).
func searchField(value string) ui.Node {
	return Input(inputClass(grow), Props{Name: "q", Type: "text", Value: value, Placeholder: "search AniList…", AutoFocus: true})
}

// searchFieldNamed renders a growing text input with a custom name/placeholder.
func searchFieldNamed(name, placeholder string) ui.Node {
	return Input(inputClass(grow), Props{Name: name, Type: "text", Placeholder: placeholder, AutoFocus: true})
}

// inputClass returns the shared input styling plus any extra utilities (e.g. width vs. flex-grow).
func inputClass(extra ...any) any {
	base := []any{Bg(theme.Bg), Border(theme.Border), Fg(theme.Fg), Rounded(RadiusLg),
		PadX(Spacing3), PadY(Spacing3), css.Raw("font", "inherit")}
	return Class(append(base, extra...)...)
}

// primaryBtn renders the filled accent submit button.
func primaryBtn(label string) ui.Node {
	return Button(Class(Bg(theme.Accent), Fg(Hex("#ffffff")), FontSemibold, Rounded(RadiusLg), PadX(Spacing4), PadY(Spacing3),
		css.Raw("border", "0"), css.Raw("cursor", "pointer"), css.Raw("font", "inherit")), Props{Type: "submit"}, label)
}

// ghostBtn renders an outlined submit button (secondary actions).
func ghostBtn(label string) ui.Node {
	return Button(Class(Fg(theme.Fg), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing4), PadY(Spacing2),
		css.Raw("background", "transparent"), css.Raw("cursor", "pointer"), css.Raw("font", "inherit")), Props{Type: "submit"}, label)
}

// eyebrow renders a mono section eyebrow.
func eyebrow(text string) ui.Node {
	return Div(Class(TextSize(TextSm), Fg(theme.Accent), Tracking(Ems(0.15))), text)
}

// sectionLabel renders an uppercase section heading.
func sectionLabel(text string) ui.Node {
	return Div(Class(TextSize(TextSm), Fg(theme.Dim), Tracking(Ems(0.12)), css.Raw("text-transform", "uppercase")), text)
}

// autoComplete maps a field name to a sensible autocomplete token.
func autoComplete(name string) string {
	switch name {
	case "username":
		return "username"
	case "password":
		return "current-password"
	}
	return ""
}

// Shared raw-CSS fragments the typed utilities don't cover (kept tiny + local).
var (
	grow       = css.Raw("flex", "1")
	fullWidth  = css.Raw("width", "100%")
	minScreen  = css.Raw("min-height", "100vh")
	centered   = css.Raw("margin", "0 auto")
	inlineFlex = css.Raw("display", "inline-flex")
)
