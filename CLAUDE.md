# CLAUDE.md

**`AGENTS.md` is canonical — read it first.** Everything there applies. This file adds
Claude-specific notes.

> **PHASE: BUILD (P1).** The build has started; the P1 foundation is in. Work `TODOS.md` phase
> by phase; the planning docs are the source of truth for scope and design.

## Claude specifics
- **Coding behavior**: load the **`karpathy-guidelines`** skill
  (`.claude/skills/karpathy-guidelines/`) before writing or refactoring code — think before
  coding, simplicity first, surgical changes, goal-driven execution. The big feature list is
  *product scope*; implement each piece simply and surgically, never gold-plated.
- **UI/UX work — ALWAYS, no exceptions**: load the **`frontend-design` skill** *before*
  touching ANY UI, every time — new components, small tweaks, CSS/layout, motion, or
  user-facing copy, not just big screens. Then follow the locked design language in
  `documents/DESIGN.md` (macOS chrome, Ubuntu-souled "Aubergine" palette, two voices,
  motion-with-restraint, reduced-motion + a11y floor).
- **Adversarial review is mandatory** for substantive changes: after doing the work, **spawn
  review subagent(s)** tasked with aggressively breaking your output — correctness, security,
  performance, and needless complexity — default to skepticism, fix every real finding, and
  re-review until it survives.
- **Subagents run sequentially, one at a time, on Sonnet — never in parallel** (Cam's standing
  preference).

## Non-negotiables (restated — easy to skip, don't)
- **Strong shared types + DTOs across all layers.** The generated `proto` package is the shared
  DTO source of truth for the Go server **and** the Go/WASM client — no `map[string]any` or
  stringly-typed data passed between layers; explicit DTOs at every boundary.
- **Document every function/method — exported *and* unexported** — plus every exported
  type/field and package, in Go doc style.
- **Extreme code quality + high performance.** A change isn't done until it's tested and has
  survived adversarial review.
- **Owner-only admin, enforced server-side** (never trust the client). **Data plane is gRPC/WS
  only**; documents (SSR/wasm/RSS/PDF) are the document plane over HTTP GET.
- **Tool routing**: programs are local (WASM) by default, backend-routed whenever CORS, a
  secret, or a server capability is involved.
- **Feature-atomic commits + logs**: one self-contained change per commit (buildable, revertible
  in isolation — never bundle features, so a bad one reverts cleanly). Keep
  `documents/DEVLOG.md` (dated narrative: what/why/broke/next) and `CHANGELOG.md`
  (Keep a Changelog) current.

## Docs discipline
- **NEVER create a new `.md` file without Cam's explicit instruction** — update the existing docs
  instead. New markdown only when Cam asks for it by name.
- Keep `README.md`, `documents/DESIGN.md`, `documents/PROJECT_LAYOUT.md`, `documents/DEVLOG.md`,
  `CHANGELOG.md`, and `TODOS.md` in sync with every decision — log the *why*, not just the outcome.
