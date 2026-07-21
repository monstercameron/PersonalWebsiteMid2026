package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/monstercameron/earlcameron/internal/adminui"
	"github.com/monstercameron/earlcameron/internal/resume"
)

// registerResumeRoutes wires the public résumé document and the (temporary) HTTP tailoring tool.
//
// `/resume` is a document-plane page (HTTP GET, print-to-PDF) and stays here. The `/admin/resume*`
// handlers are the legacy HTTP admin UI — the admin data plane now lives on AdminService (gRPC);
// these remain only until the GWC/WASM admin console replaces them, then they're deleted.
func (s *Server) registerResumeRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/resume", s.resumePage)                     // public: print-to-PDF résumé (document)
	mux.HandleFunc("/admin/resume", s.adminResume)              // legacy HTTP tailoring UI
	mux.HandleFunc("/admin/resume/tailor", s.adminResumeTailor) // legacy HTTP tailoring UI
}

// resumePage serves the canonical résumé as a clean, print-optimized HTML document.
func (s *Server) resumePage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, resume.RenderHTML(resume.Data(), ""))
}

// adminResume renders the legacy HTTP résumé tailoring tool (owner-gated).
func (s *Server) adminResume(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	key, _ := s.openAI(r.Context())
	page, err := adminui.ResumeTool(key != "", "")
	writeHTML(w, page, err)
}

// adminResumeTailor fetches a job posting URL, tailors the résumé to it, and renders the result.
// Owner-gated and POST-only (a GET with ?url= would let a SameSite=Lax navigation trigger a
// CSRF-driven SSRF). The tailoring itself is constrained to the canonical résumé (see resume.Tailor).
func (s *Server) adminResumeTailor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAdmin(w, r) {
		return
	}
	key, model := s.openAI(r.Context())
	if key == "" {
		page, err := adminui.ResumeTool(false, "Tailoring is disabled — add an OpenAI API key in settings.")
		writeHTML(w, page, err)
		return
	}
	jobURL := strings.TrimSpace(r.FormValue("url"))
	if !strings.HasPrefix(jobURL, "http://") && !strings.HasPrefix(jobURL, "https://") {
		page, err := adminui.ResumeTool(true, "Enter a full job-posting URL (http/https).")
		writeHTML(w, page, err)
		return
	}
	jobText, err := resume.FetchJobText(r.Context(), jobURL)
	if err != nil {
		page, rerr := adminui.ResumeTool(true, "Couldn't fetch that URL: "+err.Error())
		writeHTML(w, page, rerr)
		return
	}
	tailored, err := resume.Tailor(r.Context(), key, model, jobText, resume.Data())
	if err != nil {
		page, rerr := adminui.ResumeTool(true, "Tailoring failed: "+err.Error())
		writeHTML(w, page, rerr)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, resume.RenderHTML(tailored, "Tailored to: "+jobURL+" — review every line before you send it."))
}

