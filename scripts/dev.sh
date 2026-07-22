#!/usr/bin/env bash
# Local development server. Builds the wasm client + server and runs it on :8096 with convenience
# admin credentials (username / password) and a stable session secret + CashFlux sync token.
#
# NOT FOR PRODUCTION: the admin credentials here are intentionally trivial. A real deploy must set
# its own ADMIN_USERNAME/ADMIN_PASSWORD (or use first-run setup) and a strong ADMIN_SECRET — never
# these. Config defaults deliberately do NOT bake in a password, so an unconfigured production bind
# has no admin login until it's set up.
#
# Usage:  bash scripts/dev.sh
# Then:   open http://127.0.0.1:8096/admin  and log in with  username / password
set -euo pipefail
cd "$(dirname "$0")/.."

echo "==> building wasm client + server"
GOOS=js GOARCH=wasm go build -o web/static/app.wasm ./client
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/static/wasm_exec.js
go build -o bin/server.exe ./cmd/server

echo "==> serving on http://127.0.0.1:8096  (admin: username / password)"
LISTEN_ADDR="${LISTEN_ADDR:-127.0.0.1:8096}" \
  BASE_URL="${BASE_URL:-http://127.0.0.1:8096}" \
  ADMIN_USERNAME="${ADMIN_USERNAME:-username}" \
  ADMIN_PASSWORD="${ADMIN_PASSWORD:-password}" \
  ADMIN_SECRET="${ADMIN_SECRET:-dev-secret-stable}" \
  CASHFLUX_SERVER_TOKEN="${CASHFLUX_SERVER_TOKEN:-cam-sync-token}" \
  ./bin/server.exe
