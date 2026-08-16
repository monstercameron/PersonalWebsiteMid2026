#!/bin/sh
# ec-autoupdate.sh — deploy earlcameron.com's newest v* release tag when it changes.
#
# The AFF pattern: the deploy webhook (primary) and a daily systemd timer (fallback for a
# lost delivery) both run this. It compares the newest v* tag on the repo against the tag
# pinned in /opt/earlcameron/compose.yaml and, when they differ AND the image is actually
# published, hands off to ec-deploy-release.sh — which blocks on the container healthcheck,
# so a bad release fails loudly in journalctl rather than silently crash-looping.
#
# Releasing therefore stays one ritual: promote dev -> main, tag vX.Y.Z, push the tag. The
# Release workflow builds ghcr.io/monstercameron/personalwebsitemid2026:vX.Y.Z and pokes
# the hook; within moments this box is running it. `latest` and sha- tags are ignored on
# purpose — deploys pin immutable, deliberate version tags.
#
# Idempotent and safe to run by hand: `sh /usr/local/bin/ec-autoupdate.sh`.
set -eu

SRC="${EC_SRC_DIR:-/opt/earlcameron/src/PersonalWebsiteMid2026}"
APP="${EC_APP_DIR:-/opt/earlcameron}"
IMAGE="ghcr.io/monstercameron/personalwebsitemid2026"

log() { printf '[ec-autoupdate] %s\n' "$1"; }

[ -d "$SRC/.git" ] || { log "ERROR: $SRC is not a git checkout"; exit 1; }
[ -f "$APP/compose.yaml" ] || { log "ERROR: $APP/compose.yaml missing — install deploy/docker/compose.yaml"; exit 1; }

# Freshen the checkout first: the deploy scripts evolve with the repo, and an update must
# run the release procedure the RELEASE expects, not the one from bootstrap day.
git -C "$SRC" fetch -q --tags origin
git -C "$SRC" reset -q --hard origin/main

latest=$(git -C "$SRC" tag -l 'v*' --sort=-v:refname | head -n1)
if [ -z "$latest" ]; then
    log "no v* tags exist yet — nothing to deploy"
    exit 0
fi

current=$(grep -oE 'image:.*personalwebsitemid2026:[^[:space:]]+' "$APP/compose.yaml" | sed 's/^.*://' || true)
if [ "$current" = "$latest" ]; then
    log "up to date ($current)"
    exit 0
fi

# The tag can exist minutes before the Release workflow finishes publishing its image.
# Deploying into that window would fail the pull; skip quietly and let the next poke retry.
if ! docker manifest inspect "$IMAGE:$latest" >/dev/null 2>&1; then
    log "tag $latest exists but $IMAGE:$latest is not published yet — retrying next tick"
    exit 0
fi

log "updating: ${current:-none} -> $latest"
exec sh "$SRC/deploy/docker/ec-deploy-release.sh" "$latest" "$APP"
