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
go build -o "$SERVER_BIN" ./cmd/server

echo "==> done."
echo "    run: LISTEN_ADDR=127.0.0.1:8095 ./$SERVER_BIN   then open http://127.0.0.1:8095"
