# Changelog

All notable changes to earlcameron.com. Format: [Keep a Changelog](https://keepachangelog.com);
Semantic Versioning once released.

## [Unreleased]
### Added
- **Copy-to-clipboard for the minted CashFlux invite code.** The freshly-minted code callout in the
  admin "cashflux" tab now has a one-click copy button (`Copy code` → `Copied ✓`), backed by a new
  `copyToClipboard` helper in `client/grpc.go` (async Clipboard API, best-effort/fire-and-forget).
- **Admin console: CashFlux client management tab.** New "cashflux" tab in `/admin` lists enrolled
  phone/SMS clients and mints fresh, single-use, 15-minute invite codes on demand — no more editing
  `CASHFLUX_SERVER_SETUP_CODE` and restarting the server to add one person. Three new owner-only
  `AdminService` RPCs (`ListCashFluxClients`/`MintCashFluxInviteCode`/`ListCashFluxInviteCodes`),
  backed by a new `CashFluxAdmin` interface in `internal/admin` (fake-able in tests) wrapping
  CashFlux's new `pkg/embed.Admin` handle. (CashFlux-side: admin-mintable invite codes work
  alongside the existing static setup code — see CashFlux's own CHANGELOG.) Verified end-to-end in
  an isolated scratch environment (fresh owner setup → login → mint → code appears in the list) and
  by a mandatory adversarial review, which found and this shipped a fix for: the admin tab silently
  showed "no clients"/"no codes" on a real backend error indistinguishable from a genuinely empty,
  healthy deployment — now surfaces a flash message for any error that isn't the expected
  "not configured" or "session expired."

### Changed
- **Embedded CashFlux: per-person accounts instead of one shared token.** Switched from
  `cashfluxembed.NewSyncBridge` to `cashfluxembed.NewSyncAndAuthBridge`, which adds CashFlux's
  `AuthService` (phone/SMS sign-in) and `BlobService` alongside `SyncService` — everyone syncing
  against this server now has their own account instead of sharing one indistinguishable static
  token. New-account creation is gated by `CASHFLUX_SERVER_SETUP_CODE` (set on the server process,
  read directly by CashFlux — nothing to configure here): a returning, already-verified phone
  number is never asked for it again. `/grpc` and `/v1/version` stay outside `budgetGate` on
  purpose — access control now lives in AuthService's own bearer-token + setup-code gate, not in
  keeping the path secret. (CashFlux-side: `Config.SetupCode`, migration v11, and a
  `phoneOnlyAuthServer` decorator disabling username/password enrollment entirely for this
  embedding — see CashFlux's own CHANGELOG for the full mechanism and a critical bypass an
  adversarial review caught and closed before this shipped.)
- **QOTD feed: durable post history + Slack decoupled.** Every generated anime discussion post is
  now recorded in a `qotd_posts` table (one row per publish, past days kept forever); the
  `/anime/qotd.xml` feed serves the newest 30 from that history instead of a single overwritten
  settings blob (a one-time transactional startup migration moves the legacy blob over). Publishing
  now records to RSS **first** and treats Slack as best-effort delivery: a missing webhook or a
  failed Slack POST no longer blocks the feed post, and the outcome is reported in the manual Ack
  and the scheduler's log line. RSS format: channel `<link>` now points at the site (per spec, not
  the feed itself), items link to `/#anime`, and GUIDs derive from the immutable DB row id (unique
  even for same-minute publishes). Feed content is never generated on request — the handler only
  renders stored rows. New `openai.BaseURL` test seam + coverage for the record-first/Slack-optional
  contract.

### Fixed
- **Mobile horizontal overflow on the home page.** The wasm terminal's nowrap rows and text input
  set a flex `min-width:auto` floor that widened the page column past narrow viewports, clipping the
  hero and cards. `min-width:0` on the centered column, terminal frame, and input; `overflow-x:auto`
  on the terminal body so long lines scroll inside the terminal. Verified at a real 390px viewport
  (`document.scrollWidth == 390`).
- Stale GoGRPCBridge status on its project card: `v0.0.19` → `v1.0.0`.

### Changed
- **Recruiter-readability pass on the home page:** meta description + Open Graph/twitter tags in the
  SSR head (link previews + search snippet); top nav drops the owner-only `/admin` link (footer entry
  stays) and renames "budget" → "cashflux"; a secondary outlined **Read the résumé** CTA next to the
  terminal button; a one-line role-fit sentence in the contact section; the GoWebComponents card
  states its React benchmark result.
- The top-nav **cashflux** link now opens in a new tab (`target="_blank"`), matching the elsewhere
  CashFlux card — it's a separate full app, so the portfolio stays put.

### Added
- **Scheduled daily Slack posting.** The "scheduled posting" toggle now drives a real server-side
  scheduler: when enabled, the server auto-generates the anime discussion post from the saved prompt,
  posts it to Slack, and publishes it to the QOTD feed **once a day** at a configurable hour (new
  "Daily post hour" control in the RSS panel; `post_hour` on `SlackConfig`). A per-day guard prevents
  double-posting, and a failed attempt skips the day rather than retrying every tick. Verified with a
  gating unit test and a live firing test (the timer fired and attempted the post on schedule).

### Changed
- **RSS/anime panel: one generation prompt + dry-run** (replaces the QOTD prompt list). The admin RSS
  page now edits a single **generation instruction** (a textarea, saved to `SettingQOTDPrompt`) instead
  of a list of static questions. A **Dry run** button generates a preview anime discussion post from
  the (unsaved) prompt via the OpenAI Responses API — fetching the latest Anime News Network headline —
  and shows the body + the rendered RSS item, **without publishing**. **Generate & post now** generates
  from the saved prompt, posts it to Slack, and publishes it to the QOTD feed (`/anime/qotd.xml` now
  serves the single last-published post). The legacy `qotd_prompts` table is dropped on startup and its
  seeded prompts removed. New gRPC `GetPrompt`/`SavePrompt`/`DryRunPrompt` (replacing
  `ListPrompts`/`AddPrompt`/`DeletePrompt`); new `openai.Generate` Responses-API text helper.

### Added
- **CashFlux sync discovery** (`/v1/version`): the embedded sync engine now also serves the CashFlux
  frontend's discovery probe at `/v1/version` (alongside the `/grpc` tunnel), so CashFlux's "Test
  connection" succeeds when pointed at this origin — it fetches `/v1/version` for the API version and
  auth mode before connecting. Point CashFlux's Server URL at this origin (no path).
- **First-run owner setup + password reset** (`internal/admin`, `internal/store`): the deployed site
  no longer needs credentials baked into the environment. On a fresh deploy the admin client shows a
  setup screen that creates the owner account (username + password), stored as **bcrypt hashes** in a
  single-row `owner_credentials` table, and returns a one-time **recovery phrase** (6 words) plus an
  owner-chosen **hint**. Password reset uses that phrase (verified via bcrypt) or an
  `ADMIN_RECOVERY_TOKEN` env break-glass, rotates the phrase, and invalidates every prior session (a
  `pwa` token claim vs `password_changed_at`). `ADMIN_PASSWORD` still works as a bootstrap (setup is
  then closed); `ADMIN_SETUP_TOKEN`, when set, gates first-run setup to close the land-grab window.
  New public gRPC methods `AuthState`/`Setup`/`ResetPassword`; new client screens (setup, recovery
  phrase, reset). Wrong-credential paths are throttled ~1s.
- **Password gate for CashFlux** at `/budget/` (`internal/budget`, `BUDGET_PASSWORD`): a locked
  terminal-window door (typed GWC, matching the site's identity) with a password field and a
  **guest bypass**. Entering the password grants a "full" session (your synced budget); the guest
  button grants a local-only session (no sync — a guest is never given the sync token). The session
  is an HMAC-SHA256-signed, HttpOnly, path-scoped cookie (30-day TTL; a guest cannot forge a full
  session). Defaults to `ADMIN_PASSWORD`; unset disables the gate (open access).
- **RSS / anime control panel** (`internal/rss`, wasm admin → gRPC): a fully in-app RSS surface.
  Configurable **QOTD prompts** with add/delete (50 seeded defaults, `qotd_prompts` table); the daily
  prompt feed is built from them. **Spec-compliant RSS 2.0** feeds at `/anime.xml` + `/anime/qotd.xml`
  (xmlns:atom + self `atom:link`, RFC1123Z `pubDate`/`lastBuildDate`, escaped via `encoding/xml`,
  stable guids). **Slack** integration — a configurable incoming-webhook (stored server-side, never
  returned), an enable toggle, and **Post to Slack now** which composes the latest **Anime News
  Network** headline + today's prompt into a discussion/debate message. New `rss` admin tab.
- **CashFlux as a managed budgeting service** at `/budget/` (`internal/budget`): the full CashFlux
  WASM app is built (`scripts/build-cashflux.sh`) and served from this server (correct `application/wasm`
  types, no double-decompress on the self-decompressing `.gz`, SPA fallback, path-containment guard).
  Linked from the home page and admin nav.
- **Embedded CashFlux data-sync engine** (managed, in-process): the server embeds *only* CashFlux's
  sync engine — its gRPC `SyncService` over a GoGRPCBridge WebSocket tunnel at `/grpc`, via the new
  `CashFlux/pkg/embed.NewSyncBridge` — backed by an encrypted server-side SQLite store under
  `CASHFLUX_DATA_DIR`, so multi-device sync persists on this backend without running the full CashFlux
  site. The frontend's WS origin is seeded from `BASE_URL` (normalized to a bare origin); the
  auto-generated access token is logged at startup (pin with `CASHFLUX_SERVER_TOKEN`). Point
  CashFlux's "server URL" at this origin to sync.
- **Résumé variants library**: the base résumé is permanent (the diff baseline, never overwritten).
  Each tailoring is saved as a variant, shown as designed, glanceable cards — the role **title @
  company** and the **keyword chips** the tailoring emphasized (derived from the stored analysis, so
  older variants are backfilled), plus a domain source-chip and date — in a **CRUD list** at the
  bottom of the résumé tool — **view / PDF** opens the variant's print page (`/resume?variant=<id>`,
  Save as PDF), **tweak** re-opens it in the workspace, **delete** removes it, and **Apply** sets the
  active `/resume`. Over gRPC: `GetBaseResume`/`ListTailorings`/`GetTailoring`/`DeleteTailoring`;
  variants + title/company persisted in SQLite. Delete + variant page verified in-browser.
- Project scaffolding: planning docs (README, DESIGN, PROJECT_LAYOUT, DEPLOYMENT, TODOS, DEVLOG),
  agent guides (AGENTS.md, CLAUDE.md), and the `karpathy-guidelines` skill.
- Interactive design mockup (`design/mockup.html`).
- P1 ingress server foundation: `/healthz`, static file serving, SSR placeholder, graceful
  shutdown.
- `/bin` gitignored build-output directory.
- gRPC contract (`ContentService`, `ContactService`) with generated stubs via `buf` (pure Go).
- `ContentService` implementation: featured-project dataset + about copy, unit-tested.
- gRPC-over-WebSocket tunnel at `/socket` (GoGRPCBridge) with `ContentService` registered.
- `ContactService` + pure-Go SQLite store (modernc.org/sqlite): validated messages persisted.
- Standard site rendered server-side as GWC components with typed CSS (`css/u`), mobile-first
  responsive, served at `/` (SEO + no-WASM failsafe), rendered once at startup.
- WASM terminal pipeline: the GWC client builds to wasm, boots via a minimal glue script, and
  mounts over the SSR site with typed CSS (verified rendering in-browser).
- Terminal (wasm): macOS-style window — traffic-light chrome, boot log, neofetch identity splash,
  and prompt with cursor — styled with typed CSS from theme tokens; screenshot-verified against
  the mockup.
- Terminal is now **interactive**: a controlled input (GWC `UseState`/`UseEvent`) runs the
  portfolio programs (`help`/`about`/`projects`/`open`/`neofetch`/`links`/`contact`) and the
  window **expands/shrinks** to a fullscreen modal (green/red lights).
- "Elsewhere" section on the standard site + terminal `links`/`resume`: résumé download,
  LinkedIn (`earl-cameron`), YouTube (`@EarlCameron007`), GitHub — ported from the current site.
- **Anime tracker** (Go port of the current site's Node feature): AniList-backed tracking with
  two RSS feeds — `/anime.xml` (Release Radar) and `/anime/qotd.xml` (daily prompts / QOTD) —
  plus a **password-gated config page** at `/admin` (search AniList, track/untrack, run release
  checks). Verified end-to-end (search → track → feed) against live AniList data.
- **Résumé**: a clean, print-optimized HTML résumé at `/resume` (light professional document,
  Ubuntu-orange accent, "Save as PDF" action) + an owner-gated **tailoring tool** at
  `/admin/resume` that fetches a job-posting URL and re-emphasizes real résumé facts to fit it
  (never fabricates — identity fields are force-preserved; OpenAI, gated behind `OPENAI_API_KEY`).
  The standard-site "Résumé" card and terminal `resume` now point to `/resume`.
- Résumé tool shows the tailored revision as a **git-style diff of the entire résumé** — every
  section is rendered, with unchanged parts (header, skills, projects, education, job metadata) as
  context lines and the changed parts (summary + bullets) as red/green — plus **Apply / Reanalyze /
  Cancel**. **Apply** persists the tailored résumé as
  the *active* one — `GetResume` and the `/resume` page then serve it (stored in SQLite via
  `ApplyResume`); Reanalyze re-runs the tailoring; Cancel discards it.
- **Tailoring results persist to SQLite** (`tailorings` table) — each pass (which costs an OpenAI
  call) is saved and the latest reloads when you open the résumé tool (`GetLastTailoring`). Joins the
  data already in SQLite: contact messages, tracked anime, and settings.
- Admin **sub-routing**: `/admin/anime`, `/admin/resume`, `/admin/settings` deep-link to the right
  view, the URL updates on navigation (history API), and browser back/forward works.
- Résumé tool is now a **live workspace**: a live document render of the résumé (mirrors the `/resume`
  PDF), the **signals extracted from the job posting** (title/company, keywords, requirements), and a
  **rationale for each tailoring choice**. `TailorResume` returns a `TailorResult` (résumé + job
  analysis + rationales); the model is prompted to extract + explain, the résumé stays constrained.
- Settings: a **"reload models"** button that fetches the models available to the OpenAI key (saving
  the key first if it was just typed) and turns the model field into a **dropdown** of those models.
- **Settings page** (`/admin/settings`): configure the **OpenAI API key + model** from the admin UI
  (stored in the site database), so secrets no longer require env vars / SSH. A DB setting overrides
  the env default and is read live — a key added here enables the résumé tailoring tool with **no
  restart**. The key field never renders the stored value.
- **AdminService (gRPC)** — the owner-gated admin control plane now lives on the gRPC data plane
  (over the GoGRPCBridge WS tunnel), not ad-hoc HTTP: `Login`, `SearchAnime`, `ListTracked`,
  `TrackAnime`, `UntrackAnime`, `RunReleaseCheck`, `GetResume`, `TailorResume`. A unary interceptor
  enforces a signed session token (`authorization` metadata) on every method but `Login`. The
  SSRF-guarded job fetch moved to `internal/resume`. Verified with a real bufconn client/server
  test (login, interceptor rejection, authed call, disabled-tailoring precondition). The legacy
  HTTP admin pages still render until the GWC/WASM admin console (next pass) replaces them.
- Discoverability for the new features (the links were buried/absent): résumé + LinkedIn added to
  the hero link row (github · résumé · linkedin · earlcameron.com · email); a public **"Anime, on
  RSS"** section links the two feeds; a discreet **`admin`** link in the footer (still
  password-gated); and an **`anime`** command in the terminal that prints the feed URLs.
- Recruiter notes in the terminal's `notes/` folder — `cat notes/about.md`, `experience.md`,
  `skills.md`, `projects.md`, `working-style.md`: professional, tech-fit content for curious
  recruiters exploring the shell (VFS cache bumped to v2).
- Terminal **faux Unix shell** over a **localStorage-cached virtual filesystem**: ~30 commands
  (pwd/ls/cd/mkdir/touch/cp/mv/rm/cat/head/tail/echo/grep/find/sort/uniq/cut/sed/awk/wc/du/df/
  history/less/nano/curl/ssh/tar/man) with pipes `|`, chaining `&&`, `>` redirect, a cwd-aware
  prompt, and **Tab completion** (commands + paths). Verified end-to-end via chromedp (9 checks).
### Changed
- Résumé tailoring now calls the OpenAI **Responses API** (`/v1/responses`, `input` + `text.format`)
  instead of Chat Completions, and **omits `temperature`** so newer models (o-series, gpt-5, …) that
  only accept the default no longer 400. Verified end-to-end against a live posting.
- **Admin is now a WASM client over gRPC — no longer server-rendered.** The admin console (login,
  anime tracker, résumé tailor, settings) is a GoWebComponents/WASM app (`client/admin.go`,
  `adminview.go`, `grpc.go`) that consumes `AdminService` over the GoGRPCBridge WebSocket tunnel.
  `/admin*` serves only a wasm bootstrap shell; all interactivity is client-side gRPC with the JWT
  held in `localStorage` and sent in call metadata. The SSR admin (the short-lived `internal/adminui`
  package) and every admin HTTP form endpoint are removed. `AdminService` gained `GetSettings` /
  `SaveSettings` / `ListModels` so settings + the model dropdown work over gRPC. Verified end-to-end
  in a real browser: login → authenticated `ListTracked` → console renders the tracked shows.
- **Home page indexes every page**: a top nav (`work · résumé · anime · contact · admin`) plus section
  anchors, so all page links live on the single home page.
- Standard site reworked to match the mockup: hero-first order (orange-dash eyebrow, accent
  headline, sans-serif lede, glyphed social, orange launch CTA), then the terminal; boot log with
  OK badges + dotted leaders + right-aligned values; "live · interactive" title; ambient glows.
- Colors centralized into `internal/theme` design tokens (quick-ref: DESIGN.md §16).
- Minimal bootstrap `<style>` (reset + base bg + mono font-family) added to the SSR shell.
### Fixed
- Terminal: auto-scrolls to the newest line; the input keeps focus after Enter; the expanded
  modal locks the page scroll and has its own; themed scrollbar. Verified with a chromedp
  browser-interaction test (real keystrokes + clicks), not just a render check.
- Terminal modal no longer overflows the right edge (removed `width:100%` that fought the fixed
  insets); it now sizes purely from top/left/right/bottom offsets.
- Terminal: clicking anywhere in the body focuses the input (no need to hit the prompt line).
- Terminal: text is selectable again — click-to-focus no longer clears an active selection, and
  the body is explicitly `user-select: text`. (Verified via a chromedp mouse-drag selection.)
### Security
- Slack posting escapes untrusted external content: an Anime News Network headline can't inject
  mrkdwn (`@channel`/`@here` pings or a forged `<url|label>` link) into the composed Slack message
  (adversarial-review finding). The news fetch also restricts redirects to the feed's host, and QOTD
  prompts are length-capped + deduped by a unique index (race-proof seeding). The CashFlux static
  handler validates path containment explicitly (no reliance on stdlib side effects).
- Admin auth is now **username + password → signed HS256 JWT** (golang-jwt/jwt v5): the login
  (`ADMIN_USERNAME`/`ADMIN_PASSWORD`) mints a JWT with algorithm pinned to HS256 and bound to the
  owner subject; the same token gates both the HTTP cookie path and the gRPC `authorization` metadata.
  Unit-tested (round-trip, tamper, wrong-secret, wrong-subject, credential checks).
- WebSocket tunnel rejects cross-site origins (CSWSH guard) — same-origin + `ALLOWED_ORIGINS` only.
- `/admin` config page is gated by a single password (`ADMIN_PASSWORD` env) behind an HMAC-signed,
  HttpOnly session cookie; disabled entirely when no password is set.
- Admin auth hardening (from commit review): random per-process session secret when `ADMIN_SECRET`
  is unset (no forgeable known-key fallback), `Secure` cookie on https origins, and login
  rate-limiting (5 failures → 15-min lockout → 429).
- Résumé tailoring tool hardened (from review): the URL fetch blocks SSRF (dialer rejects
  loopback/private/link-local/unspecified IPs, ports 80/443 only, redirect hops re-validated,
  credentials/non-http schemes refused); the tailor endpoint is POST-only (kills CSRF-driven SSRF
  via `SameSite=Lax` GET navigation); and the model output is constrained to the canonical résumé
  skeleton (only the summary and bounded, reworded bullets can change — employers, titles, dates,
  skills, projects, and education can't be fabricated by a prompt-injecting job posting). Covered
  by unit tests.
