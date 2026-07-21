# DEVLOG — earlcameron.com

Newest first. Dated narrative of the build: what, why, what broke, what's next. Log failures too.

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
