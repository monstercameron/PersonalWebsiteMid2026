package server

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
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
// SECURITY — the Host header is attacker-controlled, and "is it a DNS name?" is NOT a sufficient
// test on its own. `curl -H 'Host: evil.example' https://www.earlcameron.com/anime.xml` reaches this
// handler, and the first version of this function would have answered with a feed advertising
// evil.example as its permanent home. On its own that only poisons the attacker's own response, but
// feeds are exactly the kind of cacheable, long-lived GET that ends up in a shared cache, and the
// output is a URL other people are invited to follow for months. So the request host is honoured
// only when it MATCHES the configured origin's host:
//
//   - BASE_URL is a real domain (the correct configuration): only that host is ever echoed, and a
//     forged Host falls back to BASE_URL. Injection is impossible.
//   - BASE_URL is an IP, a port, or empty (the misconfiguration this function exists to survive):
//     any DNS-name host is accepted, because there is no trustworthy configured value to prefer and
//     an advertised IP is a certain failure versus a hypothetical one. Fix BASE_URL to close this.
//
// Only the host is ever taken — never a path or a scheme from the client — and the scheme comes
// from X-Forwarded-Proto, which nginx sets.
func feedBaseURL(r *http.Request, configured string) string {
	host := r.Host
	if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
		// A comma-joined list means several proxies appended; the first is the client-facing one.
		host = strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	configured = strings.TrimRight(configured, "/")
	if !isDNSName(hostOnly(host)) || !hostAllowed(hostOnly(host), configured) {
		return configured
	}
	scheme := "https"
	if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
		scheme = strings.TrimSpace(strings.Split(p, ",")[0])
	} else if r.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + host
}

// hostAllowed reports whether a request host may be echoed into generated URLs. It matches the
// configured origin's host case-insensitively; when the configured origin has no usable hostname it
// allows any DNS name, which is the degraded mode described on feedBaseURL.
func hostAllowed(host, configured string) bool {
	cfgHost := configuredHost(configured)
	if !isDNSName(cfgHost) {
		return true // nothing trustworthy to compare against — see feedBaseURL.
	}
	return strings.EqualFold(host, cfgHost)
}

// configuredHost extracts the hostname from a configured base URL, tolerating a missing scheme.
func configuredHost(configured string) string {
	if configured == "" {
		return ""
	}
	if u, err := url.Parse(configured); err == nil && u.Host != "" {
		return hostOnly(u.Host)
	}
	return hostOnly(configured)
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
