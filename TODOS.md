# TODOS — earlcameron.com (mid-2026)

Status legend: `[ ]` todo · `[~]` in progress · `[x]` done · `[?]` needs a decision from Cam

## 0. Decisions still open (need Cam)
- [x] **Featured projects** — 9 chosen & wired into the mockup (count is dynamic, N):
      GoWebComponents, CashFlux, WASIBrowser, SemanticScript, SemanticAssembly, WhisperToMe,
      Vulkan Path Tracer, SemanticPortrait, GoGRPCBridge. (Gemma-4 removed per Cam. Adjust anytime.)
- [?] **Extra first-class commands?** `writing`/blog · `now` (current focus) · `guestbook`
      (bidi-stream demo)? Currently out of scope.
- [?] **AI-first copy** — react to the draft About/hero wording (honest "judgment × leverage",
      not self-deprecating, not grandiose). Tune the exact words.
- [?] **Résumé** — provide the real experience/content to author the canonical one-page PDF.
- [?] **Curated i18n languages** — which languages get reviewed bundles? (Suggest: en + a few +
      signature **Jamaican Patois**.)
- [?] **BYOK run-location** — client-direct vs **proxy-over-gRPC** (recommend proxy).
- [?] **Résumé-tailor cost/auth** — owner-gated (recommended) · public on $20 budget · BYOK.
- [?] **Live-CashFlux level** — L1 host-wasm (recommended first) vs L2 full-backend integration.
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
- [ ] Canonical one-page résumé → static PDF; `resume` / download button ($0, no AI).
- [ ] `ResumeService.Tailor` — job posting → tailored variant → **server-side Go PDF** → Blob download.
- [ ] Fixed Go PDF template (guarantees one page); agent fills slots only.
- [ ] **No-fabrication guardrail**: reorder/emphasize/rephrase existing facts ONLY.
- [ ] Cost/auth per §0 decision (recommend owner-gated + optional rate-limited public demo).
### Blog
- [ ] `BlogService`: ListPosts/GetPost (+ admin Create/Update/Delete behind auth).
- [ ] Markdown→HTML + syntax highlight; `blog` list + `read <slug>` TUI; `/blog` on site.
- [ ] RSS feed at `/blog.xml` (document-plane HTTP GET).
### Slack anime RSS
- [ ] Go cron: anime-release check → RSS feed (`/anime.xml`); optional Slack post; `anime` command.
### Live CashFlux instance
- [ ] **L1**: build CashFlux wasm, serve under `/apps/cashflux`, launch via `budget`/`cashflux`
      command + project "Launch live demo" (iframe overlay); seed demo data; lazy-load on launch.
- [ ] Keep CashFlux a separate wasm build (its own pinned GWC); mind wasm build-race/deploy hygiene.
- [ ] **L2 (deferred)**: integrate CashFlux Go backend for sync/cloud — heavy, couples two codebases.

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
Goal: **one-click install, push-to-deploy from GitHub, no brain-racking.**
- [ ] Single Go binary with `go:embed`ed wasm/assets — droplet needs no toolchain.
- [ ] CI build source: git submodule GWC (pinned v4.3.0, recursive) or `go mod vendor`.
- [ ] GitHub Actions: on push→main → build wasm+binary → SSH atomic-swap + restart.
- [ ] `deploy/install.sh` — one-shot fresh-droplet setup (user, systemd, nginx, certbot, ufw, .env).
- [ ] `deploy/update.sh` — atomic swap + `/healthz` check + **auto-rollback** (manual fallback).
- [ ] Nginx: WebSocket upgrade + long timeouts on `/socket`; TLS via certbot auto-renew.
- [ ] systemd unit (Restart=always, EnvironmentFile=.env, journald).
- [ ] SQLite in `/opt/earlcameron/data` (outside deploy dir) + nightly backup; auto-migrations on boot.
- [ ] One-time: DNS A-record → droplet IP; one GitHub SSH deploy secret.

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


