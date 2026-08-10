# DESIGN — earlcameron.com (mid-2026)

The design record. Captures **what we decided and why**, so the reasoning is reusable and we
don't relitigate. Reflects Cam's requests through 2026-07-21.

---

## 1. The thesis

A portfolio that **is** a working terminal. Not a fake terminal printing canned text — a
**friendly shell where commands launch real programs**, and the realness is the point. The
status bar's `● connected · N online` is honest telemetry from a gRPC stream, not decoration.

The single fact a sharp reader remembers: **this whole thing actually runs** — hand-built Go
from the browser to the backend. The medium is the résumé.

## 2. Positioning — AI-first, told honestly

Cam is an **AI-native systems engineer**. The brief, in his words: *"I move smarter and
faster with the LLM… I'm not cracked but with the LLM I'm an excellent and quick engineer."*

How we frame that, and the two ditches we avoid:
- **Not self-deprecation** ("just a guy with ChatGPT") — false, and the architecture disproves it.
- **Not grandiosity** ("10x cracked savant") — he disclaimed it, and inflated claims read as
  insecurity to the people worth impressing.
- **The true, compelling middle:** *systems judgment × LLM leverage = ships ambitious, real
  things fast and well.* Own "AI-first" plainly — in 2026 most engineers are cagey about it,
  so stating it confidently is a **differentiator**. Then **show, don't boast**: the proof is
  the site (his framework, his gRPC bridge, his backend) plus the breadth of shipped work,
  which *is* what velocity looks like.

Draft line (tune the words): *"I know what to build, and I use every tool I have — LLMs
included — to build it well and quickly. With the model in the loop I operate like a small
team: same taste, a fraction of the time."*

## 3. The two front doors

One page, two audiences (Cam optimizes for **both** — technical peers and recruiters):

1. **Standard site** — clean conventional portfolio (identity, featured work, contact).
   **Server-rendered HTML** on the ingress request → also the SEO surface, the **no-WASM
   failsafe**, and the primary **mobile** experience. Zero runtime comms.
2. **Friendly terminal** — the enhancement layer for the curious. Boots WASM, opens the
   gRPC-over-WS channel, runs real programs. Where the juicy details live.

**Coexistence (decided):** a **unified hero** — a real, working prompt in the hero.
**Type → the terminal grows to fullscreen. Scroll → the standard site.** No gate, no extra
click, terminal discoverable to everyone and forced on no one.

**Rule we hold:** the standard site must fully stand on its own. If content lives *only* in
the terminal, that's a bug — a recruiter who never types still gets identity + top work +
contact.

## 4. Interaction model

- **REPL scrollback** for quick things (`about`, `ls`, `help`, `clear`, `theme`).
- **Full-screen TUI takeovers** for rich things (`projects` browser, `contact` form) — like
  running `htop`/`lazygit`; `q`/Esc returns to the prompt. This is what sells "real programs."
- **Friendly, not elitist:** ghost-text autocomplete (`pro`→ghosted `jects`), Tab-complete,
  up/down history, **every command also clickable** (hints, cards, list rows "type" it for
  you), forgiving parser (`command not found` → "did you mean `projects`?").

## 5. Mobile — change the metaphor, don't shrink it

Typing shell commands on a phone is genuinely miserable, so mobile is **not** a squished
terminal:
- **Lead with the standard site** (touch-first, thumb-scroll, big tap targets). This *is* the
  "easy mobile view" Cam asked for.
- The terminal becomes **tap-driven**: a bottom bar of **command chips**
  (`about · projects · contact · ask`) and tappable output. Same programs, same gRPC —
  poke instead of type. A real keyboard input stays available but secondary.
- Desktop = terminal-forward. Mobile = content-forward with a tap-terminal.

## 6. Visual language

### Chrome
**macOS Terminal** — traffic-light buttons (close = exit terminal, green = fullscreen),
translucent blurred window, SF Mono. (Cam chose this over Windows Terminal.)

### Palette — "Aubergine" (Ubuntu-souled, dark & moody)
A macOS window running an **Ubuntu** color profile. Grounded in a real, recognized brand
vernacular → distinctive without the overused Dracula/Tokyo-Night look. Dark and moody per
Cam's ask ("Ubuntu's orange and purple").

| Token | Dark (default) | Role |
|---|---|---|
| `--bg` | `#17040F` | deep near-black aubergine ground |
| `--bg-2` | `#210A19` | raised surfaces |
| `--fg` | `#F3E9E6` | warm off-white text |
| `--dim` | `#A98BA0` | muted mauve |
| `--border` | `#3A1B2E` | aubergine-tinted (not grey) |
| `--accent` | `#E95420` | **Ubuntu orange** — prompt, cursor, CTA, active |
| `--accent-2` | `#BE7BE6` | **purple** — links, secondary highlights |

ANSI: green `#8AE234` · red `#EF5350` · yellow `#F2B840` · cyan `#4DD0E1`. Ambient moody
glows: purple top-left, orange ember bottom-right.

`theme` is a **real command** (a feature terminal people love): **aubergine · light ("paper
terminal") · nord**. The artifact also honors the viewer's OS light/dark; the in-app theme
overrides it.

### Typography — two voices
- **Mono** (`SF Mono`/system stack) = **the machine's voice** (terminal, labels, headings,
  eyebrows). No webfont — a real terminal uses the system mono, and it's CSP-safe.
- **Humanist sans** (`-apple-system`…) = **Cam's voice** (standard-site body prose).
- The two front doors literally speak differently — a *meaningful* structural choice, not
  decoration.

## 7. Motion — entice with a few earned moments, not confetti

Restraint is the strategy (scattered effects read as AI-generated). The signature moments:
1. **Boot → neofetch → live-prompt** load sequence — the hook. Honest: it narrates the real
   connection (`dialing /socket… tunnel established`). If the tunnel fails → `[!!] offline —
   serving static cache` and it degrades.
2. **Expand-to-fullscreen** — typing grows the hero prompt into the full terminal in one
   continuous motion. The signature transition.
3. **Live streaming** — gRPC output and `ask` responses type in as bytes arrive. Real motion.
4. Quiet micro-interactions: accent-glow on focus, ghost-text slide-in, hover on commands.

Everything collapses to instant under `prefers-reduced-motion`.

## 8. Technical showcase = self-selling

The architecture is a headline feature, made **visible**:
- `neofetch` declares the site is rendered by GoWebComponents (his framework), over gRPC.
- `arch` draws the request path: browser → GoGRPCBridge (WS↔gRPC) → Go gRPC server.
- "How this site works" strip on the standard site names the stack, top to bottom.

## 9. Constraints (hard)

- **gRPC + WebSocket only for browser↔server comms. The only HTTP is the ingress** (first
  page load: HTML, wasm, fonts/css). All app data flows over the WS tunnel.
- Server→OpenAI (for `ask`) is **backend egress**, same as server→SQLite — it does **not**
  violate the no-HTTP rule (that rule governs the browser↔server channel).
- **Own stack, dogfooded:** GWC client (v4.3.0, tracked via local `replace`), GoGRPCBridge,
  Go gRPC backend. Being technically interesting *is* a requirement.

## 10. Smart features — nano assistant, on a leash

`ask <question>` (and the friendly unknown-command fallback) = a shell assistant grounded in
Cam's content, streamed from a rate-limited OpenAI **nano** model. **Terminal-only** (the
standard site stays zero-comms, which also bounds cost surface).

**The reality:** under honest use the $20/mo is nearly impossible to hit — nano tokens are
cheap; a real visitor costs fractions of a cent. **The risk is abuse, not honest use:**
(1) someone scripting the endpoint as free ChatGPT; (2) a single abuser draining the month in
an hour. So the guardrails aren't polish — they're the feature:

- Server-side **spend ledger** (SQLite) → hard stop at $20/mo + **daily sub-cap (~$0.65)**.
- **Per-IP + per-session rate limits** (token bucket) — the real abuse defense.
- Tight per-request caps (cheapest nano, `max_tokens ~200`, truncated context, low temp).
- **On-topic-only, injection-resistant** system prompt (declines off-topic → kills the
  free-ChatGPT value and jailbreak attempts).
- **Answer cache** for common questions (repeats cost $0).
- **Graceful degradation** → deterministic content commands when budget/rate is spent, with a
  friendly "smart mode's resting — out of budget this month." The site never breaks.

## 11. Open decisions

See [`../TODOS.md`](../TODOS.md) §0: featured-project shortlist, any extra first-class
commands (writing/now/guestbook), final AI-first copy, résumé asset, **curated i18n
languages**, **BYOK run-location** (client-direct vs proxy-over-gRPC), **résumé-tailor cost/
auth model**, **live-CashFlux integration level** (host-wasm vs full-backend), and
**admin-auth method** (recommend WebAuthn/passkey).

## 12. Internationalization & BYOK translation (2026-07-21)

Two tiers, and — importantly — **only one is on Cam's bill.**

### Tier 1 — curated locales (free, quality, runtime cost $0)
A real i18n layer on GWC's **typed message bundles** (`gwc i18n gen` — a typo'd key/namespace
is a compile error). English is the base bundle; a handful of Cam-chosen languages get
reviewed bundles, **pre-generated once at build** (nano first pass, then spot-reviewed). These
**SSR per-locale** → SEO-visible and no-WASM-safe. Content strings (about, projects) are
served per-locale by `ContentService`; UI chrome strings come from the bundles. Command
*names* stay English (it's a CLI); their descriptions/output localize. Locale from
`Accept-Language` on ingress + a `lang` command + persisted preference.

### Tier 2 — BYOK on-demand (the long tail, visitor-funded)
Any language not pre-bundled: the visitor **brings their own OpenAI key** and the latest nano
translates the page live. The page **rewrites itself section-by-section as it streams** (a
motion moment that is the feature working in the open). Terminal-native:
`lang` · `translate <lang>` · `byok <key>` (masked) · a live **cost meter**
(`translating → Jamaican Patois · you've spent $0.0041`).

### Cost story (state it plainly)
Two AI surfaces, one bill:
- **`ask`** → Cam pays, hard-capped **$20/mo** (§10).
- **Translation** → the **visitor** pays via BYOK; **Cam pays $0 at runtime**, for any number
  of languages. Keeps the budget bounded no matter how popular translation gets.

### The pay-it-forward mechanic
**Cache every BYOK translation server-side, keyed by `(source-hash, lang)`.** The *first*
person to want a language spends a fraction of a cent with their key; **everyone after gets it
free.** Community BYOK spend slowly builds a free translation corpus. "You pay" becomes "you
pay it forward." Curated locales are authoritative; community-cached ones are labeled
*machine, best-effort* with a re-translate option.

### Run-location fork (DECISION NEEDED — recommend proxy-over-gRPC)
- **Client-direct** (WASM→OpenAI): key never touches Cam's server (best for trust/liability),
  **but** it's a browser→third-party HTTP call (dents the no-HTTP-except-ingress thesis) and
  **OpenAI CORS may block browser calls** — verify before committing.
- **Proxy-over-gRPC** *(recommended)*: key rides the existing WS/gRPC tunnel; the Go server
  calls OpenAI with the *visitor's* key. Preserves the "browser only speaks gRPC/WS" thesis,
  dodges CORS, reuses `ask` streaming. **Cost:** the key transits the server → must be
  **request-scoped, never logged, never persisted**, and said so in the UI. **Abuse-proofing:**
  the server only translates **its own known page strings, never arbitrary client text** →
  useless as a general proxy, safe by construction.

### Reality checks
- Nano translation **quality is uneven** on low-resource languages — label curated =
  *reviewed*, BYOK/community = *machine, best-effort*; offer re-translate. Don't oversell.
- **Key safety:** TLS on ingress, ephemeral in memory, never in logs/DB — stated in plain words.
- **Casual path stays simple:** standard-site gets a plain **curated-language dropdown**; BYOK
  is a terminal power-user toy. Never make a recruiter paste an API key.

### New service
- `TranslationService` — `Translate(stream)` (BYOK, server-stream so the page rewrites live);
  server pulls known source strings, uses the caller's key, writes through the `(hash,lang)`
  cache. Terminal-only for BYOK; curated locales need no service call (bundled/SSR).

### Signature suggestion
Ship **Jamaican Patois** as a curated locale — distinctive and personal to Cam's heritage,
and a delightful thing to discover via `lang patois`.

## 13. Carried-over & new app features (2026-07-21)

### 13.1 Two "planes" — clarifying the no-HTTP rule
Adding blog/RSS/résumé/CashFlux forces an honest clarification. There are **two planes**:
- **Data plane (app comms)** — browser ↔ portfolio server. **gRPC-over-WS only**, no REST. Unchanged.
- **Document plane (published resources)** — HTTP **GET** of documents/assets: SSR HTML pages,
  wasm bundles, the **RSS feeds**, the résumé PDF direct link, sitemap/robots, the CashFlux
  iframe. These are ingress/static resources and always were fine. RSS *cannot* be gRPC (feed
  readers only speak HTTP), so this boundary is real, not a loophole. The rule means: **no
  ad-hoc REST API for the app**; published documents are still served over GET.

### 13.2 Résumé + agentic tailoring
- **Canonical résumé**: one page, downloadable **PDF**, authored from real experience. `resume`
  in the terminal / a "Download résumé" button on the site → the **static PDF**, instant, $0, no AI.
- **Agentic tailoring**: paste a job posting → an agent produces a **tailored** one-page variant
  (reordered/emphasized bullets, adjusted summary, matched keywords) → renders a PDF the visitor
  **downloads in-browser** (Blob). `resume tailor` / a "Tailor to a job posting" widget.
- **PDF generation: server-side in Go**, streamed over gRPC, download via Blob URL. Keeps it
  Go-pure (no jspdf/JS) and on-model. A fixed Go template guarantees **one page**, deterministic
  layout — the agent only fills slots.
- **Hard guardrail (non-negotiable):** the agent may only **reorder, emphasize, and rephrase
  existing facts**. It must **never fabricate** experience, titles, dates, or skills. A résumé
  tool that lies is a catastrophe. Tailor from a fixed fact-set; never add claims.
- **Cost/auth (DECISION):** primarily *Cam's* tool (tailor before I apply). Options: owner-only
  behind light auth (bounded cost) · public demo on the shared $20 budget + rate limits · BYOK.
  Recommend: **owner-gated**, with an optional rate-limited public demo. (Tailoring is
  longer-output than `ask` → eats budget faster; don't leave it ungated on Cam's dime.)

### 13.3 Blog (carried from PersonalWebsite2026)
- `BlogService` (gRPC): `ListPosts`, `GetPost`; admin `Create/Update/Delete` behind auth.
- Markdown → HTML + syntax highlighting. Terminal: `blog` (list) · `read <slug>` (TUI reader).
  Standard site: `/blog`.
- **RSS feed** = document-plane HTTP GET (`/blog.xml`).

### 13.4 Slack anime RSS (carried)
- A Go **cron** checks anime releases and publishes an **RSS feed** (document-plane GET). Optional
  Slack post. Terminal: `anime` shows tracked shows / latest. Personality feature; low priority.

### 13.5 Live CashFlux instance
Cam: *"serve the cashflux wasm and impl the cashflux backend inside the server so we have our
own live instance running."* Two levels — recommend shipping L1 first:
- **L1 — host the live WASM app (recommended first).** CashFlux is local-first (client-side,
  IndexedDB), also built on GWC. Serve its wasm bundle under a route (`/apps/cashflux`), launched
  by a `budget`/`cashflux` command or a project "Launch live demo" button, in an **iframe overlay**
  (isolates its own runtime/DOM). Seed demo data. Genuinely live, visitor runs it, **$0 runtime**,
  low integration cost. Lazy-load only on launch (it's a big second wasm — mind the payload and
  the [wasm build race / deploy hygiene] from CashFlux notes). Keep it a **separate wasm build**;
  no shared runtime with the portfolio (portfolio = GWC v4.3.0, CashFlux = its own pinned v4).
- **L2 — integrate CashFlux's Go backend (later, heavy).** For server-side features (multi-device
  sync, shared/cloud state), fold CashFlux's server components into the portfolio server. This
  **couples two large codebases** and is a real project — defer until L1 is live and there's a
  concrete reason. Flagged as the single heaviest item in the whole plan.

### 13.6 Scope reality (reality-anchor note)
The plan now spans: terminal + standard site, gRPC backend, nano `ask` assistant, i18n + BYOK
translation, résumé + agentic tailoring, blog + RSS, anime cron, and a live CashFlux instance.
That is a **large, multi-phase build**, not a weekend. It's all coherent and worth building —
but it should ship in **phases** (see TODOS §9), not all at once, or it stalls. MVP first:
portfolio + terminal + résumé download + contact. Everything else layers on.

## 14. Guest vs admin (2026-07-21)

Two roles from one codebase. **Guest** = the default visitor (terminal + standard site,
unauthenticated). **Admin** = Cam, authenticated, with live control of settings, stats, and
content — **so most changes never require SSH.**

### On-theme: admin is `login` + `sudo`
No bolted-on `/admin` dashboard fighting the aesthetic. Admin lives *in the terminal*: `login`
authenticates; authed **admin programs** then appear (`settings`, `stats`, `inbox`, `posts`,
`budget`, `flags`, `cache`, `logs`, `deploy?`). The rich ones open a **full-screen TUI
dashboard** — panels for stats/settings/content/inbox, `htop`/`k9s`/`lazygit` style — with
forms where a form beats a command line. It's a genuinely cool admin console *because* it's a TUI.

### Safety net: an SSR fallback admin
Mirroring the guest two-front-doors idea, admin also has a **plain server-rendered `/admin`**
behind the same auth. If the wasm ever breaks, Cam can still get in and fix things. Robustness,
not decoration.

### What's editable live (no SSH) vs what still needs a deploy — be honest
**Live-editable in-browser:**
- **Settings/config**: contact email, theme defaults, rate-limit knobs, AI budget cap.
- **Feature flags**: toggle `ask` / translation / anime / CashFlux / maintenance mode at runtime.
- **Content**: projects (add/edit/reorder/feature), about copy, blog CRUD, résumé data, curated
  i18n strings, anime tracking list.
- **Moderation**: contact inbox, guestbook (if any), community-translation cache (approve/purge).
- **Runtime ops**: purge caches, re-seed the CashFlux demo, run the anime cron now, view logs/stats.

**Still needs the deploy pipeline (state it plainly):** shipping new **code** or **schema**
changes, and rotating **secrets** (OpenAI key, signing keys). You can't hot-swap Go safely from a
web panel, and editing live secrets through a browser is a bad idea — those stay in env/deploy.
So: *everything short of new code or secret rotation is live.*

### Persistence & propagation
A `settings` table (SQLite) read **live** by the server (no restart); flags are runtime toggles;
content edits hit the DB and `ContentService` serves them immediately.

### Security — this is the sharp edge (reality-anchor)
Admin can mutate the live site, so treat it as the real attack surface it is:
- **Auth (DECISION, recommend WebAuthn/passkey):** phishing-resistant, no shared secret.
  Fallback/alt: strong password **+ TOTP 2FA**. Never a lone env password.
- Auth rides **authed gRPC metadata** (like ai-chat-wizard's interceptor); **TLS ingress
  mandatory**; short-lived JWT + refresh; **logout / revoke all sessions**.
- **Rate-limit + lockout** on failed logins; no username enumeration; constant-time compare.
- **Audit log** (append-only): every admin change = who/what/when. Non-negotiable once a browser
  can change prod.
- **Destructive actions behind explicit confirm** (delete post, purge cache, wipe inbox).
- Least privilege; validate every admin input server-side (never trust the client).

### New service
- `AdminService` — authed RPCs for settings/flags/content/moderation/ops + `StreamStats` (live
  telemetry for the dashboard). Every mutation writes the audit log.

## 15. Engineering principles (2026-07-21)

Captured here; enforced via `AGENTS.md` / `CLAUDE.md`.
- **Two planes**: data plane = gRPC/WS only; document plane = HTTP GET (SSR, wasm, RSS, PDF, the
  CashFlux iframe). See `PROJECT_LAYOUT.md`.
- **Tool routing**: terminal programs are **local** (WASM-only) by default; **backend-routed**
  the moment CORS, a secret, a cross-origin fetch, or a server capability is involved — `ask`,
  `translate`, `contact`, `stats`, `blog`, `resume tailor`, `admin`, `anime`. Never expose a key
  or bypass CORS from the client.
- **Owner-only admin, server-enforced**: hiding commands client-side is UX; authorization is on
  every AdminService RPC. WebAuthn/passkey, audit log, least privilege (§14).
- **Strong shared types + documented funcs**: the generated `proto` package is the single DTO
  source of truth shared by the Go server and Go/WASM client — no `map[string]any` between
  layers; every function/method (exported *and* unexported) carries a doc comment.

## 16. Design tokens (code) — `internal/theme`

Colors live **once** in **`internal/theme`** (Go tokens, typed `css/u` colors) and are used across
all UI — SSR site, terminal, admin. **Never scatter ad-hoc `u.Hex()` values; import the token.** A
palette change is then one edit.

| Token (`theme.`) | Hex | Role |
|---|---|---|
| `Bg` | `#17040F` | deep aubergine ground |
| `BgRaised` | `#210A19` | raised surfaces (cards, panels) |
| `Fg` | `#F3E9E6` | warm off-white text |
| `Dim` | `#A98BA0` | muted mauve — secondary text |
| `Faint` | `#6F5364` | hints, placeholders |
| `Border` | `#3A1B2E` | aubergine-tinted border |
| `Accent` | `#E95420` | Ubuntu orange — prompt, cursor, CTA, active |
| `Accent2` | `#BE7BE6` | purple — links, secondary highlights |
| `Green` | `#8AE234` | success / status |
| `Red` | `#EF5350` | error |
| `Yellow` | `#F2B840` | warning |
| `Cyan` | `#4DD0E1` | info |

### Per-product brand tokens (`theme.Brands`, `theme.Plate`) — 2026-08-09

The Flux products have their own commissioned artwork, and the `/projects/<slug>` pages tie to
it. Two things vary per product and nothing else does:

| Slug | `Accent` | `Tint` | Sampled from |
|---|---|---|---|
| `articleflux` | `#4D84FF` | `#9AB6FF` | poster blue |
| `cashflux` | `#3FBE86` | `#8BDCBA` | poster green |
| `codeflux` | `#7D6BFF` | `#B3A8FF` | poster indigo |
| `pixelflux` | `#A86BF0` | `#E56FC8` | poster purple→magenta |
| `schemaflux` | `#3FC9C0` | `#8EE2DC` | poster teal |

Plus `Glow`, an `rgba()` string for the page's ambient background wash, and `theme.Plate`
(`#F6F4FA`) — the light surface the artwork sits on, because the posters are drawn for a near-white
ground with a dark navy wordmark and would lose half of each lockup on the aubergine page.

**The rule this does not break.** The palette stays Aubergine: ground, raised surfaces, borders, both
type voices and the single `rise` gesture are identical on every page. A product page must still read
as earlcameron.com — a visitor who clicked a project card should never wonder whose site they are on.
Accent values are brightened from the poster samples until they clear 4.5:1 against `Bg`; the poster
values are tuned for white and go muddy on aubergine. Never introduce a sixth brand without adding
its `theme.Brands` entry — `buildProjectPages` fails at boot rather than rendering a transparent
accent, which is invisible text rather than a visible error.

Spacing, radii, and font sizes are **not** re-tokenized — use `css/u` defaults directly
(`Spacing2..10`, `RadiusLg`/`RadiusXl`, `TextSm`, `FontSize(Rem(n))`, and `Md(...)` for the 768px
breakpoint).

---

*Interactive prototype: [`../design/mockup.html`](../design/mockup.html) (static mock; the
shipped site is GWC). Palette, motion, and copy are all tunable from here.*
