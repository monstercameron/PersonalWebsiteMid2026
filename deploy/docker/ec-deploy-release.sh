#!/bin/sh
# ec-deploy-release.sh — pin an image tag for earlcameron.com and roll it out.
#
# The AFF pattern, verbatim in shape (see AnimeFeedFlux scripts/deploy-release.sh, which
# this mirrors): write the tag into compose.yaml, pull, recreate, then block until the
# container's own healthcheck says healthy and fail loudly if it never does. "docker
# compose up -d succeeded" and "the site is up" are different claims; this script only
# makes the second one. A rollback IS a release to an older tag — .previous-tag records
# what was running before each attempt.
#
#   sh ec-deploy-release.sh v0.3.0
#   sh ec-deploy-release.sh v0.3.0 /opt/earlcameron   # explicit app dir
set -eu

TAG="${1:-}"
APP_DIR="${2:-/opt/earlcameron}"
HEALTH_RETRIES="${EC_HEALTH_RETRIES:-30}"
HEALTH_INTERVAL="${EC_HEALTH_INTERVAL:-10}"
CONTAINER_NAME="${EC_CONTAINER_NAME:-earlcameron}"
IMAGE_BASENAME="personalwebsitemid2026"

log() { printf '[ec-deploy-release] %s\n' "$1"; }
die() { printf '[ec-deploy-release] ERROR: %s\n' "$1" >&2; exit 1; }

[ -n "$TAG" ] || die "usage: ec-deploy-release.sh <image-tag> [app-dir] — no tag given"
case "$TAG" in
    latest) die "refusing to deploy 'latest' — pin an immutable v<semver> or sha-<commit> tag" ;;
esac

COMPOSE_FILE="$APP_DIR/compose.yaml"
[ -f "$COMPOSE_FILE" ] || die "$COMPOSE_FILE missing — install deploy/docker/compose.yaml first"

cd "$APP_DIR"
log "deploying tag: $TAG"

prev=$(grep -oE "image:.*${IMAGE_BASENAME}:[^[:space:]]+" "$COMPOSE_FILE" | sed 's/^.*://' || true)
if [ -n "$prev" ] && [ "$prev" != "$TAG" ]; then
    echo "$prev" > "$APP_DIR/.previous-tag"
    log "recorded previous tag for rollback: $prev"
fi

sed -i "s|\(image:[[:space:]]*ghcr\.io/[^:]*${IMAGE_BASENAME}\):.*|\1:${TAG}|" "$COMPOSE_FILE"
grep -q "${IMAGE_BASENAME}:${TAG}" "$COMPOSE_FILE" \
    || die "failed to write tag $TAG into $COMPOSE_FILE — check the image: line format"
log "compose.yaml now pins $TAG"

log "pulling image..."
docker compose -f "$COMPOSE_FILE" pull

log "recreating container..."
docker compose -f "$COMPOSE_FILE" up -d --remove-orphans

# SQLite has one writer, so compose's stop-then-start recreate gap is correct, and the
# loop tolerates "container not found yet" at the start rather than failing immediately.
log "waiting for healthy (up to $((HEALTH_RETRIES * HEALTH_INTERVAL))s)..."
i=0
while [ "$i" -lt "$HEALTH_RETRIES" ]; do
    status=$(docker inspect -f '{{.State.Health.Status}}' "$CONTAINER_NAME" 2>/dev/null || echo "starting")
    case "$status" in
        healthy)
            log "healthy after $((i * HEALTH_INTERVAL))s"
            log "deploy of $TAG complete"
            exit 0
            ;;
        unhealthy)
            log "container reported UNHEALTHY — last 50 log lines:"
            docker compose -f "$COMPOSE_FILE" logs --tail=50 || true
            die "release $TAG never became healthy (unhealthy) — roll back with: sh $0 \$(cat $APP_DIR/.previous-tag)"
            ;;
    esac
    i=$((i + 1))
    sleep "$HEALTH_INTERVAL"
done

log "timed out waiting for healthy — last 50 log lines:"
docker compose -f "$COMPOSE_FILE" logs --tail=50 || true
die "release $TAG never became healthy within $((HEALTH_RETRIES * HEALTH_INTERVAL))s — roll back with: sh $0 \$(cat $APP_DIR/.previous-tag)"
