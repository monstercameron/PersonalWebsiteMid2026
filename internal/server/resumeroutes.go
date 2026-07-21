package server

import (
	"fmt"
	"net/http"

	"github.com/monstercameron/earlcameron/internal/resume"
)

// registerResumeRoutes wires the public résumé document (print-to-PDF). Résumé tailoring is an admin
// action and lives on AdminService (gRPC) — consumed by the wasm admin console, not an HTTP form.
func (s *Server) registerResumeRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/resume", s.resumePage)
}

// resumePage serves the canonical résumé as a clean, print-optimized HTML document.
func (s *Server) resumePage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, resume.RenderHTML(resume.Data(), ""))
}
