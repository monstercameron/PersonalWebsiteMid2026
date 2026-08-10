//go:build js && wasm

package main

import (
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/css"
	. "github.com/monstercameron/GoWebComponents/v5/css/u"
	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/ui"

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
func consoleShell(active string, navTo func(string) ui.Handler, onLogout, onOpenBudget ui.Handler, flash string, content ui.Node) ui.Node {
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
			// Opens CashFlux already signed in: the click mints an activation code
			// against this authenticated console session and carries it over, so the
			// operator never types a credential to reach their own data. Falls back to
			// a plain link if minting fails — a broken shortcut must not become a
			// broken way in.
			Span(Class(Fg(theme.Dim), Hover(Fg(theme.Accent)), TextSize(TextSm), css.Raw("cursor", "pointer")),
				Props{OnClick: onOpenBudget}, "budget ↗"),
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

// cashfluxView renders the CashFlux admin panel: the pending-devices list, each with Pair/Reject
// actions, and — right after a successful Pair — a callout with the pairing code for the human
// cross-check against the device's own display; below that, the storage stats (database +
// artifact-blob sizes) and the enrolled-users list (signup date, provider, subscription, this
// month's request volume, with a "load more" action). configured is false when CashFlux embedding
// isn't set up on this deployment, in which case everything else is skipped. busy is the set of device ids
// currently mid-approve/reject (keyed by device_id, present+true = in flight) — a set rather than a
// single id, so one row's in-flight request never re-enables another row's buttons. users/usersMore/
// usersLoading/onLoadMoreUsers drive the users list's single page + "load more" action; storage is
// nil until the storage-stats call returns (storageStatsSection renders nothing until then).
func cashfluxView(configured bool, activation activationCodeState, pending []*sitepb.CashFluxPendingDevice, busy map[string]bool, justPaired *sitepb.CashFluxPendingDevice, pairingCode string, copied bool, onApprove func(string, *sitepb.CashFluxPendingDevice), onReject func(string), onCopy ui.Handler,
	users usersPanelState, storage *sitepb.CashFluxStorageStats) ui.Node {
	if !configured {
		return Div(Class(Flex, FlexCol, Gap(Spacing2)),
			sectionLabel("cashflux pending devices"),
			P(Class(Fg(theme.Dim), TextSize(TextSm)), "CashFlux sync isn't configured on this deployment (no CASHFLUX_DATA_DIR set)."),
		)
	}

	sections := []any{Class(Flex, FlexCol, Gap(Spacing5)),
		activationCodeSection(activation),
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
	sections = append(sections, storageStatsSection(storage))
	sections = append(sections, usersSection(users))
	return Div(sections...)
}

// activationCodeState is the activation-code panel's whole state, grouped into one value rather
// than threaded through cashfluxView as six more positional parameters.
type activationCodeState struct {
	// Code is the code currently on screen, or "" when none has been minted this visit.
	Code string
	// ExpiresAt is when Code stops working, RFC3339 in UTC. Empty whenever Code is.
	ExpiresAt string
	// Minting is true while a mint request is in flight, so the button can't be double-fired.
	Minting bool
	// Copied is true once Code has been copied to the clipboard, until the next mint.
	Copied bool
	OnMint ui.Handler
	OnCopy ui.Handler
}

// usersPanelState is the enrolled-users panel's whole state, grouped so cashfluxView keeps a
// readable signature as the panel grows.
type usersPanelState struct {
	Users []*sitepb.CashFluxUser
	// More is true when the last page came back full, so a "Load more" action is offered.
	More    bool
	Loading bool
	// ConfirmDeleteID is the one row currently showing its are-you-sure state, "" for none.
	// A single id rather than a set: two half-confirmed deletions on screen at once is a
	// misclick waiting to happen.
	ConfirmDeleteID string
	// Deleting is the set of ids with a purge in flight, keyed by user id — a set, like the
	// pending-device busy map, so one row's request never re-enables another's buttons.
	Deleting        map[string]bool
	OnLoadMore      ui.Handler
	OnAskDelete     func(string)
	OnCancelDelete  func()
	OnConfirmDelete func(string)
	// Management actions. Roles are what an account may DO; suspend freezes it
	// without losing a byte; reset clears its password and signs it out everywhere.
	OnSetRole func(userID, role string)
	OnSuspend func(userID string, suspended bool)
	OnReset   func(userID string)
	// Inviting a person: create the account, then hand them a code for it. Both are
	// needed — an account with no way to mint it a code is an account nobody can
	// ever sign in to.
	NewName   ui.State[string]
	NewRole   ui.State[string]
	OnAddUser ui.Handler
	OnMintFor func(userID string)
	// CodeForID / Code hold the most recently minted per-account code, shown on that
	// row only. Single-use and short-lived, so leaving it on screen is harmless.
	CodeForID string
	Code      string
}

// activationCodeSection renders the activation-code panel — the primary way a device gets into
// CashFlux on this deployment. Minting a code is the entire access control: it takes an admin
// session on this site, and without a code there is no way in at all (the embedded CashFlux bridge
// disables self-signup). The minted code reuses pairedCodeCallout's large-monospace-plus-copy
// treatment on purpose: it is the same kind of object, so it should be recognized at a glance
// rather than learned twice.
func activationCodeSection(a activationCodeState) ui.Node {
	buttonLabel := "Generate code"
	if a.Code != "" {
		buttonLabel = "Generate another"
	}
	if a.Minting {
		buttonLabel = "Generating…"
	}

	nodes := []any{Class(Flex, FlexCol, Gap(Spacing3), MaxWidth(Px(560))),
		sectionLabel("activate a device"),
		P(Class(Fg(theme.Dim), TextSize(TextSm)),
			"Generate a code, then enter it in CashFlux under Settings → Cloud on the device you want to sync. Each code works once and expires in five minutes. Every code opens the same account, so every device you activate shares one set of data."),
		Div(Class(Flex),
			Button(Class(Bg(theme.Accent), Fg(Hex("#ffffff")), FontSemibold, Rounded(RadiusLg), PadX(Spacing4), PadY(Spacing3),
				css.Raw("border", "0"), css.Raw("cursor", "pointer"), css.Raw("font", "inherit"), DisabledIf(a.Minting)),
				Props{OnClick: a.OnMint, Disabled: a.Minting}, buttonLabel),
		),
	}
	if a.Code != "" {
		nodes = append(nodes, activationCodeCallout(a))
	}
	return Div(nodes...)
}

// activationCodeCallout shows a freshly minted code with its own copy button and, when the server's
// timestamp parses, the wall-clock time it dies. An unparseable timestamp drops the expiry line
// rather than printing a raw RFC3339 string at someone — the code is still perfectly usable, and a
// malformed clock reading is worse than none.
func activationCodeCallout(a activationCodeState) ui.Node {
	copyLabel := "Copy code"
	if a.Copied {
		copyLabel = "Copied ✓"
	}
	footer := "single use — generate another any time"
	if t, err := time.Parse(time.RFC3339, a.ExpiresAt); err == nil {
		footer = "expires " + t.Local().Format("3:04 pm") + " · single use"
	}
	return Div(Class(Bg(theme.Bg), Border(theme.Accent), Rounded(RadiusLg), Pad(Spacing4), Flex, FlexCol, Gap(Spacing3), MaxWidth(Px(360))),
		Span(Class(TextSize(TextXs), Fg(theme.Faint), css.Raw("letter-spacing", "0.06em")), "ENTER THIS IN CASHFLUX"),
		Div(Class(Flex, ItemsCenter, JustifyBetween, Gap(Spacing3)),
			Div(Class(css.Raw("font-family", "ui-monospace,SFMono-Regular,Menlo,monospace"), FontSize(Rem(1.6)),
				css.Raw("letter-spacing", "0.08em"), FontSemibold, Fg(theme.Fg)), a.Code),
			ghostButton(copyLabel, a.OnCopy),
		),
		Span(Class(TextSize(TextSm), Fg(theme.Dim)), footer),
	)
}

// storageStatsSection renders the two on-disk-size numbers (database + artifact blobs) as a small
// stat row, human-readable rather than raw byte counts. nil (stats not loaded yet) renders nothing.
func storageStatsSection(stats *sitepb.CashFluxStorageStats) ui.Node {
	if stats == nil {
		return Span()
	}
	return Div(Class(Flex, FlexCol, Gap(Spacing2)),
		sectionLabel("storage"),
		Div(Class(Flex, Gap(Spacing3), css.Raw("flex-wrap", "wrap")),
			// Synced data leads: it is the only one of the three that moves every time
			// someone pushes. The database's own size barely does — a 17KB dataset in a
			// 284KB file is lost in the page-count rounding — and blobs stay at 0 for
			// anyone with no attachments, which together made this panel read as
			// "nothing is happening" on a server that was syncing fine.
			storageStatTile("Synced data", formatBytes(stats.GetSnapshotBytes())),
			storageStatTile("Artifact blobs", formatBytes(stats.GetBlobBytes())),
			storageStatTile("Database file", formatBytes(stats.GetDbBytes())),
		),
	)
}

// storageStatTile renders one labeled size figure.
func storageStatTile(label, value string) ui.Node {
	return Div(Class(Flex, FlexCol, Gap(Spacing1), Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing4), PadY(Spacing3), MinWidth(Px(160))),
		Span(Class(TextSize(TextXs), Fg(theme.Faint), css.Raw("letter-spacing", "0.06em"), css.Raw("text-transform", "uppercase")), label),
		Span(Class(FontSize(Rem(1.3)), FontSemibold, Fg(theme.Fg)), value),
	)
}

// usersSection renders the enrolled-users list: signup date, provider, email (or id when no email is
// on file), subscription plan/status, and this calendar month's request volume — plus a "load more"
// action when the last page came back full.
func usersSection(u usersPanelState) ui.Node {
	rows := []any{Class(Flex, FlexCol, Gap(Spacing3), MaxWidth(Px(760))),
		sectionLabel("users (" + strconv.Itoa(len(u.Users)) + ")"),
	}
	rows = append(rows, inviteRow(u))
	if len(u.Users) == 0 {
		rows = append(rows, P(Class(Fg(theme.Dim), TextSize(TextSm)), "No one has signed up yet."))
		return Div(rows...)
	}
	list := []any{Class(Flex, FlexCol, Gap(Spacing2))}
	for _, user := range u.Users {
		code := ""
		if u.CodeForID == user.GetId() {
			code = u.Code
		}
		list = append(list, userRow(user, u.ConfirmDeleteID == user.GetId(), u.Deleting[user.GetId()],
			u.OnAskDelete, u.OnCancelDelete, u.OnConfirmDelete, u.OnSetRole, u.OnSuspend, u.OnReset,
			u.OnMintFor, code))
	}
	rows = append(rows, Div(list...))
	if u.More {
		label := "Load more"
		if u.Loading {
			label = "Loading…"
		}
		rows = append(rows, ghostButton(label, u.OnLoadMore))
	}
	return Div(rows...)
}

// userRow renders one enrolled account as a scannable row: who (email, falling back to id, plus
// provider), when they signed up, their subscription (when set), this month's request volume, and
// a Delete action. Standalone component (not inlined in usersSection's loop) per the same hooks
// rule pendingDeviceRow follows: WrapHandler-backed controls in a variable-length list need their
// own stable render position.
//
// Delete is two-step and the row itself is the confirmation — no modal, and no way to erase an
// account with a single click. confirming swaps the row for its own are-you-sure state; deleting
// disables both buttons while the purge is in flight.
func userRow(u *sitepb.CashFluxUser, confirming, deleting bool, onAsk func(string), onCancel func(), onConfirm func(string),
	onSetRole func(string, string), onSuspend func(string, bool), onReset func(string),
	onMintFor func(string), code string) ui.Node {
	id := u.GetId()
	who := u.GetEmail()
	if who == "" {
		who = id
	}
	if confirming {
		return deleteConfirmRow(u, who, deleting, onCancel, onConfirm)
	}
	signedUp := "unknown signup date"
	if t := u.GetCreatedAt(); t > 0 {
		signedUp = "joined " + time.Unix(t, 0).Format("Jan 2, 2006")
	}
	meta := signedUp
	if p := u.GetProvider(); p != "" {
		meta += " · " + p
	}
	if u.GetSuspended() {
		meta += " · suspended"
	}
	if plan := u.GetSubscriptionPlan(); plan != "" {
		meta += " · " + plan
		if s := u.GetSubscriptionStatus(); s != "" {
			meta += " (" + s + ")"
		}
	}
	onDelete := ui.WrapHandler(func() { onAsk(id) })
	suspended := u.GetSuspended()
	suspendLabel := "Suspend"
	if suspended {
		suspendLabel = "Restore"
	}
	onToggleSuspend := ui.WrapHandler(func() { onSuspend(id, !suspended) })
	onResetCreds := ui.WrapHandler(func() { onReset(id) })
	onMintCode := ui.WrapHandler(func() { onMintFor(id) })
	// The owner row is the one whose deletion costs YOU data, so it carries a badge before
	// anyone reaches for the button — not only inside the confirmation.
	name := []any{Class(Flex, ItemsCenter, Gap(Spacing2)),
		Span(Class(FontSemibold, TextSize(TextSm)), who),
	}
	if u.GetIsOwner() {
		name = append(name, Span(Class(TextSize(TextXs), Fg(theme.Accent2), Border(theme.Border), Rounded(RadiusSm), PadX(Spacing2)), "your account"))
	}
	if code != "" {
		return Div(Class(Flex, FlexCol, Gap(Spacing2), Bg(theme.Bg), Border(theme.Accent), Rounded(RadiusLg), PadX(Spacing3), PadY(Spacing3)),
			Span(Class(TextSize(TextXs), Fg(theme.Faint), css.Raw("letter-spacing", "0.06em")), "ACTIVATION CODE FOR "+strings.ToUpper(who)),
			Div(Class(css.Raw("font-family", "ui-monospace,SFMono-Regular,Menlo,monospace"), FontSize(Rem(1.6)),
				css.Raw("letter-spacing", "0.08em"), FontSemibold, Fg(theme.Fg)),
				Props{Data: map[string]string{"testid": "user-activation-code"}}, code),
			Span(Class(TextSize(TextSm), Fg(theme.Dim)), "they enter this in CashFlux under Settings → Cloud · single use · 5 minutes"),
		)
	}
	return Div(Class(Flex, ItemsCenter, JustifyBetween, Gap(Spacing3), Bg(theme.BgRaised), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing3), PadY(Spacing2)),
		Div(Class(Flex, FlexCol, Gap(Spacing1)),
			Div(name...),
			Span(Class(TextSize(TextSm), Fg(theme.Dim)), meta),
		),
		Div(Class(Flex, ItemsCenter, Gap(Spacing3)),
			// What this account actually HOLDS, not how many AI requests it bought.
			// requests_this_month is written only by the metered AI proxy — the sync
			// path never touches it — so it read 0 forever for an account syncing
			// constantly, and this column was the reason the panel looked dead.
			Div(Class(Flex, FlexCol, Gap(Spacing1), css.Raw("text-align", "right")),
				Span(Class(FontSemibold, TextSize(TextSm), Fg(theme.Accent2)), userDataLabel(u)),
				Span(Class(TextSize(TextXs), Fg(theme.Faint)), userSyncedLabel(u)),
			),
			roleSelect(u, onSetRole),
			Button(Class(Fg(theme.Dim), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing3), PadY(Spacing2), TextSize(TextSm),
				css.Raw("background", "transparent"), css.Raw("cursor", "pointer"), css.Raw("font", "inherit"), Hover(Fg(theme.Accent))),
				Props{OnClick: onMintCode}, "Code"),
			Button(Class(Fg(theme.Dim), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing3), PadY(Spacing2), TextSize(TextSm),
				css.Raw("background", "transparent"), css.Raw("cursor", "pointer"), css.Raw("font", "inherit"), Hover(Fg(theme.Fg))),
				Props{OnClick: onToggleSuspend}, suspendLabel),
			Button(Class(Fg(theme.Dim), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing3), PadY(Spacing2), TextSize(TextSm),
				css.Raw("background", "transparent"), css.Raw("cursor", "pointer"), css.Raw("font", "inherit"), Hover(Fg(theme.Fg))),
				Props{OnClick: onResetCreds}, "Reset"),
			Button(Class(Fg(theme.Dim), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing3), PadY(Spacing2), TextSize(TextSm),
				css.Raw("background", "transparent"), css.Raw("cursor", "pointer"), css.Raw("font", "inherit"), Hover(Fg(theme.Red))),
				Props{OnClick: onDelete}, "Delete"),
		),
	)
}

// roleSelect is the per-row role picker. The OWNER row gets a static label instead:
// pkg/embed refuses to demote that account (it is what every activation code binds
// to), so offering a control that can only fail would be a lie about what is
// possible.
func roleSelect(u *sitepb.CashFluxUser, onSetRole func(string, string)) ui.Node {
	if u.GetIsOwner() {
		return Span(Class(TextSize(TextXs), Fg(theme.Faint)), "owner")
	}
	id := u.GetId()
	onChange := ui.WrapHandler(func(e ui.Event) { onSetRole(id, e.GetValue()) })
	opts := []any{inputBase(), Props{OnChange: onChange}}
	for _, r := range []string{"member", "viewer"} {
		opts = append(opts, Tag("option", Props{Value: r, Selected: u.GetRole() == r}, r))
	}
	return Tag("select", opts...)
}

// userDataLabel is the headline figure for a user row: how much of their data this server is
// actually holding. An account that has never pushed shows an em dash rather than "0 B", which
// would read as "synced, and empty" — a different and much more alarming thing.
func userDataLabel(u *sitepb.CashFluxUser) string {
	if u.GetLastSyncedAt() == 0 && u.GetDatasetBytes() == 0 {
		return "—"
	}
	return formatBytes(u.GetDatasetBytes())
}

// userSyncedLabel is the supporting line: when this account last pushed, and across how many
// workspaces. Relative for anything recent (the question being asked is "is this alive?"),
// absolute once it is old enough that a relative age stops being meaningful.
func userSyncedLabel(u *sitepb.CashFluxUser) string {
	ts := u.GetLastSyncedAt()
	if ts == 0 {
		return "never synced"
	}
	when := time.Unix(ts, 0)
	d := time.Since(when)
	var ago string
	switch {
	case d < time.Minute:
		ago = "synced just now"
	case d < time.Hour:
		ago = "synced " + strconv.Itoa(int(d/time.Minute)) + "m ago"
	case d < 24*time.Hour:
		ago = "synced " + strconv.Itoa(int(d/time.Hour)) + "h ago"
	default:
		ago = "synced " + when.Format("Jan 2")
	}
	if n := u.GetWorkspaces(); n > 1 {
		ago += " · " + strconv.Itoa(int(n)) + " workspaces"
	}
	return ago
}

// inviteRow creates an account for someone else. Two steps by design: the account
// exists first (with a name and a role), then you hand them a code for it. The
// alternative — one button that creates and mints together — reads as "invite" but
// leaves a half-made account behind whenever the person never uses the code.
func inviteRow(u usersPanelState) ui.Node {
	onName := ui.WrapHandler(func(e ui.Event) { u.NewName.Set(e.GetValue()) })
	onRole := ui.WrapHandler(func(e ui.Event) { u.NewRole.Set(e.GetValue()) })
	role := u.NewRole.Get()
	if role == "" {
		role = "member"
	}
	roleOpts := []any{inputBase(), Props{OnChange: onRole}}
	for _, r := range []string{"member", "viewer"} {
		roleOpts = append(roleOpts, Tag("option", Props{Value: r, Selected: role == r}, r))
	}
	return Div(Class(Flex, ItemsCenter, Gap(Spacing2), css.Raw("flex-wrap", "wrap")),
		textInput(u.NewName.Get(), onName, "name for the person you're inviting", "text", false),
		Tag("select", roleOpts...),
		ghostButton("Add person", u.OnAddUser),
	)
}

// deleteConfirmRow is the are-you-sure state a user row swaps into. It names what is destroyed
// rather than asking a generic "are you sure?", and says plainly when the target is the owner's own
// account — deleting that erases your CashFlux data on this server and signs out every device you
// activated, which is a different act from removing someone you invited.
func deleteConfirmRow(u *sitepb.CashFluxUser, who string, deleting bool, onCancel func(), onConfirm func(string)) ui.Node {
	detail := "Erases their workspaces, transactions, and attachments, and signs out every device they activated."
	if u.GetIsOwner() {
		detail = "This is YOUR account — the one every activation code opens. Erases your CashFlux data on this server and signs out every device you activated."
	}
	confirmLabel := "Delete permanently"
	if deleting {
		confirmLabel = "Deleting…"
	}
	onYes := ui.WrapHandler(func() { onConfirm(u.GetId()) })
	onNo := ui.WrapHandler(func() { onCancel() })
	return Div(Class(Flex, FlexCol, Gap(Spacing2), Bg(theme.Bg), Border(theme.Red), Rounded(RadiusLg), PadX(Spacing3), PadY(Spacing3)),
		Span(Class(FontSemibold, TextSize(TextSm)), "Delete "+who+"?"),
		Span(Class(TextSize(TextSm), Fg(theme.Dim)), detail+" This can't be undone."),
		Div(Class(Flex, Gap(Spacing2)),
			Button(Class(Bg(theme.Red), Fg(Hex("#ffffff")), FontSemibold, Rounded(RadiusLg), PadX(Spacing4), PadY(Spacing3), TextSize(TextSm),
				css.Raw("border", "0"), css.Raw("cursor", "pointer"), css.Raw("font", "inherit"), DisabledIf(deleting)),
				Props{OnClick: onYes, Disabled: deleting}, confirmLabel),
			Button(Class(Fg(theme.Fg), Border(theme.Border), Rounded(RadiusLg), PadX(Spacing4), PadY(Spacing2), TextSize(TextSm),
				css.Raw("background", "transparent"), css.Raw("cursor", "pointer"), css.Raw("font", "inherit"), DisabledIf(deleting)),
				Props{OnClick: onNo, Disabled: deleting}, "Cancel"),
		),
	)
}

// byteUnits are the binary-size unit suffixes used by formatBytes, indexed by power of 1024 above
// bytes — KB..EB is enough to cover the full int64 range, so the index is always in bounds.
var byteUnits = []string{"KB", "MB", "GB", "TB", "PB", "EB"}

// formatBytes renders a byte count as a human-readable size (e.g. "42.3 MB"), not a raw number.
func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return strconv.FormatInt(n, 10) + " B"
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	v := float64(n) / float64(div)
	// A value a whisker under a unit boundary (e.g. 1048575 bytes, one byte short of 1 MiB) can
	// still round UP to the next unit at 1-decimal precision (1023.999... -> "1024.0"). Bump the
	// unit in that case so it reads "1.0 MB" rather than the wrong "1024.0 KB".
	if r := math.Round(v*10) / 10; r >= unit && exp < len(byteUnits)-1 {
		exp++
		div *= unit
		v = float64(n) / float64(div)
	}
	return strconv.FormatFloat(v, 'f', 1, 64) + " " + byteUnits[exp]
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
func settingsView(keySet bool, models []string, model, apiKey ui.State[string], onSave, onReload ui.Handler, usage *sitepb.TerminalUsage) ui.Node {
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
		terminalUsagePanel(usage),
	)
}

// terminalUsagePanel shows how often each terminal command has been run.
//
// It exists to answer one question — does anyone use the terminal, and what do they reach for —
// which decides where the next round of terminal work goes. The panel states its own privacy
// properties on screen, because a counter whose scope is only documented in a proto file is a
// counter nobody can audit at a glance.
func terminalUsagePanel(usage *sitepb.TerminalUsage) ui.Node {
	if usage == nil {
		return Div()
	}
	rows := []ui.Node{
		Span(Class(FontSemibold, TextSize(TextSm)), "Terminal usage"),
		Span(Class(Fg(theme.Dim), TextSize(TextSm)),
			"Command names only, aggregated by day — no visitor, address, session or argument text is recorded."),
	}
	if usage.GetTotal() == 0 {
		rows = append(rows, Span(Class(Fg(theme.Faint), TextSize(TextSm)), "Nothing recorded yet."))
		return Div(Class(Flex, FlexCol, Gap(Spacing2), PadY(Spacing4),
			css.BorderTop(css.Px(1), theme.Border)), rows)
	}
	rows = append(rows, Span(Class(Fg(theme.Accent), TextSize(TextSm)),
		strconv.FormatInt(usage.GetTotal(), 10)+" commands run, all time"))
	// Top 15: the tail of a command histogram is a long list of ones, and the question this panel
	// answers is about the head of it.
	for i, c := range usage.GetCommands() {
		if i >= 15 {
			break
		}
		rows = append(rows, Div(Class(Flex, Gap(Spacing3), TextSize(TextSm)),
			Span(Class(Fg(theme.Accent2), css.Raw("min-width", "120px")), c.GetName()),
			Span(Class(Fg(theme.Dim)), strconv.FormatInt(c.GetCount(), 10))))
	}
	return Div(Class(Flex, FlexCol, Gap(Spacing2), PadY(Spacing4),
		css.BorderTop(css.Px(1), theme.Border)), rows)
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
