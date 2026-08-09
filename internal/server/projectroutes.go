package server

import (
	"net/http"
	"strings"

	"github.com/monstercameron/earlcameron/internal/content"
	"github.com/monstercameron/earlcameron/internal/site"
	"github.com/monstercameron/earlcameron/internal/theme"
)

// registerProjectRoutes wires the per-product Flux case-study pages at /projects/<slug>.
//
// These are document-plane resources (HTTP GET): server-rendered HTML, so they are crawlable,
// unfurl correctly when a recruiter pastes the link into Slack or LinkedIn, and work with no
// WebAssembly at all — which is the whole reason the standard site exists (DESIGN.md §3).
//
// The pattern is registered ahead of the "/" catch-all, and Go 1.22+ ServeMux picks the most
// specific match, so an unknown slug lands here rather than silently rendering the home page.
func (s *Server) registerProjectRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/projects/", s.projectPage)
	mux.HandleFunc("/projects", s.projectIndex)
}

// projectIndex sends bare /projects to the home page's work section. There is no separate index
// page: the home page already lists every project, and a second list would be one more surface to
// keep in sync with internal/content for no reader benefit.
func (s *Server) projectIndex(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/#work", http.StatusFound)
}

// projectPage serves one pre-rendered Flux project page.
//
// Pages are rendered once at startup (buildProjectPages) rather than per request: the content is
// a compile-time constant, so re-rendering it on every hit would burn CPU to produce identical
// bytes. An unknown slug 404s with the site's own copy instead of a bare net/http string.
func (s *Server) projectPage(w http.ResponseWriter, r *http.Request) {
	slug := strings.Trim(strings.TrimPrefix(r.URL.Path, "/projects/"), "/")
	page, ok := s.projectPages[slug]
	if !ok {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`<!doctype html><meta charset="utf-8">` +
			`<title>No such project — Earl Cameron</title>` +
			`<body style="margin:0;background:#17040f;color:#f3e9e6;` +
			`font-family:ui-monospace,SFMono-Regular,Menlo,monospace;padding:3rem">` +
			`<p>No project by that name.</p>` +
			`<p><a style="color:#e95420" href="/#work">← see the work</a></p>`))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(page)
}

// buildProjectPages renders every Flux project page to bytes, keyed by slug. It returns an error
// on the first page that fails to render, so a template mistake stops the process at boot rather
// than serving four good pages and one blank one.
func buildProjectPages(baseURL string) (map[string][]byte, error) {
	all := content.FluxProjects()
	pages := make(map[string][]byte, len(all))
	for _, p := range all {
		brand, ok := theme.Brands[p.Slug]
		if !ok {
			// A project without a palette would render with a zero-value (transparent) accent —
			// invisible text rather than a visible failure. Fail loudly at boot instead.
			return nil, &missingBrandError{slug: p.Slug}
		}
		html, err := site.RenderProjectHTML(p, brand, all, baseURL)
		if err != nil {
			return nil, err
		}
		pages[p.Slug] = []byte(html)
	}
	return pages, nil
}

// missingBrandError reports a Flux project that has no entry in theme.Brands.
type missingBrandError struct{ slug string }

// Error implements the error interface.
func (e *missingBrandError) Error() string {
	return "no theme.Brands entry for Flux project " + e.slug
}
