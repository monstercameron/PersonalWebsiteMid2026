// Package notes holds the recruiter-facing briefing that the terminal serves from ~/notes.
//
// It lives outside the client because the client is a js/wasm-only package: while this text was
// declared there, it existed solely inside the wasm binary. That made it invisible to search
// engines, to link unfurlers, to a screen reader that never reaches the terminal, and to any
// visitor whose wasm boot failed — which is to say, invisible in precisely the cases where a
// static page would still have worked. It is the most important text on the site for the audience
// the site is for, so it is shared here and rendered twice: by the terminal, and as real
// server-rendered HTML on the page.
//
// Content rules: everything here is public and professional. No claim goes in that is not also
// true in the résumé (internal/resume) — two documents making different claims about the same
// career is worse than one.
package notes

// Doc is one file under ~/notes.
type Doc struct {
	// File is the name the terminal serves it as.
	File string
	// Title is the heading used when the document is rendered as HTML.
	Title string
	// Body is the text, formatted for a monospace column.
	Body string
}

// About is the ~/about.md splash.
const About = `Earl Cameron — AI-native systems engineer · Remote

Senior software engineer, seven years at UKG building enterprise HR software at scale — cloud
operations, then product engineering, now AI product development. Full-stack: Go and C#/.NET on
the backend; React, Angular, TypeScript on the front; Claude-driven agent workflows in day-to-
day delivery. Outside work I build complete web applications in Go end to end — the UI framework
and the transport included.

Curious recruiter? ` + "`ls notes`" + ` and read on.
`

// README is the index the terminal prints from ~/notes/README.md.
const README = `notes/ — a quick briefing for recruiters.

  cat notes/experience.md      roles, scope, impact
  cat notes/skills.md          languages, frameworks, infra, AI/ML
  cat notes/projects.md        open-source work (github.com/monstercameron)
  cat notes/working-style.md   how I operate

Everything here is public and professional. The rest of the terminal is a playground —
type ` + "`help`" + ` or ` + "`projects`" + `.
`

// Experience is the roles briefing.
//
// Mirrors internal/resume.Data() role for role and date for date — the package contract is that no
// claim lives here that is not also in the résumé, and a recruiter who reads both is exactly the
// recruiter who notices when the terminal and the PDF disagree about a title. There is no EDUCATION
// block, matching the résumé (Cam's decision, 2026-08-17); nothing here should reintroduce one.
const Experience = `EXPERIENCE

UKG — Senior Software Engineer, AI Product Development                Jan 2025 – present
  Two tiger teams building AI features for Bryte AI, UKG's AI product line, and now a
  dedicated team shaping a new recruiting product. Tool-calling agents over HCM data
  and retrieval-grounded answers drawn from it. Mentoring engineers and interns;
  training the support organizations that field the product first.

UKG — Software Engineer, UKG Pro Core HR                             Jul 2021 – Jan 2025
    · Angular → React modernization of a long-lived surface (React, Node, MS SQL Server)
    · HCM Pillar Dashboard — status across product areas in one view
    · SQL Server contention (CXPACKET / SOS_SCHEDULER_YIELD); throughput & priority procs

Ultimate Software (now UKG) — Cloud Engineer                         Oct 2019 – Jul 2021
    · Led emergency war rooms for critical infrastructure incidents
    · Owned escalated customer cases and defects through to resolution
    · Built the internal dashboard teams used to track dev & infrastructure Jira cases
    · Wrote process documentation for recurring infrastructure & escalation workflows

4Geeks Academy · HeyTutor — Instructor & Tutor (contract)                  2020 – present
    · Instructor/TA on a full-stack web development program (4Geeks, 2020 – 2021)
    · HTML/CSS and JavaScript fundamentals (HeyTutor); 5.0 average student rating
`

// Skills is the capability inventory.
const Skills = `SKILLS

  AI / ML     Tool-calling agents · retrieval / RAG · evals & guardrails · OpenAI & Anthropic
              APIs · LangChain · local runtimes (llama.cpp, Ollama, LM Studio)
  Languages   Go · C# · TypeScript / JavaScript · Python   (Rust, C — exploratory)
  Backend     .NET · Node · Flask · gRPC · REST
  Frontend    React · Angular · Blazor / Razor · Tailwind · WebAssembly (Go→WASM)
  Data        MS SQL Server · PostgreSQL · MySQL · MongoDB · SQLite
  Infra       Docker · IIS · WSL2 · GCP · DigitalOcean · Nginx
  Practice    Agentic development — Claude Code and Codex on real delivery work; agent
              harnesses; spec-first; adversarial review before merge; benchmarked (see
              GoWebComponents docs/BENCHMARKS.md)
`

// Projects is the open-source inventory.
//
// ⚠️ Hand-synced with internal/content.featured and client/programs.go termProjects — the drift
// risk is tracked in TODOS §0.
const Projects = `SELECTED OPEN-SOURCE WORK   github.com/monstercameron

  CashFlux            Local-first budgeting suite in Go/WASM — 40+ pages, rules engine, AI layer.
  ArticleFlux         Self-hosted feed reader, Go all the way down — FTS5 search, TTS, multi-tenant.
  AnimeFeedFlux       AI-written RSS feeds — recipes in, spec-compliant RSS/Atom/JSON Feed out.
  GoWebComponents     A React-style UI framework in Go→WASM. This site runs on it.
  WASIBrowser         A no-JavaScript browser that renders WebAssembly apps via a custom ABI.
  GoGRPCBridge        gRPC over WebSockets for the browser — no proxy. Carries this site's traffic.
  WebGL Path Tracer   Browser path tracing — materials, physics, benchmarks. Live demo, no install.
  SemanticScript      An agent-first programming language — auditable source designed for LLMs.
  SemanticAssembly    Agent-native RISC-V assembly layer.
  SemanticPortrait    Privacy-first journaling → a living self-portrait graph via a local model.
  Snapdragon LLMs     Running 12–27B models on the Snapdragon X2 NPU/GPU (QNN / ONNX pipelines).
`

// WorkingStyle is how Cam operates.
const WorkingStyle = `HOW I WORK

  · Establish the mechanism first, challenge weak assumptions, refine the architecture, then spec.
  · Direct and technically substantive — comfortable down at low-level architecture.
  · Honest about feasibility and toolchain limits.
  · Design sense: dark, polished, engineered-intentional UIs; clean information hierarchy;
    privacy-first; explainable, reversible operations.
`

// Docs is the briefing in reading order — the order the terminal's README recommends, and the
// order the page renders them in. README itself is excluded: it is a table of contents for a
// filesystem, which means nothing on a page that shows the documents themselves.
var Docs = []Doc{
	{File: "experience.md", Title: "Experience", Body: Experience},
	{File: "skills.md", Title: "Skills", Body: Skills},
	{File: "projects.md", Title: "Open-source work", Body: Projects},
	{File: "working-style.md", Title: "How I work", Body: WorkingStyle},
}
