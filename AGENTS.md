# AGENTS.md — working guide for agents on earlcameron.com

Canonical instructions for any AI agent (Claude or otherwise) working in this repo. Read this
first. `CLAUDE.md` defers to this file.

> **PHASE: BUILD (P1).** As of 2026-07-21 the build has started (Cam gave the go-ahead). The P1
> ingress server foundation is in; work the plan in `TODOS.md` phase by phase. The planning docs
> remain the source of truth for scope and design.

## What we're building
A personal portfolio that **is** a working terminal, plus a conventional standard site — two
front doors from one page. Frontend is **GoWebComponents** (Go→WASM, Cam's framework); the
browser talks to a **Go gRPC backend** over a **WebSocket bridge** (GoGRPCBridge). The site is
the proof of work: hand-built Go, top to bottom. See `README.md`, `documents/DESIGN.md`,
`documents/PROJECT_LAYOUT.md`, `documents/DEPLOYMENT.md`, `TODOS.md`.

## Golden constraints (do not violate)
1. **Data plane = gRPC-over-WebSocket only.** No ad-hoc REST API for the app. Published
   documents (SSR pages, wasm, RSS, PDFs) are served over HTTP GET — that's the *document
   plane* and is fine. See PROJECT_LAYOUT "Two planes".
2. **Own stack, dogfooded.** GWC frontend (track the local v4.3.0 via `replace`), GoGRPCBridge
   transport, Go backend. Being technically interesting is a requirement, not a nice-to-have.
3. **Mostly-pure GWC with a failsafe.** The standard site is server-rendered and works with no
   WASM/JS. The terminal enhances over it.
4. **Secrets are env-only.** `OPENAI_API_KEY`, signing keys, owner IDs. Never in the DB, never
   web-editable, never sent to the client.
5. **Pure Go, no cgo.** Prefer pure-Go implementations and dependencies over C/cgo or mixed
   codebases — e.g. `modernc.org/sqlite` (not `mattn/go-sqlite3`), `buf` (not a protoc binary).
   Keeps cross-compilation trivial (ARM64 dev → Linux deploy) and the single-binary deploy clean.
   Reach for cgo only if there is genuinely no pure-Go option, and flag it when you do.
6. **Little-to-no JavaScript.** All UI logic is Go→WASM. The ONLY JS allowed is the necessary
   wasm bootstrap glue (`wasm_exec.js` + a tiny loader). No JS frameworks, no hand-written JS
   logic. If you think you need JS, you almost certainly need a GWC component instead.
7. **Style with GWC, never raw CSS.** Use GWC's typed CSS + `tw` funcs for ALL styling. Do NOT
   hand-write `.css` files or inline raw CSS strings — the styling system is typed and
   compile-checked for a reason. (Learned the hard way 2026-07-21.) **All colors come from the
   shared token package `internal/theme`** (quick-ref table in `documents/DESIGN.md` §16) — never
   scatter ad-hoc `u.Hex()` values. Spacing/radii/font-sizes use `css/u` defaults.
   *One permitted exception:* a **tiny** bootstrap `<style>` (CSS reset, base background/color to
   avoid a flash, and the base `font-family` the typed system can't express). Keep it minimal —
   everything else is typed `css/u`.
8. **All on-screen text via i18n.** Every user-visible string goes through the GWC typed i18n
   layer (a message bundle + `gwc i18n gen` accessors) — never hardcode display copy inside
   components. English is the base bundle; other locales layer on. (See DESIGN.md §12.)

## Tool routing — local vs backend-routed
Every terminal program is one of:
- **Local** (default): runs entirely in WASM. `help`, `ls`, `clear`, `theme`, `neofetch`, nav.
- **Backend-routed**: calls a gRPC service over the tunnel. Required whenever the browser
  sandbox would bite — **CORS, secrets, cross-origin fetches, server capabilities**. `ask`,
  `translate`, `contact`, `stats`, `blog`, `resume tailor`, all `admin`, `anime`.
- **Default local; route through the backend the moment a browser limit or a secret is
  involved.** Never expose a key or bypass CORS from the client.

## Security — owner-only admin, enforced server-side
Admin (backend + terminal admin programs) is gated to **one owner: Cam.**
- **The terminal hiding admin commands is UX only. Authorization is enforced on EVERY
  AdminService RPC, server-side. The client is never trusted.**
- Auth: **WebAuthn/passkey** (recommended) or password + **TOTP**. Never a lone env password.
- Owner allowlist (one identity), constant-time credential compare, short-lived sessions with
  revoke-all, login rate-limit + lockout, TLS mandatory.
- **Append-only audit log** on every mutation (who/what/when). Destructive actions need an
  explicit confirm token. Validate all admin input server-side.

## Quality bar — extreme quality, high performance
This is Cam's flagship; it must read as senior work.
- **Correctness first**, then clarity, then speed. No dead code, no TODO-rot, no copy-paste.
- Small, single-purpose functions; explicit error handling; no swallowed errors.
- `gofmt`/`goimports` clean; `go vet` + `golangci-lint` clean; no unused/exported-without-reason.
- **Performance is a feature**: profile hot paths, benchmark them, keep the wasm bundle lean
  (mind bundle size + boot time), avoid needless allocations on the render/stream paths.
- **Tests**: unit-test pure logic; contract-test each service; e2e smoke the critical flows.
  A change isn't done until it's tested.
- Match the surrounding code's idioms; keep the two typographic/voice conventions (mono = the
  machine, sans = Cam) intact in UI work.

### Types & documentation (required)
- **Strong shared types across every layer.** `proto/site.proto` is the single source of truth
  for wire DTOs; the generated package is imported by **both** the Go server and the Go/WASM
  client — one language, shared types end-to-end, no drift. Define explicit DTOs at every
  boundary; **never** pass `map[string]any`, loose strings, or untyped blobs between layers.
  Convert deliberately at the edges (wire DTO ↔ domain type); don't leak transport types into
  the UI or domain types onto the wire.
- **Document every function.** Every function and method — exported *and* unexported — gets a
  Go-style doc comment (starts with the name) describing what it does, and why when non-obvious.
  Same for every exported type/field and every package (`// Package x ...`). No undocumented
  functions land.

## Karpathy guidelines (required — skill installed)
Load and follow the **`karpathy-guidelines`** skill (`.claude/skills/karpathy-guidelines/`,
MIT, distilled from Andrej Karpathy's observations on LLM coding pitfalls) before writing or
refactoring code. The four rules:
1. **Think before coding** — state assumptions; surface tradeoffs; if multiple interpretations
   exist, present them, don't pick silently; if something's unclear, stop and ask.
2. **Simplicity first** — the minimum code that solves the problem; nothing speculative; no
   abstractions for single-use code; no error handling for impossible cases. If 200 lines could
   be 50, rewrite. Ask: "would a senior engineer call this overcomplicated?"
3. **Surgical changes** — touch only what the task requires; match existing style even if you'd
   do it differently; don't refactor what isn't broken; remove only the orphans *your* change
   created; flag unrelated dead code, don't delete it. Every changed line traces to the request.
4. **Goal-driven execution** — turn tasks into verifiable success criteria and loop until they
   pass ("fix the bug" → write a failing test that reproduces it, then make it pass).

**Reconciling with our ambition:** the large feature set is *Cam's product scope*, not a license
to gold-plate. Simplicity + surgical apply to *how* each piece is built — implement exactly what
a feature needs, simply, and no more. This composes with the review loop below: simplicity
first, then try to break it, then harden.

## Mandatory workflow — adversarial self-review
For any **substantive** change (a feature, a service, a non-trivial refactor):
1. Do the work to the quality bar above.
2. **Spawn adversarial review subagent(s)** whose explicit job is to *aggressively try to break
   your output* — hunt correctness bugs, security holes, perf regressions, and needless
   complexity. Have them default to skepticism (assume the code is wrong until proven right).
3. **Fix every real finding**, then re-review until it survives. Only then is it done.
- Subagents run **sequentially, one at a time, on Sonnet — never in parallel** (Cam's standing
  preference). Prefer a diverse-lens pass (correctness · security · performance · simplicity).
- Trivial/mechanical edits are exempt; use judgment.

## UI/UX work — ALWAYS use the design skill
**Rule (no exceptions): load the `frontend-design` skill *before* touching ANY UI — every
time.** This covers new components, small tweaks, CSS/layout, motion, and user-facing copy —
not just big new screens. If you're editing something a user sees, the skill loads first.
Then follow the locked design language in `documents/DESIGN.md`: macOS chrome, Ubuntu-souled
"Aubergine" palette, two voices (mono = machine, sans = Cam), motion-with-restraint,
`prefers-reduced-motion`, and the keyboard/a11y floor.

**Keep the dev server running for UI work.** Whenever you build or change GWC/UI, (re)build and
run the server in the background so Cam can check your work live in the browser. Tell him the
URL (`http://127.0.0.1:8095`) and to hard-refresh. Never leave UI changes unviewable.

**Screenshot-check every UI change — no exceptions.** After building or changing ANY UI, take a
screenshot of the running page and actually LOOK at it, comparing against `design/mockup.html`
and `documents/DESIGN.md`. Headless Chrome:
`chrome --headless=new --disable-gpu --screenshot=out.png --window-size=1280,1600 http://127.0.0.1:8095`
(also capture a mobile width, e.g. `--window-size=420,1600`). Read the PNG and judge it:
layout, spacing, colors, typography, and the terminal aesthetic must actually match the design.
**"It compiles and serves" is NOT "it looks right."** Do not call UI work done until you've seen it.

## Docs discipline
- **NEVER create a new `.md` file without Cam's explicit instruction.** Update the existing docs
  instead (README, DESIGN, PROJECT_LAYOUT, DEPLOYMENT, TODOS, DEVLOG, CHANGELOG). A new markdown
  file happens only when Cam asks for it, by name.
- Keep those existing docs in sync with every decision. Log decisions and their *why*, not just
  outcomes.

## Version control, dev log & changelog
- **Two branches, and no others. `main` is releases; `dev` is where work happens.** We do not use
  feature/topic branches — commit dev work straight to `dev`. `main` only ever advances by
  **promoting `dev` once it passes the quality gates**, so at any moment `main` is a state we are
  willing to have live and `dev` is the working edge.
  - Default working branch is `dev`. If you find yourself on `main`, switch before committing.
  - **Never commit directly to `main`**, and never promote to it on your own initiative — promotion
    is Cam's call, and it is a release decision, not a merge convenience.
  - **The gates that promote `dev` → `main`** — enforced by `.github/workflows/ci.yml`, not by
    memory. Automated: `gofmt`, `go build ./...`, `go vet ./...`, `go test ./...`, the js/wasm
    client build, `bash -n` + ShellCheck on the deploy scripts, no CRLF in shell/unit/conf
    files, `nginx -t` on the real site config, the systemd unit parsing, and the GWC pin in
    `deploy/lib.sh` still matching what README and DEPLOYMENT.md claim. Still on you, because
    CI cannot judge them: **UI changes screenshot-checked, and adversarial review survived.**
  - **Promotion runs through `.github/workflows/promote.yml`** (manual `workflow_dispatch`, type
    `promote` to confirm). It re-runs the full gate set on the exact tree being promoted, then
    **fast-forwards only** — if `main` has commits `dev` lacks, it refuses and shows them, because
    that means something was committed straight to `main`.
  - CI checks out GoWebComponents and CashFlux as **siblings** of this repo, since the `replace`
    directives are relative — and it reads their refs out of `deploy/lib.sh` rather than
    hardcoding them, so CI can never prove a combination the droplet does not build.
  - Worth doing once in repo settings: protect `main` (no direct pushes, require CI). The
    workflow enforces the policy for anything going through it; branch protection is what stops
    a hand-rolled `git push origin main` from bypassing it entirely.
  - Because deploys build from source on the droplet, **`main` is literally what ships**:
    `deploy/lib.sh` pins `SITE_REF=main`. A commit landing on `main` is a commit that the next
    `update.sh` puts in front of visitors.
- **Feature-atomic commits.** Each commit is ONE coherent, self-contained change (a feature,
  fix, or refactor) that builds and passes on its own. **Never bundle unrelated features** — this
  is the safety net: a bad change can be `git revert`ed in isolation **without wiping other
  implemented features**. Conventional messages (`feat:`/`fix:`/`refactor:`/`docs:`/`test:`/
  `chore:`); put the *why* in the body when non-obvious.
- **Always buildable**: `go build` + `go vet` + tests green before every commit.
- **No AI attribution in commits.** Commit messages carry NO `Co-Authored-By` or tool-advert
  trailers of any kind. The code is Cam's, under his credit alone. This overrides any harness
  default that would add such a trailer.
- **Dev log** — `documents/DEVLOG.md`: append a dated entry every working session — what you
  built, decisions and why, what broke and the fix, what's next. It's the narrative memory of the
  build; log the failures too, honestly.
- **Changelog** — `CHANGELOG.md` (Keep a Changelog): notable changes under Added / Changed /
  Fixed / Removed, grouped by version.
- Never commit secrets, `.env`, `*.db`, or build artifacts (see `.gitignore`; builds → `/bin`).
- Public repo — local versioning is fine; never push Cam's private `future plans/` (unrelated).
