//go:build js && wasm

package main

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/css"
	. "github.com/monstercameron/GoWebComponents/v4/css/u"
	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/ui"

	"github.com/monstercameron/earlcameron/internal/theme"
	"github.com/monstercameron/earlcameron/proto/sitepb"
)

// --- login ---

// loginView renders the centered sign-in card.
func loginView(username, password ui.State[string], onLogin, onForgot ui.Handler, flash string) ui.Node {
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
		Span(Class(Fg(theme.Dim), TextSize(TextSm), css.Raw("cursor", "pointer"), Hover(Fg(theme.Accent))),
			Props{OnClick: onForgot}, "Forgot password?"),
	)
	return Div(Class(Flex, FlexCol, ItemsCenter, JustifyCenter, Gap(Spacing4), rawMinScreen(), Pad(Spacing5)),
		card,
		A(Class(Fg(theme.Dim), TextSize(TextSm)), Props{Href: "/"}, "← back to the site"),
	)
}

// setupView is the first-run screen: create the owner account. On success the client shows the
// recovery phrase (phraseView) once. setupToken is only needed when the server sets ADMIN_SETUP_TOKEN.
func setupView(username, password, hint, setupToken ui.State[string], onSetup ui.Handler, flash string) ui.Node {
	onU := ui.WrapHandler(func(e ui.Event) { username.Set(e.GetValue()) })
	onP := ui.WrapHandler(func(e ui.Event) { password.Set(e.GetValue()) })
	onH := ui.WrapHandler(func(e ui.Event) { hint.Set(e.GetValue()) })
	onT := ui.WrapHandler(func(e ui.Event) { setupToken.Set(e.GetValue()) })
	card := Div(Class(Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusXl), Pad(Spacing6), Flex, FlexCol, Gap(Spacing4), rawWidth("100%"), MaxWidth(Px(400))),
		eyebrow("~/admin · first run"),
		H2(Class(FontSize(Rem(1.5)), FontSemibold), "Create your owner account"),
		P(Class(Fg(theme.Dim), TextSize(TextSm)), "This is a fresh deploy. Choose the credentials that will protect the admin tools. You'll get a one-time recovery phrase to save."),
		labelled("Username", textInput(username.Get(), onU, "username", "text", true)),
		labelled("Password", textInput(password.Get(), onP, "at least 8 characters", "password", false)),
		labelled("Recovery hint", textInput(hint.Get(), onH, "shown on the reset screen — keep it vague (optional)", "text", false)),
		labelled("Setup token", textInput(setupToken.Get(), onT, "only if your server requires one", "password", false)),
		flashLine(flash, theme.Red),
		primaryButton("Create account", onSetup),
	)
	return Div(Class(Flex, FlexCol, ItemsCenter, JustifyCenter, Gap(Spacing4), rawMinScreen(), Pad(Spacing5)),
		card,
		A(Class(Fg(theme.Dim), TextSize(TextSm)), Props{Href: "/"}, "← back to the site"),
	)
}

// resetView is the password-reset screen: the owner enters the recovery phrase (or the env
// break-glass token) plus a new password. The stored hint is shown to jog their memory.
func resetView(hint string, phrase, newPass ui.State[string], onReset, onBack ui.Handler, flash string) ui.Node {
	onPh := ui.WrapHandler(func(e ui.Event) { phrase.Set(e.GetValue()) })
	onNp := ui.WrapHandler(func(e ui.Event) { newPass.Set(e.GetValue()) })
	rows := []ui.Node{
		eyebrow("~/admin · reset"),
		H2(Class(FontSize(Rem(1.5)), FontSemibold), "Reset your password"),
		P(Class(Fg(theme.Dim), TextSize(TextSm)), "Enter your recovery phrase to set a new password."),
	}
	if hint != "" {
		rows = append(rows, Div(Class(Bg(theme.Bg), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing3), PadY(Spacing2), TextSize(TextSm), Fg(theme.Dim)),
			Span(Class(Fg(theme.Faint)), "hint: "), Span(hint)))
	}
	rows = append(rows,
		labelled("Recovery phrase", textInput(phrase.Get(), onPh, "the words from setup", "text", true)),
		labelled("New password", textInput(newPass.Get(), onNp, "at least 8 characters", "password", false)),
		flashLine(flash, theme.Red),
		primaryButton("Reset password", onReset),
		Span(Class(Fg(theme.Dim), TextSize(TextSm), css.Raw("cursor", "pointer"), Hover(Fg(theme.Accent))),
			Props{OnClick: onBack}, "← back to sign in"),
	)
	card := Div(append([]any{Class(Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusXl), Pad(Spacing6), Flex, FlexCol, Gap(Spacing4), rawWidth("100%"), MaxWidth(Px(400)))}, toAny(rows)...)...)
	return Div(Class(Flex, FlexCol, ItemsCenter, JustifyCenter, Gap(Spacing4), rawMinScreen(), Pad(Spacing5)), card)
}

// phraseView shows a freshly generated recovery phrase once, with a strong save-it warning, then a
// continue button. It's shown after setup (isSetup: the user is now logged in) and after a reset.
func phraseView(phrase string, isSetup bool, onContinue ui.Handler) ui.Node {
	next := "Continue to admin"
	if !isSetup {
		next = "Back to sign in"
	}
	card := Div(Class(Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusXl), Pad(Spacing6), Flex, FlexCol, Gap(Spacing4), rawWidth("100%"), MaxWidth(Px(440))),
		eyebrow("~/admin · recovery phrase"),
		H2(Class(FontSize(Rem(1.5)), FontSemibold), "Save your recovery phrase"),
		P(Class(Fg(theme.Yellow), TextSize(TextSm)), "This is shown only once. Write it down and keep it somewhere safe — it's the only way to reset your password if you forget it."),
		Div(Class(Bg(theme.Bg), Border(theme.Accent), Rounded(RadiusLg), Pad(Spacing4),
			css.Raw("font-family", "ui-monospace,SFMono-Regular,Menlo,monospace"), FontSize(Rem(1.05)),
			css.Raw("letter-spacing", "0.02em"), css.Raw("line-height", "1.7"), Fg(theme.Fg),
			css.Raw("word-spacing", "0.25em")), phrase),
		primaryButton(next, onContinue),
	)
	return Div(Class(Flex, FlexCol, ItemsCenter, JustifyCenter, Gap(Spacing4), rawMinScreen(), Pad(Spacing5)), card)
}

// labelled wraps a control with a small uppercase field label above it.
func labelled(label string, control ui.Node) ui.Node {
	return Div(Class(Flex, FlexCol, Gap(Spacing1)),
		Span(Class(TextSize(TextXs), Fg(theme.Faint), css.Raw("letter-spacing", "0.06em")), strings.ToUpper(label)),
		control,
	)
}

// toAny converts a node slice to an []any for spreading into an element's variadic children.
func toAny(nodes []ui.Node) []any {
	out := make([]any, len(nodes))
	for i, n := range nodes {
		out[i] = n
	}
	return out
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
		Div(Class(Flex, Gap(Spacing5), ItemsCenter, css.Raw("flex-wrap", "wrap")),
			tab("anime", "anime"), tab("resume", "résumé"), tab("rss", "rss"), tab("cashflux", "cashflux"), tab("settings", "settings"),
			A(Class(Fg(theme.Dim), Hover(Fg(theme.Accent)), TextSize(TextSm)), Props{Href: "/budget/", Target: "_blank", Rel: "noopener"}, "budget ↗"),
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

// resumeView renders the tailor form; when a tailoring is pending it shows the extracted job
// signals, the rationales, a git-diff of the proposed changes, and Apply/Reanalyze/Cancel; otherwise
// it live-renders the active résumé.
func resumeView(jobURL ui.State[string], onTailor, onApply, onCancel ui.Handler, canonical, tailored *sitepb.Resume, job *sitepb.JobAnalysis, rationales []*sitepb.Rationale, variants []*sitepb.TailoringMeta, onSelect func(*sitepb.TailoringMeta), onDelete func(int64)) ui.Node {
	onURL := ui.WrapHandler(func(e ui.Event) { jobURL.Set(e.GetValue()) })
	sections := []any{Class(Flex, FlexCol, Gap(Spacing4)),
		P(Class(Fg(theme.Dim), TextSize(TextSm)),
			"Your ", Span(Class(FontSemibold, Fg(theme.Fg)), "base résumé"), " is permanent. Tailor it against a job URL to create a variant — the active one is served at ",
			A(Class(Fg(theme.Accent2)), Props{Href: "/resume", Target: "_blank", Rel: "noopener"}, "/resume"), "."),
		Div(Class(Flex, Gap(Spacing2)),
			growInput(jobURL.Get(), onURL, "paste a job-posting URL…"),
			primaryButton("Tailor résumé", onTailor),
		),
	}
	if tailored != nil {
		if job != nil || len(rationales) > 0 {
			panels := []any{Class(Grid, Gap(Spacing3), css.Raw("grid-template-columns", "repeat(auto-fit,minmax(280px,1fr))"))}
			if job != nil {
				panels = append(panels, jobAnalysisPanel(job))
			}
			if len(rationales) > 0 {
				panels = append(panels, rationalesPanel(rationales))
			}
			sections = append(sections, Div(panels...))
		}
		sections = append(sections, sectionLabel("proposed changes (vs base résumé)"))
		if canonical != nil {
			sections = append(sections, diffView(canonical, tailored))
		}
		sections = append(sections, Div(Class(Flex, Gap(Spacing2), css.Raw("flex-wrap", "wrap")),
			primaryButton("Apply", onApply),
			ghostButton("Reanalyze", onTailor),
			ghostButton("Cancel", onCancel),
		))
	} else if canonical != nil {
		sections = append(sections, sectionLabel("base résumé"), resumeDocument(canonical))
	}
	sections = append(sections, sectionLabel("saved variants"), variantsList(variants, onSelect, onDelete))
	return Div(sections...)
}

// variantsList renders the saved tailoring variants as a glanceable, CRUD-able grid.
func variantsList(variants []*sitepb.TailoringMeta, onSelect func(*sitepb.TailoringMeta), onDelete func(int64)) ui.Node {
	if len(variants) == 0 {
		return P(Class(Fg(theme.Dim), TextSize(TextSm)), "No saved variants yet — tailor against a job posting to create one.")
	}
	nodes := []any{Class(Grid, Gap(Spacing3), css.Raw("grid-template-columns", "repeat(auto-fill,minmax(300px,1fr))"))}
	for _, m := range variants {
		nodes = append(nodes, variantCard(m, onSelect, onDelete))
	}
	return Div(nodes...)
}

// variantCard renders one saved variant as a scannable card: the role + company, a source chip
// (the posting's domain, linked), the date, and actions — view/PDF (opens the variant's print page),
// tweak (re-open in the workspace), and delete.
func variantCard(m *sitepb.TailoringMeta, onSelect func(*sitepb.TailoringMeta), onDelete func(int64)) ui.Node {
	title := m.GetTitle()
	if title == "" {
		title = "Tailored variant"
	}
	date := time.Unix(m.GetCreatedAt(), 0).Format("Jan 2 · 3:04 pm")
	open := ui.WrapHandler(func() { onSelect(m) })
	del := ui.WrapHandler(func() { onDelete(m.GetId()) })
	viewURL := "/resume?variant=" + strconv.FormatInt(m.GetId(), 10)

	head := []any{Class(Flex, FlexCol, Gap(Spacing1)), Span(Class(FontSemibold, Fg(theme.Fg)), title)}
	if c := m.GetCompany(); c != "" {
		head = append(head, Span(Class(Fg(theme.Accent2), TextSize(TextSm)), c))
	}

	children := []any{Class(Flex, FlexCol, Gap(Spacing3), Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusLg), Pad(Spacing4),
		css.Raw("border-left", "3px solid "+accentHex), Transition(PropAll, Ms(160), Ease), Hover(Border(theme.Accent))),
		Div(head...),
	}
	// Glanceable focus: the job keywords the tailoring emphasized.
	if kws := m.GetKeywords(); len(kws) > 0 {
		chips := []any{Class(Flex, css.Raw("flex-wrap", "wrap"), Gap(Spacing1))}
		for _, k := range kws {
			chips = append(chips, Span(Class(TextSize(TextSm), Fg(theme.Dim), css.Raw("background", "#241a22"),
				Rounded(RadiusLg), PadX(Spacing2), PadY(Spacing1)), k))
		}
		children = append(children, Div(chips...))
	}
	children = append(children,
		Div(Class(Flex, ItemsCenter, Gap(Spacing2), css.Raw("flex-wrap", "wrap")),
			A(Class(TextSize(TextSm), Fg(theme.Accent2), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing2), PadY(Spacing1)),
				Props{Href: m.GetJobUrl(), Target: "_blank", Rel: "noopener"}, "↗ "+hostOf(m.GetJobUrl())),
			Span(Class(Fg(theme.Dim), TextSize(TextSm)), date),
		),
		Div(Class(Flex, Gap(Spacing4), ItemsCenter),
			A(Class(TextSize(TextSm), FontSemibold, Fg(theme.Accent)), Props{Href: viewURL, Target: "_blank", Rel: "noopener"}, "view / PDF ↗"),
			linkButton("tweak", open, theme.Fg),
			linkButton("delete", del, theme.Red),
		),
	)
	return Div(children...)
}

// linkButton renders a borderless text button that changes color on hover (subtle inline action).
func linkButton(label string, onClick ui.Handler, hover css.Color) ui.Node {
	return Button(Class(TextSize(TextSm), FontSemibold, Fg(theme.Dim), css.Raw("background", "transparent"),
		css.Raw("border", "0"), css.Raw("cursor", "pointer"), css.Raw("font", "inherit"), Hover(Fg(hover))),
		Props{OnClick: onClick}, label)
}

// hostOf returns a posting URL's bare host (for the source chip), or "posting" if unparseable.
func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "posting"
	}
	return strings.TrimPrefix(u.Host, "www.")
}

// accentHex is the Ubuntu-orange accent (matches theme.Accent) for raw-CSS fragments.
const accentHex = "#e95420"

// diffOp is one line in a git-style diff: kind -1 removed, 0 unchanged, +1 added.
type diffOp struct {
	kind int
	text string
}

// lineDiff computes a git-style line diff of a → b via a longest-common-subsequence walk.
func lineDiff(a, b []string) []diffOp {
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var ops []diffOp
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			ops = append(ops, diffOp{0, a[i]})
			i, j = i+1, j+1
		case dp[i+1][j] >= dp[i][j+1]:
			ops = append(ops, diffOp{-1, a[i]})
			i++
		default:
			ops = append(ops, diffOp{1, b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, diffOp{-1, a[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, diffOp{1, b[j]})
	}
	return ops
}

// diffView renders the ENTIRE résumé as a git-style diff: unchanged sections (header, skills,
// projects, education, job metadata) appear as context lines, and the changed parts (summary + each
// job's bullets) show as removed/added — so the whole document is visible with the edits in place.
func diffView(before, after *sitepb.Resume) ui.Node {
	rows := []any{Class(Flex, FlexCol, css.Raw("font-family", "ui-monospace,SFMono-Regular,Menlo,monospace"),
		css.Raw("font-size", "12.5px"), Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusLg), css.Raw("overflow", "hidden"))}
	ctx := func(text string) { rows = append(rows, diffLine(diffOp{0, text})) }
	hdr := func(label string) {
		rows = append(rows, Div(Class(Fg(theme.Dim), FontSemibold, css.Raw("padding", "6px 12px"),
			css.Raw("background", "#241a22"), css.Raw("letter-spacing", ".1em")), label))
	}
	diff := func(bs, as []string) {
		for _, op := range lineDiff(bs, as) {
			rows = append(rows, diffLine(op))
		}
	}

	ctx(after.GetName())
	ctx(after.GetTitle())
	ctx(after.GetLocation() + " · " + after.GetEmail() + " · " + after.GetGithub() + " · " + after.GetLinkedin())

	hdr("SUMMARY")
	diff([]string{before.GetSummary()}, []string{after.GetSummary()})

	hdr("EXPERIENCE")
	bj, aj := before.GetJobs(), after.GetJobs()
	for i := range aj {
		ctx(aj[i].GetRole() + "  (" + aj[i].GetDates() + ")")
		ctx(aj[i].GetOrg())
		var bb []string
		if i < len(bj) {
			bb = bj[i].GetBullets()
		}
		diff(bb, aj[i].GetBullets())
	}

	hdr("SKILLS")
	for _, sk := range after.GetSkills() {
		ctx(sk.GetLabel() + ": " + sk.GetItems())
	}
	hdr("SELECTED PROJECTS")
	for _, p := range after.GetProjects() {
		ctx(p.GetName() + " — " + p.GetDesc())
	}
	hdr("EDUCATION")
	for _, e := range after.GetEducation() {
		ctx(e)
	}
	return Div(rows...)
}

// diffLine renders one diff line with git-style +/-/context coloring.
func diffLine(op diffOp) ui.Node {
	bg, fg, prefix := "transparent", "#a98ba0", "  "
	switch op.kind {
	case -1:
		bg, fg, prefix = "#3a1418", "#ef9a9a", "- "
	case 1:
		bg, fg, prefix = "#12301a", "#a5d6a7", "+ "
	}
	return Div(Class(css.Raw("background", bg), css.Raw("color", fg), css.Raw("padding", "2px 12px"), css.Raw("white-space", "pre-wrap")), prefix+op.text)
}

// jobAnalysisPanel shows what the tool extracted from the posting.
func jobAnalysisPanel(job *sitepb.JobAnalysis) ui.Node {
	items := []any{Class(Flex, FlexCol, Gap(Spacing2), Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusLg), Pad(Spacing4)),
		Div(Class(FontSemibold, Fg(theme.Accent)), "extracted from the posting")}
	if title := job.GetTitle(); title != "" || job.GetCompany() != "" {
		if job.GetCompany() != "" {
			title += " @ " + job.GetCompany()
		}
		items = append(items, Span(Class(TextSize(TextSm), FontSemibold), title))
	}
	if len(job.GetKeywords()) > 0 {
		items = append(items, Span(Class(Fg(theme.Dim), TextSize(TextSm)), "keywords"), chipRow(job.GetKeywords()))
	}
	if len(job.GetRequirements()) > 0 {
		items = append(items, Span(Class(Fg(theme.Dim), TextSize(TextSm)), "requirements"))
		for _, r := range job.GetRequirements() {
			items = append(items, Span(Class(TextSize(TextSm)), "• "+r))
		}
	}
	return Div(items...)
}

// rationalesPanel explains each tailoring decision.
func rationalesPanel(rationales []*sitepb.Rationale) ui.Node {
	items := []any{Class(Flex, FlexCol, Gap(Spacing3), Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusLg), Pad(Spacing4)),
		Div(Class(FontSemibold, Fg(theme.Accent)), "why these choices")}
	for _, r := range rationales {
		items = append(items, Div(Class(Flex, FlexCol),
			Span(Class(FontSemibold, TextSize(TextSm)), r.GetFocus()),
			Span(Class(Fg(theme.Dim), TextSize(TextSm)), r.GetReason()),
		))
	}
	return Div(items...)
}

// chipRow renders a wrapped row of pill chips.
func chipRow(values []string) ui.Node {
	nodes := []any{Class(Flex, css.Raw("flex-wrap", "wrap"), Gap(Spacing2))}
	for _, v := range values {
		nodes = append(nodes, Span(Class(TextSize(TextSm), Fg(theme.Fg), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing2), PadY(Spacing1)), v))
	}
	return Div(nodes...)
}

// resumeDocument renders the résumé as a light "paper" preview, mirroring the /resume PDF layout.
func resumeDocument(r *sitepb.Resume) ui.Node {
	sec := func(t string) ui.Node {
		return Div(Class(css.Raw("font-size", "11px"), css.Raw("letter-spacing", ".12em"), css.Raw("text-transform", "uppercase"),
			css.Raw("color", "#c1440f"), css.Raw("font-weight", "700"), css.Raw("margin-top", "16px")), t)
	}
	items := []any{Class(css.Raw("background", "#ffffff"), css.Raw("color", "#1b1420"), Rounded(RadiusXl),
		css.Raw("padding", "28px 32px"), css.Raw("font-family", "-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif"), css.Raw("line-height", "1.5")),
		Div(Class(css.Raw("font-size", "24px"), css.Raw("font-weight", "700")), r.GetName()),
		Div(Class(css.Raw("color", "#e95420"), css.Raw("font-weight", "600")), r.GetTitle()),
		Div(Class(css.Raw("color", "#555"), css.Raw("font-size", "13px"), css.Raw("margin-top", "4px")),
			r.GetLocation()+" · "+r.GetEmail()+" · "+r.GetGithub()+" · "+r.GetLinkedin()),
		sec("Summary"), Div(r.GetSummary()),
		sec("Experience"),
	}
	for _, j := range r.GetJobs() {
		items = append(items,
			Div(Class(css.Raw("display", "flex"), css.Raw("justify-content", "space-between"), css.Raw("margin-top", "8px")),
				Span(Class(css.Raw("font-weight", "700")), j.GetRole()),
				Span(Class(css.Raw("color", "#666"), css.Raw("font-size", "13px")), j.GetDates())),
			Div(Class(css.Raw("color", "#444"), css.Raw("font-size", "14px")), j.GetOrg()))
		ul := []any{Class(css.Raw("margin", "4px 0 0"), css.Raw("padding-left", "18px"))}
		for _, b := range j.GetBullets() {
			ul = append(ul, Tag("li", b))
		}
		items = append(items, Tag("ul", ul...))
	}
	items = append(items, sec("Skills"))
	for _, sk := range r.GetSkills() {
		items = append(items, Div(Class(css.Raw("font-size", "14px")),
			Span(Class(css.Raw("color", "#e95420"), css.Raw("font-weight", "600")), sk.GetLabel()+": "), Span(sk.GetItems())))
	}
	items = append(items, sec("Selected Projects"))
	pul := []any{Class(css.Raw("margin", "4px 0 0"), css.Raw("padding-left", "18px"))}
	for _, p := range r.GetProjects() {
		pul = append(pul, Tag("li", Span(Class(css.Raw("font-weight", "700")), p.GetName()), Span(" — "+p.GetDesc())))
	}
	items = append(items, Tag("ul", pul...), sec("Education"))
	eul := []any{Class(css.Raw("margin", "4px 0 0"), css.Raw("padding-left", "18px"))}
	for _, e := range r.GetEducation() {
		eul = append(eul, Tag("li", e))
	}
	items = append(items, Tag("ul", eul...))
	return Div(items...)
}

// --- cashflux device pairing ---

// cashfluxView renders the CashFlux device-pairing panel: the pending-devices list, each with
// Pair/Reject actions, and — right after a successful Pair — a callout with the pairing code for the
// human cross-check against the device's own display. configured is false when CashFlux embedding
// isn't set up on this deployment, in which case everything else is skipped. busy is the set of
// device ids currently mid-approve/reject (keyed by device_id, present+true = in flight) — a set
// rather than a single id, so one row's in-flight request never re-enables another row's buttons.
func cashfluxView(configured bool, pending []*sitepb.CashFluxPendingDevice, busy map[string]bool, justPaired *sitepb.CashFluxPendingDevice, pairingCode string, copied bool, onApprove func(string, *sitepb.CashFluxPendingDevice), onReject func(string), onCopy ui.Handler) ui.Node {
	if !configured {
		return Div(Class(Flex, FlexCol, Gap(Spacing2)),
			sectionLabel("cashflux pending devices"),
			P(Class(Fg(theme.Dim), TextSize(TextSm)), "CashFlux sync isn't configured on this deployment (no CASHFLUX_DATA_DIR set)."),
		)
	}

	sections := []any{Class(Flex, FlexCol, Gap(Spacing5)),
		Div(Class(Flex, FlexCol, Gap(Spacing2), MaxWidth(Px(560))),
			sectionLabel("pending devices"),
			P(Class(Fg(theme.Dim), TextSize(TextSm)),
				"A device asking to sync shows up here. Pair it to create an account and mint a pairing code — read the code out (or have them compare it against their own screen) before they accept, so you're pairing the device you meant to."),
		),
	}
	if justPaired != nil {
		sections = append(sections, pairedCodeCallout(justPaired, pairingCode, copied, onCopy))
	}
	sections = append(sections, pendingDeviceList(pending, busy, onApprove, onReject))
	return Div(sections...)
}

// pairedCodeCallout highlights the pairing code from the most recent approval — large, monospace,
// with a one-click copy button — mirroring phraseView's "shown once, save it" treatment for the
// recovery phrase. copyLabel switches to a confirmation once copied, rather than a separate toast/timer.
func pairedCodeCallout(device *sitepb.CashFluxPendingDevice, code string, copied bool, onCopy ui.Handler) ui.Node {
	copyLabel := "Copy code"
	if copied {
		copyLabel = "Copied ✓"
	}
	return Div(Class(Bg(theme.Bg), Border(theme.Accent), Rounded(RadiusLg), Pad(Spacing4), Flex, FlexCol, Gap(Spacing3), MaxWidth(Px(360))),
		Span(Class(TextSize(TextXs), Fg(theme.Faint), css.Raw("letter-spacing", "0.06em")), "READ THIS TO “"+strings.ToUpper(device.GetLabel())+"”"),
		Div(Class(Flex, ItemsCenter, JustifyBetween, Gap(Spacing3)),
			Div(Class(css.Raw("font-family", "ui-monospace,SFMono-Regular,Menlo,monospace"), FontSize(Rem(1.6)),
				css.Raw("letter-spacing", "0.08em"), FontSemibold, Fg(theme.Fg)), code),
			ghostButton(copyLabel, onCopy),
		),
		Span(Class(TextSize(TextSm), Fg(theme.Dim)), "confirm it matches what their device shows before they accept"),
	)
}

// pendingDeviceList renders unresolved pairing requests as scannable rows with Pair/Reject actions.
func pendingDeviceList(devices []*sitepb.CashFluxPendingDevice, busy map[string]bool, onApprove func(string, *sitepb.CashFluxPendingDevice), onReject func(string)) ui.Node {
	if len(devices) == 0 {
		return P(Class(Fg(theme.Dim), TextSize(TextSm)), "No devices waiting to pair.")
	}
	nodes := []any{Class(Flex, FlexCol, Gap(Spacing2))}
	for _, d := range devices {
		nodes = append(nodes, pendingDeviceRow(d, busy[d.GetDeviceId()], onApprove, onReject))
	}
	return Div(nodes...)
}

// pendingDeviceRow renders one pending device request: label, requested/expiry time, and its own
// Pair/Reject buttons. It's a standalone component (not inlined in a loop) per the CashFlux hooks
// gotcha — On*/WrapHandler-backed controls in a variable-length list must live in their own component
// so each row gets a stable render position.
func pendingDeviceRow(d *sitepb.CashFluxPendingDevice, busy bool, onApprove func(string, *sitepb.CashFluxPendingDevice), onReject func(string)) ui.Node {
	deviceID := d.GetDeviceId()
	requested := time.Unix(d.GetRequestedAt(), 0).Format("Jan 2 · 3:04 pm")
	expires := "expires " + time.Unix(d.GetExpiresAt(), 0).Format("Jan 2 · 3:04 pm")

	pairLabel, rejectLabel := "Pair", "Reject"
	if busy {
		pairLabel, rejectLabel = "…", "…"
	}
	onPair := ui.WrapHandler(func() { onApprove(deviceID, d) })
	onDecline := ui.WrapHandler(func() { onReject(deviceID) })

	return Div(Class(Flex, ItemsCenter, JustifyBetween, Gap(Spacing3), Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing3), PadY(Spacing2)),
		Div(Class(Flex, FlexCol, Gap(Spacing1)),
			Span(Class(FontSemibold, TextSize(TextSm)), d.GetLabel()),
			Span(Class(TextSize(TextSm), Fg(theme.Dim)), "requested "+requested+" · "+expires),
		),
		Div(Class(Flex, Gap(Spacing2)),
			Button(Class(Bg(theme.Accent), Fg(Hex("#ffffff")), FontSemibold, Rounded(RadiusLg), PadX(Spacing4), PadY(Spacing3),
				css.Raw("border", "0"), css.Raw("cursor", "pointer"), css.Raw("font", "inherit"), DisabledIf(busy)),
				Props{OnClick: onPair, Disabled: busy}, pairLabel),
			Button(Class(Fg(theme.Fg), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing4), PadY(Spacing2),
				css.Raw("background", "transparent"), css.Raw("cursor", "pointer"), css.Raw("font", "inherit"), DisabledIf(busy)),
				Props{OnClick: onDecline, Disabled: busy}, rejectLabel),
		),
	)
}

// --- settings view ---

// settingsView renders the OpenAI key + model form (a dropdown when models are known). onReload
// re-fetches the model list from OpenAI using the stored key.
func settingsView(keySet bool, models []string, model, apiKey ui.State[string], onSave, onReload ui.Handler) ui.Node {
	onKey := ui.WrapHandler(func(e ui.Event) { apiKey.Set(e.GetValue()) })
	onModel := ui.WrapHandler(func(e ui.Event) { model.Set(e.GetValue()) })

	keyPlaceholder := "sk-…"
	if keySet {
		keyPlaceholder = "a key is set — leave blank to keep it"
	}

	var modelField ui.Node = textInput(model.Get(), onModel, "gpt-4o-mini", "text", false)
	modelHint := "save a key, then hit “reload models” to load the list"
	switch {
	case len(models) > 0:
		modelHint = "pick a model available to your key"
		opts := []any{inputBase(rawWidth("100%")), Props{OnChange: onModel}}
		for _, m := range models {
			opts = append(opts, Tag("option", Props{Value: m, Selected: m == model.Get()}, m))
		}
		modelField = Tag("select", opts...)
	case keySet:
		modelHint = "no models loaded yet — hit “reload models” (or the key may be invalid)"
	}

	// The model row pairs the field with a reload button that re-fetches the list from OpenAI.
	modelControl := Div(Class(Flex, Gap(Spacing2), ItemsCenter),
		Div(Class(css.Raw("flex", "1")), modelField),
		ghostButton("reload models", onReload),
	)

	return Div(Class(Flex, FlexCol, Gap(Spacing4), MaxWidth(Px(520))),
		P(Class(Fg(theme.Dim), TextSize(TextSm)), "The OpenAI key is stored in the backend database and used server-side."),
		settingRow("OpenAI API key", "openai key ("+keyStatus(keySet)+")", textInput(apiKey.Get(), onKey, keyPlaceholder, "password", false)),
		settingRow("OpenAI model", modelHint, modelControl),
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

// --- rss / anime control panel ---

// rssView renders the RSS control panel: the public feed links, the single QOTD generation prompt
// (edit → dry-run preview → save), and the Slack "generate & publish" config.
func rssView(promptText ui.State[string], onSavePrompt, onDryRun ui.Handler, dryRunning bool, preview *sitepb.PostPreview,
	slackWebhook ui.State[string], slackSet, slackEnabled bool, slackHour ui.State[int], onToggleSlack, onSaveSlack, onPostNow ui.Handler) ui.Node {
	onPrompt := ui.WrapHandler(func(e ui.Event) { promptText.Set(e.GetValue()) })
	onWebhook := ui.WrapHandler(func(e ui.Event) { slackWebhook.Set(e.GetValue()) })
	onHour := ui.WrapHandler(func(e ui.Event) {
		if h, err := strconv.Atoi(strings.TrimSpace(e.GetValue())); err == nil && h >= 0 && h <= 23 {
			slackHour.Set(h)
		}
	})

	webhookPlaceholder := "https://hooks.slack.com/services/…"
	if slackSet {
		webhookPlaceholder = "a webhook is set — leave blank to keep it"
	}
	schedule := "scheduled posting: off"
	if slackEnabled {
		schedule = "scheduled posting: on"
	}
	dryLabel := "Dry run — generate a test post"
	if dryRunning {
		dryLabel = "Generating…"
	}

	return Div(Class(Flex, FlexCol, Gap(Spacing5)),
		Div(Class(Flex, FlexCol, Gap(Spacing2)),
			sectionLabel("public rss feeds"),
			Div(Class(Flex, Gap(Spacing4), css.Raw("flex-wrap", "wrap"), TextSize(TextSm)),
				A(Class(Fg(theme.Accent2)), Props{Href: "/anime.xml", Target: "_blank", Rel: "noopener"}, "↗ /anime.xml — Release Radar"),
				A(Class(Fg(theme.Accent2)), Props{Href: "/anime/qotd.xml", Target: "_blank", Rel: "noopener"}, "↗ /anime/qotd.xml — QOTD"),
			),
		),
		Div(Class(Flex, FlexCol, Gap(Spacing3), MaxWidth(Px(760))),
			sectionLabel("anime discussion prompt"),
			P(Class(Fg(theme.Dim), TextSize(TextSm)),
				"One instruction that turns the latest anime headline into a discussion post for Slack + the QOTD feed. Edit it, dry-run to preview what it generates, then save."),
			Textarea(inputBase(rawWidth("100%"), css.Raw("min-height", "150px"), css.Raw("resize", "vertical"), css.Raw("line-height", "1.5")),
				Props{Value: promptText.Get(), OnInput: onPrompt, Placeholder: "Write an instruction for the model…"}),
			Div(Class(Flex, Gap(Spacing3), ItemsCenter, css.Raw("flex-wrap", "wrap")),
				primaryButton("Save prompt", onSavePrompt),
				ghostButton(dryLabel, onDryRun),
			),
			previewBlock(preview),
		),
		Div(Class(Flex, FlexCol, Gap(Spacing3), MaxWidth(Px(560))),
			sectionLabel("slack — generate & publish"),
			P(Class(Fg(theme.Dim), TextSize(TextSm)),
				"Generates a post from the saved prompt and publishes it to the QOTD RSS feed; if a Slack webhook is set, it also posts to your Slack channel. Turn on scheduled posting to have the server do this automatically once a day."),
			settingRow("Webhook URL ("+keyStatus(slackSet)+")", "stored in the backend, never shown",
				textInput(slackWebhook.Get(), onWebhook, webhookPlaceholder, "password", false)),
			settingRow("Daily post hour (0–23, server time)", "when scheduled posting is on, the post fires once a day around this hour",
				textInput(strconv.Itoa(slackHour.Get()), onHour, "9", "number", false)),
			Div(Class(Flex, Gap(Spacing3), ItemsCenter, css.Raw("flex-wrap", "wrap")),
				ghostButton(schedule, onToggleSlack),
				primaryButton("Save Slack config", onSaveSlack),
				ghostButton("Generate & post now", onPostNow),
			),
		),
	)
}

// previewBlock renders a dry-run result — the headline used, the generated post, and the rendered RSS
// XML — or an error. It returns an empty span when there is no preview yet.
func previewBlock(p *sitepb.PostPreview) ui.Node {
	if p == nil {
		return Span()
	}
	if p.GetError() != "" {
		return Div(Class(TextSize(TextSm), Fg(theme.Red), Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing3), PadY(Spacing3)),
			"⚠ "+p.GetError())
	}
	rows := []any{Class(Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusLg), Pad(Spacing4), Flex, FlexCol, Gap(Spacing3)),
		sectionLabel("preview — not published"),
	}
	if p.GetNews() != "" {
		rows = append(rows, Span(Class(TextSize(TextXs), Fg(theme.Faint)), "based on headline: "+p.GetNews()))
	}
	rows = append(rows,
		Span(Class(FontSemibold, TextSize(TextSm)), p.GetTitle()),
		P(Class(TextSize(TextSm), css.Raw("white-space", "pre-wrap"), css.Raw("line-height", "1.5")), p.GetBody()),
		Tag("pre", Class(TextSize(TextXs), Fg(theme.Dim), Bg(theme.Bg), Border(theme.Border), Rounded(RadiusMd), Pad(Spacing3),
			css.Raw("overflow-x", "auto"), css.Raw("white-space", "pre"), css.Raw("max-height", "220px"),
			css.Raw("font-family", "ui-monospace,SFMono-Regular,Menlo,monospace")), p.GetRss()),
	)
	return Div(rows...)
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
