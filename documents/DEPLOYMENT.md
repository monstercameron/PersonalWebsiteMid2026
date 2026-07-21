# DEPLOYMENT — Ubuntu + Nginx on DigitalOcean

> **Status: PLAN.** Blueprint for when we build. Goal, in Cam's words: **one-click to install,
> one action to update from GitHub — without racking my brain.**

## The principle that makes it painless
Ship **one self-contained Go binary** with **all assets embedded** (`go:embed` the built
wasm, `wasm_exec.js`, css, fonts). Deploying is then: drop one file, restart one service. The
droplet stays dumb — no Go toolchain, no npm, no build steps on the server.

```
Internet ──443/TLS──► Nginx (reverse proxy, Let's Encrypt)
                         │  proxy_pass 127.0.0.1:8095
                         ▼
                   earlcameron (single Go binary, systemd)
                     ├─ /socket   gRPC-over-WebSocket (app data plane)
                     └─ /         wasm + SSR + RSS + PDF (document plane)
                   SQLite file on disk (persistent, outside the deploy dir)
```

## Two flows, both brain-free

### 1. First install — one script, once (`deploy/install.sh`)
Run once on a fresh Ubuntu droplet. Idempotent. It:
- `apt install nginx certbot python3-certbot-nginx`
- creates the `earlcameron` user + `/opt/earlcameron/{releases,current,data}`
- installs the systemd unit + the Nginx site (below)
- `certbot --nginx -d earlcameron.com -d www.earlcameron.com` (TLS + auto-renew)
- `ufw allow 'Nginx Full' && ufw allow OpenSSH`
- prompts once for secrets → writes `/opt/earlcameron/.env` (chmod 600)
- pulls the first release, enables + starts the service

One command: `curl -fsSL https://raw.githubusercontent.com/monstercameron/earlcameron/main/deploy/install.sh | sudo bash`.

### 2. Updates — just `git push` (recommended)
**Push-to-deploy via GitHub Actions.** On push to `main`, CI builds the wasm + the Linux
binary (with embedded assets), then SSHes to the droplet and runs the atomic swap + restart.
Cam pushes; the site updates. No SSH, no thinking.

Fallback (manual, one command on the droplet): `sudo /opt/earlcameron/deploy/update.sh` —
fetches the latest release artifact, atomic-swaps `current`, restarts, health-checks, and
**auto-rolls-back** if `/healthz` fails.

## Build source (a real decision — flagged)
Our `go.mod` uses local `replace`s to `../GoWebComponents` (+ its vendored GoGRPCBridge). Those
paths **don't exist in CI**. To build on a GitHub runner, pick one:
- **Git submodule (recommended)**: add GoWebComponents as a submodule pinned to the v4.3.0
  audit commit (it already vendors GoGRPCBridge as its own submodule); CI checks out
  `--recursive`; the `replace` points at the submodule path. Clean, reproducible, matches how
  GWC itself vendors the bridge.
- **`go mod vendor`** committed: fully self-contained CI build, larger repo.

Decide before wiring CI. Default: submodule.

## The Nginx bit that actually matters — WebSocket proxying
The gRPC tunnel is a long-lived WebSocket, so the proxy must upgrade and not time out:

```nginx
server {
  server_name earlcameron.com www.earlcameron.com;
  listen 443 ssl;   # http2 on; certbot manages the ssl_* lines

  location /socket {                         # the gRPC-over-WS tunnel
    proxy_pass http://127.0.0.1:8095;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_read_timeout 3600s;                # don't kill live streams
    proxy_send_timeout 3600s;
  }
  location / {                               # document plane
    proxy_pass http://127.0.0.1:8095;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
  }
}
# port 80 server_block redirects → https (certbot adds this)
```
TLS here also satisfies the admin/BYOK requirement that keys never travel in cleartext.

## systemd unit (`/etc/systemd/system/earlcameron.service`)
```ini
[Unit]
Description=earlcameron.com
After=network.target

[Service]
User=earlcameron
WorkingDirectory=/opt/earlcameron/current
EnvironmentFile=/opt/earlcameron/.env
ExecStart=/opt/earlcameron/current/server
Restart=always
RestartSec=2

[Install]
WantedBy=multi-user.target
```
Logs go to journald (`journalctl -u earlcameron`). Auto-restart on crash, start on boot.

## Atomic, reversible updates (`deploy/update.sh` sketch)
```bash
set -euo pipefail
rel="/opt/earlcameron/releases/$(date +%s)"        # timestamp passed in by CI
install -D artifact/server "$rel/server"
prev="$(readlink -f /opt/earlcameron/current || true)"
ln -sfn "$rel" /opt/earlcameron/current
systemctl restart earlcameron
sleep 1
curl -fsS http://127.0.0.1:8095/healthz || {        # rollback on failed health check
  [ -n "$prev" ] && ln -sfn "$prev" /opt/earlcameron/current && systemctl restart earlcameron
  echo "deploy failed, rolled back"; exit 1; }
```
Keeps the last N releases for instant rollback.

## Data, secrets, migrations
- **SQLite lives in `/opt/earlcameron/data/` — outside the deploy dir**, so updates never touch
  it. Nightly backup (cron `sqlite3 .backup` → off-box / DO Spaces).
- **Secrets in `/opt/earlcameron/.env`** (chmod 600, `EnvironmentFile`): `OPENAI_API_KEY`,
  `ADMIN_OWNER_IDS`, signing keys. Never in git, never in the binary, never web-editable.
- **Migrations** run automatically on server start (idempotent, versioned), so an update that
  changes the schema self-applies. Back up before migrating.

## Reality checks
- **DO droplet arch**: build the binary for the droplet's arch (`GOOS=linux GOARCH=amd64`, or
  `arm64` on an ARM droplet). CI sets this.
- **Zero-downtime**: systemd restart is a ~1s blip — fine for a portfolio; not true zero-downtime.
  Good enough; revisit only if it ever matters.
- **First deploy needs**: domain DNS A-record → droplet IP, and one GitHub deploy secret
  (SSH key) for push-to-deploy. Both are one-time.
- **DO App Platform** would be even less ops, but you chose a droplet + Nginx, so this is the fit.
```
