# PROJECT_LAYOUT — planned Go structure (not yet scaffolded)

> **Status: PLAN.** We are in the planning phase; no `go.mod` or code exists yet. This
> documents the *proper* Go layout we'll create when we move to build (P1). It's the
> blueprint, not the build.

## Directory tree

```
earlcameron/                     module github.com/monstercameron/earlcameron  (go 1.26)
├── go.mod                       # local replaces → ../GoWebComponents + its GoGRPCBridge
├── proto/
│   └── site.proto               # gRPC contract (generated *.pb.go committed alongside)
├── cmd/
│   └── server/main.go           # ingress entrypoint (one listener)
├── internal/                    # private packages (not importable outside the module)
│   ├── server/                  # http mux: /socket tunnel + document plane; graceful stop
│   ├── config/                  # env config + runtime feature flags (live-editable)
│   ├── store/                   # SQLite (pure-Go modernc.org/sqlite): content, inbox,
│   │                            #   blog, AI ledger, translation cache, settings, audit
│   ├── content/                 # ContentService — projects/about, localized
│   ├── contact/                 # ContactService — inbox
│   ├── assistant/               # `ask` nano model + hard budget ledger
│   ├── translation/             # BYOK translation (known-strings only) + pay-forward cache
│   ├── blog/                    # BlogService + RSS
│   ├── resume/                  # ResumeService — tailoring + server-side Go PDF
│   ├── anime/                   # anime cron → RSS
│   ├── admin/                   # owner-only auth/authz/audit (server-enforced)
│   ├── system/                  # SystemService — live process facts (`stats`/`uptime`/Ping)
│   │                            #   + the terminal command-name allowlist, shared with the client
│   ├── notes/                   # the ~/notes briefing text, shared by the terminal and the SSR
│   │                            #   page — NOT in client/, which is wasm-only and so invisible
│   │                            #   to crawlers, unfurlers and any failed boot
│   └── telemetry/               # live stats stream
├── client/                      # GoWebComponents frontend (GOOS=js GOARCH=wasm)
│   ├── main.go                  # mounts the terminal, or the admin console under /admin
│   ├── terminal.go              # the component: hooks, effects, key bindings
│   ├── termctl.go               # termCtl — streaming output + command dispatch
│   ├── lineedit.go              # history navigation, readline kills, did-you-mean (pure logic)
│   ├── shell.go / shellutil.go  # the faux shell: pipes, &&, >, ~30 commands, completion
│   ├── vfs.go                   # localStorage-backed virtual filesystem (+ seed repair)
│   ├── programs.go              # portfolio programs as data (progRow) → styled nodes OR text
│   ├── palette.go               # terminal-only `theme`, via --t-* custom properties
│   ├── system.go                # measured boot metrics, `stats`, `uptime`
│   ├── bench.go / tour.go / eggs.go / deeplink.go / contactform.go / telemetry.go
│   └── admin.go / adminview.go  # the owner console
├── web/
│   ├── static/                  # built app.wasm(.gz), wasm_exec.js, css, fonts (gitignored builds)
│   └── data/                    # runtime sqlite (gitignored)
├── documents/                   # DESIGN.md, PROJECT_LAYOUT.md, ...
└── design/                      # mockup.html (static prototype)
```

## Conventions
- **Standard Go layout**: `cmd/` entrypoints, `internal/` private packages, one package per
  responsibility, no `pkg/` until something is genuinely reusable across repos.
- **Dependency direction**: `cmd → internal/server → internal/<service> → internal/store`.
  No cycles; services depend on `store` interfaces, not concretions.
- **Two build targets**: native (server, `cmd` + `internal`) and `js/wasm` (`client/**`,
  build-tagged `//go:build js && wasm`). They share `proto` types only.
- **Secrets**: env only (`OPENAI_API_KEY`, signing keys, `ADMIN_OWNER_IDS`). Never in the DB,
  never web-editable, never sent to the client.

## Two planes (the no-HTTP-except-ingress rule, precisely)
- **Data plane** (browser ↔ server app comms): **gRPC over WebSocket only** (`/socket`, via
  GoGRPCBridge). No ad-hoc REST API.
- **Document plane** (published GET resources): SSR HTML pages, wasm/assets, **RSS feeds**,
  the résumé PDF, the CashFlux iframe, sitemap/robots. Served over HTTP GET — as they must be.

## Tool routing — local vs backend-routed programs
Every terminal program declares its kind:
- **Local** (default): runs entirely in WASM, no round-trip. E.g. `help`, `ls`, `clear`,
  `theme`, `neofetch`, `matrix`, client-side nav.
- **Backend-routed**: calls a gRPC service over the tunnel. Use this whenever the browser
  sandbox would otherwise bite — **CORS, secrets, cross-origin fetches, server capabilities**.
  E.g. `ask` (secret key + budget), `translate` (BYOK proxy dodges OpenAI CORS), `contact`,
  `stats`, `blog`/`read`, `resume tailor` (PDF gen), all `admin` programs, `anime` (RSS source).
- Rule: **default local; route through the backend the moment a browser limitation or a secret
  is involved.** Never expose a key or bypass CORS from the client.

## Build recipe (for when we start — verified against the ai-chat-wizard example)
1. `go mod init github.com/monstercameron/earlcameron`; add the two local `replace`s.
2. `protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/site.proto`
3. Build client: `GOOS=js GOARCH=wasm` (via the `gwc` tool or raw `go build`); copy
   `wasm_exec.js` from `$(go env GOROOT)/lib/wasm/`.
4. Server mounts `grpctunnel.BuildBridgeHandler(grpcSrv, cfg)` at `/socket`; serves static +
   SSR at `/`. Default `127.0.0.1:8095`.

## Proto contract (planned — `proto/site.proto`)

```proto
syntax = "proto3";
package site.v1;
option go_package = "github.com/monstercameron/earlcameron/proto;sitepb";

service ContentService {          // one source of truth for terminal + standard site
  rpc GetAbout(LocaleRequest) returns (About);
  rpc ListProjects(LocaleRequest) returns (ProjectList);
  rpc GetProject(ProjectRequest) returns (Project);
}
service ContactService { rpc SendMessage(ContactMessage) returns (Ack); }
service SystemService  { rpc StreamStatus(Empty) returns (stream Status); }   // stats/top + footer
service AssistantService { rpc Ask(AskRequest) returns (stream Token); }      // nano, $20 cap
service TranslationService { rpc Translate(TranslateRequest) returns (stream TranslatedChunk); } // BYOK, known-strings only
service BlogService { rpc ListPosts(LocaleRequest) returns (PostList); rpc GetPost(PostRequest) returns (Post); }
service ResumeService { rpc Tailor(TailorRequest) returns (stream PdfChunk); } // no-fabrication
service AdminService {          // OWNER-ONLY, every rpc authorized server-side
  rpc GetSettings(Empty) returns (Settings);
  rpc UpdateSettings(Settings) returns (Ack);
  rpc SetFlag(Flag) returns (Ack);
  rpc StreamStats(Empty) returns (stream AdminStats);
  rpc ListInbox(Empty) returns (InboxList);
  rpc UpsertProject(Project) returns (Ack);
  rpc DeletePost(PostRequest) returns (Ack);   // destructive → confirm token
  rpc PurgeCache(CacheKey) returns (Ack);
}
// messages: Project{id,name,status,glyph,blurb,long,repo,demo,tags[]}, ContactMessage,
// Status{uptime,commit,online,ai_spent}, AskRequest/Token, TranslateRequest{byok_key,
// target_lang,page}/TranslatedChunk, Post, TailorRequest/PdfChunk, Settings, AdminStats, ...
```
