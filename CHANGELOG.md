# Changelog

All notable changes to earlcameron.com. Format: [Keep a Changelog](https://keepachangelog.com);
Semantic Versioning once released.

## [Unreleased]
### Added
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
  Linked from the admin nav. Frontend-hosting only — server-side sync is a separate larger project.
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
