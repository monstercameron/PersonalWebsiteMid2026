# Changelog

All notable changes to earlcameron.com. Format: [Keep a Changelog](https://keepachangelog.com);
Semantic Versioning once released.

## [Unreleased]
### Added
- Project scaffolding: planning docs (README, DESIGN, PROJECT_LAYOUT, DEPLOYMENT, TODOS, DEVLOG),
  agent guides (AGENTS.md, CLAUDE.md), and the `karpathy-guidelines` skill.
- Interactive design mockup (`design/mockup.html`).
- P1 ingress server foundation: `/healthz`, static file serving, SSR placeholder, graceful
  shutdown.
- `/bin` gitignored build-output directory.
- gRPC contract (`ContentService`, `ContactService`) with generated stubs via `buf` (pure Go).
- `ContentService` implementation: featured-project dataset + about copy, unit-tested.
- gRPC-over-WebSocket tunnel at `/socket` (GoGRPCBridge) with `ContentService` registered.
- `ContactService` + pure-Go SQLite store (modernc.org/sqlite): validated messages persisted.
- Standard site rendered server-side as GWC components with typed CSS (`css/u`), mobile-first
  responsive, served at `/` (SEO + no-WASM failsafe), rendered once at startup.
- WASM terminal pipeline: the GWC client builds to wasm, boots via a minimal glue script, and
  mounts over the SSR site with typed CSS (verified rendering in-browser).
- Terminal (wasm): macOS-style window — traffic-light chrome, boot log, neofetch identity splash,
  and prompt with cursor — styled with typed CSS from theme tokens; screenshot-verified against
  the mockup. Static for now; the interactive engine + gRPC programs are next.
### Changed
- Standard site reworked to match the mockup: hero-first order (orange-dash eyebrow, accent
  headline, sans-serif lede, glyphed social, orange launch CTA), then the terminal; boot log with
  OK badges + dotted leaders + right-aligned values; "live · interactive" title; ambient glows.
- Colors centralized into `internal/theme` design tokens (quick-ref: DESIGN.md §16).
- Minimal bootstrap `<style>` (reset + base bg + mono font-family) added to the SSR shell.
### Security
- WebSocket tunnel rejects cross-site origins (CSWSH guard) — same-origin + `ALLOWED_ORIGINS` only.
