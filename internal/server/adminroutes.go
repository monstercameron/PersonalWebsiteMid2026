package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/monstercameron/earlcameron/internal/anime"
)

// registerAdminRoutes wires the public anime RSS feeds and the admin console shell.
//
// The admin UI is a wasm app (client/) that talks to the backend purely over gRPC (AdminService,
// via the GoGRPCBridge tunnel). Every /admin* path serves the same shell; the client routes
// internally and holds its JWT in localStorage. There are no admin HTTP form endpoints. The RSS
// feeds and the résumé page remain on the document plane (HTTP GET).
func (s *Server) registerAdminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/anime.xml", s.animeFeed)     // public RSS: tracked releases
	mux.HandleFunc("/anime/qotd.xml", s.qotdFeed) // public RSS: daily prompts
	mux.HandleFunc("/admin", s.adminShell)        // admin console (wasm)
	mux.HandleFunc("/admin/", s.adminShell)       // client-routed sub-paths
}

// animeFeed serves the tracked-releases RSS feed.
func (s *Server) animeFeed(w http.ResponseWriter, r *http.Request) {
	list, err := s.anime.List(r.Context())
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	_, _ = fmt.Fprint(w, s.anime.TrackedFeedXML(list, s.cfg.BaseURL))
}

// qotdFeed serves the daily-prompts RSS feed.
func (s *Server) qotdFeed(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	_, _ = fmt.Fprint(w, anime.QuestionFeedXML(s.cfg.BaseURL, time.Now()))
}

// adminShell serves the admin console shell: a minimal document that mounts #admin-root and boots
// the wasm client. All interactivity happens client-side over gRPC.
func (s *Server) adminShell(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, adminShellHTML)
}

// adminShellHTML is the wasm bootstrap document for the admin app. The <script> lines are the same
// wasm glue used by the public site; the base styles are minimal (the css/u utilities are injected
// at runtime by the wasm client). The wasm main() routes to the admin app for /admin* paths.
const adminShellHTML = `<!doctype html><html lang="en"><head><meta charset="utf-8">` +
	`<meta name="viewport" content="width=device-width,initial-scale=1"><title>admin · Earl Cameron</title>` +
	`<style>*{box-sizing:border-box}html,body{margin:0}body{color:#f3e9e6;` +
	`font-family:ui-monospace,"SF Mono",SFMono-Regular,Menlo,"Cascadia Code","JetBrains Mono",monospace;` +
	`background:radial-gradient(60vw 50vw at 12% -8%,rgba(190,123,230,.16),transparent 60%),` +
	`radial-gradient(55vw 55vw at 105% 115%,rgba(233,84,32,.14),transparent 55%),#17040f;` +
	`background-attachment:fixed;min-height:100vh}input::placeholder{color:#a98ba0}a{color:#be7be6;text-decoration:none}` +
	`select{color:#f3e9e6;background:#210a19}</style></head><body>` +
	`<div id="admin-root"></div>` +
	`<script src="/static/wasm_exec.js"></script>` +
	`<script>const go=new Go();WebAssembly.instantiateStreaming(fetch("/static/app.wasm"),go.importObject)` +
	`.then(function(r){go.run(r.instance);}).catch(function(e){console.error("wasm boot failed",e);});</script>` +
	`</body></html>`
