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
  the mockup.
- Terminal is now **interactive**: a controlled input (GWC `UseState`/`UseEvent`) runs the
  portfolio programs (`help`/`about`/`projects`/`open`/`neofetch`/`links`/`contact`) and the
  window **expands/shrinks** to a fullscreen modal (green/red lights).
- "Elsewhere" section on the standard site + terminal `links`/`resume`: résumé download,
  LinkedIn (`earl-cameron`), YouTube (`@EarlCameron007`), GitHub — ported from the current site.
- Recruiter notes in the terminal's `notes/` folder — `cat notes/about.md`, `experience.md`,
  `skills.md`, `projects.md`, `working-style.md`: professional, tech-fit content for curious
  recruiters exploring the shell (VFS cache bumped to v2).
- Terminal **faux Unix shell** over a **localStorage-cached virtual filesystem**: ~30 commands
  (pwd/ls/cd/mkdir/touch/cp/mv/rm/cat/head/tail/echo/grep/find/sort/uniq/cut/sed/awk/wc/du/df/
  history/less/nano/curl/ssh/tar/man) with pipes `|`, chaining `&&`, `>` redirect, a cwd-aware
  prompt, and **Tab completion** (commands + paths). Verified end-to-end via chromedp (9 checks).
### Changed
- Standard site reworked to match the mockup: hero-first order (orange-dash eyebrow, accent
  headline, sans-serif lede, glyphed social, orange launch CTA), then the terminal; boot log with
  OK badges + dotted leaders + right-aligned values; "live · interactive" title; ambient glows.
- Colors centralized into `internal/theme` design tokens (quick-ref: DESIGN.md §16).
- Minimal bootstrap `<style>` (reset + base bg + mono font-family) added to the SSR shell.
### Fixed
- Terminal: auto-scrolls to the newest line; the input keeps focus after Enter; the expanded
  modal locks the page scroll and has its own; themed scrollbar. Verified with a chromedp
  browser-interaction test (real keystrokes + clicks), not just a render check.
- Terminal modal no longer overflows the right edge (removed `width:100%` that fought the fixed
  insets); it now sizes purely from top/left/right/bottom offsets.
- Terminal: clicking anywhere in the body focuses the input (no need to hit the prompt line).
- Terminal: text is selectable again — click-to-focus no longer clears an active selection, and
  the body is explicitly `user-select: text`. (Verified via a chromedp mouse-drag selection.)
### Security
- WebSocket tunnel rejects cross-site origins (CSWSH guard) — same-origin + `ALLOWED_ORIGINS` only.
