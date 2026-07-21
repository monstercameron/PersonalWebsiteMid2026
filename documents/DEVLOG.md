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
