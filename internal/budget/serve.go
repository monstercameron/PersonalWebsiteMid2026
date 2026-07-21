// Package budget hosts the CashFlux budgeting app as a "managed" (server-served) frontend.
//
// CashFlux (github.com/monstercameron/CashFlux) is a separate, local-first Go/WASM project: all
// user data lives in the browser's IndexedDB, and the app has no server-side data plane of its
// own. "Managed" here means we serve its already-built static SPA (index.html, wasm_exec.js, the
// compiled main.wasm/.gz, and its vendored JS/fonts/audio/icons) from this server under a mount
// path, so a visitor gets the full budgeting app without needing the CashFlux repo locally. It
// does NOT add server-side sync, accounts, or storage for CashFlux's data — that would be a
// separate, larger project (see the WIRING note in the repo root for how this package's handler
// is meant to be mounted, and what a fuller managed backend would still need).
//
// The static payload is produced by scripts/build-cashflux.sh, which builds (or, on build
// failure, copies a prebuilt) CashFlux's WASM frontend into web/cashflux/. This package only
// knows how to serve that directory; it does not build it.
package budget

import (
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// DefaultRoot is the on-disk directory the handler serves from when built via Handler(). It is
// relative to the server process's working directory, matching how the rest of this project
// serves static assets (e.g. internal/server's "web/static").
const DefaultRoot = "web/cashflux"

// indexFile is the SPA entry point served for "/" and for any path that doesn't resolve to a
// real file under the served root — client-side routing (the CashFlux router) then takes over.
const indexFile = "index.html"

// Handler returns an http.Handler that serves the built CashFlux frontend from DefaultRoot
// ("web/cashflux"). Mount it behind http.StripPrefix so the app sees root-relative paths, e.g.:
//
//	mux.Handle("/budget/", http.StripPrefix("/budget/", budget.Handler()))
//
// See the WIRING section in this project for the full mounting instructions.
func Handler() http.Handler {
	return NewHandler(DefaultRoot)
}

// NewHandler returns an http.Handler that serves a built CashFlux frontend from the given root
// directory. Splitting this out from Handler lets callers (and tests) point at an arbitrary
// directory instead of the compiled-in default.
func NewHandler(root string) http.Handler {
	return &spaHandler{root: root}
}

// spaHandler serves a single-page app's static build directory: real files are served as-is with
// an explicit, correct Content-Type; any other path (one with no file extension, e.g. a
// client-side route like "/reports") falls back to the SPA's index.html so the in-app router can
// take over. Paths that look like a missing asset (they have a file extension) 404 instead of
// silently returning index.html, so a broken/renamed asset reference is loud, not swallowed.
type spaHandler struct {
	root string
}

// ServeHTTP resolves the request path against the served root and writes the matching file (or
// the SPA fallback), setting an explicit Content-Type in every case — see contentTypeFor for why
// this handler never lets the standard library guess.
func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clean := path.Clean("/" + r.URL.Path)
	name := clean
	if name == "/" {
		name = "/" + indexFile
	}

	full := filepath.Join(h.root, filepath.FromSlash(strings.TrimPrefix(name, "/")))
	// Explicit containment: never touch the filesystem for a path that resolves outside the root
	// (defense in depth — filepath.Join is OS-native, so on Windows a "..\" could otherwise escape).
	if rel, err := filepath.Rel(h.root, full); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		http.NotFound(w, r)
		return
	}
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		if path.Ext(clean) != "" {
			// Looks like a missing asset (has an extension) — a real 404, not a route.
			http.NotFound(w, r)
			return
		}
		full = filepath.Join(h.root, indexFile)
	}

	w.Header().Set("Content-Type", contentTypeFor(full))
	http.ServeFile(w, r, full)
}

// contentTypeFor returns the Content-Type this handler serves a file with, based on its name.
//
// It never relies on the standard library's extension-based guessing for the two cases that
// matter most here:
//
//   - "*.wasm" must be "application/wasm" so WebAssembly.instantiateStreaming can stream-compile
//     it; some OS mime databases don't know the extension and Go would otherwise fall back to
//     "application/octet-stream", which instantiateStreaming rejects in some browsers.
//   - "*.wasm.gz" is served as opaque, still-gzipped bytes: CashFlux's own loader
//     (web/index.html) fetches it and pipes the raw bytes through a client-side
//     DecompressionStream itself. Critically, this handler must NEVER set a Content-Encoding
//     header for it — if it did, the browser's HTTP layer would transparently decompress the
//     body before JS ever sees it, and the app's manual DecompressionStream would then fail
//     trying to decompress already-decompressed bytes. contentTypeFor only ever sets
//     Content-Type, so that hazard can't happen by omission.
//
// Everything else falls back to the standard mime package, with a hard default of
// "application/octet-stream" for anything unrecognized (fonts, audio, images, manifest, etc. all
// resolve via their normal extensions).
func contentTypeFor(name string) string {
	switch {
	case strings.HasSuffix(name, ".wasm.gz"):
		return "application/wasm"
	case strings.HasSuffix(name, ".wasm"):
		return "application/wasm"
	case strings.HasSuffix(name, ".webmanifest"):
		return "application/manifest+json"
	case strings.HasSuffix(name, ".woff2"):
		return "font/woff2"
	}
	if ct := mime.TypeByExtension(filepath.Ext(name)); ct != "" {
		return ct
	}
	return "application/octet-stream"
}
