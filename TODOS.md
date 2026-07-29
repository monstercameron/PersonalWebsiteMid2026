# TODOS — earlcameron.com (mid-2026)

Status legend: `[ ]` todo · `[~]` in progress · `[x]` done · `[?]` needs a decision from Cam

## RSS + CashFlux managed-service effort (tracking — 2026-07-22)
_(No Claude Code todo tool exists in-session; tracked here per project convention.)_
- [x] RSS QOTD reworked to a single **generation prompt** (textarea, save) + **dry-run** preview
      (OpenAI Responses API against the latest Anime News Network headline). E2E'd via chromedp.
- [x] **Post to Slack now** — generate from the saved prompt → post discussion to Slack → publish to
      the QOTD feed. Manual path done + tested.
- [x] RSS **spec compliance** — RSS 2.0 tests (`TestPublishedFeedXMLSpecCompliance`, `TestTrackedFeedXML`)
      + Slack mrkdwn injection-escaping tests. Passing.
- [x] **CashFlux managed sync** embedded (gRPC `/grpc` + `/v1/version`, encrypted server store),
      sync-transfer logging, and the sync-page encryption decision panel. E2E'd.
- [x] **Scheduled Slack posting** — the "scheduled posting" toggle now drives a real server-side daily
      scheduler (configurable post hour); auto-generates + posts + publishes once/day.
- [ ] Managed-service billing/subscription/console — **out of scope per Cam** ("just the data sync
      engine, not the whole site"). Intentionally not built.

## 0. Decisions still open (need Cam)
- [x] **Featured projects** — 9 chosen & wired into the mockup (count is dynamic, N). Order is the
      billing, since sitepb.Project has no featured flag — **headliners first: CashFlux,
      ArticleFlux, GoWebComponents, GoGRPCBridge**, then WebGL Path Tracer, WASIBrowser,
      SemanticScript, SemanticAssembly, SemanticPortrait. Nine also fills the desktop grid's three
      columns exactly: row 1 the shipped apps + their framework, the path tracer's live demo in the
      centre cell of row 2, the three Semantic\* research projects as row 3. (Gemma-4 and
      *vkPathTracer* removed per Cam; ArticleFlux added and WhisperToMe dropped 2026-07-28; the
      *WebGL* path tracer added 2026-07-28 — chosen over vkPathTracer because it has a live GitHub
      Pages demo a recruiter can click. Adjust anytime — but change all three lists, see below, and
      re-check the grid rather than appending.)
      > **Known gap:** with WhisperToMe gone the grid has no on-device-ML card, while the hero copy
      > claims "on-device ML" and the résumé lists Snapdragon NPU / QNN / ONNX Runtime GenAI / INT4.
      > `Gemma4-12B-SnapdragonX2Elite` is public and would close it — Cam's call, deliberately open.
      > Three places read the same set and must not drift: `internal/content.featured` (canonical,
      > feeds the SSR grid and gRPC), `client/programs.go termProjects` (terminal `projects`), and
      > `client/vfs.go projectsMD` (`/projects.md`).
- [?] **Extra first-class commands?** `writing`/blog · `now` (current focus) · `guestbook`
      (bidi-stream demo)? Currently out of scope.
- [?] **AI-first copy** — react to the draft About/hero wording (honest "judgment × leverage",
      not self-deprecating, not grandiose). Tune the exact words.
- [~] **Résumé** — canonical content authored from Cam's profile in internal/resume/Data(); shipped
      at `/resume`. Open: Cam to review/correct the exact wording + fill any gaps.
- [?] **Curated i18n languages** — which languages get reviewed bundles? (Suggest: en + a few +
      signature **Jamaican Patois**.)
- [?] **BYOK run-location** — client-direct vs **proxy-over-gRPC** (recommend proxy).
- [x] **Résumé-tailor cost/auth** — decided **owner-gated** (ADMIN_PASSWORD + OPENAI_API_KEY).
      Budget cap + optional public demo tracked as §8 follow-ups.
- [x] ~~**Live-CashFlux level** — L1 host-wasm vs L2 full-backend integration.~~ **Resolved
      2026-07-29: neither.** CashFlux becomes its own service (§13.A).
- [?] **CashFlux split line** (§13.A) — does the portfolio keep the CashFlux admin panels and
      activation-code minting, or do those move into CashFlux's own admin?
- [?] **CashFlux hostname** (§13.A) — `budget.earlcameron.com`, to parallel `feed.earlcameron.com`?
- [?] **Admin-auth method** — **WebAuthn/passkey** (recommended) vs password + TOTP 2FA.
- [?] **CI build source** — git submodule GWC pinned to v4.3.0 (recommended) vs `go mod vendor`.
- [!] **Toolchain** — install `protoc` + `protoc-gen-go`/`-grpc` before the first gRPC service
      (Go plugins via `go install`; protoc via winget/scoop/release). Currently NOT installed.

## 1. Design (in progress)
- [x] Concept: portfolio-as-terminal; two front doors (terminal + standard site).
- [x] Front-door model: **unified hero** — live prompt, type→terminal, scroll→site.
- [x] Chrome: **macOS Terminal**; Palette: **Aubergine** (Ubuntu orange + purple, dark/moody).
- [x] Type: two voices — mono (terminal) / sans (standard site).
- [x] Interactive mockup v1 (`design/mockup.html`, published artifact).
- [ ] Review mockup with Cam; iterate palette/motion/copy.
- [ ] Mobile tap-view: refine chip set + guided flow (currently keyboard hidden on ≤720px).
- [ ] Motion pass: boot sequence, expand-to-fullscreen, streamed output, reduced-motion.
- [ ] Standard-site visual polish (work grid, how-it-works strip, contact).
- [ ] Accessibility: keyboard nav, focus states, screen-reader path via the standard site,
      `prefers-reduced-motion`, contrast check on all three themes.

## 2. Backend — Go gRPC server
- [x] `go mod init` (minimal module; local `replace`s for GWC/bridge added when first imported).
- [ ] Add GWC + GoGRPCBridge deps + local `replace`s (when the client/tunnel land).
- [ ] `proto/site.proto`: ContentService, ContactService, SystemService, AssistantService.
- [ ] Generate stubs (`protoc … --go-grpc_out`); commit generated files.  ⛔ blocked on protoc.
- [~] Server bootstrap: `http.Server` + mux + `/healthz` + SSR placeholder + graceful shutdown
      **DONE & verified** (build/vet clean, `/healthz`→200). Pending: `/socket` tunnel, static wasm.
- [ ] SQLite store: contact messages, AI spend ledger, cached answers.
- [x] Graceful shutdown (Shutdown on SIGINT/SIGTERM).
- [ ] Server-render the **standard site** as real HTML (SEO + no-WASM failsafe) — placeholder so far.

## 3. Frontend — GWC WASM client
- [ ] `client/main.go`: `ui.Run("#app", Root)`; copy `wasm_exec.js` from GOROOT.
- [ ] gRPC dial via `grpctunnel.BuildTunnelConn(ctx, {Target:"/socket", …})` + reconnect/backoff.
- [ ] Boot sequence component (honest connection states; offline → static fallback).
- [ ] Terminal engine: parser, history, Tab/ghost autocomplete, clickable commands,
      forgiving "did you mean?" (Levenshtein).
- [ ] Programs: `about` `projects` `open` `contact` `ask` `stats` `arch` `neofetch`
      `theme` `ls` `resume` `help` `clear`.
- [ ] Full-screen TUI takeovers (projects browser, contact form) with q/Esc to exit.
- [ ] Standard-site components (hero, work grid, how-it-works, contact, footer badge).
- [ ] Theme system: aubergine · light · nord (live `theme` switch, persisted).
- [ ] Responsive + mobile tap-terminal (command chips, big touch input).

## 4. Smart features — nano assistant (guardrails are the feature)
- [ ] `AssistantService.Ask` (server-stream) — tokens stream into the terminal.
- [ ] Server holds the OpenAI key (env); browser never sees it.
- [ ] **Spend ledger** (SQLite): tokens→USD, monthly total, hard stop at **$20**.
- [ ] **Daily sub-cap** (~$0.65/day) so one day can't drain the month.
- [ ] **Per-IP + per-session rate limits** (token bucket) — the real abuse defense.
- [ ] Tight per-request caps: cheapest nano model, `max_tokens ~200`, truncated context, low temp.
- [ ] **On-topic-only, injection-resistant** system prompt (grounded in my content; declines
      off-topic → kills "free ChatGPT" abuse).
- [ ] **Answer cache** for common questions (repeats cost $0).
- [ ] Graceful degradation → deterministic content commands when budget/rate exhausted.
- [ ] Confirm exact nano model + pricing; compute the hard cap from real token cost at build.

## 5. Content
- [ ] Real project data (name, status, blurb, long, tags, links) → ContentService.
- [ ] Final About / hero copy (AI-first positioning).
- [ ] neofetch / arch text (architecture as a selling point).
- [ ] Résumé asset.

## 6. Build, deploy, quality
- [ ] Build script: wasm build + `wasm_exec.js` copy + asset pipeline.
- [ ] Deploy target (DigitalOcean/Ubuntu + Nginx ingress → Go server; TLS; WS upgrade).
- [ ] E2E smoke: boot, a few commands, contact submit, theme switch, mobile view.
- [ ] Lighthouse / perf on the standard-site (SSR) surface.

## 7. i18n & BYOK translation
- [ ] Route all UI copy through GWC typed message bundles (`gwc i18n gen`); en base bundle.
- [ ] Curated locales: pre-generate (nano) + spot-review the chosen languages; per-locale SSR.
- [ ] `ContentService` returns localized content per locale (fallback → en).
- [ ] Locale detect: `Accept-Language` on ingress + `lang` command + persisted preference.
- [ ] `TranslationService.Translate` (server-stream, BYOK) — page rewrites live as it streams.
- [ ] BYOK key handling: request-scoped, **never logged/persisted**; TLS ingress; stated in UI.
- [ ] Abuse-proof: translate **only known source strings**, never arbitrary client text.
- [ ] `(source-hash, lang)` cache → pay-it-forward (first BYOK visitor funds a language, rest free).
- [ ] Live cost meter; label curated=reviewed, community=machine best-effort + re-translate.
- [ ] Standard-site: simple curated-language dropdown (no key). BYOK stays terminal-only.

## 8. Carried-over & new features
### Résumé + agentic tailoring
- [x] Canonical one-page résumé → **print-optimized HTML at `/resume`** (browser Save-as-PDF; no
      static-PDF asset needed); `resume` command + site card link to it. (internal/resume)
- [x] Résumé **tailoring tool** at `/admin/resume` — job posting URL → server fetch → OpenAI →
      tailored variant, rendered as the same print-to-PDF page. (chose HTML Save-as-PDF over a
      server-side Go PDF; document-plane HTTP, owner-gated — not a gRPC service.)
- [x] **No-fabrication guardrail**: system prompt forbids invention; identity/contact fields are
      force-preserved after the model returns; empty result falls back to the canonical résumé.
- [x] Cost/auth: **owner-gated** (behind ADMIN_PASSWORD + OPENAI_API_KEY), per §0 recommendation.
- [ ] Follow-ups: per-request/monthly OpenAI budget cap (Cam's $20/mo); SSRF hardening on the URL
      fetch (block private/loopback targets); optional rate-limited public demo.
### Blog
- [ ] `BlogService`: ListPosts/GetPost (+ admin Create/Update/Delete behind auth).
- [ ] Markdown→HTML + syntax highlight; `blog` list + `read <slug>` TUI; `/blog` on site.
- [ ] RSS feed at `/blog.xml` (document-plane HTTP GET).
### Slack anime RSS
- [x] Go anime tracker: AniList search/track/untrack + release-check → **`/anime.xml`** (Release
      Radar RSS) and **`/anime/qotd.xml`** (daily-prompt RSS); password-gated config at `/admin`.
      (internal/anime, internal/store/anime.go). Optional Slack post + `anime` terminal command TBD.
### Live CashFlux instance — ⚠️ SUPERSEDED 2026-07-29 by §13.A (decouple into its own service)
- [x] ~~**L1**: build CashFlux wasm, serve under `/apps/cashflux`~~ — shipped as `/budget/`, and now
      being undone: the decision flipped from hosting it here to running it as its own service.
- [ ] ~~Keep CashFlux a separate wasm build~~ — see §13.A; the whole `replace` goes away.
- [ ] ~~**L2 (deferred)**: integrate CashFlux Go backend~~ — **rejected.** L2 was "couple the two
      codebases harder"; §13.A goes the other way. Kept here so the reversal is legible.

## 9. Phasing (ship in order — don't build it all at once)
- [ ] **P1 · MVP**: standard site + terminal (about/projects/contact/neofetch/theme) + résumé
      download + contact over gRPC. Deployable, sells Cam on its own.
- [ ] **P2 · Smart + live**: `ask` (nano, $20 guardrails) + live CashFlux L1 + blog + RSS.
- [ ] **P3 · Global + agentic**: i18n curated + BYOK translation + résumé agentic tailoring.
- [ ] **P4 · Extras**: anime RSS, guestbook(?), CashFlux L2 (only if a real need appears).

> Admin lands in **P2** (basic auth + settings/stats/inbox), expanding through P3/P4 as each
> feature it manages ships.

## 10. Guest vs admin
### Auth (security-critical — do it right the first time)
- [ ] Admin auth per §0 decision — **WebAuthn/passkey** (recommend) or password + TOTP.
- [ ] Authed gRPC metadata + interceptor; **TLS ingress mandatory**; short JWT + refresh.
- [ ] Login rate-limit + lockout; no user enumeration; constant-time compare.
- [ ] Session management: logout, **revoke all sessions**.
- [ ] **Append-only audit log** (who/what/when) for every admin mutation.
- [ ] Destructive actions behind explicit confirm; validate all admin input server-side.
### Admin surfaces
- [ ] `login` command → unlocks authed admin programs in the terminal.
- [ ] Full-screen **TUI dashboard** (`admin`): stats · settings · content · inbox · budget panels.
- [ ] **SSR `/admin` fallback** behind same auth (works even if wasm is down).
- [ ] `AdminService`: settings/flags/content/moderation/ops RPCs + `StreamStats`.
### Live-editable (no SSH)
- [ ] `settings` table read live (no restart); runtime **feature flags**.
- [ ] Content editing (projects/about/blog/résumé/i18n/anime) → DB → served immediately.
- [ ] Moderation: contact inbox, community-translation cache.
- [ ] Runtime ops: purge caches, re-seed CashFlux demo, run anime cron, view logs/stats.
### Boundary (stays in deploy pipeline)
- [ ] Document clearly: new **code/schema** and **secret rotation** are NOT web-editable.

## 11. Deploy — Ubuntu + Nginx on DigitalOcean (see documents/DEPLOYMENT.md)
Goal: **one-click install, one action to update, no brain-racking.**
- [x] ~~Single Go binary with `go:embed`ed wasm/assets~~ — **rejected, see DEPLOYMENT.md.** Asset
      paths are CWD-relative and CashFlux alone is ~120 MB; a release is a *directory* swapped by
      symlink instead. The atomic unit was the point, not the file count.
- [x] Build source decided: **droplet builds from source**, three sibling checkouts, because
      `go.mod` pins GWC + CashFlux by relative `replace`. No submodule, no vendor, no CI.
- [x] `deploy/install.sh` — one-shot fresh-droplet setup (packages, user, swap-if-small, Go
      toolchain read from `go.mod`, clones, `.env` + generated secrets, systemd, nginx, ufw,
      first build, certbot).
- [x] `deploy/update.sh` — pull → build → stage → atomic swap + `/healthz` + **auto-rollback**.
- [x] `deploy/rollback.sh` — manual undo (`--list`, by name, or back-one), symlink-only.
- [x] Nginx: WebSocket upgrade + long timeouts on `/socket` **and `/grpc`**; gzip for
      `application/wasm`; `X-Forwarded-Proto` for the budget gate's Secure cookie; TLS via certbot.
- [x] systemd unit (Restart=always, EnvironmentFile=.env, journald, hardened —
      **no `MemoryDenyWriteExecute`**, wazero JITs).
- [x] SQLite in `/opt/earlcameron/data` (outside deploy dir); auto-migrations on boot.
- [ ] **Nightly backup of `data/` off-box** (`sqlite3 .backup` → DO Spaces). Still the one
      irreplaceable thing with no copy.
- [ ] One-time: DNS A-record → droplet IP.
- [ ] Push-to-deploy from GitHub (optional follow-on; `update.sh` is the manual equivalent and
      needs no deploy secret).

## 12. Terminal — live tracker (Cam's asks)
> Native Claude Code TodoWrite isn't available this session — tracking here instead.
- [x] Interactive input (type + Enter runs commands) — GWC hooks.
- [x] Faux portfolio programs: help/about/projects/open/neofetch/links/ls/contact/echo/clear.
- [x] Expand to fullscreen **modal** (green light) + shrink (red light).
- [x] Auto-scroll to newest line (like a real terminal).
- [x] Keep input **focus** after Enter.
- [x] Lock page scroll in the modal + its own scroll; themed scrollbar.
- [x] Verified with a **chromedp** browser-interaction test (real keystrokes/clicks).
- [x] Fix: modal falling off the right edge (removed width:100% vs fixed inset). Verified.
- [x] **Virtual filesystem** cached in localStorage (persists across reloads). Verified.
- [x] **~30 bash commands** (pwd/ls/cd/mkdir/touch/cp/mv/rm/cat/less/head/tail/nano/echo/grep/
      find/sort/uniq/cut/sed/awk/wc/du/df/history/curl/ssh/tar/man) + pipes `|`, `&&`, `>`. Verified.
- [x] **Tab auto-completion** (commands + paths), Ubuntu-style. Verified.
- [ ] Wire portfolio programs to **real gRPC** (ContentService) instead of faux.
- [ ] i18n all terminal + site copy.



## 13. Current effort (2026-07-29) — Cam's four asks
> These supersede the older CashFlux-hosting plan (§8 "Live CashFlux instance" L1/L2 and the
> 2026-07-21 Goal B block below): the decision has flipped from *bundle it in* to *split it out*.

### A. Decouple CashFlux into its own service
**What actually couples the two today** — each of these has to be undone, so scope it against this
list, not against "it's just a mount":
1. `go.mod` **requires** `github.com/monstercameron/CashFlux` with `replace => ../CashFlux`. CashFlux
   still pins **GWC /v4 v4.2.0** while this site is on **/v5**, so one build graph carries two GWC
   majors. Dropping this is the single biggest win of the split.
2. `internal/budget/` (`gate.go`, `gatepage.go`, `serve.go` + tests) — password gate and static
   handler, mounted at `internal/server/server.go` → `mux.Handle("/budget/", …)`.
3. `scripts/build-cashflux.sh` stages the CashFlux wasm into `web/cashflux/` from a **hardcoded
   `C:/Users/mreca/Desktop/CashFlux`**, and `deploy/install.sh` clones a *third sibling checkout*
   onto the droplet to build it there.
4. The **`/grpc` WebSocket tunnel carries CashFlux's `SyncService` inside this server**, with
   `CASHFLUX_SERVER_TOKEN` in the environment and a matching nginx upgrade block.
5. The wasm admin console owns CashFlux's control plane — device pairing, activation-code minting,
   user list, storage stats (`client/admin.go`, `client/adminview.go`, `AdminService` RPCs).
6. CashFlux is **~120 MB of the release directory** (§11) — it dominates deploy size and build time
   for a site that is otherwise a single Go binary plus a wasm bundle.

**Target shape:** CashFlux runs as its own service on its own host — the same shape ArticleFlux
already has at `feed.earlcameron.com` — and this site links out to it instead of hosting it.

- [ ] **Decision (needs Cam, see §0):** does the portfolio keep *any* CashFlux surface — the admin
      panels and invite/activation minting — or do those move into CashFlux's own admin too?
- [ ] **Decision (needs Cam, see §0):** hostname. `budget.earlcameron.com` is the obvious parallel to
      `feed.earlcameron.com`.
- [ ] Stand up the new host: own systemd unit, own nginx server block, own TLS cert, own data dir.
- [ ] Move `SyncService` + the `/grpc` tunnel out of this server; CashFlux serves its own tunnel on
      its own origin (this removes `CASHFLUX_SERVER_TOKEN` from this `.env`).
- [ ] Move device pairing / activation codes / user list / storage stats out of `client/admin.go`,
      `client/adminview.go` and `AdminService` into CashFlux's own admin.
- [ ] Drop the `require` + `replace` from `go.mod` → this module stops carrying GWC v4 entirely.
      **Verify** afterwards that `go mod graph` has exactly one GWC major.
- [ ] Delete `internal/budget/`, `scripts/build-cashflux.sh`, and the `web/cashflux/` staging step;
      remove the third checkout from `deploy/install.sh` and `deploy/update.sh`.
- [ ] Keep `/budget/` alive as a **301 to the new host** — it's linked from the nav, the `~/elsewhere`
      card, and anything already bookmarked. Do not let it 404.
- [ ] Repoint the nav link, the `~/elsewhere` card, and the CashFlux project card at the new host.
- [ ] Deploy hygiene: separate release dirs per service, so a CashFlux build can never break the
      portfolio's atomic symlink swap; `/healthz` checked per service, rollback per service.
- [ ] Docs to update on landing: `documents/DEPLOYMENT.md`, `documents/PROJECT_LAYOUT.md`,
      `README.md` (architecture diagram lists CashFlux as vendored), `AGENTS.md` "own stack" line.

### B. Links + copy — recruiter-grade
The audience is a hiring manager who gives the page ~30 seconds. Every link must land somewhere
that pays that attention back, and every line must earn its space.
- [ ] **Link audit, every destination on the page**: does it resolve, does it dead-end, does it
      demand an account? (Precedent: `feed.earlcameron.com/` root put a visitor on a *sign-in form* —
      fixed 2026-07-29 by pointing at `/home`. Assume the next one is just as silent.)
- [ ] Automate it: a link-check in CI over every internal + external URL the site renders.
- [ ] Hero copy pass — the claim density is high ("AI-native", "ship at a pace that used to take a
      team"); make each claim cash out in something on the page.
- [ ] Project blurbs: lead with the hardest thing solved and quantify it, not the feature list.
- [ ] Status chips — audit whether `prototype` / `research` read as range or as *unfinished*.
- [ ] **Close the on-device-ML gap** (§0): hero and résumé both claim it, no card backs it.
- [ ] Terminal first-contact copy: the welcome line and `help` are the first things typed — they
      should sell, not just list.
- [ ] `neofetch` / `arch` text — architecture as the pitch.
- [ ] CTA hierarchy: résumé, contact, and the two live apps should be findable without scrolling.
- [ ] Meta/OG title, description and preview image — the copy that renders when the link is *shared*.

### C. Terminal — fully tested and feature-packed
**Current state:** 9 files in `client/`, **zero Go tests**, and `e2e/` holds only CashFlux
`auth-flows` + `sync-flows` — nothing drives the terminal. Programs read from hardcoded slices.
- [ ] Go unit tests for the pure logic — command parser, pipes/`&&`/`>`, the vfs, tab completion,
      history. This is ordinary Go; it does not need a browser.
- [ ] Playwright e2e: boot, run every program, tab-complete, history, fullscreen/shrink, vfs
      persistence across reload, mobile viewport. **Use a `keyboard.type` delay** — headless renders
      this page at ~0.4fps and full-speed typing silently drops keystrokes (see DEVLOG 2026-07-29).
- [ ] Golden-output tests for program rendering, so copy changes are diffs and not surprises.
- [ ] Wire programs to **real gRPC `ContentService`** instead of faux slices (also §12) — today
      `client/programs.go termProjects` must be hand-synced with `internal/content.featured`.
- [ ] Missing programs: `ask` (§4), `stats`, `arch`, `theme`, `login` (§10), `blog` (§8), plus the
      `writing` / `now` / `guestbook` decisions still open in §0.
- [ ] Full-screen TUI takeovers — projects browser and contact form (§3).
- [ ] a11y: keyboard-only path, visible focus, and a screen-reader story for a terminal UI.

### D. CSS refine, responsive, and a real mobile mode
**Current state:** `internal/theme` is a single `theme.go`; `internal/site/site.go` has **5 `Md()`
breakpoint uses and 19 `css.Raw` escapes**; `client/*.go` has **zero breakpoints** — the terminal and
the admin console do not respond to width at all.
- [ ] Token audit: grow `internal/theme` into a full scale (spacing, type, radius, elevation) so the
      `css.Raw` escape count drops instead of creeping. Every escape should be a *documented* gap in
      `css/u`, not a shortcut.
- [ ] Per-section responsive pass at 390 / 768 / 1024 / 1440: nav, hero, work grid, how-it-works,
      elsewhere, anime, contact, footer.
- [ ] **Terminal responsive** — it has no width handling today; it is the centrepiece of the page.
- [ ] **Mobile mode** (upgrades §1's "mobile tap-view"): command chips instead of typing, large touch
      input, sensible default view without a hardware keyboard, and a decision on whether the
      terminal opens fullscreen on small screens.
- [ ] Admin console responsive (`client/admin.go`, `client/adminview.go` — 0 breakpoints).
- [ ] Quality floor, enforced not assumed: `prefers-reduced-motion`, `:focus-visible`, tap targets.
- [ ] Visual-regression screenshots at three widths in CI, so responsive work stays fixed.

## 14. Recruiter refinement (2026-07-29) — translate ambition into evidence
> Thesis, and the rule for every ticket below: **do not remove ambition to make the site
> recruiter-friendly — translate the ambition into evidence.** The unusual projects are the
> advantage. The missing layer is measurable execution, ownership, architectural explanation, and
> stated professional relevance. Supersedes the looser §13.B copy pass, which folds in here.

### ⚠️ Data reality check — done first, because it kills one of the planned sections
Measured on the CashFlux repo 2026-07-29, before designing anything around it:
- **Merged PRs: `0`.** Every one of **2,745 commits** went straight to `main`. The proposed
  "cumulative merged PRs over time" chart **cannot be built** — there is no PR data to plot.
- **Commits per week: 1179 · 626 · 227 · 124 · 408 · 181** (W25–W30 2026). The curve is
  front-loaded and *declining*. Plotted as-is it tells a "big burst, then tapering off" story,
  which is the opposite of sustained velocity. It also puts ~67 commits/day on screen, which
  invites the question of how much was agent-generated rather than answering it.
- **What is real and defensible instead:** **26 dated releases** in `CHANGELOG.md` (1.0.2 on
  2026-07-07 → 1.5.0 on 2026-07-24), on top of 7 git tags, each release naming shipped capability.
  Also measured: **221 internal packages · 50 routes · 6,102 non-test Go files · 3,669 test files**.
- [ ] **Decision (needs Cam):** chart **cumulative releases with annotated feature milestones**
      (recommended — real data, rising curve, maps directly to "velocity = shipped capability"), or
      drop the chart and lead with the static metric strip alone.

### A. Information architecture — featured case studies vs Labs
- [x] Four featured projects, promoted to case-study cards rather than equal portfolio tiles:
      **CashFlux** (product engineering + sustained execution), **ArticleFlux** (AI orchestration +
      media pipeline), **GoWebComponents** (framework/platform engineering), **WASIBrowser**
      (systems vision + architectural ambition).
- [x] Everything else demoted to a visually quieter **Labs** section — the visitor must be able to
      tell production-grade work from major bets from experiments from language research.
      ⚠️ **This drops GoGRPCBridge out of the featured four**, which contradicts the 2026-07-28
      instruction to bill it fourth. Flagged to Cam; one-line swap if he wants it back.
- [x] Homepage order: hero → proof strip → featured four → CashFlux velocity → capabilities
      ("how this site works") → Labs → elsewhere → personal (anime) → contact.
- [ ] **Still missing from that order: professional experience.** The site has no experience section
      at all — UKG is one clause in the hero and the rest lives only in `/resume`. A recruiter
      scanning the page cannot see the work history without opening another document.

### B. Hero — recruiter-efficient first screen
Within the first viewport: role + specialization, seniority signal, core competencies, résumé,
contact, and one immediate proof point.
- [x] Reposition from "AI-native systems engineer" — recruiters read "systems engineer" narrowly as
      OS/networking/embedded/infra-admin — to **"Senior software engineer building AI-native
      products, developer platforms, and unconventional systems."** Keep "systems" in support text.
- [x] Name the employer and the work: currently building agentic systems at UKG.
- [x] **Proof strip**: "Creator of CashFlux, ArticleFlux, GoWebComponents and WASIBrowser" — faster
      to process than inferring identity from a nine-card grid.
- [x] Primary actions visible without scrolling: View work · Résumé · GitHub · Contact.

### C. The terminal — a reward, not a gate
- [x] Frame the intent so it does not read as the site's navigation.
      ⚠️ **The critique's own framing was wrong here and was not adopted:** the terminal is not
      something to do *while the wasm loads* — on the public site the terminal **is** the wasm
      (`client/main.go` mounts only `Terminal` into `#term-root`; the rest of the page is
      server-rendered Go). It cannot exist before the binary is up. The shipped copy says so, which
      is also the stronger claim: "Everything above is server-rendered Go. The terminal below *is*
      the WebAssembly."
- [x] Show suggested commands on the surface instead of expecting visitors to guess.
- [ ] Revisit once §13.D lands: on mobile the terminal should not be the first thing between a
      recruiter and the content.

### D. CashFlux as primary velocity evidence
- [x] "Built at high velocity" metric strip — 26 releases · 221 packages · 50 routes · 2,998 tests ·
      six weeks. Shipped 2026-07-29 and confirmed by review as stronger than a bare PR chart.
- [ ] **Build timeline — now OPTIONAL supporting evidence, not the proof.** The five metrics already
      carry speed; a timeline would only add *progressive delivery* (which capability landed when).
      Put it behind a "See the build timeline" disclosure or inside the case-study page — it must not
      dominate the homepage. Still no PR data (zero merged PRs), so it plots the 26 CHANGELOG
      releases; each milestone date must be verified against `CHANGELOG.md` before publishing.
- [ ] Metric strip — releases, active development weeks, feature modules, routes, tests, languages.
      **No lines-of-code headline**: LOC rewards duplication and verbosity and recruiters know it.
- [ ] Milestone annotations: transaction import · budgets · goals · reports · household ownership ·
      financial to-do system · local-first assistant · wasm deployment. Verify each against
      `CHANGELOG.md` before publishing a date next to it.

### E. CashFlux link — own instance, framed
Linking the real instance beats a sanitized mockup: it proves he uses what he builds.
- [ ] **Blocked — needs Cam:** does a **guest/demo mode with a separate dataset** exist? The framing
      copy ("my actual instance; guest mode loads a demo dataset; my personal financial data never
      enters the guest session") is only publishable if it is *true*. Today `/budget/` is behind a
      password gate. **Do not ship the claim before the mechanism.**
- [ ] Two distinct CTAs once it exists: **Explore guest demo** and **View engineering case study**.
- [ ] Note the interaction with §13.A: CashFlux is moving to its own host, so this copy should land
      on the new host, not `/budget/`.

### F. The CashFlux case study — the highest-leverage single item
> A homepage creates interest; a case study creates confidence — and it is the thing a hiring
> manager can circulate internally.
- [ ] One page, in this order: product overview · problem and motivation · major capabilities ·
      architecture diagram · development timeline and velocity · hard engineering problems · key
      decisions and tradeoffs · screenshots or live demo · what to improve next · repo and live app.
- [ ] **Needs Cam's own words** for: why he built it, what he personally owned, and the two or three
      genuinely hard problems. These are judgment and authorship claims — they must not be
      synthesised. Everything else (capabilities, architecture, metrics, timeline) is sourceable
      from the repo.
- [ ] Then repeat the shape for ArticleFlux and GoWebComponents, one at a time.

### H. Résumé — accomplishment-oriented, not task-oriented (2026-07-29 review)
- [x] Title aligned with the site's positioning.
- [ ] **Needs Cam's numbers.** The experience bullets read as tasks: "Built a Unified Search
      micro-frontend", "Prototyped Bryte ChangeJob". Each needs scope, adoption, performance or
      delivery outcome — users served, latency moved, teams onboarded, ship date held. These are
      facts only Cam has; they must not be estimated or inferred. Until they land, the homepage
      sells him better than the résumé does.
- [ ] Note for whoever edits it: `/resume` prefers the stored `active_resume` setting over
      `resume.Data()`, so a code edit silently does nothing while an override is applied.

### G. Per-featured-project answers (each must answer five things fast)
- [ ] 1 · What is it, in one jargon-free sentence.
- [ ] 2 · Why he built it — the product-judgment signal.
- [ ] 3 · What he personally owned — the thing hiring managers actually need.
- [ ] 4 · What was technically hard — two or three problems, not twelve technologies.
- [ ] 5 · The evidence — metrics, demo, benchmark, or architecture diagram.

## Goal (2026-07-21): RSS control panels + CashFlux managed service
Tracking (Claude Code TodoWrite tool is not exposed in this session; tracked here instead).

### A. RSS features + in-app control panels  [subagent: internal/rss]
- [x] Configurable QOTD prompts — full CRUD in-app (add/list/delete), store-backed (`qotd_prompts`)
- [x] Spec-compliant RSS 2.0 (atom:self, RFC1123Z dates, guid, lastBuildDate) — /anime.xml, /anime/qotd.xml
- [x] Anime news fetch (Anime News Network RSS) → discussion/debate composer
- [x] Slack integration: post news + QOTD to a configurable company Slack channel (webhook), toggle + "post now"
- [x] RSS/Slack control panel in the wasm admin (gRPC): QOTD CRUD, Slack config, post-now, feed links
- [x] e2e + screenshot the control panel; adversarial review → refine (5 findings fixed: Slack escape, guid, seed race, len cap, redirect host)

### B. CashFlux as a managed budgeting service  [subagent: internal/budget]
- [x] Build CashFlux wasm frontend → serve from this server at /budget (internal/budget handler)
- [x] Server integration so it runs as a managed/hosted app (serve + assess backend sync)
- [x] Link from site/admin; e2e + screenshot; adversarial review → refine

### Orchestration
- [x] 2 parallel Sonnet subagents (disjoint new packages, no shared-file collision)
- [x] Wire each package into proto/AdminService + wasm-admin views + server routes (me)
- [x] Adversarial Sonnet review per feature; refine until clean
- [~] e2e + screenshot both (done); commit
