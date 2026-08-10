#!/usr/bin/env bash
# Build the earlcameron.com artifacts: the wasm client, the wasm_exec.js shim, and the server.
# Artifacts land in web/static (wasm + shim) and bin (server binary) — all gitignored.
set -euo pipefail
cd "$(dirname "$0")/.."

echo "==> building wasm client (GOOS=js GOARCH=wasm)"
GOOS=js GOARCH=wasm go build -o web/static/app.wasm ./client

echo "==> copying wasm_exec.js from the Go toolchain"
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/static/wasm_exec.js

echo "==> building server"
# The binary name carries the platform's executable suffix — "server.exe" on Windows,
# plain "server" on the Ubuntu droplet — so deploy/ can reference one predictable path.
SERVER_BIN="bin/server$(go env GOEXE)"
# Stamp the build so the terminal's `stats` can report which commit is serving. An unstamped build
# reports "dev", which is the honest answer for one — the point of the command is that its numbers
# are real, so a made-up version would defeat it.
VERSION="$(git rev-parse --short HEAD 2>/dev/null || echo dev)"
if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
  VERSION="$VERSION-dirty"
fi
go build -ldflags "-X github.com/monstercameron/earlcameron/internal/system.Version=$VERSION" -o "$SERVER_BIN" ./cmd/server

echo "==> done."
echo "    run: LISTEN_ADDR=127.0.0.1:8095 ./$SERVER_BIN   then open http://127.0.0.1:8095"
