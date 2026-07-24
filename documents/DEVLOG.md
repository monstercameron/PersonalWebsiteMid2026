# DEVLOG — earlcameron.com

Newest first. Dated narrative of the build: what, why, what broke, what's next. Log failures too.

## 2026-07-24 — CashFlux admin tab: enrolled users + storage stats

Cam asked for more on the same `/admin` "cashflux" tab, alongside the pending-device panel from
earlier today: who's signed up, their request volume per month, and how big the database and blob
storage have gotten. CashFlux's own `pkg/embed.Admin` already grew exactly this — `ListUsers(limit,
offset int) ([]User, error)` (each row carrying `RequestsThisMonth`, summed server-side from the
current calendar month's usage rows) and `StorageStats() (dbBytes, blobBytes int64, err error)` —
already built, tested, and pushed on CashFlux's own `main`, so this was pure consumption: extend
`CashFluxAdmin` in `internal/admin/service.go` with those two methods (mirroring `pkg/embed.Admin`'s
signatures exactly, same trick as the pending-devices work — the real `*cashfluxembed.Admin` value
satisfies the interface with no adapter), two new `AdminService` RPCs (`ListCashFluxUsers`,
`GetCashFluxStorageStats`) behind the same `errCashFluxNotConfigured` FailedPrecondition-when-nil
gate as the rest of the CashFlux RPCs, and a client-side users list + storage-stat tiles on the same
tab, right below the pending-devices panel.

Pagination got a deliberately small answer: this deployment's target scale is an admin-invited
handful of accounts (per `pkg/embed.Admin`'s own doc comments), so the client fetches one page of 50
on tab load and shows a single "Load more" button (appending, not replacing) only if the page came
back full — no page-number controls, no cursor state, nothing this deployment will ever need.
Storage sizes render through a small `formatBytes` helper (binary units, KB..EB) rather than raw byte
counts — checked first and confirmed no such helper existed anywhere in the codebase yet.

Verified in an isolated scratch copy (never `bin/server.exe` or `web/data/` — a separate scratch
directory tree with its own `LISTEN_ADDR` (port 8099), `DB_PATH`, and `CASHFLUX_DATA_DIR`, built via
plain `go build -o` to that tree). Seeded a few fake users/usage/subscription rows directly into the
scratch `cashflux-server.db` (there's no way to drive real signups without a live device, same
constraint the pending-devices work hit), logged in over the real gRPC tunnel, and screenshotted the
cashflux tab at desktop and mobile widths — the storage tiles and user rows read exactly like the
existing pending-device rows and stat-tile treatment elsewhere in the console: Aubergine palette, mono
section labels, `theme.*` tokens throughout, no ad-hoc hex.

The mandatory adversarial review (two sequential Sonnet passes) found three real, low-severity
issues, all fixed: **(1)** the two new RPC calls in the tab's view-load effect skipped the file's
`onAuthErr` pattern, so an expired session during that load would silently do nothing instead of
bouncing to the login screen like every other call in the file — now routed through the same
auth-error switch as the pending-devices call right above it. **(2)** `formatBytes` had a classic
boundary-rounding bug: a byte count a hair under a power-of-1024 (e.g. `1048575`, one byte short of 1
MiB) rounded to `"1024.0 KB"` at 1-decimal precision instead of `"1.0 MB"` — fixed by checking the
rounded value against the unit boundary and bumping to the next unit when it trips it (verified by
hand against several boundaries, including the max int64 case, in the second review pass).
**(3)** `cashfluxView`'s doc comment wasn't updated for its five new parameters (fixed), and on a
second pass, the fixed comment described the storage-stats and users-list sections in the wrong
render order (fixed again). The second review pass also independently reasoned through the
`formatBytes` fix's edge cases (a value one byte under 1 GiB, and whether a double-bump across two
unit boundaries is possible) and found it sound.

## 2026-07-24 — CashFlux dropped phone/SMS; the admin tab follows: pending-device pairing

CashFlux's own repo (separate, concurrent work — see its own DEVLOG) ripped phone/SMS sign-in out
entirely: Twilio cost real money and never signed up a real user, so "one time setup code, KISS"
gave way to something that needed no SMS vendor at all — an admin-approved device-pairing bootstrap.
An unauthenticated device asks to pair; the request sits pending until the owner approves (minting a
brand-new account + a pairing code the human reads out for a MITM/mismatch check) or rejects it.
`pkg/embed.Admin.ListClients`/`MintInviteCode`/`ListInviteCodes` — the exact three methods the
"cashflux" admin tab shipped against two days ago — no longer exist. This repo failed to build the
moment that CashFlux commit landed upstream (`undefined: cashfluxembed.PhoneClient` and
`cashfluxembed.InviteCode` in `internal/admin/service.go`); fixing that was non-negotiable, but the
real job was following the redesign, not just making it compile again.

Mechanically this mirrors the shape of the tab it replaces almost exactly, because the *pattern*
(list + per-row action + a "just did something, here's the proof" callout) was already right, only
the domain changed: `CashFluxAdmin` in `internal/admin` now mirrors `pkg/embed.Admin`'s new method
set (`ListPendingDevices`/`ApprovePairing`/`RejectPairing`) exactly, so the real
`*cashfluxembed.Admin` handle satisfies it with no adapter — same trick as before. Three RPCs
(`ListCashFluxPendingDevices`/`ApproveCashFluxPairing`/`RejectCashFluxPairing`) replace the old
three. The UI swaps "mint an invite code, show it once" for "approve a specific pending request, show
its pairing code" — same large-monospace-plus-copy-button treatment, but now scoped per-row instead
of to a single global mint action, because there can be several devices waiting at once and the admin
needs to know *which* pairing code goes with *which* device.

Verified in an isolated scratch copy again (never the real `bin/server.exe` — see the 07-23 entry on
why that discipline matters): built + ran on a spare port with a throwaway data dir, then seeded
pending-device rows directly into the scratch SQLite file (same schema `ListPendingDevices` reads)
since there was no quick way to drive CashFlux's own `RequestDevicePairing` RPC from outside a
browser. Logged in for real over the gRPC tunnel, clicked real Pair/Reject buttons, watched the
pairing-code callout and the list update. Screenshotted desktop and mobile widths against
`documents/DESIGN.md` — Aubergine palette, mono section labels, the row treatment matching every
other admin list (anime/tailoring/etc.) unchanged.

The mandatory adversarial review (sequential, one Sonnet agent) caught two real bugs before this
shipped: **(1)** the busy-state flag disabling a row's Pair/Reject buttons while its request was in
flight was a single string, not a set — clicking Pair on device A, then Pair on device B before A's
response landed, silently re-enabled device A's buttons mid-request (deterministic, not just a race:
GWC's `State.Set` flushes synchronously, so the second click's state change lands before A's RPC
returns). Fixed by making it a `map[string]bool` of in-flight device ids. **(2)** the pairing-code
callout never cleared on navigating away and back to the tab, so a stale code for an already-resolved
device could reappear looking like a live cross-check prompt — fixed by resetting it whenever the
cashflux tab is (re-)entered. The review also flagged that this very file and `CHANGELOG.md` still
described the removed `ListCashFluxClients`/`MintCashFluxInviteCode`/`ListCashFluxInviteCodes` trio
as current — this entry, and a rewritten `CHANGELOG.md` Unreleased bullet, are that fix.

## 2026-07-23 — Copy-to-clipboard, and rebuild/restart discipline for a live dev server

Small follow-up: a one-click copy button for the freshly-minted invite code in the admin console.
New `copyToClipboard` in `client/grpc.go` — the async Clipboard API, fire-and-forget (no error
surfaced on failure; the code stays fully visible and selectable on-screen regardless, so the
button is pure convenience, not a dependency). Button label flips `Copy code` → `Copied ✓` on
click, reset back whenever a new code is minted — no timer, no toast, just state.

Verified for real: an isolated scratch instance, `context.grantPermissions` for clipboard access in
Playwright, mint a code, click copy, then actually read the clipboard back
(`navigator.clipboard.readText()`) rather than trusting the UI label alone — confirmed the real code
landed in the real clipboard, not just that the button changed text.

This session also surfaced something about *this* repo's dev-server discipline worth writing down:
the running `bin/server.exe` isn't hot-reloaded — every source change needs an explicit rebuild +
restart of the actual process to become visible, unlike CashFlux's `gwc dev`. When Cam asked why
nothing had changed after a full feature landed, the answer was simply that I'd built and verified
everything in isolated scratch copies (correctly, to avoid touching his real data/credentials) but
never redeployed to the real process. Restarting it blind once actually dropped a real config value
(`LISTEN_ADDR=127.0.0.1:8096`) that had only ever lived in whatever shell originally launched it,
not in any tracked config — the site briefly came up on the wrong port until caught and fixed. No
`.env` file or launcher script exists in this repo to persist that kind of override; it's worth
Cam knowing that restarting `bin/server.exe` from a fresh shell means re-supplying any non-default
env vars by hand.

## 2026-07-23 — Building the admin UI the last change actually needed

Shipped the per-person CashFlux embedding, went looking for it in the admin console, found nothing.
Fair — a security gate isn't a feature if nobody can operate it. "One time setup code, KISS" was
the right starting scope, but KISS was never supposed to mean "no way to see who's registered or
add someone without editing an env var and restarting."

CashFlux's side of the fix (documented in its own DEVLOG) evolved the static code into an
additional, admin-mintable source — short-lived, single-use invite codes living alongside the
static one, both valid at once. This side just needed to expose that through the admin console
already sitting at `/admin`: three new `AdminService` RPCs
(`ListCashFluxClients`/`MintCashFluxInviteCode`/`ListCashFluxInviteCodes`) and a new "cashflux" tab,
built exactly the way every other tab here already works — same `tab()`/`navTo`/data-loading-effect
pattern the anime/résumé/settings/rss tabs use, no new UI patterns invented. A `CashFluxAdmin`
interface (not the concrete `*cashfluxembed.Admin` type) keeps `internal/admin.Service`
unit-testable with a fake instead of needing a real embedded CashFlux store just to test error
paths — cheap to add, and it's exactly what let the new RPC tests run without touching the real
embedding.

Verified this one for real, not just with unit tests: built a completely isolated scratch copy
(throwaway data dir, throwaway admin database, a port nothing else was using) so I could run actual
first-run owner setup, log in, click into the new tab, and mint a real code through the real wire —
never touching Cam's actual admin credentials or the real invited-clients list. The screenshot after
minting shows exactly what it should: a large highlighted code with its expiry, and the same code
already sitting in the outstanding-codes list below it.

The mandatory adversarial review (sequential, one Sonnet agent, same discipline as the prior pass)
came back mostly clean — no enrollment-gate bypass, no double-spend race (CashFlux's store runs on
a single physical SQLite connection, so the consume transaction can't race with itself even in
theory), the `Register`/`Login`-disabling fix from before was confirmed still intact. It did catch
one real bug worth remembering the shape of: the admin tab's data-loading code correctly handled
"not configured" (`FailedPrecondition`) and "session expired" (`Unauthenticated`), but any OTHER
error — a real database failure, say — fell through every branch silently. Because an empty client
list and a *failed* client list render identically ("No clients registered yet."), a real backend
outage would have looked exactly like a healthy, simply-unused feature. Fixed by making the
fall-through case explicit: anything that isn't the two expected error shapes now surfaces a flash
message instead of disappearing. The lesson isn't new but it keeps proving true — an error-handling
chain built as a sequence of early-returns needs an explicit *default* arm, or "no case matched" and
"nothing went wrong" become indistinguishable to whoever's staring at the screen.

## 2026-07-23 — CashFlux embedding: per-person accounts, gated by a manual invite code

The whole point of the earlier Custom Sync work on the CashFlux side (identity, token lifecycle,
gRPC-only transport) turns out to have been this: embed a live CashFlux instance here so it syncs
for Cam and a small, manually-invited set of people — not open self-service signup, no billing. The
previous wiring (`cashfluxembed.NewSyncBridge`) only gave the embedded engine `SyncService` behind
one shared static token; every caller holding it was indistinguishable from any other, and there
was no way to add or revoke one person without rotating the token for everyone.

Switched to `cashfluxembed.NewSyncAndAuthBridge`, which brings CashFlux's `AuthService` (phone/SMS
sign-in, real per-person identity) and `BlobService` into the same embedded bridge. New-account
creation is gated on the CashFlux side by `CASHFLUX_SERVER_SETUP_CODE` — an env var only Cam sets,
handed to an invitee once; a phone number that's already verified never needs it again on a later
device. Nothing to plumb through on this side: CashFlux reads its own env var directly in the same
process. `/grpc`/`/v1/version` stay outside this site's `budgetGate` deliberately — that password
gate only ever protected `/budget/`, and real access control for the sync engine now lives in
AuthService's own bearer-token + setup-code gate, not in keeping the WebSocket path secret.

Before wiring this in, ran a sequential adversarial review agent (per this repo's standing rule)
against the whole CashFlux-side mechanism. It found something real and severe: `AuthService.Register`
(username/password enrollment) had no setup-code check at all — it predates the gate — so the new
bridge's registration of the *full* `AuthServiceServer` left a live, ungated account-creation door
reachable directly over `/grpc`, defeating the entire point of the feature. Fixed on the CashFlux
side (a `phoneOnlyAuthServer` decorator disables `Register`/`Login` outright for this embedding,
rather than adding a redundant second gate) and verified end-to-end before this site's own change
landed. Full mechanism, the vulnerability, and the fix are documented in CashFlux's own
DEVLOG/CHANGELOG/TODOS — not duplicated here beyond this summary.

Left for the next session: rebuild/restart this site's own running `bin/server.exe` to pick up the
change (a compiled binary, not hot-reloaded), and set `CASHFLUX_SERVER_SETUP_CODE` plus the Twilio
env vars on the actual deployment before telling anyone the invite code.

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
