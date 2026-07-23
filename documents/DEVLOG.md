# DEVLOG — earlcameron.com

Newest first. Dated narrative of the build: what, why, what broke, what's next. Log failures too.

## 2026-07-23 — QOTD feed: durable post history + Slack decoupled

Cam asked to make sure the RSS engine generates on schedule, the format is right, nothing is
generated on the fly, and **past days are properly recorded**. Audit found the last point broken:
each publish overwrote the single `qotd_published` settings blob, so the feed only ever held one
item — and publishing was hard-coupled to Slack (no webhook → no RSS post at all).

Built: a `qotd_posts` table (one row per publish, kept forever; the feed serves the newest 30 via
`RecentQOTDPosts`), a transactional startup migration that moves the legacy blob into the table
exactly once, and a reordered `publishDiscussion`: generate → **record to RSS first** (this can
error) → best-effort Slack (missing webhook or failed POST no longer blocks the publish, and the
outcome lands in the Ack *and* the scheduler's log line — `PostScheduledIfDue` now returns the
message so a broken webhook isn't silent). Scheduling itself was already sound (minute ticker,
enabled+hour gates, claim-the-day-slot-before-posting). Format fixes: channel `<link>` now points
at the site, not the feed (RSS 2.0 spec); items link to `/#anime`; GUIDs now come from the
immutable DB row id, so two publishes in the same minute can't collide (readers de-dupe on guid).

Adversarial review caught four real issues in the first cut, all fixed: non-transactional
migration (crash between insert+delete → duplicate row next boot), minute-stamped GUID collisions
(double-click "post now"), the scheduler discarding the outcome message, and zero test coverage of
the decoupling. Added `openai.BaseURL` as a test seam and
`TestPublishDiscussionRecordsAndDecouplesSlack` (httptest OpenAI + failing Slack: both branches
record the post). Live-verified on a scratch server: seeded legacy blob + history row → feed
served both items newest-first, RFC1123Z dates, unique GUIDs. Known leftovers: no WAL/busy_timeout
pragmas on the shared SQLite handle (pre-existing), same-day manual+scheduled posts share a
day-granular title.

## 2026-07-23 — recruiter-readability pass on the home page

Cam asked for a review of how the home page reads to recruiters/hiring managers, then "refine it".
The review's findings, applied: (1) **mobile horizontal overflow** — the real bug. Flex items
default to `min-width:auto`, so the terminal's nowrap rows (prompt line, boot lines) plus the text
input's intrinsic width set a min-content floor that inflated the centered column past a 390px
viewport, clipping the hero/h1/cards. Fix: `min-width:0` on the `center()` column and the terminal
frame, `overflow-x:auto` on the terminal body (wide lines scroll inside the terminal — real
terminal behavior), `min-width:0` on the input. (2) SSR head now carries a meta description +
OG/twitter tags so recruiter-shared links get preview cards and Google gets a real snippet.
(3) Top nav: dropped `/admin` (owner-only; the discreet footer entry stays), renamed the opaque
"budget" label to "cashflux". (4) A quiet outlined "Read the résumé" secondary CTA next to the
terminal button — recruiters' primary action was buried as a text link. (5) One role-fit line at
the end of contact. (6) Content: GoGRPCBridge status was stale (`v0.0.19` → `v1.0.0`), GWC blurb
gains its quantified claim ("Benchmarked head-to-head with React — faster on overall geomean").

Adversarial review caught two real problems in the first cut: the new `<a>` button rendered with
the browser-default underline (there is NO sitewide link reset — `css.Preflight()` is never called,
and the mockup's `a{text-decoration:none}` was never ported; fixed locally on the button, the
sitewide gap is still open), and the blurb originally said "faster overall", which overstates the
mixed per-benchmark record — softened to the geomean claim the bench data actually supports.
Also flagged but deferred: no `og:image` (needs an asset), no `prefers-reduced-motion` gating
anywhere in SSR output (pre-existing, DESIGN.md promises it).

What broke along the way: headless-Chrome `--window-size=390` screenshots kept showing the clip
even after the fix — a stale cached `app.wasm` in the reused profile. Playwright with a real 390px
viewport is the trustworthy check: `document.scrollWidth == 390`, zero elements past the viewport
after wasm boot, clean full-page screenshots at 390/1440. Left alone (flagged for Cam): the orange
"Launch the live terminal" CTA is a no-op div in the SSR page (nothing wires it), and the anime
section's placement between elsewhere and contact is a taste call.

## 2026-07-22 — scheduled daily Slack posting (the toggle was dead)

Follow-up: the "scheduled posting" toggle stored `SettingSlackEnabled` but nothing acted on it — the
only way to post was the manual "post now". Made it real: a server-side minute-ticker
(`Server.runSlackScheduler`) calls a new `Service.PostScheduledIfDue(ctx, now)` that gates on
enabled + a configurable post hour (`SettingSlackPostHour`, exposed as `post_hour` on `SlackConfig`
with a "Daily post hour" input in the RSS panel) + a per-day guard (`SettingSlackLastPost`). It
claims the day's slot before posting so a failure skips the day instead of retrying every tick.
Factored the manual and scheduled paths onto a shared `publishDiscussion`. Tested: a deterministic
gating unit test (`TestPostScheduledIfDue`) plus a live run — configured it for the current hour with
a dead webhook and watched the server log `scheduled slack post failed` when the timer fired, proving
the whole chain (timer → gate → generate → post attempt → per-day guard) works end-to-end.

## 2026-07-22 — RSS panel: single generation prompt + dry-run

Cam wanted the /admin/rss page to drop the old QOTD prompt list and instead hold ONE editable prompt
he can save, plus a dry-run that generates a non-published post to test it. Reframed the "prompt" from
a static question into a generation instruction: it's fed the latest Anime News Network headline and a
model writes the discussion post. New `openai.Generate` (Responses API, free-text) backs it. Backend:
`GetPrompt`/`SavePrompt` over `SettingQOTDPrompt`; `DryRunPrompt` generates + returns a preview (body +
rendered RSS item) without touching the feed; `PostToSlackNow` now generates from the saved prompt,
posts to Slack, and stores the result as `SettingQOTDPublished`, which the `/anime/qotd.xml` handler
serves. Removed the orphaned multi-prompt machinery my change created — `store/qotd.go`, `rss/qotd.go`
(DefaultPrompts/SeedPrompts/DailyPrompt), `rss.QuestionFeedXML` and their tests — and dropped the
`qotd_prompts` table on startup so old seeded rows are gone (left the unrelated, already-dead
`anime.QuestionFeedXML` alone). Client: a textarea + Save/Dry-run buttons + a preview block (body +
RSS `<pre>`). Verified via chromedp: login → /admin/rss renders the textarea (pre-loaded default) +
buttons, no list; Dry run with no key shows the graceful "add an OpenAI key" error.

## 2026-07-21 — first-run owner setup + password reset

Cam wanted the deployed site to set up its own username/password on first run (not baked into env),
plus a reset strategy — he chose a recovery phrase with a hint, backed by an env break-glass. Built
bottom-up: a single-row `owner_credentials` table (bcrypt password + recovery hashes, plaintext
hint, nanosecond `password_changed_at`); a `recovery` package (embedded 286-word list, crypto/rand
rejection-free phrase generation, bcrypt helpers); a rewritten `Sessions` that prefers the stored
account, falls back to the `ADMIN_PASSWORD` env bootstrap, and reports `NeedsSetup` when neither
exists. Setup is guarded three ways — closed once an account exists, refused when env creds manage
auth (so a stranger can't seize an env-configured box), and gated by `ADMIN_SETUP_TOKEN` when set.
Reset verifies the phrase (or `ADMIN_RECOVERY_TOKEN`), rotates it, and bumps `password_changed_at`,
which a `pwa` JWT claim checks so a password change invalidates every prior token. New public gRPC
`AuthState`/`Setup`/`ResetPassword`; new WASM screens (setup, one-time phrase, reset). Tested:
comprehensive session unit tests (setup guards, reset, break-glass, token invalidation,
weak-password, setup-token, env-bootstrap) plus a chromedp drive of the whole flow — setup → phrase
(`orchid jelly pumpkin pouch canyon quartz`) → console → logout → forgot → reset — and a fresh-load
check that the hint ("first pet") renders on the reset screen. One test caught a real bug: a
second-resolution `password_changed_at` collided with the token's own mint second, so a same-second
password change didn't invalidate the token — switched to nanoseconds.

## 2026-07-21 — password gate + guest bypass for CashFlux

Cam wanted CashFlux on the home page for quick access, but gated with a password — with a guest
mode that bypasses it. He picked the semantics: password → your synced budget, guest → the full app
but local-only (no sync, my data never loads). Since CashFlux's sync config lives in its own
IndexedDB (not reachable from outside without touching the frozen frontend), the guest/full split is
enforced naturally by the sync token: a guest is simply never given it. So the gate is pure
server-side access control — an HMAC-signed, HttpOnly, path-scoped mode cookie in front of the SPA.
Built as a locked terminal-window door in the site's own visual language (mac chrome, `$ unlock
cashflux` prompt, aubergine). One bug caught in testing: `http.StripPrefix("/budget/")` strips the
leading slash, so the POST /__enter match had to clean the path first — before the fix every POST
fell through to the gate page (no cookie set). Next: the bigger ask — first-run credential setup +
a password-reset strategy for the deployed site.

## 2026-07-21 — CashFlux becomes a managed service (embedded sync engine)

Cam wanted CashFlux hosted here as a managed budgeting app with server-side storage, but explicitly
"just the data sync engine, not the whole site." So rather than embed CashFlux's full HTTP mux
(billing/portal/console/AI), the server now embeds only its gRPC `SyncService` over a GoGRPCBridge
tunnel at `/grpc`, backed by an encrypted server-side SQLite store. Added `CashFlux/pkg/embed.NewSyncBridge`
(upstream) on a sync-only `NewSyncBridgeHandler` twin of CashFlux's full bridge — same interceptor
chain, minus AIService. Adversarial review confirmed the narrowing is sound (`/grpc` is the sole
bridge mount the full server used; a pure sync client needs nothing else) but caught the real
landmine: token-mode auto-mints a random token each boot that the embed never surfaced, so sync was
unauthenticatable — `NewSyncBridge` now returns the generated token and we log it at startup. Also
seed CashFlux's WS app-origin from `BASE_URL`, normalized to a bare `scheme://host` so a trailing
slash doesn't silently disable the whole engine. Next: gate `/budget/` on the home page with a
password (guest-mode bypass).

## 2026-07-21 — planning + P1 foundation
**Planning (this session).** Locked: the concept (portfolio-as-terminal + a conventional
standard site, two front doors, unified hero); the "Aubergine" design language (macOS chrome,
Ubuntu-souled palette, two typographic voices); the architecture (GWC WASM client ↔
gRPC-over-WebSocket via GoGRPCBridge ↔ Go backend; no-HTTP-except-ingress data plane vs a
document plane); the feature set (`ask` nano assistant with a hard $20/mo budget, i18n + BYOK
real-time translation, résumé + agentic tailoring, blog + RSS, anime RSS, a live CashFlux
instance, owner-only admin); the deploy plan (Ubuntu + Nginx on DigitalOcean, one-click install,
push-to-deploy); and the agent governance (karpathy-guidelines + frontend-design skills,
mandatory adversarial review, strong shared types, documented functions). Built an interactive
design mockup at `design/mockup.html`.

**P1 foundation (build started).** Minimal Go module + ingress server: `cmd/server`,
`internal/config`, `internal/server` — stdlib only, no speculative deps (Karpathy #2). Serves
`/healthz`, static files, an SSR placeholder; graceful shutdown on SIGINT/SIGTERM.
**Verified:** `go build ./...` and `go vet ./...` clean; `/healthz` → `ok [200]`; `/` → placeholder
`[200]`. Established `/bin` as the gitignored build-output dir.

**Broke / notes.** The *old* site's `better-sqlite3` wouldn't compile on Node 26 / ARM64 —
irrelevant here (new stack is pure Go). **`protoc` is NOT installed** — that's the next gate
before any gRPC service.

**Next.** Install `protoc` + Go plugins → write `proto/site.proto` → implement `ContentService`
(projects/about), the single source both the terminal and the standard site read from.

### Build progress (P1, same day)
Used `buf` (pure Go) instead of a protoc binary — cleaner on ARM64. Shipped as atomic commits:
gRPC contract (Content + Contact); `ContentService` (9 projects + about, tested); the
gRPC-over-WebSocket tunnel at `/socket` (GoGRPCBridge) with services registered; `ContactService`
+ a pure-Go SQLite store (modernc.org/sqlite, tested); and the **standard site** rendered
server-side as GWC components with typed CSS (`css/u`), mobile-first responsive (`@media 768px`),
served at `/` — live at http://127.0.0.1:8095.

**Broke / corrected.** (1) Hand-wrote a raw `.css` file for SSR — wrong for this stack; reverted
and switched to GWC's typed `css/u` per Cam (portfolio-site's raw-Tailwind approach is the
anti-pattern; `examples/public/typed-css/counter.go` is the template). (2) Commit security review
caught a **CSWSH** hole — my dev `CheckOrigin: return true` allowed any origin; replaced with a
same-origin + allow-list validator (tested).

**Terminal + UI (done, same day).** WASM pipeline verified in-browser via headless-Edge
screenshots — **new discipline: always screenshot-check UI against the mockup** (added to the
agent files after I once shipped an unverified layout Cam caught). Standard site: responsive
project **grid** (typed `css.Property` for `grid-template-columns`, since `css/u` lacks it),
tighter vertical rhythm, and a minimal bootstrap `<style>` for the mono font-family the typed
system can't express (Cam OK'd a tiny bootstrap). Terminal: built the **macOS-window visual**
(traffic-light chrome, boot log, neofetch, prompt) in wasm with typed CSS — now matches the
mockup. Added `scripts/build.sh`. Design tokens centralized to `internal/theme`.

**Next.** Make the terminal interactive: controlled input → command dispatch → gRPC programs
(projects/about/contact over the tunnel) + local programs (help/theme/clear/neofetch).

### Carried-over features + auth hardening (same day)
Ported two features off Cam's current site and closed a security round.

**Anime tracker (Go).** Replaced the old Node feature. `internal/anime` queries AniList's public
GraphQL (search + refresh); `internal/store/anime.go` persists `tracked_anime`; two RSS 2.0 feeds
ship at `/anime.xml` (Release Radar) and `/anime/qotd.xml` (daily discussion prompts). A strong
config page at `/admin` (search with cover art, track/untrack, run release check) is password-gated.
Verified end-to-end: search *Frieren* → track → it appears in `/anime.xml` (curl + screenshot).

**Résumé.** A print-optimized HTML résumé at `/resume` (light professional document, Ubuntu-orange
accent, "Save as PDF") — chose browser Save-as-PDF over authoring/maintaining a static PDF or a
server-side Go PDF; simpler and always in sync. `internal/resume` holds the structured content +
renderer. Owner-gated **tailoring tool** at `/admin/resume`: paste a job-posting URL → server
fetches it (HTML→text) → OpenAI re-emphasizes *existing* facts to fit. Guardrails: system prompt
forbids fabrication, and identity/contact fields are force-overwritten with the originals after the
model returns (defense in depth against prompt injection). Disabled with a clear notice when
`OPENAI_API_KEY` is unset. Site card + terminal `resume` now point to `/resume`.

**Broke / corrected.** Commit security review flagged three real issues in the admin gate, all
fixed and verified: (1) **auth-bypass** — the session secret fell back to a *known* constant, so
anyone could forge an admin cookie; now a random per-process 32-byte secret when `ADMIN_SECRET` is
unset (set it to persist sessions across restarts). (2) **insecure-cookie** — session cookie now
`Secure` when `BASE_URL` is https. (3) **brute-force** — `/admin/login` is rate-limited (5 fails →
15-min lockout → 429); confirmed 5×200 then 429. Behind nginx the limiter is effectively global on
one RemoteAddr — fine for a single-owner gate.

**Next.** Résumé-tailor follow-ups: OpenAI budget cap ($20/mo) + SSRF guard on the URL fetch
(block private/loopback). Then wire the terminal's portfolio programs to real gRPC.

### RSS control panel + CashFlux managed service (same day)
Two features, built by **two parallel Sonnet subagents on disjoint new packages** (worktree isolation
was off-limits — it breaks the `../GoWebComponents` relative replace — and two agents on the shared
proto/server/client tree would collide, so each agent owned a standalone package: `internal/rss`,
`internal/budget`), then wired into the gRPC/WASM admin by hand and **adversarially reviewed** (one
fresh Sonnet reviewer per feature) with a fix loop.

**RSS** (ported from the old Node `PersonalWebsite2026/api`): configurable QOTD prompts (50 seeded,
`qotd_prompts` + unique-index dedup), spec-compliant RSS 2.0 via `encoding/xml` (atom:self, RFC1123Z,
stable guid), Anime News Network fetch, and a Slack incoming-webhook poster that composes the latest
headline + today's prompt into a discussion topic — all controllable from a new `rss` admin tab over
`AdminService`. **CashFlux** is built to WASM and hosted at `/budget/` as a managed SPA.

**Broke / corrected (adversarial review).** RSS reviewer caught a real **HIGH**: unescaped ANN
headlines flowing into Slack `<url|label>` mrkdwn — a hostile headline could `@channel`-ping or forge
a link; fixed with Slack escaping (+test). Also: unstable RSS guid (re-showed tracked shows every
episode), a seed TOCTOU race (→ unique index + INSERT OR IGNORE), prompt length cap, news
redirect-host restriction. CashFlux reviewer caught a path-containment gap (stdlib-side-effect
reliance) → explicit `filepath.Rel` guard. Verified in-browser: RSS panel + 50 prompts + Slack
config, CashFlux dashboard boots at `/budget/`, both feeds valid XML. Tests + vet green.
