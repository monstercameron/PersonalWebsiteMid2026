package server

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/monstercameron/earlcameron/internal/rss"
)

// feedBaseURL decides the origin the RSS feeds print inside themselves — the channel <link>, the
// atom:self href, and every item guid.
//
// It exists because those URLs outlive the request. A feed reader stores them and follows them for
// months, so an origin that was merely *reachable* at generation time is not good enough: on
// 2026-07-29 both public feeds were advertising `http://167.99.232.99:8080` — a raw IP on a
// non-standard port — to anyone who subscribed through the real domain, because that is what
// BASE_URL happened to be set to on the box.
//
// So: prefer the hostname the request actually arrived on when it is a real name, and fall back to
// the configured BASE_URL otherwise. A request that came in on a name proves that name resolves
// here; the configured value proves nothing.
//
// The Host header is attacker-controlled, so this deliberately cannot be pointed at another origin:
// a candidate is accepted only if it is a DNS name (an IP literal is rejected, which is the whole
// point), and only the host is taken — never a path or a scheme from the client. The scheme comes
// from X-Forwarded-Proto, which nginx sets and which a direct client cannot forge past it.
func feedBaseURL(r *http.Request, configured string) string {
	host := r.Host
	if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
		// A comma-joined list means several proxies appended; the first is the client-facing one.
		host = strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	if isDNSName(hostOnly(host)) {
		scheme := "https"
		if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
			scheme = strings.TrimSpace(strings.Split(p, ",")[0])
		} else if r.TLS == nil {
			scheme = "http"
		}
		return scheme + "://" + host
	}
	return strings.TrimRight(configured, "/")
}

// hostOnly strips a :port suffix, tolerating IPv6 literals in brackets.
func hostOnly(hostport string) string {
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		return h
	}
	return hostport
}

// isDNSName reports whether host is a routable domain name rather than an IP literal, "localhost",
// or empty. The dot requirement is what rejects both bare IPs and single-label internal names.
func isDNSName(host string) bool {
	if host == "" || net.ParseIP(host) != nil {
		return false
	}
	return strings.Contains(host, ".") && !strings.ContainsAny(host, " /\\")
}

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

// animeFeed serves the tracked-releases RSS feed (spec-compliant RSS 2.0 via internal/rss).
func (s *Server) animeFeed(w http.ResponseWriter, r *http.Request) {
	list, err := s.anime.List(r.Context())
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	xml, err := rss.TrackedFeedXML(list, feedBaseURL(r, s.cfg.BaseURL), time.Now())
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	_, _ = fmt.Fprint(w, xml)
}

// qotdFeedItemCap bounds how many posts the QOTD feed serves. Older posts stay recorded in the
// qotd_posts table; they just fall off the feed.
const qotdFeedItemCap = 30

// qotdFeed serves the QOTD RSS feed from the stored post history: the newest posts, one item per
// published day (an empty but valid feed until the first publish). The content is never generated
// on request — this only renders what the scheduler/manual publish already recorded.
func (s *Server) qotdFeed(w http.ResponseWriter, r *http.Request) {
	posts, err := s.store.RecentQOTDPosts(r.Context(), qotdFeedItemCap)
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	xml, err := rss.PublishedFeedXML(posts, feedBaseURL(r, s.cfg.BaseURL), time.Now())
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	_, _ = fmt.Fprint(w, xml)
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
