#!/usr/bin/env bash
# Build (or fall back to copying) the CashFlux WASM frontend and stage it under
# web/cashflux/ so this server can host it as a "managed" budgeting app.
#
# CashFlux (C:\Users\mreca\Desktop\CashFlux) is a separate, local-first Go/WASM project with its
# own module, its own build tooling (gwc), and its own gitignored build outputs. We never write
# INTO that tree's source files; the only thing we touch there is `go build -o <tmpdir>/...`,
# which the Go toolchain treats as an ordinary out-of-tree build (no files change under
# CashFlux/web except its own pre-existing, gitignored web/bin/ if the WASM_OUT temp step below
# ends up pointed there — by default it does not).
#
# Usage:
#   scripts/build-cashflux.sh
#
# Env overrides:
#   CASHFLUX_SRC       Path to the CashFlux checkout (default: C:\Users\mreca\Desktop\CashFlux)
#   BUDGET_MOUNT_PATH  URL path this server will mount the app under (default: /budget/).
#                      Rewritten into the copied index.html's <base href> so root-relative
#                      asset references (./bin/main.wasm, ./wasm_exec.js, ...) resolve correctly
#                      when served from a subpath instead of the domain root.
#
# Output: web/cashflux/ (index.html, wasm_exec.js, bin/main.wasm[.gz], vendored JS, fonts,
# audio, icons, manifest, service worker) — idempotent, safe to re-run.
set -euo pipefail
cd "$(dirname "$0")/.."
PROJECT_ROOT="$(pwd)"

CASHFLUX_SRC="${CASHFLUX_SRC:-C:/Users/mreca/Desktop/CashFlux}"
BUDGET_MOUNT_PATH="${BUDGET_MOUNT_PATH:-/budget/}"
OUT_DIR="$PROJECT_ROOT/web/cashflux"

echo "==> CashFlux source:  $CASHFLUX_SRC"
echo "==> Output directory: $OUT_DIR"
echo "==> Mount path:       $BUDGET_MOUNT_PATH"

if [ ! -f "$CASHFLUX_SRC/main.go" ] || [ ! -f "$CASHFLUX_SRC/web/index.html" ]; then
  echo "!! CashFlux checkout not found or missing main.go / web/index.html at: $CASHFLUX_SRC" >&2
  exit 1
fi

mkdir -p "$OUT_DIR/bin"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

BUILD_MODE="fresh-build"
echo "==> attempting fresh WASM build (GOOS=js GOARCH=wasm go build .)"
if ( cd "$CASHFLUX_SRC" && GOOS=js GOARCH=wasm go build -o "$TMP_DIR/main.wasm" . ); then
  echo "==> build succeeded"
  echo "==> gzip-compressing main.wasm -> main.wasm.gz"
  gzip -9 -c "$TMP_DIR/main.wasm" > "$TMP_DIR/main.wasm.gz"
  cp "$TMP_DIR/main.wasm" "$OUT_DIR/bin/main.wasm"
  cp "$TMP_DIR/main.wasm.gz" "$OUT_DIR/bin/main.wasm.gz"
else
  echo "!! fresh build failed; falling back to CashFlux's existing prebuilt artifact" >&2
  BUILD_MODE="fallback-prebuilt"
  if [ ! -f "$CASHFLUX_SRC/web/bin/main.wasm" ]; then
    echo "!! no prebuilt $CASHFLUX_SRC/web/bin/main.wasm to fall back to either — aborting" >&2
    exit 1
  fi
  cp "$CASHFLUX_SRC/web/bin/main.wasm" "$OUT_DIR/bin/main.wasm"
  if [ -f "$CASHFLUX_SRC/web/bin/main.wasm.gz" ]; then
    cp "$CASHFLUX_SRC/web/bin/main.wasm.gz" "$OUT_DIR/bin/main.wasm.gz"
  else
    gzip -9 -c "$OUT_DIR/bin/main.wasm" > "$OUT_DIR/bin/main.wasm.gz"
  fi
fi

echo "==> copying static SPA assets (index.html, wasm_exec.js, vendored JS, fonts, audio, icons, PWA files)"
# NOTE: web/admin and web/portal are CashFlux's own separate sub-apps (admin console, billing
# portal) — intentionally excluded; this build only stages the main budgeting SPA.
STATIC_FILES=(
  index.html
  favicon.svg icon-192.png icon-512.png apple-touch-icon.png
  manifest.webmanifest sw.js fonts.css
  d3.min.js chart.js countup.js muzak.js wonder.js
  mermaid.min.js mermaid.js marked.min.js purify.min.js
)
for f in "${STATIC_FILES[@]}"; do
  cp "$CASHFLUX_SRC/web/$f" "$OUT_DIR/$f"
done

# wasm_exec.js is handled separately because CashFlux GITIGNORES it: it is a toolchain
# artifact, not source, so a fresh clone (i.e. the deploy droplet) does not have one and
# the loop above would die on a missing file. Prefer the checkout's copy when present —
# that is the local-dev case and keeps behaviour identical — and otherwise take it from
# the Go toolchain doing the build, which is the copy that actually matches the wasm we
# just compiled.
if [ -f "$CASHFLUX_SRC/web/wasm_exec.js" ]; then
  cp "$CASHFLUX_SRC/web/wasm_exec.js" "$OUT_DIR/wasm_exec.js"
else
  echo "==> CashFlux checkout has no web/wasm_exec.js (gitignored); using the toolchain's"
  cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$OUT_DIR/wasm_exec.js"
fi

mkdir -p "$OUT_DIR/fonts" "$OUT_DIR/audio"
cp -r "$CASHFLUX_SRC/web/fonts/." "$OUT_DIR/fonts/"
cp "$CASHFLUX_SRC/web/audio/"*.mp3 "$OUT_DIR/audio/" 2>/dev/null || true

# Stamp the service-worker cache key with a hash of the wasm we just staged.
#
# sw.js PRECACHES ./bin/main.wasm.gz under `const CACHE = "cashflux-vNNN"` and only
# evicts caches whose key differs from it. That constant is bumped by hand on
# release — so shipping a new wasm without bumping it leaves every returning
# browser serving the OLD binary from Cache Storage, forever. A hard refresh does
# not help: it bypasses the HTTP cache, not a service worker answering from its own
# store. (This bit us for a full day of builds on 2026-07-24.)
#
# Deriving the key from the artifact makes the failure impossible rather than
# merely documented: the key changes exactly when the bytes change, sw.js therefore
# differs, the browser re-installs it, and the stale precache is evicted.
WASM_HASH="$(sha256sum "$OUT_DIR/bin/main.wasm.gz" | cut -c1-12)"
echo "==> stamping sw.js cache key with wasm hash $WASM_HASH"
sed -i "s/^const CACHE = \".*\";/const CACHE = \"cashflux-${WASM_HASH}\";/" "$OUT_DIR/sw.js"
grep -q "cashflux-${WASM_HASH}" "$OUT_DIR/sw.js" || { echo "!! failed to stamp sw.js cache key" >&2; exit 1; }

echo "==> rewriting <base href> in index.html for mount path $BUDGET_MOUNT_PATH"
sed -i "s|<base href=\"/\" />|<base href=\"${BUDGET_MOUNT_PATH}\" />|" "$OUT_DIR/index.html"

echo "==> done ($BUILD_MODE). Artifact sizes:"
du -h "$OUT_DIR/bin/main.wasm" "$OUT_DIR/bin/main.wasm.gz" 2>/dev/null || true
du -sh "$OUT_DIR" 2>/dev/null || true
echo "==> run 'go build ./internal/budget/' and see internal/budget/serve.go for the HTTP handler."
