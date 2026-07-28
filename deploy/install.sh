#!/usr/bin/env bash
# One-shot setup for a fresh Ubuntu droplet. Idempotent — safe to re-run.
#
#   curl -fsSL https://raw.githubusercontent.com/monstercameron/PersonalWebsiteMid2026/main/deploy/install.sh | sudo bash
#
# or, from a checkout:  sudo deploy/install.sh
#
# Env overrides:
#   DOMAIN=earlcameron.com   CERTBOT_EMAIL=you@example.com
#   SKIP_TLS=1               skip certbot (use when DNS does not point here yet)
#   GWC_REF=v5.0.1           GoWebComponents ref to pin
#
# WHY THE DROPLET BUILDS FROM SOURCE
# go.mod pins GoWebComponents and CashFlux through RELATIVE replace directives
# (../GoWebComponents, ../CashFlux). Neither is consumable from the module proxy at the
# pinned state, so a build needs all three checkouts side by side — which is exactly what
# this script lays down under /opt/earlcameron/src. The cost is a Go toolchain and ~2 GB
# of RAM on the box; the benefit is that updating is `git pull` and nothing else.
set -euo pipefail

SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -f "$SELF_DIR/lib.sh" ]; then
    # shellcheck source=deploy/lib.sh
    source "$SELF_DIR/lib.sh"
else
    # Piped from curl: no checkout yet. Clone first, then use its lib.sh.
    APP_ROOT="${APP_ROOT:-/opt/earlcameron}"
fi

DOMAIN="${DOMAIN:-earlcameron.com}"
CERTBOT_EMAIL="${CERTBOT_EMAIL:-}"
SKIP_TLS="${SKIP_TLS:-0}"

[ "$(id -u)" -eq 0 ] || { echo "run as root (sudo)"; exit 1; }

# ---------------------------------------------------------------- system packages
echo "==> installing system packages"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq \
    nginx certbot python3-certbot-nginx \
    git curl ca-certificates ufw gzip coreutils util-linux

# ---------------------------------------------------------------- app user + layout
APP_USER="${APP_USER:-earlcameron}"
if ! id -u "$APP_USER" >/dev/null 2>&1; then
    echo "==> creating service user $APP_USER"
    # A system user with a real home at $APP_ROOT: the Go build cache lives there, and
    # nologin keeps it out of SSH while still allowing runuser to execute builds.
    useradd --system --home-dir "$APP_ROOT" --create-home --shell /usr/sbin/nologin "$APP_USER"
fi

mkdir -p "$APP_ROOT"/{src,releases,data,data/cashflux,go,.cache}
chown -R "$APP_USER:$APP_USER" "$APP_ROOT"
chmod 750 "$APP_ROOT"

# ---------------------------------------------------------------- swap
# Building two wasm binaries (this client, plus CashFlux's ~94 MB main.wasm) will OOM a
# 1 GB droplet outright, and the failure looks like a mysterious "signal: killed" from the
# compiler. Add swap when the box is small and has none.
mem_kb="$(awk '/MemTotal/ {print $2}' /proc/meminfo)"
swap_kb="$(awk '/SwapTotal/ {print $2}' /proc/meminfo)"
if [ "$mem_kb" -lt 2097152 ] && [ "$swap_kb" -lt 1048576 ] && [ ! -f /swapfile ]; then
    echo "==> small droplet with no swap; adding a 2G swapfile so the wasm build can finish"
    fallocate -l 2G /swapfile || dd if=/dev/zero of=/swapfile bs=1M count=2048
    chmod 600 /swapfile
    mkswap /swapfile >/dev/null
    swapon /swapfile
    grep -q '^/swapfile' /etc/fstab || echo '/swapfile none swap sw 0 0' >>/etc/fstab
fi

# ---------------------------------------------------------------- source checkouts
SRC_DIR="$APP_ROOT/src"
SITE_REPO="${SITE_REPO:-https://github.com/monstercameron/PersonalWebsiteMid2026.git}"
SITE_REF="${SITE_REF:-main}"
GWC_REPO="${GWC_REPO:-https://github.com/monstercameron/GoWebComponents.git}"
GWC_REF="${GWC_REF:-v5.0.1}"
CASHFLUX_REPO="${CASHFLUX_REPO:-https://github.com/monstercameron/CashFlux.git}"
CASHFLUX_REF="${CASHFLUX_REF:-main}"

clone_or_fetch() {
    local repo="$1" dir="$2" ref="$3"
    if [ -d "$dir/.git" ]; then
        echo "==> updating $(basename "$dir") -> $ref"
        runuser -u "$APP_USER" -- git -C "$dir" fetch --tags --prune origin
    else
        echo "==> cloning $(basename "$dir") -> $ref"
        # --recurse-submodules: GoWebComponents vendors GoGRPCBridge as a submodule and
        # replaces it by relative path in its own go.mod.
        runuser -u "$APP_USER" -- git clone --recurse-submodules "$repo" "$dir"
    fi
    runuser -u "$APP_USER" -- git -C "$dir" checkout -q --detach "$ref" 2>/dev/null \
        || runuser -u "$APP_USER" -- git -C "$dir" checkout -q "$ref"
    runuser -u "$APP_USER" -- git -C "$dir" submodule update --init --recursive || true
}

clone_or_fetch "$SITE_REPO" "$SRC_DIR/PersonalWebsiteMid2026" "$SITE_REF"
clone_or_fetch "$GWC_REPO"  "$SRC_DIR/GoWebComponents"        "$GWC_REF"
clone_or_fetch "$CASHFLUX_REPO" "$SRC_DIR/CashFlux"           "$CASHFLUX_REF"

SITE_DIR="$SRC_DIR/PersonalWebsiteMid2026"

# ---------------------------------------------------------------- Go toolchain
# Ubuntu's packaged golang trails the `go` directive in go.mod by years, so read the
# required version out of go.mod rather than hardcoding a second copy of that fact here.
GO_VERSION="$(awk '/^go [0-9]/ {print $2; exit}' "$SITE_DIR/go.mod")"
GO_VERSION="${GO_VERSION:-1.26.4}"
GO_ARCH="$(dpkg --print-architecture)"   # amd64 on a standard droplet, arm64 on ARM
if [ "$(/usr/local/go/bin/go version 2>/dev/null | awk '{print $3}')" != "go$GO_VERSION" ]; then
    echo "==> installing Go $GO_VERSION ($GO_ARCH)"
    tarball="/tmp/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    curl -fsSL -o "$tarball" "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "$tarball"
    rm -f "$tarball"
    echo 'export PATH=$PATH:/usr/local/go/bin' >/etc/profile.d/go.sh
fi
/usr/local/go/bin/go version

# ---------------------------------------------------------------- secrets
if [ ! -f "$APP_ROOT/.env" ]; then
    echo "==> generating $APP_ROOT/.env"
    install -m 600 -o "$APP_USER" -g "$APP_USER" "$SITE_DIR/deploy/env.example" "$APP_ROOT/.env"
    # Fill the two secrets that must never be blank or guessable. Everything else is
    # left for Cam to edit — an unset OPENAI_API_KEY disables the AI features cleanly
    # rather than failing, which is the right default for an unattended first boot.
    sed -i "s|^ADMIN_SECRET=.*|ADMIN_SECRET=$(openssl rand -hex 32)|" "$APP_ROOT/.env"
    sed -i "s|^ADMIN_SETUP_TOKEN=.*|ADMIN_SETUP_TOKEN=$(openssl rand -hex 16)|" "$APP_ROOT/.env"
    sed -i "s|^BASE_URL=.*|BASE_URL=https://$DOMAIN|" "$APP_ROOT/.env"
    sed -i "s|^CASHFLUX_SERVER_APP_ORIGIN=.*|CASHFLUX_SERVER_APP_ORIGIN=https://$DOMAIN|" "$APP_ROOT/.env"
    echo "    setup token: $(grep '^ADMIN_SETUP_TOKEN=' "$APP_ROOT/.env" | cut -d= -f2)"
    echo "    (you need it to claim the owner account at https://$DOMAIN/admin)"
else
    echo "==> $APP_ROOT/.env exists; leaving it alone"
fi
chown "$APP_USER:$APP_USER" "$APP_ROOT/.env"
chmod 600 "$APP_ROOT/.env"

# ---------------------------------------------------------------- systemd + nginx
echo "==> installing systemd unit"
install -m 644 "$SITE_DIR/deploy/earlcameron.service" /etc/systemd/system/earlcameron.service
systemctl daemon-reload
systemctl enable earlcameron >/dev/null

echo "==> installing nginx site for $DOMAIN"
# Only the domain is substituted. The upstream block is named `earlcameron_app` with no
# dot, so it is deliberately not matched by this pattern.
sed "s/earlcameron\.com/$DOMAIN/g" \
    "$SITE_DIR/deploy/nginx/earlcameron.conf" >/etc/nginx/sites-available/earlcameron
ln -sfn /etc/nginx/sites-available/earlcameron /etc/nginx/sites-enabled/earlcameron
# The default site answers on :80 for any Host and will shadow this one on a fresh box.
rm -f /etc/nginx/sites-enabled/default
nginx -t
systemctl reload nginx

# ---------------------------------------------------------------- firewall
echo "==> configuring ufw"
ufw allow OpenSSH >/dev/null
ufw allow 'Nginx Full' >/dev/null
ufw --force enable >/dev/null
ufw status verbose | head -5

# ---------------------------------------------------------------- first build + start
echo "==> building and activating the first release"
# Invoked through `bash` rather than executed directly: these scripts are authored on
# Windows, where git does not record the executable bit, so a clone can land them 644.
bash "$SITE_DIR/deploy/update.sh" --skip-pull

# ---------------------------------------------------------------- TLS
if [ "$SKIP_TLS" = "1" ]; then
    echo "==> SKIP_TLS=1; run certbot yourself once DNS points here:"
    echo "    certbot --nginx -d $DOMAIN -d www.$DOMAIN"
else
    echo "==> requesting a certificate for $DOMAIN"
    # Needs the A record for $DOMAIN pointing at this droplet already. If it does not,
    # this is the one step that fails — and it fails safely: the site keeps serving on
    # :80 and you can re-run certbot later.
    if [ -n "$CERTBOT_EMAIL" ]; then
        certbot --nginx --non-interactive --agree-tos -m "$CERTBOT_EMAIL" \
            -d "$DOMAIN" -d "www.$DOMAIN" --redirect || warn_tls=1
    else
        certbot --nginx -d "$DOMAIN" -d "www.$DOMAIN" --redirect || warn_tls=1
    fi
    if [ "${warn_tls:-0}" = "1" ]; then
        echo "!!  certbot failed (DNS not pointing here yet?). The site is up on http://$DOMAIN"
        echo "!!  Re-run: certbot --nginx -d $DOMAIN -d www.$DOMAIN"
    else
        # HSTS is commented out in the shipped config on purpose; now that TLS works it
        # is safe to turn on.
        sed -i 's|# add_header Strict-Transport-Security|add_header Strict-Transport-Security|' \
            /etc/nginx/sites-available/earlcameron
        nginx -t && systemctl reload nginx
    fi
fi

echo
echo "==> done."
echo "    status:  systemctl status earlcameron"
echo "    logs:    journalctl -u earlcameron -f"
echo "    update:  sudo $SITE_DIR/deploy/update.sh"
echo "    undo:    sudo $SITE_DIR/deploy/rollback.sh"
