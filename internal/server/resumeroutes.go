package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/monstercameron/earlcameron/internal/resume"
	"github.com/monstercameron/earlcameron/internal/store"
)

// registerResumeRoutes wires the public résumé document (print-to-PDF). Résumé tailoring is an admin
// action and lives on AdminService (gRPC) — consumed by the wasm admin console, not an HTTP form.
func (s *Server) registerResumeRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/resume", s.resumePage)
}

// resumePage serves the active résumé (the applied override if one exists, else the canonical) as a
// clean, print-optimized HTML document.
func (s *Server) resumePage(w http.ResponseWriter, r *http.Request) {
	doc := resume.Data()
	if v, _ := s.store.GetSetting(r.Context(), store.SettingActiveResume); v != "" {
		var dom resume.Resume
		if json.Unmarshal([]byte(v), &dom) == nil && dom.Name != "" {
			doc = dom
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, resume.RenderHTML(doc, ""))
}
