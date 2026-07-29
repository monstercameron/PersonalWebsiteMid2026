# DEVLOG — earlcameron.com

Newest first. Dated narrative of the build: what, why, what broke, what's next. Log failures too.

## 2026-07-29 — The "stale root render" that wasn't, and the metadata that was

A review reported something alarming: the bare root URL appeared to serve the *old* homepage —
former headline, Lauderhill, nine equal projects, old email — while clicking through gave the new
one. That would mean an SSR/hydration split or a cache serving stale HTML to crawlers, which is
worse than any copy problem.

**It was not happening.** `curl -L https://www.earlcameron.com/` returns the new content with zero
occurrences of `Lauderhill` or `mr.e.cameron`, and `/` and `/work` are **byte-identical** documents
(33,883 bytes each — `/work` is not a route, it is an in-page anchor served by the same shell). No
divergent templates, no cache split, no hydration replacing a stale shell. Two things produced the
impression: the first fetch almost certainly predated yesterday's deploy landing, since production
was stuck on `bb0f044` until 11:52 UTC — and one genuine defect underneath it.

**The genuine defect: the `<title>`, `og:title` and `meta description` still said "AI-native systems
engineer".** The hero had been repositioned; the head had not. Those four strings are read first and
by the most people — the browser tab, the search snippet, the preview card when the link is pasted
into Slack or LinkedIn — and they were the last ones updated. The review's own citation was titled
with the old positioning, which is exactly the failure. Fixed, with the reason recorded next to them
so a future H1 change takes them along.

**Where a claim had no receipt.** GoWebComponents said it was "benchmarked head-to-head with React —
faster on overall geomean" with nothing to click. The numbers were already in the GWC repo under
`docs/benchmarks`; they were just unreachable from the card. Third links now come from an
`evidenceLinks` map keyed on project id rather than a new `sitepb.Project` field, because adding one
means regenerating protos and protoc is unavailable here.

**Two reported problems that were not problems, checked rather than assumed.** The "ambiguous blank
input" on the CashFlux lock page is `<input type="hidden" name="mode" value="guest">` — a DOM parse
sees it, a visitor never does; the page has exactly one visible field, the password. And `/budget/`
answering a bare `curl` with a bodyless 401 is deliberate: the lock page renders only for a document
navigation, because a service worker fetching `./bin/main.wasm.gz` without a session would otherwise
cache an HTML lock page under the WASM's URL and hang the next boot. Neither was changed. The lock
page *copy* was improved, which was the real suggestion.

**One trap worth recording.** `/resume` prefers a stored `active_resume` override from the database
over `resume.Data()`, so editing the Go file appeared to do nothing locally — the dev database has
an override applied from testing the tailoring feature. Production has none: it renders `Remote` and
`cam@earlcameron.com`, which are code values, so the new title lands there on deploy. Anyone
wondering why a résumé edit "didn't take" should check that setting first.

Also: `gofmt -l` flagged a dozen files this session, including several never touched. That is the
CRLF artifact a `git checkout main`/`dev` round-trip leaves in the working tree, not real drift —
`git show :<file> | gofmt -d` returns zero lines for every changed file, and `git status` sees only
the four that were actually edited. Do not "fix" it with `gofmt -w`; that rewrites line endings and
manufactures a repo-wide diff.

## 2026-07-29 — Motion: one gesture, and no JavaScript to deliver it

DESIGN.md §7 is explicit that restraint is the strategy and that scattered effects read as
AI-generated, so the pass uses **one** gesture everywhere — `rise`, a 10px lift with a fade — played
two ways: staggered across the hero at load (0/70/140/210/280/350ms), and tied to the element's own
viewport progress for every section below it. Six different gestures is what makes a page feel
generated; one gesture reused is what makes it feel composed.

**The scroll reveal needs no JavaScript.** `animation-timeline: view()` is native CSS scroll-driven
animation — no IntersectionObserver, no observer bookkeeping, nothing added to the wasm. That matters
here beyond elegance: the sections are server-rendered, so a JS-driven reveal would have made the
SSR content depend on the wasm having booted. The degradation is safe by construction: a browser
that does not support the property ignores the timeline and runs the animation once at load, landing
on the same final state.

**The failure mode to guard was content stuck invisible**, since `animation-fill-mode: both` holds
an element at its "from" frame. So it was checked rather than assumed: scroll the whole page, then
read computed opacity for every section, heading and the footer — in normal mode *and* with
`prefers-reduced-motion: reduce`. Zero elements below opacity 0.99 in both. And to prove the effect
is not silently a no-op, the inverse: `#labs` reads opacity 0 while off-screen, 13 animations run on
the page, hovering a card computes `matrix(1,0,0,1,0,-2)`, and expanding the terminal fires
`scrim-in` + `term-open`.

Reduced motion is a real collapse, not a shortened effect: `animation-name: none` with the final
opacity and transform applied directly.

The terminal expand is the honest cheap version of DESIGN.md's "signature transition" — a true FLIP
from inline frame to modal needs measurement before paint. The modal arrives from 98.5% scale and
6px low behind a fading scrim, which reads as one movement rather than a cut.

## 2026-07-29 — Copyable RSS URLs, and the base URL that had to be threaded

The feed cards were bare links, which is the wrong shape for an RSS address: nobody *clicks* a feed
URL, they paste it into a reader. Each card now shows the address in a readonly field with a copy
button beside it.

**Two affordances, deliberately.** The button is the fast path but it can fail for reasons that
aren't bugs — `navigator.clipboard` does not exist outside a secure context, and the write can be
rejected — and it also depends on the wasm having booted at all. So the field carries the URL either
way and the button says what actually happened (`copied` / `copy failed` / `select it instead`)
rather than pretending. If the binding never runs, the address is still there to select by hand.

**The base URL had to be threaded.** The cards were rendering relative paths, and a relative
`/anime.xml` is worthless in a feed reader — the one place these strings are meant to go. `config`
already had `BaseURL`; it just never reached the SSR layer, so it now passes through
`site.RenderHTML` → `Page` → `animeRadar`.

**Where the handler lives.** The buttons are SSR markup, so they exist before the wasm runs and
can't be given handlers at render time — the same constraint as the hero CTA. One delegated
`data-copy-url` listener on the document, bound in `main` before `ui.Run`, covers all of them and
needs no cleanup. `closest()` resolves clicks that land inside the button rather than on it. The
promise handlers are released from whichever of then/catch fires, since a promise settles once and
an unreleased `js.Func` leaks for the life of the page.

Verified by reading the clipboard back in the browser, not by trusting the label: click →
`navigator.clipboard.readText()` returns `http://127.0.0.1:8096/anime.xml`, and the button returns
to "copy" after the flash.

## 2026-07-29 — Terminal chrome: hover glyphs and an escape hatch

Two small things that make the terminal read as a window rather than a picture of one. The traffic
lights now reveal glyphs on hover, and the reveal is a *group* hover — `u.GroupHover` compiles to
`.group:hover &`, so the wrapper carries the literal class `group` concatenated with its generated
one, and all three dots light together the way macOS does rather than one at a time.

The glyph choice is deliberate and is *not* the macOS mapping. Here red shrinks the fullscreen modal
and green expands it; nothing closes. So red gets `−` and green `⤢`, and yellow — which is inert
decoration — gets no glyph at all, because a glyph would make it look clickable.

Escape now shrinks. It is bound on `document`, not on the input: the terminal lets you select text
in the scrollback, and selecting blurs the input, so an input-scoped handler would strand a visitor
in a fullscreen overlay with a key that does nothing. Verified both paths — with the input focused,
and after an explicit `blur()` — the modal drops from 810px back to its inline 460px.

## 2026-07-29 — Recruiter refinement, and the metric that would have been a lie

Cam brought a detailed recruiter/hiring-manager critique: keep the ambition, add the evidence layer.
Featured case studies vs a quieter Labs shelf; a recruiter-efficient first screen; the terminal
framed as a reward rather than a gate; and CashFlux as the primary velocity proof, charted as
**cumulative merged PRs over time** with milestone annotations. TODOS §14 now carries the whole
thing. Most of it is right and most of the first pass is built. Two findings changed the plan.

**The PR chart cannot be built.** CashFlux has **zero merged pull requests**. All **2,745 commits**
went straight to `main`. There is no PR data to plot, so the centrepiece of the proposed velocity
section does not exist. Nor is the obvious substitute safe: commits per week run
1179 · 626 · 227 · 124 · 408 · 181, a **front-loaded and declining** curve. Plotted honestly it says
"big burst, then tapering", the opposite of sustained velocity, and it puts ~67 commits/day on
screen — which invites the question of how much was agent-written instead of answering it. What is
real: **26 documented releases** in `CHANGELOG.md`, each naming shipped capability. That is the
chart if he wants one, and the decision is logged for him rather than made for him.

**A metric was nearly published wrong.** The first count of the CashFlux test suite used `find`,
which walked untracked build output and returned **3,669 test files**. `git ls-files` says **638**
test files and **2,998** test functions — the `find` number overstated by roughly 6×. That number
was one step from being printed on a public page aimed at people who verify claims. Anything
measured for this site now goes through `git ls-files`, and the rule is written into the code
comment above the strip. The published set: 26 releases · 221 packages · 50 routes · 2,998 tests ·
six weeks. No lines-of-code headline — LOC rewards duplication and readers know it.

**What shipped.** `splitTiers` divides `content.featured` positionally: the first four are billed
case studies in wide two-up cards that now carry the long description as well as the blurb; the rest
fall into a new `~/labs` ruled list. The tier difference is carried by *form* — cards vs a list —
because size alone reads as an accident. Reordering the content slice now re-tiers the site, which
is documented where the slice lives. The hero leads with "Senior software engineer building AI-native
products, developer platforms, and unconventional systems", names UKG, and carries a proof strip
built *from the featured slice* so it cannot drift. The terminal lost the loud orange button to
"View the work", with its commands shown as chips rather than guessed.

**The critique was wrong about the terminal, and the first copy shipped its error.** It described
the terminal as something to occupy a visitor "while the WASM binary initializes", and that line
went onto the page. It is self-contradictory here: `client/main.go` mounts *only* `Terminal` into
`#term-root` on the public site — the terminal **is** the wasm. It cannot entertain anyone before
the binary it lives in has loaded. Cam caught it. The truth is also the better line, because the
rest of the page is server-rendered Go: "Everything above is server-rendered Go. The terminal below
*is* the WebAssembly." The lesson worth keeping is narrower than "check copy": an outside critique
carries assumptions about the architecture, and those assumptions need checking against the code
before their words go on the page.

**Note the conflict:** billing WASIBrowser as the fourth case study pushes **GoGRPCBridge into
Labs** — the same project Cam asked to bill fourth yesterday. Flagged rather than quietly resolved;
it is a one-line swap either way.

**Two responsive bugs caught by screenshotting rather than reasoning.** `minmax(420px,1fr)` with
`auto-fill` does not collapse below its own minimum, so the featured grid forced 54px of horizontal
scroll at 390px — fixed with `min(420px,100%)`. And the new headline at 1.9rem ran six lines on a
phone and pushed "View the work" below the fold, defeating the point of an actionable first screen;
1.6rem on mobile puts the CTA at y=736 against an 844px fold.

## 2026-07-29 — Four new workstreams on TODOS, and a CTA that did nothing

**TODOS §13** now carries Cam's four current asks, each scoped against what the code actually does
rather than against the slogan:

*Decouple CashFlux.* The coupling is six-deep, and the `go.mod` line is the one that matters: this
module **requires** CashFlux and replaces it with `../CashFlux`, and CashFlux still pins **GWC /v4
v4.2.0** while this site runs **/v5** — one build graph, two GWC majors. Under that sit
`internal/budget/`, the hardcoded `C:/Users/mreca/Desktop/CashFlux` in `scripts/build-cashflux.sh`,
a third checkout cloned by `install.sh`, **CashFlux's `SyncService` running inside this server's
`/grpc` tunnel**, its entire control plane in this wasm admin console, and ~120 MB of release
directory. The target is the shape ArticleFlux already has: its own host, linked out to. Two
decisions went to §0 (does the portfolio keep the CashFlux admin panels; what hostname).

*Terminal testing.* `client/` is 9 files with **zero Go tests**, and `e2e/` only drives CashFlux
auth/sync. The parser, pipes, vfs and completion are pure logic and testable natively — that's the
cheap half, and it isn't done.

*CSS and mobile.* `internal/site/site.go` has **5 breakpoints and 19 `css.Raw` escapes**;
`client/*.go` has **zero breakpoints**. The terminal — the centrepiece — does not respond to width
at all, and neither does the admin console.

*Copy and links.* Framed around the failure mode this session already produced twice: a link that
resolves but dead-ends. Includes a CI link-check, because that class of bug is silent.

**The launch CTA was decoration.** "▶ Launch the live terminal" was a `<div>` with `cursor:pointer`
and no handler attached to anything. The reason it was never wired: it is server-rendered in
`internal/site`, above `#term-root`, so it lives outside the wasm tree and can't be given a handler
at render time. Fix is a DOM binding from inside the component on mount, with the ID as the contract
between the two, plus cleanup on unmount. It now expands *and* focuses — expanding without focus
hands someone a shell that ignores their typing. Verified in the browser: after the click
`document.activeElement` is `term-input`, the terminal is 1230×810 in a 1280×900 viewport, and typed
input runs. Also switched `<div>` → `<button>` so it is keyboard-reachable at all.

**Profile scrub.** No city on the site: hero eyebrow, `notes/about.md`, and the résumé's `Location`
all read "Remote", and the contact address is `cam@earlcameron.com` everywhere (5 call sites). The
résumé mattered most — it gets downloaded and forwarded by strangers. **Still present:** Florida
International University and Miami Dade College in the education section, which narrow him to South
Florida. Those are credentials rather than a location line, so they were left alone — flagged for
Cam rather than silently stripped.

**Footer.** `Border(theme.Border)` sets four sides, so the footer was a box floating under the page
— cards are boxes here, endings are not. Now `css.BorderTop(css.Px(1), theme.Border)` on a real
`<footer>`, with the credit line turned into evidence (both frameworks link to source) and one
accent note, "zero npm". The separator before it was dropped after the mobile screenshot showed the
line wrapping so that a bare "·" started the second line.

## 2026-07-29 — ArticleFlux as a first-class link, and the URL that would have dead-ended

Cam asked for ArticleFlux to sit alongside CashFlux as a first-class destination, pointing at
`https://feed.earlcameron.com/`. It now appears in exactly the two places CashFlux does: the top nav
(`articleflux`, new tab, next to `cashflux`) and an `~/elsewhere` card. It is deliberately *not* in
the terminal's `links` — CashFlux isn't either, and parity is the point.

**The URL needed changing to do what he asked.** The host root resolves to the reader, which requires
an account: a visitor with no login lands on "Sign in to your reader — No account? Whoever runs this
server creates one with `articleflux adduser`". A recruiter following a first-class link to a login
wall is worse than no link. `/home` is the public front door — why it exists, reading, what it
learns, what leaves the box, how it is built, plus its own live-demo and source links — so the
constant `articleFluxURL` points there, with the reasoning on the constant so nobody "fixes" it back
to the root later.

The card copy says "my feed reader — running live", not CashFlux's "try it live". A visitor can see
it and read how it works; they can't actually use it without an account, and the copy shouldn't
promise what the sign-in form will refuse. The open question for Cam is whether to mint a read-only
guest account and upgrade both the copy and the link to the reader itself.

Side effect worth noting: `~/elsewhere` was five cards in a 3+2 ragged grid. Six makes it 3×2.

## 2026-07-28 — Re-cutting the featured grid, and what a nine-card grid costs

Cam asked to drop WhisperToMe, put ArticleFlux in its place, and lead with CashFlux, ArticleFlux,
GoWebComponents, GoGRPCBridge. ArticleFlux was already in `featured` — what he was looking at was a
**server binary built on 2026-07-25**, three days stale, still serving the old set. Worth remembering
when a content change "doesn't take": `scripts/dev.sh` rebuilds and restarts, and nothing else does.

The removal left eight cards, which is the wrong number: the desktop grid is
`repeat(auto-fill,minmax(260px,1fr))` inside a 1000px column, so it lands on three columns and eight
leaves a hole. Cam asked for a ninth.

**The candidate list turned into an editorial question, not a technical one.** Two path tracers exist
in his repos. `vkPathTracer` is his own — C++23, Vulkan/D3D12/Metal, Emscripten/WebGPU, Jolt physics,
Lua scripting, a deterministic headless benchmark mode. `pathtracer` is JS/WebGL and, by its own
README, a fork of Evan Wallace's demo — but heavily rebuilt (a dozen material models, Rapier physics,
SDF/CSG primitives, OBJ/STL/PLY/glTF/GLB import, scene tree and inspector, benchmark panel with
shareable score cards) and, decisively, it has a **live GitHub Pages demo**. Cam picked it on exactly
that ground: a recruiter can click it. Right call for the audience — a clickable artifact beats an
unclickable one — and the attribution problem is handled by crediting the original in the card's
`Long`, which is where the terminal's `open pathtracer` shows it. The repo README leads with the same
credit, so the disclosure is consistent in both places a reader can land.

**What the re-cut costs, recorded so it isn't rediscovered later:** WhisperToMe was the only
on-device-ML card. The hero copy says "on-device ML" and the résumé lists Snapdragon NPU / QNN / ONNX
Runtime GenAI / INT4, and there is now nothing on the grid behind either. `Gemma4-12B-SnapdragonX2Elite`
is public and would close it. Left open deliberately — nine slots, and Cam spent the ninth on reach
rather than on backing a claim. Logged in `TODOS.md` §0.

Two smaller things came in mid-pass: CashFlux had a GitHub Pages demo that its card never linked
(verified 200, added), and the projects section now carries `github.com/monstercameron ↗` on the
`~/projects · featured` eyebrow row. It sits on the eyebrow rather than under the grid because that's
the "where am I" line — a visitor scanning cards for repos shouldn't have to pass nine of them to
find the profile. It right-aligns to the content column on desktop and wraps under the shell path on
mobile.

**One false alarm worth writing down.** Driving the wasm terminal's `projects` command headless typed
`proets` — two dropped keystrokes. Not a bug: headless Chrome renders this page at roughly 0.4fps, so
`keyboard.type()` at full speed outruns the input handler. With `delay: 180` the command runs clean.
Any future terminal e2e needs that delay, or it will report phantom input bugs.

## 2026-07-28 — Deploy for real, and a v5 migration that was already overdue

Cam asked to get the repo ready for a DigitalOcean Ubuntu + Nginx droplet. The first thing recon
turned up was that **the repo did not build.** `go.mod` required `GoWebComponents/v4` and replaced
it with `../GoWebComponents`, but that checkout had moved onto the `v5` branch, whose `go.mod`
declares the `/v5` module path. So every build was failing on "no required module provides package
.../v5/…" before any deploy question mattered.

Cam's call was to go to v5. The migration itself was nearly nothing — v5 is explicitly additive,
and this repo only touches four GWC packages (`css`, `css/u`, `html/shorthand`, `ui`) across eight
files, so it was an import-path rewrite. **One real break:** v5.0.0's `css/u/exports.go` re-exports
seven names — `Track`, `Repeat`, `LinearGradient`, `RadialGradient`, `Stop`, `Circle`, `Ellipse` —
that `html/shorthand` already declares as element constructors. Five files here dot-import both, and
Go rejects that eagerly: `Track redeclared in this block`. The fix belonged upstream, not here: that
file's own header says its purpose is to let you dot-import `u` alongside `shorthand`, and it already
withholds names that would shadow (`Gap`, `Bg`, `Rounded`, `Border`). These seven were the same
oversight against a different package. Dropped them, released as **GWC v5.0.1**, pinned here.

**On the deploy design, two plan items got rejected on contact with the code.**

The plan said one `go:embed`-ed binary. It can't be, and shouldn't be: the server resolves
`web/static` and `web/cashflux` by CWD-relative path, and CashFlux's frontend is ~120 MB of wasm,
fonts and audio. What the single-binary plan was really buying was *atomicity* — code and frontend
swapping together. A release directory behind a symlink buys that directly. `WorkingDirectory` in
the unit points at the symlink, not a release, which is the whole trick.

The plan also said GitHub Actions push-to-deploy. Cam chose build-on-droplet, which removes the
problem the plan had flagged as undecided: the relative `replace` directives that don't exist in CI
*do* exist if you clone the three repos as siblings, which is what `install.sh` does.

**Things that would have broken a real deploy, found by dry-running the droplet layout** (clean
clones of all three repos into a scratch dir, then building):

- `scripts/build-cashflux.sh` copies `web/wasm_exec.js` out of the CashFlux checkout — and CashFlux
  **gitignores** it, because it's a toolchain artifact. A fresh clone has no copy, so the staging
  loop would `set -e` out on the first droplet build. Now falls back to `$(go env GOROOT)`'s copy,
  which is byte-identical and, on the droplet, is the one that actually matches the wasm just built.
- `build.sh` only ever emitted `bin/server.exe`. Now `bin/server$(go env GOEXE)`.
- Nginx needed the WebSocket upgrade on **`/grpc`**, not just `/socket` — `/grpc` is the embedded
  CashFlux sync engine's tunnel. The DEPLOYMENT.md sketch had only `/socket`, which would have left
  the budget app failing "Test connection" with nothing in the logs to explain it.
- `MemoryDenyWriteExecute=true` is the obvious systemd hardening line to add and would have been a
  live-site bug: `ncruces/go-sqlite3` runs SQLite as wasm through wazero, which allocates W^X pages.
  Left out, with the reason written in the unit so nobody adds it back.
- Windows git doesn't record the exec bit, so a cloned `update.sh` can land 644. `install.sh` invokes
  it through `bash`, and `.gitattributes` now pins `eol=lf` so a CRLF shebang can't reach Ubuntu.
- `update.sh` pulls this repo and then **re-execs itself once**. bash reads a script incrementally as
  it runs it; a pull that rewrites the running file resumes at a byte offset into new content.

**Not done:** nothing backs `data/` up off-box. It's the only irreplaceable thing on the droplet.
Logged in TODOS §11.

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

## 2026-07-24 — Activation codes for the embedded CashFlux

**Why.** Cam: "the server makes an activation code, the client uses the code and it takes the
creds from the privacy lock. this locks the feature to me only as I am the only one with access
to the portfolio site." The device-pairing flow shipped earlier today has the device ask and the
admin approve; he wants it the other way round, and he wants to type no credentials at all. The
security model is exactly "who can reach `/admin`" — which, for a site only he logs into, is him.

**What.** One new RPC (`MintCashFluxActivationCode`) over CashFlux's new
`pkg/embed.Admin.MintActivationCode`, and a panel that leads the cashflux tab: Generate code →
6-digit code in the existing accent callout with expiry + copy. Every code binds to a single
fixed owner account (`device:owner`), created on the first mint, so activating a phone and a
laptop puts them in the same dataset instead of two islands. Pending-device pairing stays for
the multi-person case; it just isn't the headline any more.

**The part that mattered.** The RPC and the button were an hour; the reason nothing had ever
synced was three bugs on CashFlux's first-sync path (see CashFlux's DEVLOG for the detail — an
unseeded first upload, a `crypto.randomUUID` invoked detached which panics the whole wasm app,
and a blob-before-workspace ordering deadlock). All three end with the sync chip reading
"Synced", which is why the symptom read as "connected but nothing transfers". Fixed in CashFlux,
verified here end to end on a throwaway port + data dir rather than against Cam's live instance.

**Ops note.** The live dev server was restarted onto the new binary with dev.sh's env
(`LISTEN_ADDR`, `BASE_URL`, `CASHFLUX_SERVER_TOKEN=cam-sync-token`, `ADMIN_*`) — the `BASE_URL`
and token pins from yesterday's incident still matter and were preserved. Logs now go to
`server_restart3.log`.

## 2026-07-24 — Deleting CashFlux users from the console

Cam: "I need to be able to delete users from the portfolio site." One RPC over CashFlux's
new `pkg/embed.Admin.DeleteUser`, plus the row-level UI.

**Two-step, no modal.** The row swaps into its own are-you-sure state. The confirmation
names what is destroyed ("erases their workspaces, transactions, and attachments, and signs
out every device they activated") rather than asking a generic "are you sure?" — a
confirmation you can't read tells you nothing you didn't already know.

**The owner row is called out twice.** `device:owner` is the account every activation code
opens, so deleting it erases Cam's own CashFlux data and signs out every device he
activated. It carries a "your account" badge in the list, before anyone reaches for the
button, and its confirmation says so explicitly. The flag is computed server-side against
`cashfluxembed.OwnerAccountID` — the client is never asked to guess which id is special.

Deliberately still possible, not blocked: he asked for delete, and a console that refuses
the one row you most need to clear when starting over is worse than one that warns properly.

`usersSection`/`userRow` grew handlers, so the users params moved into a `usersPanelState`
struct rather than adding four more positional args to `cashfluxView` — same treatment
`activationCodeState` got.

Verified against a real synced account, not an empty one, and with Cancel exercised before
the real delete.

## 2026-07-24 — Stats that move

Cam: "I dont see any updates on the server side stats." He was right, and the panel was the
liar again — a pattern for the day. Neither figure it showed CAN reflect sync:

- "requests this month" counts metered AI calls. `AddUsage` is reached from exactly two
  places, the AI proxy and a dev-seed helper. Nothing in the sync path touches it. On this
  deployment — no AI, sync only — it is structurally 0 forever.
- the database's own page count barely moves. His real dataset is 17KB; the file is 284KB.

So the console showed a billing metric and a rounding error, and he read the obvious
conclusion: nothing is arriving. The data had been arriving the whole time.

Fixed by reporting what an account actually HOLDS: snapshot bytes, workspace count, last
push. Snapshot bytes is the only one of the three storage figures that changes on every
push, so it leads the panel; the database file moved to last.

Two small calls worth keeping. A never-synced account renders "—" and "never synced", not
"0 B" — "0 B" reads as "synced, and empty", which is a much more alarming claim than "hasn't
synced". And the zero time is sent as literal 0, because `time.Time{}.Unix()` is
-62135596800 and would have rendered as a date in year 1.

## 2026-07-24 — Authn/authz build-out

Implemented phases 1–4 of the auth plan: roles, account management, set-a-password, and the
one-credential handoff. CashFlux side is in its own repo; here it is the RPCs, the console
controls, the gate change, and the test rig.

**The gate stopped being a second password.** It now recognises a visitor arriving with a
live activation code — one this console minted moments earlier from an authenticated session
— and opens on that basis. Peeking at the code rather than consuming it matters: a consuming
check would let anyone burn a legitimate code by visiting a URL.

**Test rig split.** `harness.mjs` holds the hermetic rig and the helpers; `sync-flows.mjs` and
`auth-flows.mjs` are scenario files. Two of the failures during this work were the suite's own
bad assertions — sampling a sync state once when it settles, and counting a control that
appears with the signed-in state rather than with the page. Both now wait. A flaky suite
spends its credibility fast, and these had already earned some by catching two real bugs.

**Known gap, stated plainly:** the e2e suites do not prove viewer/suspension enforcement.
Reaching a second IDENTITY requires the device-initiated pairing flow, which did not cooperate
in the harness. That enforcement is covered by Go tests driving SyncService directly — the
authoritative level for it — but the end-to-end gap is real.
