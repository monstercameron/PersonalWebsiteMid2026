# DEPLOYMENT — Ubuntu + Nginx on DigitalOcean

> **Status: BUILT.** `deploy/` holds the real scripts. Goal, in Cam's words: **one-click to
> install, one action to update — without racking my brain.**

## ⚡ Current flow: Docker, à la AnimeFeedFlux (2026-08-16)

The sections below this one describe the ORIGINAL on-box-build flow; it still works and is
the documented rollback, but the current deployment is a container, built in CI, pulled by
the box — the same shape AnimeFeedFlux proved end-to-end (including its failure path) on
2026-08-16:

```
tag vX.Y.Z on main ──► Release workflow (.github/workflows/release.yml)
                          verify (3-repo workspace, build/vet/test)
                          build deploy/docker/Dockerfile ──► ghcr.io/monstercameron/personalwebsitemid2026:vX.Y.Z
                          POST https://earlcameron.com/internal/deploy-hook  (X-Ec-Deploy-Token)
                                    │
Droplet: nginx ──► webhook daemon (127.0.0.1:9309, shared with AFF) ──► ec-autoupdate.sh
                          newest v* tag vs compose.yaml pin, image published?
                          └─► ec-deploy-release.sh: pin → pull → up -d → BLOCK on healthcheck
                              (.previous-tag recorded first; daily timer = lost-delivery fallback)
```

What changed and what did not:

- **The droplet stops building.** No Go toolchain, no two-wasm-binary OOM risk, no swap
  dependency — the Release workflow builds the image from the same three-sibling workspace
  ci.yml already assembles (refs still read from `deploy/lib.sh`, one source of pins).
- **The runtime contract is untouched.** Container publishes to `127.0.0.1:8095` (nginx
  config unchanged), bind-mounts the existing `/opt/earlcameron/data` at `/data`, and reads
  the same `/opt/earlcameron/.env` (compose re-asserts the three container-path keys:
  `LISTEN_ADDR=:8095`, `DB_PATH=/data/site.db`, `CASHFLUX_DATA_DIR=/data/cashflux`).
- **Rollback is two commands.** `sh ec-deploy-release.sh $(cat /opt/earlcameron/.previous-tag)`
  rolls to the prior image; the legacy systemd flow below is the deeper fallback
  (`systemctl start earlcameron` after `docker compose down` — same data files either way).
- **The promote-doesn't-deploy trap (§ below) is retired for real deploys**: releases key
  off tag pushes, which a human makes with a real token, not off `workflow_run` chains that
  `GITHUB_TOKEN` pushes never start.

Pieces: `deploy/docker/{Dockerfile, compose.yaml, ec-deploy-release.sh, ec-autoupdate.sh,
ec-autoupdate.service, ec-autoupdate.timer, webhook-site.conf.example}`. Secrets:
`/etc/earlcameron/deploy-hook-secret` on the box, mirrored to the repo's
`EC_DEPLOY_HOOK_SECRET` actions secret. Dry-run the image without tagging via the
workflow's `workflow_dispatch` (build, no push, no poke).

```
Internet ──443/TLS──► Nginx (reverse proxy, Let's Encrypt)
                         │  proxy_pass 127.0.0.1:8095
                         ▼
                   earlcameron (Go binary, systemd)
                     ├─ /socket   gRPC-over-WebSocket   (this site's data plane)
                     ├─ /grpc     gRPC-over-WebSocket   (embedded CashFlux sync engine)
                     └─ /         wasm + SSR + RSS + PDF (document plane)
                   SQLite in /opt/earlcameron/data — outside every release
```

## What we ship, and why it is not a single binary

The original plan here was one `go:embed`-ed binary. **It isn't**, deliberately: the server
resolves `web/static` and `web/cashflux` by *working-directory-relative* path
(`internal/server/server.go`, `internal/budget.DefaultRoot`), and CashFlux's frontend alone
is ~120 MB of wasm, fonts and audio. So a release is a **directory** — binary plus its
assets — and the atomic unit is the symlink that points at it. Swapping code and frontend
together is the property that matters; one-file-ness was only ever a means to it.

## The droplet builds from source

`go.mod` pins GoWebComponents and CashFlux through **relative `replace` directives**
(`../GoWebComponents`, `../CashFlux`). Neither is consumable from the module proxy at the
pinned state, so a build needs all three checkouts side by side:

```
/opt/earlcameron/
  src/
    PersonalWebsiteMid2026/     ← this repo            (branch main)
    GoWebComponents/            ← replace target       (tag v5.0.1)
    CashFlux/                   ← replace target       (branch main)
  releases/<utc>-<sha>/{server, web/static, web/cashflux}
  current -> releases/<...>     ← the symlink systemd runs from
  data/{site.db, cashflux/}     ← never touched by a deploy
  .env                          ← chmod 600
```

The cost is a Go toolchain and ~2 GB of RAM on the box. The benefit is that updating is
`git pull` and nothing else — no CI, no artifact registry, no deploy key.

> **GWC is pinned to `v5.0.1`, and the version is load-bearing.** Five files here
> dot-import `css/u` and `html/shorthand` together; in v5.0.0 `css/u` re-exported seven
> names that `html/shorthand` already declares, so those files did not compile at all.
> v5.0.1 is the release that fixes it. Do not move the pin backwards.

## Two flows

### 1. First install — once, on a fresh droplet

```bash
curl -fsSL https://raw.githubusercontent.com/monstercameron/PersonalWebsiteMid2026/main/deploy/install.sh | sudo bash
# or, from a checkout:  sudo deploy/install.sh
```

Idempotent. It installs nginx/certbot/git/ufw, creates the `earlcameron` system user and
`/opt/earlcameron`, **adds a 2 GB swapfile if the droplet has <2 GB RAM and no swap**
(building two wasm binaries OOMs a 1 GB box, and the failure looks like an unexplained
`signal: killed` from the compiler), installs the Go toolchain at the version read out of
`go.mod`, clones the three repos, generates `.env` with fresh secrets, installs the systemd
unit and the nginx site, enables ufw, builds and activates the first release, then runs
certbot.

Useful overrides: `DOMAIN=`, `CERTBOT_EMAIL=`, `SKIP_TLS=1` (when DNS doesn't point at the
droplet yet), `GWC_REF=`.

It prints the generated `ADMIN_SETUP_TOKEN` — you need it to claim the owner account at
`/admin`, and it is what stops a stranger claiming a freshly deployed site first.

### 2. Update

```bash
sudo /opt/earlcameron/src/PersonalWebsiteMid2026/deploy/update.sh
```

Pulls all three repos, rebuilds, stages a new `releases/<utc>-<sha>/`, flips the symlink,
restarts, and **health-checks — rolling back automatically if the new release fails.**

## ⚠️ Promoting `dev` → `main` does NOT deploy this repo (2026-08-10)

Verified against the live webhook delivery log, because the failure is completely silent:
the promotion is green, `main` really does move, and the box keeps serving the old commit.

`deployhook` (in the ArticleFlux repo, serving both sites) fires on a `workflow_run` matching
a `(workflow name, branch)` pair. The pair configured for **this** repo is `("CI", "main")`.
But `promote.yml` fast-forwards `main` with a push authenticated by `GITHUB_TOKEN`, and GitHub
does not start workflow runs from `GITHUB_TOKEN` events — so **no CI run on `main` ever happens
after a promotion**, and the hook waits for something that will never arrive.

Both obvious attempts are declined, each with a green 200:

| dispatched from | hook's answer |
|---|---|
| `main` | `no deploy: workflow is "Promote dev to main", not "CI"` |
| `dev`  | `no deploy: branch is "dev", not "main"` |

**So, after promoting, deploy with:**

```bash
gh workflow run ci.yml --ref main      # produces CI/main → the pair the hook accepts
```

Confirm it actually took, rather than trusting the 200 — the response body says which:

```bash
gh api repos/monstercameron/PersonalWebsiteMid2026/hooks/658081377/deliveries \
  --jq '[.[] | select(.action=="completed")][0].id'   # then …/deliveries/<id> --jq .response.payload
```

A working delivery answers `deploying monstercameron/PersonalWebsiteMid2026 at <sha>`.

**The real fix is in the ArticleFlux repo, not this one:** give this repo the same second
trigger ArticleFlux already has — `("Promote dev to main", "dev")` — so promotion deploys on
its own. ArticleFlux's `deploy/README.md` documents that topology and explains why the pair is
pinned to the dispatch ref; this repo simply never got the second entry.

## Two details worth knowing about `update.sh`

- It **pulls this repo first, then re-execs itself once** (`EC_UPDATE_REEXEC`). bash reads
  a script incrementally as it runs, so a pull that rewrote `update.sh` mid-run would
  resume at a byte offset into different content.
- A failed build **never touches the running site**: building happens in the source tree,
  and the swap happens only after `bin/server` exists.

### 3. Roll back

```bash
sudo deploy/rollback.sh            # back one release
sudo deploy/rollback.sh --list     # what's on disk, and which is live
sudo deploy/rollback.sh 20260728T140233Z-a1b2c3d
```

`update.sh` already auto-rolls-back a release that fails its health check. `rollback.sh` is
for the other case: the deploy came up healthy and is serving, but the change was wrong.
It only moves the symlink — no rebuild, no network. If the rollback target is *also*
unhealthy it restores what was live and says so.

Note it does **not** move the source tree; the next `update.sh` will rebuild the same commit.
Revert the commit, or pass `SITE_REF=<sha>`.

## The Nginx bits that actually matter

`deploy/nginx/earlcameron.conf`, installed to `sites-available/earlcameron`. It ships
**HTTP-only on purpose** — `certbot --nginx` rewrites it in place to add the 443 block and
the redirect, and shipping a pre-baked 443 block is the usual reason a first deploy fights
certbot instead of finishing.

- **WebSocket upgrade on `/socket` *and* `/grpc`.** `/grpc` is the embedded CashFlux sync
  engine's tunnel and `/v1/version` is the handshake its frontend probes first. Miss
  `/grpc` and the budget app fails "Test connection" with no obvious cause. The regex is
  `^/(socket|grpc)(/|$)` because `internal/server` registers both `/socket` and `/socket/`.
- **`proxy_read_timeout 3600s`** on those locations — `StreamStatus` and `Ask` are
  long-lived server-streams that the 60s default would cut every minute.
- **`X-Forwarded-Proto`** on `location /` — `internal/budget/gate.go` trusts it to decide
  whether its session cookie gets the `Secure` flag. Without it, nginx terminates TLS and
  every cookie is issued insecure.
- **gzip including `application/wasm`** — `app.wasm` is ~26 MB and the Go file server does
  not compress. This is the only thing between a visitor and a 26 MB download.
- **No `expires` on `/static/`** — `app.wasm` is not content-hashed, so a long cache
  lifetime serves the previous release's wasm after a deploy. Go's `http.FileServer`
  already does `Last-Modified`/`304` revalidation, which gives the caching without the
  staleness.

## systemd

`deploy/earlcameron.service` → `/etc/systemd/system/`. `Restart=always`, journald logging
(`journalctl -u earlcameron -f`), `Type=exec`, and `KillSignal=SIGTERM` with a 30s stop
timeout so the server's existing `http.Server.Shutdown` drain actually gets to finish.

`WorkingDirectory` is the **`current` symlink, not a fixed release** — that indirection is
what makes the swap work, since the asset paths are CWD-relative.

Hardened with `ProtectSystem=strict`, `ProtectHome`, `NoNewPrivileges`, `PrivateTmp`,
`RestrictAddressFamilies` and friends, with `ReadWritePaths=/opt/earlcameron/data` as the
single writable path.

> **`MemoryDenyWriteExecute` is deliberately absent.** The CashFlux sync engine pulls in
> `ncruces/go-sqlite3`, which runs SQLite as WebAssembly through **wazero** — and wazero's
> compiler allocates W^X pages at runtime. Denying write+execute mappings kills the budget
> app on its first query. Same reason there is no aggressive `SystemCallFilter`:
> `mmap`/`mprotect` are load-bearing on that path.

## Data, secrets, migrations

- **SQLite lives in `/opt/earlcameron/data/`**, outside every release, so updates never
  touch it and the release tree itself can stay read-only. `DB_PATH` and
  `CASHFLUX_DATA_DIR` in `.env` point there.
- **Secrets in `/opt/earlcameron/.env`** (chmod 600, systemd `EnvironmentFile`). Generated
  from `deploy/env.example`, which documents every key the code actually reads —
  `install.sh` fills `ADMIN_SECRET` and `ADMIN_SETUP_TOKEN` with `openssl rand`. Never in
  git, never in the binary, never web-editable.
- **Migrations** run on start, so a schema-changing update self-applies.

## Still open

- **No backup job yet.** `data/` is the only irreplaceable thing on the droplet and nothing
  copies it off-box. Nightly `sqlite3 .backup` → DO Spaces is the obvious shape.
- **Restart is a ~1s blip**, not zero-downtime. Fine for a portfolio.
- **First deploy needs** a DNS A-record for the domain → droplet IP. That is the one step
  certbot cannot do for you, and the one that fails if you skip it (safely: the site keeps
  serving on :80 and you re-run certbot).
