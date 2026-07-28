# earlcameron.com — terminal portfolio (mid-2026)

A personal portfolio that **is** a working terminal. Every command a visitor types is a
real program that fetches its data over gRPC. The whole site is the proof of work: it's
rendered by a Go→WASM UI framework I wrote, talking to a Go gRPC backend over a
WebSocket bridge I built. No REST, no JavaScript framework.

> **Positioning:** I'm an **AI-native systems engineer** — I pair real systems judgment
> with LLMs in the loop and ship ambitious things fast. This site demonstrates that, it
> doesn't just claim it.

## The two front doors

The site serves two audiences from one page (SSR on the ingress request):

1. **Standard site** — a clean, conventional portfolio (identity, featured work, contact).
   Server-rendered HTML, touch-friendly, SEO-visible, zero runtime comms. This is also the
   **no-WASM failsafe** and the primary **mobile** experience.
2. **Friendly terminal** — the enhancement layer for the curious, where the juicy details
   live. Boots WASM, opens the gRPC-over-WebSocket channel, runs real programs.

A live prompt sits in the hero: **type → the terminal grows to fullscreen; scroll → the
standard site.** No gate, no forced choice. On mobile the terminal is **tap-driven**
(command chips), because typing shell commands on a phone is miserable.

## Architecture

```
┌───────────────┐  browser: GoWebComponents (Go→WASM), zero JS framework
│  your browser │
└───────┬───────┘
        │  gRPC-over-WebSocket  (one same-origin socket, no proxy)
┌───────┴───────┐
│  GoGRPCBridge │  my WS↔gRPC tunnel  (grpctunnel)
└───────┬───────┘
┌───────┴───────┐
│  Go gRPC srv  │  content · contact · live status · ask(nano)
└───────────────┘
```

**The only HTTP is the ingress** (first page load: HTML, wasm, fonts/css). Everything after
is gRPC over the WebSocket tunnel. Server→OpenAI (for `ask`) is backend egress, not part of
the browser↔server channel — it does not violate the no-HTTP rule.

Two planes, to be precise: the **data plane** (browser↔server app comms) is gRPC/WS only; the
**document plane** (published GET resources — SSR pages, wasm, **RSS feeds**, the résumé PDF,
the CashFlux iframe, sitemap/robots) is served over HTTP GET, as it must be (feed readers can't
speak gRPC). The rule bans an ad-hoc REST *API*, not published documents.

## Stack

| Layer | Choice |
|---|---|
| Frontend | **GoWebComponents v5.0.1** (Go→WASM, React-style, my framework) |
| Transport | **GoGRPCBridge** `grpctunnel` — gRPC over WebSocket, same-origin `/socket` |
| Backend | **Go gRPC server** — one `http.Server` serves both the tunnel and the wasm/SSR assets |
| Data | SQLite (contact messages, AI spend ledger, cached answers) |
| AI | rate-limited OpenAI **nano** model, hard-capped at **$20/mo** (see DESIGN) |
| i18n | GWC typed message bundles + **BYOK** real-time nano translation (visitor-funded) |
| Apps | live **CashFlux** instance (its own wasm, hosted, local-first) via `budget`/`cashflux` |
| Build | `GOOS=js GOARCH=wasm` via the `gwc` tool; `protoc` for the proto contract |

## Design language

macOS Terminal chrome (traffic lights, SF Mono, blur) wrapping an **Ubuntu-souled
"Aubergine" palette** — deep aubergine ground, Ubuntu orange accent, purple secondary; dark
and moody. `theme` is a real command (aubergine · light · nord). Two typographic voices:
**mono = the machine (terminal), sans = Cam (standard site).** See
[`documents/DESIGN.md`](documents/DESIGN.md).

## gRPC services

- `ContentService` — `ListProjects`, `GetProject`, `GetAbout` (stream). Powers `projects`,
  `open`, and the standard-site work grid from one source, so they never drift.
- `ContactService` — `SendMessage` (unary). Stored server-side.
- `SystemService` — `StreamStatus` (server-stream). Live uptime, deployed commit, visitors
  online. Powers `stats`/`top` and the footer badge.
- `AssistantService` — `Ask` (server-stream). The rate-limited nano assistant. Terminal-only.
- `TranslationService` — `Translate` (server-stream, **BYOK**). Real-time page translation on
  the visitor's own key; `(source-hash, lang)` cache pays it forward. Terminal-only.
- `BlogService` — `ListPosts`/`GetPost` (+ authed admin CRUD). RSS at `/blog.xml`.
- `ResumeService` — `Tailor` (job posting → tailored one-page **PDF**, server-side Go, Blob
  download). No-fabrication guardrail. Canonical résumé is a static PDF.
- `AdminService` — authed settings/flags/content/moderation/ops RPCs + `StreamStats`. Every
  mutation writes an append-only audit log.

## Admin & security

Two roles from one codebase: **guest** (default, unauthenticated) and **admin** (Cam). Admin
is on-theme — `login` unlocks admin programs and a full-screen **TUI dashboard** (stats,
settings, content, inbox, budget), with a plain **SSR `/admin` fallback** behind the same auth
in case wasm is down. Goal: **operate the site without SSH.**

Live-editable in-browser: settings, runtime feature flags, all content (projects/about/blog/
résumé/i18n/anime), moderation, and runtime ops (purge caches, re-seed demo, run cron). Still
requires the deploy pipeline: **new code/schema and secret rotation** — those never go through
a browser.

Security (this is the real attack surface): **WebAuthn/passkey** auth (recommended) or password
+ TOTP — never a lone env password; TLS mandatory; login rate-limit + lockout; short-lived JWT;
revoke-all-sessions; **append-only audit log**; destructive actions behind explicit confirm;
all admin input validated server-side.

## Repo layout (planned)

```
proto/            # .proto contract + generated *_grpc.pb.go / *.pb.go
client/           # GWC WASM app  (ui.Run("#app", Root))
  main.go
  app/            # terminal + standard-site components
server/           # Go gRPC server + tunnel handler + static/SSR
  cmd/server/
internal/         # content, store (sqlite), assistant (nano + budget), telemetry
documents/        # DESIGN.md and other design/process notes
design/           # mockup.html — interactive design prototype (static, not GWC)
third_party/
  GoGRPCBridge/   # vendored (git submodule) — replace-directive target
```

## Build & run

> Prereqs: Go ≥1.26, `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc`. GWC and CashFlux are
> pinned to **sibling local checkouts** via `replace` directives, so `../GoWebComponents` and
> `../CashFlux` must exist next to this repo. GWC must be on **v5.0.1 or newer** — earlier v5
> fails to compile here (`css/u` shadowed `html/shorthand`, and five files dot-import both).

```bash
# 1. deps — GWC and CashFlux are wired via local replace directives in go.mod:
#    replace github.com/monstercameron/GoWebComponents/v5 => ../GoWebComponents
#    replace github.com/monstercameron/CashFlux           => ../CashFlux

# 2. generate the proto contract
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/site.proto

# 3. build the client wasm (GOOS=js GOARCH=wasm)  +  copy wasm_exec.js
#    from $(go env GOROOT)/lib/wasm/wasm_exec.js

# 4. run the server (serves tunnel at /socket + static/SSR at /)
go run ./server/cmd/server        # default 127.0.0.1:8095
```

Design prototype (no toolchain needed): open [`design/mockup.html`](design/mockup.html) in a
browser, or view the published version. It is a **static mock** — the shipped site is GWC.

## Deploy

Ubuntu + Nginx on DigitalOcean, **one command to install and one to update**:

```bash
curl -fsSL https://raw.githubusercontent.com/monstercameron/PersonalWebsiteMid2026/main/deploy/install.sh | sudo bash
sudo deploy/update.sh      # pull → build → atomic swap → health check → auto-rollback
sudo deploy/rollback.sh    # undo a deploy that came up healthy but was wrong
```

The droplet **builds from source** (three sibling checkouts, because `go.mod` pins GWC and
CashFlux by relative `replace`), a release is a *directory* of binary + assets swapped by
symlink, and SQLite lives outside every release. Nginx terminates TLS and upgrades the
WebSocket tunnels; systemd runs it hardened, with journald logs. See
[`documents/DEPLOYMENT.md`](documents/DEPLOYMENT.md).

## Status

**Build phase — P1 foundation.** Ingress server runs (`/healthz` → 200, SSR placeholder);
`go build`/`go vet` clean. Next: install `protoc`, then the gRPC contract + first services.
Live backlog + progress in [`TODOS.md`](TODOS.md).
