// Package content serves the portfolio's projects and about copy.
//
// It implements sitepb.ContentServiceServer from a static in-memory dataset — the single source
// of truth read by BOTH the terminal (`projects`/`open`, `about`) and the server-rendered
// standard site, so the two front doors never drift. Content is English-only for now; the
// locale field is accepted and ignored until localized bundles land.
package content

import (
	"context"

	"github.com/monstercameron/earlcameron/proto/sitepb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service implements sitepb.ContentServiceServer over the static dataset built in New.
type Service struct {
	sitepb.UnimplementedContentServiceServer
	about    *sitepb.About
	projects []*sitepb.Project
}

// New returns a Service populated with Cam's about copy and featured projects.
func New() *Service {
	return &Service{about: aboutCopy(), projects: featured()}
}

// Projects returns the featured projects in display order (for the SSR site, which reads
// content directly rather than over gRPC).
func (s *Service) Projects() []*sitepb.Project { return s.projects }

// About returns the identity/thesis copy (for direct SSR use).
func (s *Service) About() *sitepb.About { return s.about }

// GetAbout returns the identity/thesis copy. The locale is currently ignored (English only).
func (s *Service) GetAbout(_ context.Context, _ *sitepb.LocaleRequest) (*sitepb.About, error) {
	return s.about, nil
}

// ListProjects returns the featured projects in display order.
func (s *Service) ListProjects(_ context.Context, _ *sitepb.LocaleRequest) (*sitepb.ProjectList, error) {
	return &sitepb.ProjectList{Projects: s.projects}, nil
}

// GetProject returns the project with the given id, or a NotFound error if none matches.
func (s *Service) GetProject(_ context.Context, req *sitepb.ProjectRequest) (*sitepb.Project, error) {
	for _, p := range s.projects {
		if p.Id == req.GetId() {
			return p, nil
		}
	}
	return nil, status.Errorf(codes.NotFound, "no project %q", req.GetId())
}

// aboutCopy returns the AI-first positioning copy shown in the hero and `about`.
func aboutCopy() *sitepb.About {
	return &sitepb.About{
		Headline: "AI-native systems engineer. I ship ambitious things, fast.",
		Body: "I'm an AI-first engineer. Not a 10x savant — something more useful: I know what to " +
			"build, and I use every tool I have, LLMs included, to build it well and quickly. Deep " +
			"systems work — a Go→WASM UI framework, on-device LLMs, a JS-free browser — paired with " +
			"AI leverage that turns weeks into days. With the model in the loop I operate like a " +
			"small team: same taste, a fraction of the time.",
	}
}

// featured returns the curated set of projects, in display order.
//
// Order IS the billing, and since 2026-07-29 it is also the *tier*: there is no featured flag on
// sitepb.Project, so site.splitTiers takes the first site.featuredCount (4) as the billed case
// studies — CashFlux (product engineering), ArticleFlux (AI orchestration), GoWebComponents
// (framework/platform), WASIBrowser (systems ambition) — and everything after them renders in the
// quieter Labs shelf. The SSR page, the terminal's `projects`, and /projects.md all read this one
// slice in this one order.
//
// Reordering therefore re-tiers the site. Moving a project above index 4 promotes it to a case
// study; pushing one below demotes it to Labs. GoGRPCBridge sits at index 4 — first in Labs — as
// the nearest thing to a fifth headliner.
func featured() []*sitepb.Project {
	const gh = "https://github.com/monstercameron/"
	return []*sitepb.Project{
		{Id: "cashflux", Name: "CashFlux", Status: "shipping", Glyph: "◈",
			Tags: []string{"Go", "WASM", "gRPC", "SQLite"}, Repo: gh + "CashFlux",
			Demo:  "https://monstercameron.github.io/CashFlux/",
			Blurb: "Local-first budgeting suite — 40+ pages, live charts, a rules engine, an AI layer. All Go/WASM, no JS framework.",
			Long:  "A full personal-finance app built entirely in Go compiled to WebAssembly on my own framework. Offline-first, enterprise-grade test suite, motion system, i18n. Runs live on this site — try `budget`."},
		{Id: "articleflux", Name: "ArticleFlux", Status: "shipping", Glyph: "◨",
			Tags: []string{"Go", "WASM", "gRPC", "FTS5"}, Repo: gh + "ArticleFlux",
			Demo:  "https://monstercameron.github.io/ArticleFlux/",
			Blurb: "A self-hosted feed reader that is Go all the way down — the server, the client, and the CSS. Real gRPC in the browser; the only JavaScript that ships is a boot shim.",
			Long:  "Google Reader's key map over a virtualised list at firehose scale — 151 subscriptions, 3,621 items — with SQLite FTS5 search, tags, notes, per-source hues and text-to-speech. Multi-tenant and deduplicated: a popular feed is polled once no matter how many people subscribe. The live demo is the shipping client with only the transport swapped."},
		{Id: "gwc", Name: "GoWebComponents", Status: "v5.0.1", Glyph: "⟠",
			Tags: []string{"Go", "WASM", "framework"}, Repo: gh + "GoWebComponents",
			Demo:  "https://monstercameron.github.io/GoWebComponents/",
			Blurb: "A React-style UI framework in Go→WASM. Hooks, a fiber runtime, SSR/hydration, typed HTML. Benchmarked head-to-head with React — faster on overall geomean. This site runs on it.",
			Long:  "The framework rendering the page you are reading — component model, reconciler, router, server functions, devtools. Benchmarked against React and winning on several axes. Zero npm."},
		{Id: "wasibrowser", Name: "WASIBrowser", Status: "prototype", Glyph: "◵",
			Tags: []string{"Go", "wasmtime", "C"}, Repo: gh + "WASIBrowser",
			Blurb: "A no-JavaScript browser. Renders WebAssembly apps directly via a custom ABI (Blitz + wasmtime).",
			Long:  "A browser where pages are WASM, not HTML+JS — a reactified-C component ABI, a Go host, and a real storefront demo over authenticated RPC."},
		{Id: "grpcbridge", Name: "GoGRPCBridge", Status: "v1.1.1", Glyph: "⇄",
			Tags: []string{"Go", "gRPC", "WebSocket"}, Repo: gh + "GoGRPCBridge",
			Demo:  "https://monstercameron.github.io/GoGRPCBridge/",
			Blurb: "gRPC over WebSockets for the browser — no Envoy, no proxy. This site talks to its backend through it.",
			Long:  "The tunnel that lets a Go/WASM browser client speak real gRPC to a Go server over one same-origin socket. It carries every request on this site."},
		{Id: "pathtracer", Name: "WebGL Path Tracer", Status: "demo", Glyph: "◐",
			Tags: []string{"WebGL", "GLSL", "rendering"}, Repo: gh + "pathtracer",
			Demo:  "https://monstercameron.github.io/pathtracer/",
			Blurb: "Path tracing live in a browser tab — a dozen material models, rigid-body physics, depth of field, and a repeatable benchmark harness. Open the demo and watch it converge.",
			Long:  "An extensively rebuilt fork of Evan Wallace's WebGL path tracer. Progressive refinement with temporal AA, glass/PBR/volumetric/spectral/toon materials, Rapier physics, SDF and CSG primitives, OBJ/STL/PLY/glTF/GLB import, an editor with scene tree and inspector, and a benchmark panel with baseline comparison and shareable score cards."},
		{Id: "semanticscript", Name: "SemanticScript", Status: "research", Glyph: "⧉",
			Tags: []string{"language", "runtime", "JIT"}, Repo: gh + "SemanticScript",
			Blurb: "An agent-first programming language — explicit, auditable source designed to be written and read by LLMs.",
			Long:  "A language + toolchain betting that agent-authored code wants different ergonomics: checkable structure, module-scoped names, a JIT that links the native runtime with no build step."},
		{Id: "semanticassembly", Name: "SemanticAssembly", Status: "research", Glyph: "⎇",
			Tags: []string{"RISC-V", "assembly", "agents"}, Repo: gh + "SemanticAssembly",
			Blurb: "Agent-native assembly — a RISC-V-first layer with a soft semantic overlay that advises, never gates.",
			Long:  "What assembly looks like when agents write it: RV32-first, additive semantics, and checks hardened against real agent mistakes at the metal level."},
		{Id: "semanticportrait", Name: "SemanticPortrait", Status: "prototype", Glyph: "◮",
			Tags: []string{"local-LLM", "graph", "journaling"}, Repo: gh + "SemanticPortrait",
			Blurb: "A journaling app that turns entries into a living self-portrait graph via a local model.",
			Long:  "A Windows journaling + self-portrait app: entries feed a local-model entry→graph loop, on a security-reviewed local-first store."},
	}
}
