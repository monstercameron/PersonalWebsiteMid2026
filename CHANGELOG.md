# Changelog

All notable changes to earlcameron.com. Format: [Keep a Changelog](https://keepachangelog.com);
Semantic Versioning once released.

## [Unreleased]
### Changed
- **Release metadata now matches the pinned build inputs.** The site requires Go 1.26.5 and
  GoWebComponents v5.0.1, matching the CashFlux and framework revisions checked out by CI and
  deployment.
- **Project card actions are now `code · benchmarks · live demo`.** The evidence link moved ahead of
  the demo — it is the receipt for a claim the card makes in its own text, and last place read as an
  afterthought next to the thing people actually click. "demo" became "live demo" in both tiers so a
  visitor does not have to guess whether it runs.
- **Résumé projects carry the numbers now**, ordered to match the site's featured four so a reader
  arriving from either direction meets the same story: CashFlux with 221 packages / 2,998 tests / 26
  releases in six weeks, ArticleFlux with 151 subscriptions and 3,621 items, GoWebComponents with the
  React comparison. Every figure is counted from the repositories, not estimated. The **employment**
  bullets are still task-shaped and still need Cam's own scope and outcome figures — the one part of
  the résumé that cannot be sourced from a repo (TODOS §14.H).

### Fixed
- **Every CashFlux homepage link now opens the standalone service.** The top navigation,
  `~/elsewhere` card, and project-card live demo all point to `https://budget.earlcameron.com`
  instead of the portfolio's legacy `/budget/` mount or the retired GitHub Pages demo.
- **The RSS feeds were advertising a raw IP to their own subscribers.** Both public feeds printed
  `http://167.99.232.99:8080` as the channel `<link>`, the `atom:self` href and every item guid —
  *including when served from `https://www.earlcameron.com`* — because that is what `BASE_URL` was
  set to on the droplet. Feed URLs outlive the request: a reader stores them and follows them for
  months, so this shipped a raw IP on a non-standard port into every subscription. `feedBaseURL` now
  prefers the hostname the request actually arrived on when it is a real DNS name, falling back to
  the configured value. A forged `Host` cannot redirect the advertised origin: IP literals,
  `localhost`, single-label names and anything containing a path separator are rejected, and the
  scheme comes from `X-Forwarded-Proto`, not the client. Covered by table tests. The droplet's
  `BASE_URL` was also corrected to the real origin, which is what the homepage's feed field renders
  from — it had been showing the IP too.
- **Security: Host-header injection in that same fix.** The first version accepted any DNS-name
  `Host`, so `curl -H 'Host: evil.example' …/anime.xml` would have produced a feed advertising
  `evil.example` as its permanent home — and a feed is exactly the cacheable, long-lived GET that
  reaches a shared cache. The request host is now echoed **only when it matches the configured
  origin's host** (case-insensitively); anything else falls back to `BASE_URL`, so injection is
  impossible whenever `BASE_URL` names a real domain. The permissive path survives only when
  `BASE_URL` has no usable hostname — the misconfiguration this function exists to rescue — and is
  documented as such. Thirteen table cases, including forged `Host`, forged `X-Forwarded-Host`, and
  a subdomain of the real host.
- **The page title, description and OG tags still carried the old positioning.** The hero had moved
  to "Senior software engineer building AI-native products…" while `<title>` still said "AI-native
  systems engineer", so every crawler, search snippet and pasted-link preview reported the narrower
  framing — the strings read *first* and by the most people were the last ones updated. All four now
  track the hero, with a note to change them in the same commit as any future H1.
- **The résumé title was narrower than the site's.** Now "Senior Software Engineer — AI Systems,
  Developer Platforms & WebAssembly". The résumé travels separately from the site, so a stale title
  there quietly undoes the repositioning for whoever only ever sees the PDF.

### Added
- **A `benchmarks ↗` link on the GoWebComponents card.** "Benchmarked head-to-head with React —
  faster on overall geomean" was the least verifiable sentence on the page; the numbers existed in
  the GWC repo's `docs/benchmarks` and simply were not reachable from here. Third links come from a
  small `evidenceLinks` map rather than a new proto field, since protoc is unavailable (TODOS §2).

### Changed
- **CashFlux copy is measured rather than promotional.** "40+ pages" → "50 routes";
  "enterprise-grade test suite" → "221 internal packages behind 2,998 tests", and the long
  description now names the shipped modules instead of asserting quality.
- **WASIBrowser leads with the result**, not the vocabulary: "A browser runtime where applications
  ship as WebAssembly components instead of JavaScript bundles." The reactified-C ABI detail stays
  one sentence later for the reader who wants it.
- **CashFlux lock page states the personal-use angle up front** — "the same CashFlux deployment I
  use personally" — so the password box reads as conviction rather than as a closed door, with guest
  isolation stated as information rather than defence.
- **Motion pass — one gesture, used twice, no JavaScript.** DESIGN.md §7 asks for a few earned
  moments rather than confetti, so the whole site shares a single `rise` gesture (a 10px lift with a
  fade): staggered at load across the hero, and tied to `animation-timeline: view()` for every
  section below it. That is native CSS scroll-driven animation — no observer, no JS, and browsers
  without support simply run it once at load and land on the same final state, so nothing is ever
  left invisible. Plus card hover-lift, link colour transitions, a `:focus-visible` accent ring
  (keyboard only), and the terminal's signature expand — scrim fade with the modal arriving from
  98.5% scale. **Everything collapses to nothing under `prefers-reduced-motion`**, verified by
  auditing computed opacity in both modes after a full-page scroll.
- **Copy-to-clipboard on the RSS feeds, plus the URL in a field beside it.** Each feed card now
  prints its **absolute** address in a readonly, selectable input with a `copy` button next to it;
  the button reports `copied` / `copy failed` / `select it instead` rather than failing silently.
  Two affordances on purpose — the Clipboard API is unavailable outside a secure context and the
  button depends on the wasm having booted, so the field is the fallback that always works.
  Absolute URLs required threading `config.BaseURL` through `site.RenderHTML` → `Page` →
  `animeRadar`: a relative `/anime.xml` is useless in the one place these are meant to be pasted.
  Wiring lives in `client/clipboard.go` as one delegated `data-copy-url` listener, bound in `main`
  before the terminal mounts, since the buttons belong to the SSR shell and not to any component.
- **Terminal window chrome behaves like a window.** The traffic lights reveal their glyphs on hover
  of the cluster (`u.GroupHover`, so all of them light together the way macOS does) — `−` on red,
  which shrinks the fullscreen modal, and `⤢` on green, which expands it. Yellow is inert decoration
  and gets no glyph so it never looks clickable. The glyphs describe what these buttons actually do
  rather than copying the macOS close/minimize/zoom mapping, since nothing here closes.
- **Escape shrinks the fullscreen terminal.** Bound on `document` rather than the input, so it still
  works after clicking into the scrollback to select text — losing input focus can no longer trap a
  visitor in the overlay. Bound only while expanded. The title bar now says "esc or red to shrink".
- **Project tiers: four featured case studies, everything else in Labs.** `site.splitTiers` takes
  the first `featuredCount` (4) of `content.featured` — CashFlux, ArticleFlux, GoWebComponents,
  WASIBrowser — and renders them as wide two-up cards carrying both the blurb *and* the long
  description; the remainder renders as a quieter ruled list under a new `~/labs` section. The tier
  difference is carried by form, not just size, so a visitor can tell billed work from experiments.
- **Hero proof strip** — "creator of CashFlux · ArticleFlux · GoWebComponents · WASIBrowser", built
  from the featured slice so it cannot drift out of sync with the grid below it.
- **"Built at high velocity" evidence section** — CashFlux measured, not described: 26 documented
  releases, 221 internal packages, 50 routes, 2,998 tests, six weeks from first commit. Counted from
  tracked files with `git ls-files`. No lines-of-code figure, deliberately.

### Changed
- **Hero repositioned for recruiters.** "AI-native systems engineer. I ship ambitious things, fast."
  → "Senior software engineer building AI-native products, developer platforms, and unconventional
  systems", with the employer named ("currently building agentic systems at UKG"). "Systems
  engineer" reads narrowly to a recruiter — OS, networking, embedded, infra admin — and undersold
  the product and platform work. The first screen now carries role, specialization, employer, proof
  strip, and **View the work · Read the résumé · Contact** above the fold at 390px.
- **The terminal is a reward, not a gate.** It no longer owns the loud primary button; the work
  does. Its suggested commands (`projects`, `open cashflux`, `neofetch`) are shown as chips instead
  of guessed at, with "open it fullscreen" as a quiet inline control. The copy states what the
  terminal actually is — "Everything above is server-rendered Go. The terminal below **is** the
  WebAssembly" — because on the public site the wasm binary mounts nothing else.
- **The hero's "Launch the live terminal" CTA now launches the terminal.** It was a `<div>` with
  `cursor:pointer` and no handler — pure decoration. It is now a `<button id="term-launch">` that the
  wasm terminal binds on mount (`client/terminal.go`): one click expands the terminal to fullscreen
  **and focuses its input**, so a visitor can type immediately. Expanding by any route now focuses
  the input, and the button is keyboard-reachable.

### Changed
- **No city anywhere on the site.** "LAUDERHILL, FL" → "REMOTE" in the hero eyebrow, the terminal's
  `notes/about.md`, and the résumé's `Location`. Contact address is now `cam@earlcameron.com`
  (5 call sites: hero social link, contact section, terminal `links` + `contact`, résumé).
- **Footer redesigned.** It was a full 1px box — `Border()` sets four sides — floating under the
  page, reading as a stray rectangle rather than an ending. Now a single hairline rule above the
  content, a real `<footer>` element, with the credit line rewritten to close the page's argument:
  GoWebComponents and GoGRPCBridge are links to Cam's own repos, "zero npm" is the only accent down
  there, and `admin` drops to Faint so it stops competing with the copyright.
- **ArticleFlux is a first-class destination**, matching CashFlux exactly: a top-nav entry
  (`articleflux`, opens in its own tab) and an `~/elsewhere` card. Both point at
  `https://feed.earlcameron.com/home`, **not** the host root — the root resolves to the reader, which
  requires an account, so an unauthenticated visitor would land on a sign-in form. `/home` is the
  public front door. The target lives in one constant, `site.articleFluxURL`.
- **WebGL Path Tracer joins the featured grid** (`pathtracer`) — browser path tracing with a dozen
  material models, Rapier physics, SDF/CSG primitives, mesh import and a benchmark harness, with a
  live GitHub Pages demo. Picked over the native `vkPathTracer` specifically because a recruiter can
  click it; the card's `Long` credits the Evan Wallace original it was forked from and rebuilt on.
- **A GitHub link in the projects section eyebrow** — `github.com/monstercameron ↗`, right-aligned
  on the `~/projects · featured` line so the full repo list is reachable without scrolling past nine
  cards. Wraps below the path on narrow viewports.
- **CashFlux card now links its GitHub Pages demo** (verified 200), so the site's flagship project is
  clickable from the grid and not only through the nav's embedded `/budget/` instance.
- **Deploy: the droplet story is real.** `deploy/` now holds `install.sh` (one-shot fresh-droplet
  setup), `update.sh` (pull → build → stage → atomic symlink swap → `/healthz` → **auto-rollback**),
  `rollback.sh` (manual undo, symlink-only, `--list`/by-name/back-one), a shared `lib.sh`, a hardened
  systemd unit, an Nginx site, and `env.example` documenting every key the code actually reads.
  Nginx upgrades **`/grpc` as well as `/socket`** — `/grpc` is the embedded CashFlux sync tunnel, and
  missing it fails "Test connection" with no obvious cause — and gzips `application/wasm`, which is
  the only thing between a visitor and a 26 MB `app.wasm`. The systemd unit deliberately omits
  `MemoryDenyWriteExecute`: `ncruces/go-sqlite3` runs SQLite through wazero, which JITs W^X pages, so
  the hardening flag that looks obviously correct would kill the budget app on its first query.
- `.gitattributes` pinning `eol=lf` for shell/unit/conf files, so a Windows-authored script cannot
  reach Ubuntu with a CRLF shebang.

### Changed
- **Featured projects re-cut and re-ordered.** WhisperToMe is out; the order is now CashFlux,
  ArticleFlux, GoWebComponents, GoGRPCBridge, WebGL Path Tracer, WASIBrowser, SemanticScript,
  SemanticAssembly, SemanticPortrait. Nine entries fill the desktop grid's three columns exactly, so
  the order is now a layout decision as well as a billing one — row 1 the shipped apps plus the
  framework they run on, the path tracer's live demo in the centre cell, the three `Semantic*`
  research projects as one bottom row. All three lists that must not drift were updated together:
  `internal/content.featured`, `client/programs.go termProjects`, `client/vfs.go projectsMD`.
  **Note:** dropping WhisperToMe leaves no on-device-ML card behind the hero's "on-device ML" claim
  and the résumé's Snapdragon/QNN/ONNX line — tracked in `TODOS.md` §0.
- **Frontend moved to GoWebComponents v5** (`/v4` → `/v5`, `replace` retargeted). The repo did not
  build before this: the local GWC checkout had moved to the `v5` module path while `go.mod` still
  required `/v4`. Migration was import-path-only — v5 is additive — except for one genuine API
  break, fixed upstream and released as **GWC v5.0.1**: `css/u` re-exported seven names
  (`Track`, `Repeat`, `LinearGradient`, `RadialGradient`, `Stop`, `Circle`, `Ellipse`) that
  `html/shorthand` already declares, so the five files here that dot-import both failed with
  `redeclared in this block`. **The GWC pin must stay ≥ v5.0.1.**
- `scripts/build.sh` names the binary `bin/server$(go env GOEXE)` — `server.exe` on Windows, `server`
  on the droplet — so `deploy/` has one predictable path.
- `scripts/build-cashflux.sh` falls back to the Go toolchain's `wasm_exec.js` when the CashFlux
  checkout lacks one. CashFlux gitignores that file, so a fresh clone — i.e. every droplet — has no
  copy and the staging loop died on it. The toolchain's copy is byte-identical and actually matches
  the wasm just compiled.

### Added (earlier)
- **Admin console: CashFlux account management.** Each user row now carries a role picker
  (member/viewer), Suspend/Restore, Reset, and Delete, plus the account's role and suspended state
  in the list. Four new owner-only RPCs over CashFlux's new `pkg/embed` surface. Refusals from
  CashFlux (demoting the owner, an unknown role, a taken username) arrive as `InvalidArgument`
  carrying their reason, so the flash says WHY rather than "failed". The owner row deliberately
  renders a static "owner" label instead of a picker: `pkg/embed` refuses to demote that account, so
  offering a control that can only fail would misrepresent what is possible.
- **One-click, one-credential entry to CashFlux.** The console's "budget ↗" now mints an activation
  code against its own authenticated session and opens `/budget/?activate=<code>`; the client
  redeems and strips it. The budget gate recognises a live code and opens on that basis — demanding
  a second, unrelated password from someone who just proved themselves to mint it proves nothing.
  Opened with `noopener` because the URL carries a single-use credential.
- **End-to-end suites.** `e2e/harness.mjs` (hermetic rig: own port, site DB, CashFlux data dir,
  fresh owner), `e2e/sync-flows.mjs` (21 checks: pairing, upload, hydration, re-login) and
  `e2e/auth-flows.mjs` (9 checks: handoff, owner protection, management actions). Assertions go
  through the real UIs, and the payload is a uniquely-named transaction typed on one browser and
  read back on another — byte counts and status labels were green through two real bugs these
  suites then caught.

- **Admin console: CashFlux stats that actually move.** Cam: "I don't see any updates on the
  server side stats." He was right and the panel was at fault — neither figure it showed CAN
  reflect sync. "Requests this month" counts metered AI calls (`AddUsage` is reached only from the
  AI proxy; nothing in the sync path touches it), so it reads 0 forever for an account that syncs
  constantly; and the database's own size barely moves, since a 17KB dataset landing in a 284KB
  file is lost in the page-count rounding. A server syncing perfectly well therefore presented as a
  dead panel. Now: the storage panel leads with **Synced data** (total dataset-snapshot bytes — the
  one number that changes on every push), then artifact blobs, then the database file; and each
  user row's right column shows **how much of their data this server holds** and **when they last
  pushed** ("1.3 MB / synced just now", plus a workspace count when there's more than one) instead
  of a request count that was structurally always zero. A never-synced account shows an em dash and
  "never synced" rather than "0 B", which would read as the much more alarming "synced, and
  empty". Backed by CashFlux's new `UserSyncSummary`/`SnapshotBytes`; three new proto fields on
  `CashFluxUser` plus `snapshot_bytes` on `CashFluxStorageStats`. 3 new tests; verified against a
  real synced account (1,363,769 bytes on the server → "1.3 MB" in the panel).
- **Admin console: delete CashFlux users.** Each row in the users list gets a Delete action that
  permanently purges that account and everything it owns — workspaces, dataset snapshots, artifact
  blobs on disk, AI keys, and every refresh token, so it cannot be resurrected by a session that
  outlived it. Two-step and the row itself is the confirmation (no modal, no single-click erasure):
  the row swaps for an are-you-sure state that names what is destroyed rather than asking a generic
  "are you sure?", with Cancel as the escape hatch and both buttons disabled while the purge is in
  flight. The owner's own account — the one every activation code opens — carries a "your account"
  badge in the list and gets its own confirmation wording, because deleting your CashFlux data is a
  different act from removing an invited person; `CashFluxUser.is_owner` is set server-side against
  CashFlux's own constant so the console can never disagree about which row that is. New owner-only
  `DeleteCashFluxUser` RPC over `pkg/embed.Admin.DeleteUser`; `deleted=false` (already gone) comes
  back as a normal response, not an error. The users list and storage figures re-read on success.
  9 new tests. Verified end to end on an isolated instance against a REAL synced account — 1.5 MB
  database, 7 blobs, workspace, snapshot, refresh token — all zero afterwards including the files on
  disk, with the audit row retained.
- **Admin console: "Activate a device" — generate a CashFlux activation code.** The "cashflux" tab
  now leads with a Generate-code button; the minted 6-digit code appears in an accent-bordered
  callout with its expiry and a copy button (the same treatment the pairing-code callout already
  uses, since it's the same kind of object). Typed into any CashFlux client's Settings → Cloud, it
  signs that device in — no pending request to approve first, no username or password anywhere.
  This is the whole access-control story for the embedded instance: minting a code requires an admin
  session on this site, and CashFlux's embedded bridge disables self-signup, so nobody without that
  session can get in. New owner-only `MintCashFluxActivationCode` RPC behind the same
  FailedPrecondition-when-not-configured gate as the rest, backed by CashFlux's new
  `pkg/embed.Admin.MintActivationCode`. Every code binds to one owner account, so a second activated
  device shares the first one's data. The expiry is normalized to UTC on the wire and rendered in
  local time. 4 new tests; verified end to end on an isolated scratch instance (separate port + data
  dir, fresh owner setup, Playwright-driven: mint → activate → 1.36 MB dataset and 7 artifact blobs
  land in the scratch server's SQLite store).
- **Admin console: CashFlux users + storage stats.** The same "cashflux" tab now shows the
  enrolled-users list (email/id, provider, signup date, subscription plan/status, and current
  calendar-month request volume — via CashFlux's new `pkg/embed.Admin.ListUsers`) and a storage
  panel (database size + total artifact-blob size, human-readable via a new `formatBytes` helper —
  via `pkg/embed.Admin.StorageStats`), below the existing pending-devices panel. Two new
  owner-only `AdminService` RPCs, `ListCashFluxUsers` (paged; a single "Load more" button appends
  further pages — no page-number UI, matching this deployment's small admin-invited user-set
  scale) and `GetCashFluxStorageStats`, both behind the same FailedPrecondition-when-not-configured
  gate as the rest of the CashFlux RPCs. `CashFluxAdmin` in `internal/admin` gained `ListUsers`/
  `StorageStats`, mirroring `pkg/embed.Admin`'s signatures exactly (no adapter needed). Verified in
  an isolated scratch environment (separate port/data dir, seeded users directly into the scratch
  SQLite store, desktop + mobile screenshots) and by two sequential mandatory adversarial reviews,
  which found and this shipped fixes for: (1) the two new RPC calls skipped the file's `onAuthErr`
  session-expiry pattern; (2) `formatBytes` mis-rounded values a hair under a power-of-1024 (e.g.
  `1048575` → "1024.0 KB" instead of "1.0 MB"); (3) a stale/then-misordered doc comment on
  `cashfluxView` after its parameter list grew.
- **Admin console: CashFlux pending-devices tab.** CashFlux dropped phone/SMS sign-in entirely
  (Twilio cost money, never signed up a real user) in favor of an admin-approved device-pairing
  bootstrap: an unauthenticated device asks to pair, and the owner approves or rejects it from here.
  The "cashflux" tab in `/admin` now lists unresolved pairing requests (label, requested/expiry
  time) with per-row **Pair**/**Reject** actions. Pairing mints a brand-new CashFlux account and
  shows the pairing code large and monospace with a one-click copy button (`Copy code` → `Copied ✓`,
  the same `copyToClipboard` helper from `client/grpc.go`) — the owner reads it to (or cross-checks
  it against) the person on the device before they accept, a human MITM/mismatch check. Three
  owner-only `AdminService` RPCs (`ListCashFluxPendingDevices`/`ApproveCashFluxPairing`/
  `RejectCashFluxPairing`), replacing the earlier `ListCashFluxClients`/`MintCashFluxInviteCode`/
  `ListCashFluxInviteCodes` trio this same tab shipped with days ago (superseded before release —
  see CashFlux's own CHANGELOG for the pairing-bootstrap mechanism). `CashFluxAdmin` in
  `internal/admin` now mirrors `pkg/embed.Admin`'s new method set exactly, so the real
  `*cashfluxembed.Admin` value satisfies it with no adapter. Verified end-to-end in an isolated
  scratch environment (seeded pending-device rows via the real embedded SQLite store, real Pair/
  Reject clicks over the real wire, desktop + mobile screenshots) and by a mandatory adversarial
  review, which found and this shipped fixes for: (1) the in-flight "busy" flag guarding the
  Pair/Reject buttons was a single scalar, so clicking one device's Pair button while another's was
  still in flight re-enabled the first device's buttons mid-request — now a per-device-id set; (2)
  the pairing-code callout didn't clear on navigating away and back, so a stale code for an
  already-resolved device could reappear looking like a live cross-check prompt — now reset whenever
  the tab is (re-)entered.

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
