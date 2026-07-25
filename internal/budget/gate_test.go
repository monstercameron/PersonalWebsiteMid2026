package budget

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestGateNeverAnswersAssetRequestsWithHTML is the regression guard for a cache-poisoning bug:
// a service worker precaches "./bin/main.wasm.gz" in the background with whatever cookies it
// has — none, on a first visit — and the gate used to answer EVERY unauthenticated request with
// the lock page at 200. The SW then cached HTML under the wasm's URL, and the next boot tried to
// instantiate a login page as WebAssembly and hung with nothing to act on.
func TestGateNeverAnswersAssetRequestsWithHTML(t *testing.T) {
	g := NewGate("hunter2", "test-secret")
	app := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("APP")) })
	h := g.Wrap(app)

	// A service worker fetching the wasm: no session, no text/html in Accept.
	req := httptest.NewRequest(http.MethodGet, "/bin/main.wasm.gz", nil)
	req.Header.Set("Sec-Fetch-Mode", "no-cors")
	req.Header.Set("Accept", "*/*")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("asset request status = %d, want 401", rec.Code)
	}
	if body := rec.Body.String(); strings.Contains(body, "<html") || strings.Contains(body, "locked") {
		t.Fatalf("asset request got HTML back (%d bytes) — a service worker would cache this as wasm", len(body))
	}

	// A real navigation still gets the door.
	nav := httptest.NewRequest(http.MethodGet, "/", nil)
	nav.Header.Set("Sec-Fetch-Mode", "navigate")
	navRec := httptest.NewRecorder()
	h.ServeHTTP(navRec, nav)
	if navRec.Code != http.StatusOK || !strings.Contains(navRec.Body.String(), "<html") {
		t.Fatalf("navigation status = %d, want 200 with the gate page", navRec.Code)
	}
}

// TestGateFallsBackToAcceptHeader covers a client that sends no Sec-Fetch-Mode: the Accept header
// still separates a page navigation from an asset fetch.
func TestGateFallsBackToAcceptHeader(t *testing.T) {
	g := NewGate("hunter2", "test-secret")
	h := g.Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))

	asset := httptest.NewRequest(http.MethodGet, "/bin/main.wasm.gz", nil)
	asset.Header.Set("Accept", "*/*")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, asset)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("asset without Sec-Fetch-Mode: status = %d, want 401", rec.Code)
	}

	nav := httptest.NewRequest(http.MethodGet, "/", nil)
	nav.Header.Set("Accept", "text/html,application/xhtml+xml")
	navRec := httptest.NewRecorder()
	h.ServeHTTP(navRec, nav)
	if navRec.Code != http.StatusOK {
		t.Fatalf("navigation without Sec-Fetch-Mode: status = %d, want 200", navRec.Code)
	}
}
