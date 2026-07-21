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
go build -o bin/server.exe ./cmd/server

echo "==> done."
echo "    run: LISTEN_ADDR=127.0.0.1:8095 ./bin/server.exe   then open http://127.0.0.1:8095"
