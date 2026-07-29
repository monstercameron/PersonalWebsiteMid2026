// Package resume renders a clean, print-optimized HTML résumé (save-as-PDF friendly) and holds
// the structured résumé data the tailoring tool refines.
package resume

import (
	"fmt"
	"html"
	"strings"
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
		Title: "Senior Software Engineer — AI Systems, Developer Platforms & WebAssembly",
		// Availability, not a place — no city on the résumé by Cam's instruction (2026-07-29). The
		// résumé is downloaded and forwarded by strangers; it is the last document to carry one.
		Location: "Remote",
		Email:    "cam@earlcameron.com",
		GitHub:   "github.com/monstercameron",
		LinkedIn: "linkedin.com/in/earl-cameron",
		Summary: "Senior software engineer who ships production systems across the stack — Go and " +
			"C#/.NET on the backend, React/Angular/TypeScript on the front — by pairing deep systems " +
			"judgment with LLMs in the loop. Recent focus: agentic systems, AI infrastructure, and " +
			"on-device inference.",
		Jobs: []Job{{
			Role: "Software Engineer (P3 / Senior)", Org: "UKG", Dates: "2020 — present",
			Bullets: []string{
				"Modernized a large Angular surface to React (React, Tremor, Node, MS SQL Server).",
				"Built a Unified Search micro-frontend and a \"48 Hours\" ChatAssistant.",
				"Prototyped Bryte ChangeJob; delivered the HCM Pillar Dashboard.",
				"Wrote Go store/send benchmarks and client-per-core capacity experiments.",
				"Built provider registries and authentication/session systems.",
				"Investigated SQL performance (CXPACKET, SOS_SCHEDULER_YIELD); tuned throughput and priority stored procedures.",
			},
		}},
		Skills: []SkillGroup{
			{"Languages", "Go · C# · TypeScript/JavaScript · Python"},
			{"Frontend", "React · Angular · Blazor/Razor · Tailwind · WebAssembly (Go→WASM)"},
			{"Backend", ".NET · Node · Flask · gRPC · REST"},
			{"Data", "MS SQL Server · MySQL · MongoDB · SQLite"},
			{"Infra", "Docker · IIS · WSL2 · GCP · DigitalOcean · Nginx"},
			{"AI / ML", "OpenAI & Anthropic APIs · LangChain · FAISS · Whisper · on-device inference (Snapdragon NPU, QNN, ONNX Runtime GenAI, INT4)"},
		},
		// Ordered and quantified to match the site's featured four, so a reader who arrives from
		// either direction meets the same story. Every figure here is counted from the repository
		// (git ls-files / CHANGELOG), not estimated — see internal/site.velocityStrip. The projects
		// are the one part of this résumé that can carry hard numbers today; the employment bullets
		// above still need Cam's own scope and outcome figures (TODOS §14.H).
		Projects: []Project{
			{"CashFlux", "Local-first budgeting platform in Go/WASM — transactions, budgets, goals, reports, household ownership, rules engine and an on-device assistant. 221 internal packages behind 2,998 tests and 26 documented releases, six weeks from first commit."},
			{"ArticleFlux", "Self-hosted feed reader, Go all the way down — real gRPC in the browser, SQLite FTS5 search and text-to-speech, running 151 subscriptions and 3,621 items. Multi-tenant and deduplicated: a popular feed is polled once regardless of subscriber count."},
			{"GoWebComponents", "A React-style UI framework in Go→WASM — hooks, a fiber runtime, SSR/hydration and typed HTML. Benchmarked head-to-head with React and faster on overall geomean; my portfolio runs on it."},
			{"WASIBrowser", "A browser runtime where applications ship as WebAssembly components instead of JavaScript bundles — a custom component ABI over a Go host."},
			{"GoGRPCBridge", "gRPC over WebSockets for the browser — no Envoy, no proxy. Carries every request on my site."},
			{"On-device LLMs", "Running 12–27B models on the Snapdragon X2 NPU/GPU (QNN / ONNX Runtime GenAI, INT4)."},
		},
		Edu: []string{
			"B.S. Information Technology — Florida International University",
			"A.S. Information Technology — Miami Dade College",
		},
	}
}

// RenderHTML renders the résumé as a self-contained, print-optimized HTML document. `tailoredNote`
// is shown as a banner when the résumé was tailored to a job (empty for the canonical version).
func RenderHTML(r Resume, tailoredNote string) string {
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="en"><head><meta charset="utf-8">`)
	b.WriteString(`<meta name="viewport" content="width=device-width,initial-scale=1">`)
	fmt.Fprintf(&b, `<title>%s — Résumé</title>`, esc(r.Name))
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
	b.WriteString(`<ul class="projects">`)
	for _, p := range r.Projects {
		fmt.Fprintf(&b, `<li><b>%s</b> — %s</li>`, esc(p.Name), esc(p.Desc))
	}
	b.WriteString(`</ul>`)

	section(&b, "Education")
	b.WriteString(`<ul class="edu">`)
	for _, e := range r.Edu {
		fmt.Fprintf(&b, `<li>%s</li>`, esc(e))
	}
	b.WriteString(`</ul>`)

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
}
`
