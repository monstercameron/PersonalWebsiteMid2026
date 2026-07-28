#!/usr/bin/env bash
# Pull, rebuild, and swap in a new release — with an automatic rollback if the new one
# does not come up healthy.
#
#   sudo /opt/earlcameron/src/PersonalWebsiteMid2026/deploy/update.sh
#   sudo ... /deploy/update.sh --skip-pull     # build what is already checked out
#
# The invariant: `current` only ever moves to a release that has answered /healthz. A
# failed build never touches the running site, because the build happens in the source
# tree and the swap happens after.
set -euo pipefail

SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=deploy/lib.sh
source "$SELF_DIR/lib.sh"

require_root

SKIP_PULL=0
[ "${1:-}" = "--skip-pull" ] && SKIP_PULL=1

# ------------------------------------------------------------------ pull, then re-exec
# bash reads a script incrementally as it runs it, so a `git pull` that rewrites THIS
# file mid-execution makes the shell resume at a byte offset into different content.
# Pull first, then hand off to the freshly-pulled copy exactly once.
if [ "$SKIP_PULL" = "0" ] && [ -z "${EC_UPDATE_REEXEC:-}" ]; then
    log "pulling $SITE_REF in PersonalWebsiteMid2026"
    as_app git -C "$SITE_DIR" fetch --tags --prune origin
    as_app git -C "$SITE_DIR" checkout -q "$SITE_REF"
    as_app git -C "$SITE_DIR" pull --ff-only origin "$SITE_REF"
    log "re-executing the updated deploy script"
    EC_UPDATE_REEXEC=1 exec bash "$SELF_DIR/update.sh" "$@"
fi

if [ "$SKIP_PULL" = "0" ]; then
    for pair in "$GWC_DIR:$GWC_REF" "$CASHFLUX_DIR:$CASHFLUX_REF"; do
        dir="${pair%:*}"; ref="${pair##*:}"
        log "updating $(basename "$dir") -> $ref"
        as_app git -C "$dir" fetch --tags --prune origin
        # Detached checkout for tags, branch checkout + ff-only pull for branches.
        if as_app git -C "$dir" checkout -q --detach "$ref" 2>/dev/null; then :; else
            as_app git -C "$dir" checkout -q "$ref"
            as_app git -C "$dir" pull --ff-only origin "$ref"
        fi
        as_app git -C "$dir" submodule update --init --recursive || true
    done
fi

# ------------------------------------------------------------------ build
log "building server + client wasm"
as_app bash "$SITE_DIR/scripts/build.sh"

log "building and staging the CashFlux frontend"
# CASHFLUX_SRC must point at the sibling checkout; the script's default is Cam's Windows
# path. BUDGET_MOUNT_PATH rewrites <base href> so the SPA's relative asset URLs resolve
# under /budget/ instead of the domain root.
as_app env CASHFLUX_SRC="$CASHFLUX_DIR" BUDGET_MOUNT_PATH=/budget/ \
    bash "$SITE_DIR/scripts/build-cashflux.sh"

[ -x "$SITE_DIR/bin/server" ] || die "build produced no bin/server — aborting before any swap"

# ------------------------------------------------------------------ stage the release
sha="$(as_app git -C "$SITE_DIR" rev-parse --short HEAD)"
rel="$RELEASES_DIR/$(date -u +%Y%m%dT%H%M%SZ)-$sha"
log "staging release $(basename "$rel")"

mkdir -p "$rel/web"
install -m 755 "$SITE_DIR/bin/server" "$rel/server"
# The server resolves web/static and web/cashflux relative to its working directory, so
# the assets have to travel WITH the binary — that is what makes the symlink flip swap
# code and frontend together instead of leaving a new binary serving old wasm.
cp -a "$SITE_DIR/web/static"   "$rel/web/static"
cp -a "$SITE_DIR/web/cashflux" "$rel/web/cashflux"
chown -R "$APP_USER:$APP_USER" "$rel"

# ------------------------------------------------------------------ swap + verify
prev="$(current_release)"
log "activating $(basename "$rel")"
activate_release "$rel"

if health_check 30; then
    log "healthy: $(basename "$rel") is live"
    prune_releases
    exit 0
fi

warn "new release failed its health check"
if [ -n "$prev" ] && [ -d "$prev" ]; then
    warn "rolling back to $(basename "$prev")"
    activate_release "$prev"
    if health_check 30; then
        warn "rolled back; the previous release is healthy again"
    else
        die "ROLLBACK ALSO UNHEALTHY — check: journalctl -u $SERVICE_NAME -n 100"
    fi
    # Keep the failed release on disk rather than pruning it; it is the evidence.
    die "deploy failed and was rolled back. Failed build kept at $rel"
fi

die "deploy failed and there is no previous release to roll back to. journalctl -u $SERVICE_NAME -n 100"
