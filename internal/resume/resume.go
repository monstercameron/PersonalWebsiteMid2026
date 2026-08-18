// Package resume renders a clean, print-optimized HTML résumé (save-as-PDF friendly) and holds
// the structured résumé data the tailoring tool refines.
package resume

import (
	"fmt"
	"html"
	"strings"

	"github.com/monstercameron/earlcameron/internal/site"
)

// Resume is the structured résumé content.
type Resume struct {
	Name     string
	Title    string
	Location string
	Email    string
	GitHub   string
	LinkedIn string
	Summary  string
	Jobs     []Job
	Skills   []SkillGroup
	Projects []Project
	Edu      []string
}

// Job is one work entry.
type Job struct {
	Role, Org, Dates string
	Bullets          []string
}

// SkillGroup is a labeled row of skills.
type SkillGroup struct{ Label, Items string }

// Project is one selected project.
type Project struct{ Name, Desc string }

// Data returns Cam's résumé (professional / tech-fit content only).
func Data() Resume {
	return Resume{
		Name: "Earl Cameron",
		// Tracks the site's positioning (internal/site hero + <title>). The résumé is forwarded
		// separately from the site, so a narrower title here quietly undoes the repositioning for
		// exactly the reader who only ever sees the PDF.
		// Two specializations, not three. WebAssembly used to sit here as a third; a title naming
		// three specializations reads as someone mid-rebrand who has not decided what to lead with,
		// and costs the benefit of the doubt before the first bullet. WebAssembly is still all over
		// the skills and projects, where a reader who cares about it will find it.
		//
		// "Developer Platforms" gave way to "Full-Stack" (2026-08-18): the roles Cam is actually
		// approached for are product-engineering roles wanting someone who ships across the stack,
		// and "developer platforms" reads as internal-tooling work to a product company. The platform
		// story is not lost — it is the whole of the projects section, where it is shown, not claimed.
		Title: "Senior Software Engineer — Full-Stack & AI Systems",
		// Availability, not a place — no city on the résumé by Cam's instruction (2026-07-29). The
		// résumé is downloaded and forwarded by strangers; it is the last document to carry one.
		Location: "Remote",
		Email:    "cam@earlcameron.com",
		GitHub:   "github.com/monstercameron",
		LinkedIn: "linkedin.com/in/earl-cameron",
		// Names the domain (enterprise HR / HCM), not only the employer. UKG is a household name in HR
		// software and nowhere else; a reader hiring for anything benefits-, payroll- or workforce-
		// adjacent will not make that connection from four letters, and the domain is the most
		// transferable thing here after the code. "Full-stack" is stated outright for the same reason:
		// the stack list implies it, but product companies filter on the word.
		// Three sentences, not one 45-word chain of em-dashes: tenure and trajectory, then stack,
		// then the differentiator. The first sentence has to survive being read alone, because on a
		// fast screen it often is.
		Summary: "Senior software engineer, seven years at UKG building enterprise HR software at " +
			"scale, promoted twice — cloud operations, then product engineering, now AI product " +
			"development. Full-stack: Go and C#/.NET on the backend, React/Angular/TypeScript on the " +
			"front, with Claude-driven agent workflows in day-to-day delivery. Outside work I build " +
			"complete web applications in Go — UI framework, transport and styling in one language.",
		// One employer, three titles, oldest last — the shape a recruiter scans for promotion in
		// under five seconds. Split from the single flat "UKG 2020 — present" entry that hid two
		// promotions and seven years of tenure inside one date range. Cloud Engineer is dated to
		// Jul 2021 rather than the Aug 2021 on LinkedIn: the month of overlap is an artifact of the
		// Ultimate Software → UKG transition, and overlapping dates on one résumé read as an error.
		//
		// Content rule from Cam (2026-08-17): nothing non-production from UKG. The innovation-award
		// chatbot and the Bryte ChangeJob proof-of-concept were prototypes and are deliberately
		// absent — prototypes invite "what shipped?" and the answer has to be somewhere else.
		// The org line carries the internal team, not just the employer. Seven years at one company
		// otherwise looks static on paper; "Core HR" moving to "AI Product Development" shows the
		// same movement inside UKG that a job-hopper's résumé shows across logos, and it is the
		// answer to the unspoken screener question of what someone did with seven years in one place.
		Jobs: []Job{{
			Role: "Senior Software Engineer", Org: "UKG — AI Product Development", Dates: "Jan 2025 — Present",
			Bullets: []string{
				// Trajectory first (Cam's correction, 2026-08-18): two tiger teams on Bryte AI, then the
				// dedicated product team. Bryte AI is UKG's publicly announced AI brand and is safe to
				// name — but it needs the three-word gloss, because a brand a reader outside UKG has
				// never heard of is not a credential, it is a word. The recruiting product is unreleased
				// and stays generic: an unshipped internal codename on a document strangers forward is a
				// disclosure, not an achievement. "a dedicated AI product development team" was cut back
				// to "a dedicated team" because the org line one line above already says exactly that,
				// and the thinnest section on the page cannot afford to say a thing twice.
				"Moved through two tiger teams building AI features for Bryte AI, UKG's AI product line, and now build on a dedicated team shaping a new recruiting product.",
				// Named techniques, confirmed by Cam (2026-08-17): tool-calling agents, retrieval, and
				// evals/guardrails/tracing. The previous wording — "LLM-backed services and the internal
				// tooling engineers work with them through" — was every 2026 résumé's sentence and gave
				// an interviewer nothing to ask about, which put the most important question of the
				// screen on Cam to answer cold. Precise nouns are the substitute for the metrics he
				// cannot release.
				"Build agentic systems and AI infrastructure for UKG's HCM products — tool-calling agents over HCM data and retrieval-grounded answers drawn from it.",
				// An eval/guardrail/tracing bullet stood here and was removed at Cam's instruction
				// (2026-08-18). The AI / ML skills row still lists "evals & guardrails", which is the only
				// remaining claim to that work on the page — if that goes too, the résumé stops claiming
				// it entirely, which is a positioning decision rather than a wording one.
				"Mentor engineers and interns, and run training sessions for the support organizations that field the product first.",
			},
		}, {
			Role: "Software Engineer", Org: "UKG — UKG Pro, Core HR", Dates: "Jul 2021 — Jan 2025",
			Bullets: []string{
				"Modernized a large Angular surface to React, moving a long-lived product area onto the current stack (React, Tremor, Node, MS SQL Server).",
				// "Built a Unified Search micro-frontend spanning HCM product areas" was here, inherited
				// from the previous résumé. Cam did not do that work (2026-08-18) — removed. Nothing goes
				// back in that he has not confirmed, however plausible it sounds against the rest.
				"Delivered the HCM Pillar Dashboard, consolidating status across product areas into one view.",
				// A "provider registries and the authentication and session layer behind them" bullet
				// stood here until Cam struck it (2026-08-18) as meaningless to a reader — it named an
				// internal shape with no product, no user and no outcome attached, which is what an
				// inherited bullet degrades into once the person who wrote it has moved on. It was the
				// third line removed from this résumé that nobody in this session had ever confirmed.
				// The Go store/send benchmarks and per-core capacity analysis bullet stood here and was
				// removed at Cam's instruction (2026-08-18). Consequence worth knowing rather than
				// quietly absorbing: the employment history now contains no Go evidence at all, so every
				// Go claim on the page rests on the projects section. That is survivable — the projects
				// are public and checkable, which employment bullets never are — but a screener filtering
				// Experience for "Go" will not find it there.
				"Diagnosed and fixed SQL Server contention (CXPACKET, SOS_SCHEDULER_YIELD) and tuned throughput and priority stored procedures.",
			},
		}, {
			Role: "Cloud Engineer", Org: "Ultimate Software (now UKG)", Dates: "Oct 2019 — Jul 2021",
			Bullets: []string{
				"Led emergency war rooms for critical infrastructure incidents, coordinating diagnosis and resolution across engineering, operations and support.",
				"Owned escalated customer cases and defects end to end, from reproduction through fix and direct communication with the customer.",
				"Designed and built the internal dashboard teams used to track development and infrastructure Jira cases.",
				"Wrote the process documentation teams follow for recurring infrastructure and escalation workflows.",
			},
		}, {
			// Two contracts, one entry. Separately they are two thin blocks competing with UKG for a
			// recruiter's attention; together they are six years of teaching other people to write
			// code, which is the strongest available counterweight to no degree on the page.
			Role: "Instructor & Tutor (contract)", Org: "4Geeks Academy · HeyTutor", Dates: "2020 — Present",
			Bullets: []string{
				"Instructed and mentored students through a full-stack web development program at 4Geeks Academy (2020 — 2021), as both instructor and teaching assistant.",
				"Tutor HTML/CSS and JavaScript fundamentals through HeyTutor (2020 — present); 5.0 average student rating.",
			},
		}},
		// AI / ML leads, because the title does. With Languages/Frontend/Backend/Data/Infra above it,
		// the section quietly argued that AI was the fifth-most important thing on the page while the
		// header claimed it was the first — the résumé's own ordering contradicting its own headline.
		Skills: []SkillGroup{
			// No Snapdragon NPU / QNN / ONNX Runtime GenAI / INT4 / Whisper here, by Cam's instruction
			// (2026-08-18): the on-device inference work is an esoteric hardware experiment, and listing
			// it beside the production agent work invites a screener to read the whole row as hobby
			// tinkering. Precision costs nothing here; breadth does.
			{"AI / ML", "Tool-calling agents · retrieval / RAG · evals & guardrails · OpenAI & Anthropic APIs · LangChain · local runtimes (llama.cpp, Ollama, LM Studio)"},
			// Named as a practice, not a tool list. "Claude Code, Cursor, Copilot" reads as a
			// subscription; how he works with agents is the part that is actually a skill. The
			// GoWebComponents pointer is deliberate: "benchmark-driven" is a label until the reader
			// knows a published benchmark backs it, and it sits three paragraphs below.
			{"Practice", "Agentic development — Claude Code and Codex on real delivery work · agent harnesses · spec-first · adversarial review · benchmarked (see GoWebComponents)"},
			{"Languages", "Go · C# · TypeScript/JavaScript · Python"},
			{"Backend", ".NET · Node · Flask · gRPC · REST"},
			{"Frontend", "React · Angular · Blazor/Razor · Tailwind · WebAssembly (Go→WASM)"},
			{"Data", "MS SQL Server · PostgreSQL · MySQL · MongoDB · SQLite"},
			{"Infra", "Docker · IIS · WSL2 · GCP · DigitalOcean · Nginx"},
		},
		// Three projects, by Cam's instruction (2026-08-18): CashFlux, ArticleFlux, GoWebComponents.
		// WASIBrowser and WhisperToMe are gone and GoGRPCBridge lost its own entry — six entries was
		// a list, and a list of six dilutes the three that matter. GoGRPCBridge survives inside the
		// GoWebComponents description rather than disappearing: it is the transport half of the
		// "written entirely in Go" claim, and that claim does not land with the framework alone.
		// Every figure here is counted from the repository
		// (git ls-files / CHANGELOG), not estimated — see internal/site.velocityStrip. The projects
		// are the one part of this résumé that carries hard numbers: the employment bullets are
		// deliberately figure-free (Cam has no releasable UKG metrics, and inventing them is the one
		// unrecoverable résumé mistake).
		Projects: []Project{
			// The velocity figures now name the mechanism that produced them. Stated bare, "221 packages
			// and 2,998 tests in six weeks" is the most inflated-sounding line on the page and invites
			// the reader to invent an unflattering explanation; naming the agent workflow is both the
			// honest answer and the proof of the practice the summary claims.
			{"CashFlux", "Local-first budgeting platform in Go/WASM — transactions, budgets, goals, reports, household ownership, rules engine and an on-device assistant. 221 internal packages behind 2,998 tests and 26 documented releases, six weeks from first commit, built with the agent-harness workflow above."},
			// "SQLite FTS5 rather than a search service" is the one build-vs-buy call on the page. Every
			// other line is "I built X"; a senior screen also looks for something the candidate chose
			// not to build.
			{"ArticleFlux", "Self-hosted feed reader, Go all the way down — real gRPC in the browser, text-to-speech, and search on SQLite FTS5 rather than a separate search service. Runs 151 subscriptions and 3,621 items; multi-tenant and deduplicated, so a popular feed is polled once regardless of subscriber count."},
			// The framing used to be "no JavaScript, no npm" — twice, as if avoiding a technology were
			// the achievement. That is the tell of NIH hobby work to the reader who matters. The reason
			// Cam actually did it (one language, one toolchain) is a tradeoff, and stating the tradeoff
			// is what separates an infrastructure decision from an ideology.
			{"GoWebComponents", "A React-style UI framework in Go→WASM — hooks, a fiber runtime, SSR/hydration and typed HTML. Its companion GoGRPCBridge carries gRPC over WebSockets straight to the browser, so one language, one type system and one debugger cover the whole stack from database to pixel. Benchmarked head-to-head against React — faster on overall geomean, methodology and results in docs/BENCHMARKS.md. My portfolio runs on both."},
		},
		// No Education section, by Cam's decision (2026-08-17): the bachelor's is unfinished, and a
		// résumé is the wrong place to either claim it or explain it. Seven years, a senior title and
		// six years of teaching carry the credential weight instead. The section is omitted rather
		// than emptied — see RenderHTML, which now skips it when this slice is empty.
		Edu: nil,
	}
}

// RenderHTML renders the résumé as a self-contained, print-optimized HTML document. `tailoredNote`
// is shown as a banner when the résumé was tailored to a job (empty for the canonical version).
func RenderHTML(r Resume, tailoredNote string) string {
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="en"><head><meta charset="utf-8">`)
	b.WriteString(`<meta name="viewport" content="width=device-width,initial-scale=1">`)
	fmt.Fprintf(&b, `<title>%s — Résumé</title>`, esc(r.Name))
	// Same icon block as the rest of the site. A résumé opens in its own tab, often the one a
	// recruiter leaves parked for days, so it is the last page that should show a generic globe.
	b.WriteString(site.FaviconLinks())
	b.WriteString(`<style>` + resumeCSS + `</style></head><body>`)
	if tailoredNote != "" {
		fmt.Fprintf(&b, `<div class="banner no-print">%s</div>`, esc(tailoredNote))
	}
	b.WriteString(`<div class="page">`)

	// header
	fmt.Fprintf(&b, `<header><h1>%s</h1><div class="role">%s</div><div class="contact">%s · <a href="mailto:%s">%s</a> · %s · %s</div></header>`,
		esc(r.Name), esc(r.Title), esc(r.Location), esc(r.Email), esc(r.Email), esc(r.GitHub), esc(r.LinkedIn))

	section(&b, "Summary")
	fmt.Fprintf(&b, `<p class="summary">%s</p>`, esc(r.Summary))

	section(&b, "Experience")
	for _, j := range r.Jobs {
		fmt.Fprintf(&b, `<div class="job"><div class="jobhead"><b>%s</b><span>%s</span></div><div class="org">%s</div><ul>`,
			esc(j.Role), esc(j.Dates), esc(j.Org))
		for _, bl := range j.Bullets {
			fmt.Fprintf(&b, `<li>%s</li>`, esc(bl))
		}
		b.WriteString(`</ul></div>`)
	}

	section(&b, "Skills")
	b.WriteString(`<table class="skills">`)
	for _, g := range r.Skills {
		fmt.Fprintf(&b, `<tr><td class="k">%s</td><td>%s</td></tr>`, esc(g.Label), esc(g.Items))
	}
	b.WriteString(`</table>`)

	section(&b, "Selected Projects")
	// One line telling the reader where all of this lives. Every project below is public, but the
	// only address on the page was in the contact line four sections up — so a reader who wanted to
	// verify a claim had to go hunting for which repo was which, which most will not do.
	fmt.Fprintf(&b, `<p class="projnote">All repositories at <a href="https://%s">%s</a>.</p>`, esc(r.GitHub), esc(r.GitHub))
	b.WriteString(`<ul class="projects">`)
	for _, p := range r.Projects {
		fmt.Fprintf(&b, `<li><b>%s</b> — %s</li>`, esc(p.Name), esc(p.Desc))
	}
	b.WriteString(`</ul>`)

	// Rendered only when there is education to render. An empty section is worse than no section:
	// the heading is exactly what a reader looking for a degree stops on, and finding it blank
	// answers the question louder than omitting it does.
	if len(r.Edu) > 0 {
		section(&b, "Education")
		b.WriteString(`<ul class="edu">`)
		for _, e := range r.Edu {
			fmt.Fprintf(&b, `<li>%s</li>`, esc(e))
		}
		b.WriteString(`</ul>`)
	}

	b.WriteString(`</div>`) // .page
	b.WriteString(`<div class="actions no-print"><button onclick="window.print()">Save as PDF</button> <a href="/">← back to the site</a></div>`)
	b.WriteString(`</body></html>`)
	return b.String()
}

func section(b *strings.Builder, title string) { fmt.Fprintf(b, `<h2>%s</h2>`, esc(title)) }
func esc(s string) string                      { return html.EscapeString(s) }

// resumeCSS is a clean, professional, print-first stylesheet (light document, subtle accent).
const resumeCSS = `
*{box-sizing:border-box}
body{margin:0;background:#e9e6ea;color:#1b1420;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif;line-height:1.45}
.page{max-width:820px;margin:28px auto;background:#fff;padding:48px 56px;box-shadow:0 10px 40px -12px rgba(0,0,0,.3)}
header{border-bottom:2px solid #e95420;padding-bottom:14px;margin-bottom:8px}
h1{font-size:30px;margin:0;letter-spacing:-.01em}
.role{color:#e95420;font-weight:600;margin-top:2px}
.contact{color:#555;font-size:13px;margin-top:8px}
.contact a{color:#555}
h2{font-size:12px;letter-spacing:.14em;text-transform:uppercase;color:#e95420;margin:22px 0 8px;border-bottom:1px solid #eee;padding-bottom:4px}
.summary{margin:0}
.job{margin-bottom:12px}
.jobhead{display:flex;justify-content:space-between;align-items:baseline}
.jobhead span{color:#666;font-size:13px}
.org{color:#444;font-size:14px;margin-bottom:4px}
ul{margin:6px 0 0;padding-left:18px}
li{margin:3px 0}
table.skills{border-collapse:collapse;width:100%;font-size:14px}
table.skills td{padding:3px 0;vertical-align:top}
table.skills td.k{color:#e95420;font-weight:600;width:110px;white-space:nowrap}
.projects li b,.edu li{color:#1b1420}
.projnote{margin:0 0 6px;font-size:13px;color:#555}
.projnote a{color:#c1440f}
.banner{max-width:820px;margin:16px auto 0;background:#fff3e6;border:1px solid #e95420;color:#8a3b12;padding:10px 16px;border-radius:8px;font-size:14px}
.actions{max-width:820px;margin:16px auto 40px;display:flex;gap:16px;align-items:center}
.actions button{background:#e95420;color:#fff;border:0;padding:10px 18px;border-radius:8px;font:inherit;font-weight:600;cursor:pointer}
.actions a{color:#e95420}
@media print{
  body{background:#fff}
  .page{box-shadow:none;margin:0;max-width:none;padding:0.5in 0.6in}
  .no-print{display:none!important}
  h2{color:#c1440f}
  header{border-color:#c1440f}
  /* Page-break discipline. The résumé now runs to two pages, and the default break behaviour
     strands a section heading alone at the foot of page one and splits a bullet across the fold —
     both of which read as carelessness on the one document whose job is to look careful. Headings
     and role lines stay with what follows them; individual bullets stay whole. Whole job blocks are
     deliberately NOT kept together: a six-bullet block that cannot fit the remaining space would
     leave a third of a page blank, which is worse than a clean break between bullets. */
  h2{break-after:avoid;page-break-after:avoid}
  .jobhead,.org{break-after:avoid;page-break-after:avoid}
  li,tr{break-inside:avoid;page-break-inside:avoid}
}
`
