#!/usr/bin/env bash
# Run the client/ unit tests.
#
# The terminal's logic — the parser, the history, the line editor, the virtual filesystem — lives
# in a package built only for GOOS=js/GOARCH=wasm, so `go test ./...` skips it entirely: it does
# not appear in the build at all, and a suite CI never runs is not a suite. This runs it through
# node using the wasm exec shim that ships with the Go toolchain.
#
# Usage:  bash scripts/test-wasm.sh [extra go test flags]
set -euo pipefail
cd "$(dirname "$0")/.."

SHIM="$(go env GOROOT)/lib/wasm/wasm_exec_node.js"
if [ ! -f "$SHIM" ]; then
  echo "wasm exec shim not found at $SHIM" >&2
  exit 1
fi
if ! command -v node >/dev/null 2>&1; then
  echo "node is required to run the js/wasm tests" >&2
  exit 1
fi

echo "==> go test (js/wasm) ./client"
GOOS=js GOARCH=wasm go test -exec "node $SHIM" "$@" ./client/
