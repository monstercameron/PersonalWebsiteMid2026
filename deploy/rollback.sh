#!/usr/bin/env bash
# Roll back to a previous release. No rebuild, no network — it only moves the symlink,
# so it is the fastest way out of a bad deploy.
#
#   sudo deploy/rollback.sh              # go back one release
#   sudo deploy/rollback.sh --list       # show what is on disk and which is live
#   sudo deploy/rollback.sh 20260728T140233Z-a1b2c3d   # go to a specific release
#
# update.sh already rolls back automatically when a new release fails its health check.
# This script is for the other case: the deploy came up healthy and IS serving traffic,
# but the change turned out to be wrong.
set -euo pipefail

SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=deploy/lib.sh
source "$SELF_DIR/lib.sh"

live="$(current_release)"

# --list is read-only, so it does not need root.
if [ "${1:-}" = "--list" ]; then
    echo "releases in $RELEASES_DIR (newest first):"
    for d in $(ls -1d "$RELEASES_DIR"/*/ 2>/dev/null | sort -r); do
        d="${d%/}"
        if [ "$d" = "$live" ]; then
            printf '  * %s   <- live\n' "$(basename "$d")"
        else
            printf '    %s\n' "$(basename "$d")"
        fi
    done
    exit 0
fi

require_root

target=""
if [ -n "${1:-}" ]; then
    target="$RELEASES_DIR/$1"
    [ -d "$target" ] || die "no such release: $1  (try: $0 --list)"
else
    # Newest release that is not the live one. Names are UTC timestamps, so reverse
    # lexical order is reverse chronological order.
    for d in $(ls -1d "$RELEASES_DIR"/*/ 2>/dev/null | sort -r); do
        d="${d%/}"
        [ "$d" = "$live" ] && continue
        target="$d"
        break
    done
    [ -n "$target" ] || die "no previous release to roll back to (try: $0 --list)"
fi

[ "$target" != "$live" ] || die "$(basename "$target") is already live"

log "rolling back: $(basename "${live:-none}") -> $(basename "$target")"
activate_release "$target"

if health_check 30; then
    log "healthy: $(basename "$target") is live"
    log "NOTE: the source tree is still at the newer commit. The next update.sh will"
    log "      rebuild and deploy it again — revert the commit, or pass SITE_REF=<sha>."
    exit 0
fi

warn "the rollback target is ALSO unhealthy"
if [ -n "$live" ] && [ -d "$live" ]; then
    warn "restoring $(basename "$live")"
    activate_release "$live"
    health_check 30 || die "nothing healthy left. journalctl -u $SERVICE_NAME -n 100"
fi
die "rollback target failed its health check; restored $(basename "$live")"
