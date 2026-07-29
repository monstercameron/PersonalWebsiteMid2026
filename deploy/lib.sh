#!/usr/bin/env bash
# Shared constants and helpers for deploy/{install,update,rollback}.sh.
#
# Sourced, never executed. Everything that both activates a release and verifies one
# lives here, because update.sh and rollback.sh must agree on what "live" means — if
# they drift, a rollback stops being a reliable undo of a deploy.

# Layout on the droplet:
#   /opt/earlcameron/src/{PersonalWebsiteMid2026,GoWebComponents,CashFlux}
#                           the three checkouts, kept as SIBLINGS because go.mod uses
#                           relative replace directives (../GoWebComponents, ../CashFlux)
#   /opt/earlcameron/releases/<utc>-<sha>/{server,web/}   built artifacts, immutable
#   /opt/earlcameron/current -> releases/<...>            the symlink systemd runs from
#   /opt/earlcameron/data/                               SQLite, outside every release
#   /opt/earlcameron/.env                                secrets, chmod 600
APP_USER="${APP_USER:-earlcameron}"
APP_ROOT="${APP_ROOT:-/opt/earlcameron}"
SRC_DIR="$APP_ROOT/src"
SITE_DIR="$SRC_DIR/PersonalWebsiteMid2026"
GWC_DIR="$SRC_DIR/GoWebComponents"
CASHFLUX_DIR="$SRC_DIR/CashFlux"
RELEASES_DIR="$APP_ROOT/releases"
CURRENT_LINK="$APP_ROOT/current"
DATA_DIR="$APP_ROOT/data"
ENV_FILE="$APP_ROOT/.env"
SERVICE_NAME="earlcameron"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:8095/healthz}"
GO_ROOT="/usr/local/go"
# How many old releases to keep. Each is a ~26 MB wasm plus a ~120 MB CashFlux tree, so
# this is a disk-space decision as much as a rollback-depth one.
KEEP_RELEASES="${KEEP_RELEASES:-5}"

# Upstream refs, and the asymmetry between them is a POLICY rather than an accident — it
# decides which promotions reach visitors and which do not.
#
#   SITE_REF      main        promoting this repo deploys, and the webhook does it
#   CASHFLUX_REF  main        promoting CashFlux is picked up by the NEXT deploy of this
#                             site — see the note below, it does not trigger one itself
#   GWC_REF       a TAG       promoting GoWebComponents deploys NOTHING, deliberately
#
# GWC is pinned because it is a library with a major-version contract that this site and
# CashFlux both resolve through relative `replace` directives; tracking its `main` would
# let an unrelated library commit change what is served without anybody promoting
# anything. v5.0.1 specifically is the first v5 release in which css/u stops shadowing
# html/shorthand, which this site's five dot-importing files require to compile at all.
# Moving it is a deliberate edit here, and that is the right amount of friction.
#
# CashFlux tracking `main` means this site always builds the latest promoted CashFlux —
# but only when a deploy runs. A CashFlux promotion does not currently start one, because
# no deployhook target names that repository. The full topology, and the one-line fix, is
# in ArticleFlux/deploy/README.md → "What deploys what".
SITE_REPO="${SITE_REPO:-https://github.com/monstercameron/PersonalWebsiteMid2026.git}"
SITE_REF="${SITE_REF:-main}"
GWC_REPO="${GWC_REPO:-https://github.com/monstercameron/GoWebComponents.git}"
GWC_REF="${GWC_REF:-v5.0.1}"
CASHFLUX_REPO="${CASHFLUX_REPO:-https://github.com/monstercameron/CashFlux.git}"
CASHFLUX_REF="${CASHFLUX_REF:-main}"

log()  { printf '\033[1;35m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m!!\033[0m  %s\n' "$*" >&2; }
die()  { printf '\033[1;31mxx\033[0m  %s\n' "$*" >&2; exit 1; }

# require_root refuses to continue unless we can install units and restart services.
require_root() {
    [ "$(id -u)" -eq 0 ] || die "run as root (sudo $0)"
}

# as_app runs a command as the service user with a Go environment rooted in the app dir.
# The caches must NOT land in /root: builds run as the app user so every artifact it
# produces is already owned correctly and never needs a chown pass.
as_app() {
    runuser -u "$APP_USER" -- env \
        HOME="$APP_ROOT" \
        GOPATH="$APP_ROOT/go" \
        GOMODCACHE="$APP_ROOT/go/pkg/mod" \
        GOCACHE="$APP_ROOT/.cache/go-build" \
        PATH="$GO_ROOT/bin:/usr/local/bin:/usr/bin:/bin" \
        "$@"
}

# health_check polls the local health endpoint until it answers or the budget runs out.
# Returns non-zero on failure so callers can roll back. The service needs a moment to
# bind and to run migrations, so a single immediate curl would produce false failures.
health_check() {
    local tries="${1:-30}" i
    for ((i = 1; i <= tries; i++)); do
        if curl -fsS --max-time 3 "$HEALTH_URL" >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
    done
    return 1
}

# current_release prints the release directory the symlink resolves to, or nothing.
current_release() {
    [ -L "$CURRENT_LINK" ] && readlink -f "$CURRENT_LINK" || true
}

# activate_release points `current` at $1 and restarts the service.
#
# The two-step symlink dance is the atomic part: `ln -sfn` on an existing directory
# symlink unlinks before it relinks, leaving a window with no `current` at all. Creating
# a temporary link and rename(2)-ing it over the old one has no such window.
activate_release() {
    local rel="$1"
    [ -d "$rel" ] || die "release does not exist: $rel"
    [ -x "$rel/server" ] || die "release has no executable server binary: $rel"

    ln -sfn "$rel" "$CURRENT_LINK.tmp"
    mv -Tf "$CURRENT_LINK.tmp" "$CURRENT_LINK"
    chown -h "$APP_USER:$APP_USER" "$CURRENT_LINK"
    systemctl restart "$SERVICE_NAME"
}

# prune_releases deletes all but the newest $KEEP_RELEASES, never the live one.
#
# Every step is guarded, and deliberately so: callers run with `set -euo pipefail`,
# and pruning happens AFTER the new release is already live and healthy. An
# unguarded `ls | sort | tail` returns non-zero the first time the releases dir is
# short or empty, which would abort a successful deploy at the very last step and
# report a green rollout as a failure. Nothing here is worth failing a deploy over.
prune_releases() {
    local live old_list
    live="$(current_release || true)"
    # shellcheck disable=SC2012  # release names are UTC timestamps, so `ls` sorts correctly
    old_list="$(ls -1d "$RELEASES_DIR"/*/ 2>/dev/null | sort -r | tail -n +"$((KEEP_RELEASES + 1))" || true)"
    [ -n "$old_list" ] || return 0
    printf '%s\n' "$old_list" | while read -r old; do
        old="${old%/}"
        [ -n "$old" ] || continue
        [ "$old" = "$live" ] && continue
        log "pruning old release $(basename "$old")"
        rm -rf "$old" || true
    done
    return 0
}
