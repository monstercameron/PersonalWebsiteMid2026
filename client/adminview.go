//go:build js && wasm

package main

import (
	"strconv"

	"github.com/monstercameron/GoWebComponents/v4/css"
	. "github.com/monstercameron/GoWebComponents/v4/css/u"
	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/ui"

	"github.com/monstercameron/earlcameron/internal/theme"
	"github.com/monstercameron/earlcameron/proto/sitepb"
)

// --- login ---

// loginView renders the centered sign-in card.
func loginView(username, password ui.State[string], onLogin ui.Handler, flash string) ui.Node {
	onU := ui.WrapHandler(func(e ui.Event) { username.Set(e.GetValue()) })
	onP := ui.WrapHandler(func(e ui.Event) { password.Set(e.GetValue()) })
	card := Div(Class(Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusXl), Pad(Spacing6), Flex, FlexCol, Gap(Spacing4), rawWidth("100%"), MaxWidth(Px(380))),
		eyebrow("~/admin"),
		H2(Class(FontSize(Rem(1.5)), FontSemibold), "Sign in"),
		P(Class(Fg(theme.Dim), TextSize(TextSm)), "Owner access to the anime tracker and résumé tools."),
		textInput(username.Get(), onU, "username", "text", true),
		textInput(password.Get(), onP, "password", "password", false),
		flashLine(flash, theme.Red),
		primaryButton("Sign in", onLogin),
	)
	return Div(Class(Flex, FlexCol, ItemsCenter, JustifyCenter, Gap(Spacing4), rawMinScreen(), Pad(Spacing5)),
		card,
		A(Class(Fg(theme.Dim), TextSize(TextSm)), Props{Href: "/"}, "← back to the site"),
	)
}

// --- console shell ---

// consoleShell renders the header nav + the active view's content.
func consoleShell(active string, navTo func(string) ui.Handler, onLogout ui.Handler, flash string, content ui.Node) ui.Node {
	tab := func(v, label string) ui.Node {
		c := theme.Dim
		if v == active {
			c = theme.Accent
		}
		return Span(Class(Fg(c), TextSize(TextSm), css.Raw("cursor", "pointer")), Props{OnClick: navTo(v)}, label)
	}
	header := Div(Class(Flex, ItemsCenter, JustifyBetween, Gap(Spacing3), PadY(Spacing4)),
		A(Class(FontSemibold, Fg(theme.Accent)), Props{Href: "/"}, "~/earlcameron"),
		Div(Class(Flex, Gap(Spacing5), ItemsCenter),
			tab("anime", "anime"), tab("resume", "résumé"), tab("settings", "settings"),
			ghostButton("logout", onLogout),
		),
	)
	body := []any{Class(Flex, FlexCol, Gap(Spacing5), rawCentered(), MaxWidth(Px(1000)), PadX(Spacing6), PadY(Spacing5)), header}
	if flash != "" {
		body = append(body, Div(Class(Fg(theme.Accent2), TextSize(TextSm)), flash))
	}
	body = append(body, content)
	return Div(body...)
}

// --- anime view ---

// animeView renders the search row, results, and tracked list.
func animeView(query ui.State[string], onSearch, onCheck ui.Handler, results, tracked []*sitepb.Anime, trackFn func(int32, bool)) ui.Node {
	onQ := ui.WrapHandler(func(e ui.Event) { query.Set(e.GetValue()) })
	sections := []any{Class(Flex, FlexCol, Gap(Spacing4))}
	sections = append(sections,
		Div(Class(Flex, Gap(Spacing4), ItemsCenter, Fg(theme.Dim), TextSize(TextSm), css.Raw("flex-wrap", "wrap")),
			Span("feeds:"),
			A(Class(Fg(theme.Accent2)), Props{Href: "/anime.xml", Target: "_blank", Rel: "noopener"}, "/anime.xml"),
			A(Class(Fg(theme.Accent2)), Props{Href: "/anime/qotd.xml", Target: "_blank", Rel: "noopener"}, "/anime/qotd.xml"),
			ghostButton("run release check", onCheck),
		),
		Div(Class(Flex, Gap(Spacing2)),
			growInput(query.Get(), onQ, "search AniList…"),
			primaryButton("Search", onSearch),
		),
	)
	if len(results) > 0 {
		sections = append(sections, sectionLabel("results"), animeGrid(results, trackFn))
	}
	sections = append(sections, sectionLabel("tracked ("+strconv.Itoa(len(tracked))+")"))
	if len(tracked) == 0 {
		sections = append(sections, P(Class(Fg(theme.Dim), TextSize(TextSm)), "Nothing tracked yet — search above and hit “track”."))
	} else {
		sections = append(sections, animeGrid(tracked, trackFn))
	}
	return Div(sections...)
}

// animeGrid renders a responsive grid of anime cards.
func animeGrid(cards []*sitepb.Anime, trackFn func(int32, bool)) ui.Node {
	nodes := []any{Class(Grid, Gap(Spacing3), css.Raw("grid-template-columns", "repeat(auto-fill,minmax(250px,1fr))"))}
	for _, a := range cards {
		nodes = append(nodes, animeCardNode(a, trackFn))
	}
	return Div(nodes...)
}

// animeCardNode renders one show with a track/remove action wired to trackFn.
func animeCardNode(a *sitepb.Anime, trackFn func(int32, bool)) ui.Node {
	id, isTracked := a.GetAnilistId(), a.GetTracked()
	onClick := ui.WrapHandler(func() { trackFn(id, !isTracked) })
	btn := primaryButton("track", onClick)
	if isTracked {
		btn = ghostButton("remove", onClick)
	}
	meta := a.GetFormat() + " · " + a.GetStatus() + " · " + strconv.Itoa(int(a.GetEpisodes())) + " eps"
	if a.GetSeasonYear() > 0 {
		meta += " · " + strconv.Itoa(int(a.GetSeasonYear()))
	}
	return Div(Class(Flex, Gap(Spacing3), Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusLg), Pad(Spacing3)),
		Img(Class(css.Raw("width", "56px"), css.Raw("height", "80px"), css.Raw("object-fit", "cover"), Rounded(RadiusLg), Bg(theme.Border)),
			Props{Src: a.GetCoverImage(), Alt: ""}),
		Div(Class(Flex, FlexCol, Gap(Spacing2)),
			Span(Class(FontSemibold, TextSize(TextSm)), a.GetTitle()),
			Span(Class(Fg(theme.Dim), TextSize(TextSm)), meta),
			btn,
		),
	)
}

// --- résumé view ---

// resumeView renders the job-URL tailor form and the tailored result.
func resumeView(jobURL ui.State[string], onTailor ui.Handler, tailored *sitepb.Resume) ui.Node {
	onURL := ui.WrapHandler(func(e ui.Event) { jobURL.Set(e.GetValue()) })
	sections := []any{Class(Flex, FlexCol, Gap(Spacing4)),
		P(Class(Fg(theme.Dim), TextSize(TextSm)),
			"Canonical résumé: ", A(Class(Fg(theme.Accent2)), Props{Href: "/resume", Target: "_blank", Rel: "noopener"}, "/resume"),
			" — paste a job URL below to tailor a variant."),
		Div(Class(Flex, Gap(Spacing2)),
			growInput(jobURL.Get(), onURL, "paste a job-posting URL…"),
			primaryButton("Tailor résumé", onTailor),
		),
	}
	if tailored != nil {
		sections = append(sections, tailoredCard(tailored))
	}
	return Div(sections...)
}

// tailoredCard renders the tailored résumé data for review.
func tailoredCard(r *sitepb.Resume) ui.Node {
	items := []any{Class(Flex, FlexCol, Gap(Spacing3), Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusLg), Pad(Spacing4)),
		Span(Class(FontSemibold, Fg(theme.Accent)), r.GetName()+" — "+r.GetTitle()),
		Span(Class(Fg(theme.Dim), TextSize(TextSm)), r.GetSummary()),
	}
	for _, j := range r.GetJobs() {
		items = append(items, Span(Class(FontSemibold, TextSize(TextSm)), j.GetRole()+" · "+j.GetOrg()+" · "+j.GetDates()))
		bl := []any{Class(Flex, FlexCol, Gap(Spacing2))}
		for _, b := range j.GetBullets() {
			bl = append(bl, Span(Class(Fg(theme.Dim), TextSize(TextSm)), "• "+b))
		}
		items = append(items, Div(bl...))
	}
	items = append(items, P(Class(Fg(theme.Dim), TextSize(TextSm)), "Review carefully before use."))
	return Div(items...)
}

// --- settings view ---

// settingsView renders the OpenAI key + model form (a dropdown when models are known).
func settingsView(keySet bool, models []string, model, apiKey ui.State[string], onSave ui.Handler) ui.Node {
	onKey := ui.WrapHandler(func(e ui.Event) { apiKey.Set(e.GetValue()) })
	onModel := ui.WrapHandler(func(e ui.Event) { model.Set(e.GetValue()) })

	keyPlaceholder := "sk-…"
	if keySet {
		keyPlaceholder = "a key is set — leave blank to keep it"
	}

	var modelField ui.Node = textInput(model.Get(), onModel, "gpt-4o-mini", "text", false)
	modelHint := "add a key above and save to load available models"
	switch {
	case len(models) > 0:
		modelHint = "models available to your key"
		opts := []any{inputBase(rawWidth("100%")), Props{OnChange: onModel}}
		for _, m := range models {
			opts = append(opts, Tag("option", Props{Value: m, Selected: m == model.Get()}, m))
		}
		modelField = Tag("select", opts...)
	case keySet:
		modelHint = "couldn't load models from OpenAI (key invalid?) — type a model id"
	}

	return Div(Class(Flex, FlexCol, Gap(Spacing4), MaxWidth(Px(520))),
		P(Class(Fg(theme.Dim), TextSize(TextSm)), "The OpenAI key is stored in the backend database and used server-side."),
		settingRow("OpenAI API key", "openai key ("+keyStatus(keySet)+")", textInput(apiKey.Get(), onKey, keyPlaceholder, "password", false)),
		settingRow("OpenAI model", modelHint, modelField),
		primaryButton("Save settings", onSave),
	)
}

// settingRow renders a labeled settings field with a hint.
func settingRow(label, hint string, field ui.Node) ui.Node {
	return Div(Class(Flex, FlexCol, Gap(Spacing2)),
		Span(Class(FontSemibold, TextSize(TextSm)), label),
		Span(Class(Fg(theme.Dim), TextSize(TextSm)), hint),
		field,
	)
}

// --- shared bits ---

// inputBase returns the shared input/select styling plus extras.
func inputBase(extra ...any) any {
	base := []any{Bg(theme.Bg), Border(theme.Border), Fg(theme.Fg), Rounded(RadiusLg), PadX(Spacing3), PadY(Spacing3), css.Raw("font", "inherit")}
	return Class(append(base, extra...)...)
}

// textInput renders a styled, controlled text/password input.
func textInput(value string, onInput ui.Handler, placeholder, typ string, autofocus bool) ui.Node {
	return Input(inputBase(rawWidth("100%")), Props{Value: value, OnInput: onInput, Placeholder: placeholder, Type: typ, AutoFocus: autofocus, AutoComplete: "off"})
}

// growInput is a controlled input that flexes to fill its row.
func growInput(value string, onInput ui.Handler, placeholder string) ui.Node {
	return Input(inputBase(css.Raw("flex", "1")), Props{Value: value, OnInput: onInput, Placeholder: placeholder, Type: "text"})
}

// primaryButton renders the filled accent button.
func primaryButton(label string, onClick ui.Handler) ui.Node {
	return Button(Class(Bg(theme.Accent), Fg(Hex("#ffffff")), FontSemibold, Rounded(RadiusLg), PadX(Spacing4), PadY(Spacing3),
		css.Raw("border", "0"), css.Raw("cursor", "pointer"), css.Raw("font", "inherit")), Props{OnClick: onClick}, label)
}

// ghostButton renders an outlined button.
func ghostButton(label string, onClick ui.Handler) ui.Node {
	return Button(Class(Fg(theme.Fg), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing4), PadY(Spacing2),
		css.Raw("background", "transparent"), css.Raw("cursor", "pointer"), css.Raw("font", "inherit")), Props{OnClick: onClick}, label)
}

// eyebrow renders a mono accent eyebrow.
func eyebrow(text string) ui.Node {
	return Div(Class(TextSize(TextSm), Fg(theme.Accent), Tracking(Ems(0.15))), text)
}

// sectionLabel renders an uppercase section heading.
func sectionLabel(text string) ui.Node {
	return Div(Class(TextSize(TextSm), Fg(theme.Dim), Tracking(Ems(0.12)), css.Raw("text-transform", "uppercase")), text)
}

// flashLine renders a colored status/error line (empty renders nothing).
func flashLine(text string, color css.Color) ui.Node {
	if text == "" {
		return Span()
	}
	return Div(Class(Fg(color), TextSize(TextSm)), text)
}

// keyStatus describes whether a secret is configured.
func keyStatus(set bool) string {
	if set {
		return "configured"
	}
	return "not set"
}

func rawWidth(v string) any { return css.Raw("width", v) }
func rawMinScreen() any     { return css.Raw("min-height", "100vh") }
func rawCentered() any      { return css.Raw("margin", "0 auto") }
