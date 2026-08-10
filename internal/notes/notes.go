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

Senior software engineer (UKG, since 2020). I build production systems across the stack — Go,
C#/.NET on the backend; React, Angular, TypeScript on the front — and I ship fast by pairing deep
systems judgment with LLMs in the loop. Recently focused on agentic systems and AI infrastructure.

Curious recruiter? ` + "`ls notes`" + ` and read on.
`

// README is the index the terminal prints from ~/notes/README.md.
const README = `notes/ — a quick briefing for recruiters.

  cat notes/experience.md      roles, impact, education
  cat notes/skills.md          languages, frameworks, infra, AI/ML
  cat notes/projects.md        open-source work (github.com/monstercameron)
  cat notes/working-style.md   how I operate

Everything here is public and professional. The rest of the terminal is a playground —
type ` + "`help`" + ` or ` + "`projects`" + `.
`

// Experience is the roles-and-education briefing.
const Experience = `EXPERIENCE

UKG — Software Engineer (P3 / Senior)                                    2020 – present
  HCM domain, and more recently an agentic-systems / AI-infrastructure org (agents & AI tooling).
  Selected work:
    · Angular → React modernization (React, Tremor, Node, MS SQL Server)
    · Unified Search micro-frontend
    · "48 Hours" ChatAssistant
    · Bryte ChangeJob proof-of-concept
    · HCM Pillar Dashboard
    · Go store/send benchmarks; client-per-core capacity experiments
    · Provider registries; authentication & session systems
    · SQL performance work (CXPACKET / SOS_SCHEDULER_YIELD); throughput & priority stored procs

EDUCATION
  B.S. Information Technology — Florida International University (FIU)
  A.S. Information Technology — Miami Dade College
`

// Skills is the capability inventory.
const Skills = `SKILLS

  Languages   Go · C# · TypeScript / JavaScript · Python   (Rust, C — exploratory)
  Frontend    React · Angular · Blazor / Razor · Tailwind · WebAssembly (Go→WASM)
  Backend     .NET · Node · Flask · gRPC · REST
  Data        MS SQL Server · MySQL · MongoDB · SQLite
  Infra       Docker · IIS · WSL2 · GCP · DigitalOcean · Nginx
  AI / ML     OpenAI & Anthropic APIs · LangChain · FAISS · Whisper · Stable Diffusion ·
              local LLM runtimes · on-device inference (Snapdragon NPU, QNN, ONNX Runtime GenAI,
              INT4 quantization)
  Workflow    AI-native — Claude Code, Copilot, Cursor; heavy but measured LLM use
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
