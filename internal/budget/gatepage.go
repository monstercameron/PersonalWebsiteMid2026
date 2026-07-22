package budget

import (
	"github.com/monstercameron/GoWebComponents/v4/css"
	. "github.com/monstercameron/GoWebComponents/v4/css/u"
	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/ui"

	"github.com/monstercameron/earlcameron/internal/theme"
)

// monoFont is the machine voice — the terminal typeface shared with the rest of the site. font-family
// isn't a css/u helper, so use the typed escape hatch.
var monoFont = css.Raw("font-family", `ui-monospace,"SF Mono",SFMono-Regular,Menlo,"Cascadia Code","JetBrains Mono",monospace`)

// sansFont is the prose voice (proportional sans) for the explainer copy.
var sansFont = css.Raw("font-family", `-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif`)

// renderGatePage renders the CashFlux gate as a complete, self-contained HTML document: a locked
// terminal window (matching the site's portfolio-as-terminal identity) with a password field and a
// guest bypass. errMsg, when non-empty, is shown as an error above the form.
//
// The whole node tree is built inside this call because css/u emits its rules at node-construction
// time into the process-global sink that css.StyleBlock() then harvests — so the markup must be
// built here, immediately before RenderToString, with nothing resetting the sink in between.
func renderGatePage(errMsg string) string {
	markup, err := ui.RenderToString(gate(errMsg))
	if err != nil {
		// RenderToString only fails on a malformed tree, which this static page is not; fall back to a
		// plain door so the app is never unreachable behind a render bug.
		return `<!doctype html><meta charset="utf-8"><title>CashFlux</title>` +
			`<form method="post" action="/budget/__enter"><input type="password" name="password">` +
			`<button type="submit">Unlock</button></form>` +
			`<form method="post" action="/budget/__enter"><input type="hidden" name="mode" value="guest">` +
			`<button type="submit">Continue as guest</button></form>`
	}
	head := `<!doctype html><html lang="en"><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<title>CashFlux — locked</title>` +
		`<style>*{box-sizing:border-box}html,body{margin:0;height:100%}` +
		`body{color:#f3e9e6;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif;` +
		`background:radial-gradient(60vw 50vw at 12% -8%,rgba(190,123,230,.16),transparent 60%),` +
		`radial-gradient(55vw 55vw at 105% 115%,rgba(233,84,32,.14),transparent 55%),#17040f;` +
		`background-attachment:fixed;overflow-x:hidden}` +
		`@media (prefers-reduced-motion:no-preference){.cf-field{transition:border-color .15s ease,box-shadow .15s ease}}</style>` +
		css.StyleBlock() + `</head><body>`
	return head + markup + `</body></html>`
}

// gate builds the locked-window card.
func gate(errMsg string) ui.Node {
	return Div(Class(css.Raw("min-height", "100dvh"), Flex, FlexCol, ItemsCenter, JustifyCenter, PadX(Spacing5), PadY(Spacing6)),
		Div(Class(WFull, MaxWidth(Px(430)), css.Raw("max-width", "min(430px,100%)"), Flex, FlexCol, Gap(Spacing4)),
			card(errMsg),
			back(),
		),
	)
}

// card is the terminal-window-chrome panel: a title bar with traffic-light dots, then the prompt,
// copy, and the two entry paths.
func card(errMsg string) ui.Node {
	return Div(Class(
		Bg(theme.BgRaised),
		css.Raw("border", "1px solid "+string(theme.TermBorder)),
		css.Raw("border-radius", "12px"),
		css.Raw("overflow", "hidden"),
		css.Raw("box-shadow", "0 24px 60px -20px rgba(0,0,0,.6)"),
	),
		titleBar(),
		Div(Class(Flex, FlexCol, Gap(Spacing4), Pad(Spacing6)),
			prompt(),
			explainer(),
			IfElse(errMsg != "", errorNote(errMsg), Span()),
			passwordForm(),
			divider(),
			guestForm(),
		),
	)
}

// titleBar is the mac-style window chrome: three traffic-light dots and a centered label.
func titleBar() ui.Node {
	dot := func(hex string) ui.Node {
		return Span(Class(css.Raw("width", "11px"), css.Raw("height", "11px"), css.Raw("border-radius", "999px"),
			css.Raw("display", "inline-block"), css.Raw("background", hex)))
	}
	return Div(Class(Flex, ItemsCenter, Gap(Spacing3), PadX(Spacing4), PadY(Spacing3),
		Bg(theme.TermBar), css.Raw("border-bottom", "1px solid "+string(theme.TermBorder))),
		Div(Class(Flex, ItemsCenter, Gap(Spacing2)), dot("#ff5f57"), dot("#febc2e"), dot("#28c840")),
		Span(Class(monoFont, TextSize(TextXs), Fg(theme.Faint), css.Raw("margin", "0 auto"),
			css.Raw("letter-spacing", "0.04em")), "cashflux — locked"),
	)
}

// prompt is the signature element: a terminal command line tying the door to the site's identity.
func prompt() ui.Node {
	return Div(Class(monoFont, TextSize(TextSm), Flex, ItemsCenter, Gap(Spacing2)),
		Span(Class(Fg(theme.Accent)), "$"),
		Span(Class(Fg(theme.Fg)), "unlock cashflux"),
		Span(Class(Fg(theme.Accent), css.Raw("width", "8px"), css.Raw("height", "1.05em"),
			css.Raw("background", string(theme.Accent)), css.Raw("display", "inline-block"),
			css.Raw("opacity", "0.9"))),
	)
}

// explainer states, in plain language, what the two paths mean.
func explainer() ui.Node {
	return P(Class(sansFont, TextSize(TextSm), Fg(theme.Dim), LineHeight(Num(1.5))),
		Span("This is my budgeting app. Enter the password for "),
		Span(Class(Fg(theme.Fg)), "your synced budget"),
		Span(", or continue as a "),
		Span(Class(Fg(theme.Fg)), "guest"),
		Span(" to try it locally on this device (nothing syncs, my data never loads)."),
	)
}

// errorNote shows a wrong-password message.
func errorNote(msg string) ui.Node {
	return Div(Class(sansFont, TextSize(TextSm), Fg(theme.Red),
		css.Raw("background", "rgba(239,83,80,.10)"), css.Raw("border", "1px solid rgba(239,83,80,.35)"),
		css.Raw("border-radius", "8px"), PadX(Spacing3), PadY(Spacing2)),
		msg,
	)
}

// fieldClass is the shared input styling: terminal-flavored, with a visible accent focus ring.
func fieldClass() any {
	return Class("cf-field", monoFont, WFull, TextSize(TextSm), Fg(theme.Fg),
		Bg(theme.Bg), css.Raw("border", "1px solid "+string(theme.Border)),
		css.Raw("border-radius", "8px"), PadX(Spacing3), PadY(Spacing3),
		css.Raw("outline", "none"),
		Focus(css.Raw("border-color", string(theme.Accent))),
		Focus(css.Raw("box-shadow", "0 0 0 3px rgba(233,84,32,.25)")),
	)
}

// passwordForm posts the password to the gate for a full session.
func passwordForm() ui.Node {
	return Form(Props{Method: "post", Action: "/budget/__enter"},
		Class(Flex, FlexCol, Gap(Spacing3)),
		Label(Class(sansFont, TextSize(TextXs), Fg(theme.Faint), css.Raw("letter-spacing", "0.06em")), "PASSWORD"),
		Input(fieldClass(), Props{Type: "password", Name: "password", AutoFocus: true}),
		Button(Props{Type: "submit"}, Class(
			monoFont, TextSize(TextSm), FontSemibold, css.Raw("cursor", "pointer"),
			Bg(theme.Accent), Fg(Hex("#1a0a04")), css.Raw("border", "none"),
			css.Raw("border-radius", "8px"), PadX(Spacing4), PadY(Spacing3),
			Hover(css.Raw("filter", "brightness(1.08)")),
		), "Unlock →"),
	)
}

// divider is a labelled "or" separator between the two entry paths.
func divider() ui.Node {
	rule := Span(Class(css.Raw("flex", "1"), css.Raw("height", "1px"), css.Raw("background", string(theme.Border))))
	return Div(Class(Flex, ItemsCenter, Gap(Spacing3)),
		rule,
		Span(Class(sansFont, TextSize(TextXs), Fg(theme.Faint)), "or"),
		Span(Class(css.Raw("flex", "1"), css.Raw("height", "1px"), css.Raw("background", string(theme.Border)))),
	)
}

// guestForm posts the guest bypass for a local-only session.
func guestForm() ui.Node {
	return Form(Props{Method: "post", Action: "/budget/__enter"},
		Input(Props{Type: "hidden", Name: "mode", Value: "guest"}),
		Button(Props{Type: "submit"}, Class(
			sansFont, TextSize(TextSm), WFull, css.Raw("cursor", "pointer"),
			css.Raw("background", "transparent"), Fg(theme.Accent2),
			css.Raw("border", "1px solid "+string(theme.Border)),
			css.Raw("border-radius", "8px"), PadX(Spacing4), PadY(Spacing3),
			Hover(Fg(theme.Fg)), Hover(css.Raw("border-color", string(theme.Accent2))),
		), "Continue as guest →"),
	)
}

// back is a quiet link home, so the door isn't a dead end.
func back() ui.Node {
	return A(Props{Href: "/"}, Class(sansFont, TextSize(TextXs), Fg(theme.Faint),
		css.Raw("text-align", "center"), Hover(Fg(theme.Dim))), "← back to earlcameron.com")
}
