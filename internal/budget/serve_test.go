package budget

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// newFixture builds a tiny fake CashFlux build directory (index.html, wasm_exec.js,
// bin/main.wasm, bin/main.wasm.gz) under a temp dir, so tests never touch the real ~20MB
// artifact web/cashflux/ produces.
func newFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "index.html"), "<html><body>cashflux fixture</body></html>")
	writeFile(t, filepath.Join(root, "wasm_exec.js"), "// fake go wasm glue")
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	writeFile(t, filepath.Join(root, "bin", "main.wasm"), "\x00asm-fixture-not-real-wasm")
	writeFile(t, filepath.Join(root, "bin", "main.wasm.gz"), "\x1f\x8b-fake-gzip-bytes")

	return root
}

func writeFile(t *testing.T, name, contents string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// TestServeIndexAtRoot verifies "/" serves the SPA's index.html.
func TestServeIndexAtRoot(t *testing.T) {
	h := NewHandler(newFixture(t))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "<html><body>cashflux fixture</body></html>" {
		t.Fatalf("body = %q, want the fixture index.html contents", got)
	}
}

// TestServeWasmContentType verifies main.wasm is served with the application/wasm content type
// required for WebAssembly.instantiateStreaming to accept it.
func TestServeWasmContentType(t *testing.T) {
	h := NewHandler(newFixture(t))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/bin/main.wasm", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/wasm" {
		t.Fatalf("Content-Type = %q, want application/wasm", ct)
	}
}

// TestServeWasmGzNoContentEncoding verifies main.wasm.gz is served as opaque bytes (its own
// Content-Type, but crucially NO Content-Encoding) — CashFlux's client decompresses it manually
// via DecompressionStream, so a server-set Content-Encoding: gzip would cause the browser to
// transparently decompress it first and break that client-side pipeline.
func TestServeWasmGzNoContentEncoding(t *testing.T) {
	h := NewHandler(newFixture(t))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/bin/main.wasm.gz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/wasm" {
		t.Fatalf("Content-Type = %q, want application/wasm", ct)
	}
	if enc := rec.Header().Get("Content-Encoding"); enc != "" {
		t.Fatalf("Content-Encoding = %q, want unset", enc)
	}
}

// TestServeSPAFallback verifies a client-side route with no file extension (e.g. "/reports",
// which only exists inside the CashFlux in-app router) falls back to index.html rather than
// 404ing, so a deep-link/hard-refresh on a sub-route still boots the app.
func TestServeSPAFallback(t *testing.T) {
	h := NewHandler(newFixture(t))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/reports", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "<html><body>cashflux fixture</body></html>" {
		t.Fatalf("body = %q, want the fixture index.html contents (SPA fallback)", got)
	}
}

// TestServeMissingAssetIs404 verifies a path that looks like a real asset (has a file extension)
// but doesn't exist 404s instead of silently returning index.html — a renamed/missing asset
// reference should be loud, not swallowed as a fake "route".
func TestServeMissingAssetIs404(t *testing.T) {
	h := NewHandler(newFixture(t))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/does-not-exist.png", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
