package server

import (
	"fmt"
	"html"
	"net/http"
	"strings"

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
	adminPage(w, "résumé tool", resumeToolBody(s.cfg.OpenAIKey != "", ""))
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
	if s.cfg.OpenAIKey == "" {
		adminPage(w, "résumé tool", resumeToolBody(false, "Tailoring is disabled — set OPENAI_API_KEY."))
		return
	}
	jobURL := strings.TrimSpace(r.FormValue("url"))
	if !strings.HasPrefix(jobURL, "http://") && !strings.HasPrefix(jobURL, "https://") {
		adminPage(w, "résumé tool", resumeToolBody(true, "Enter a full job-posting URL (http/https)."))
		return
	}
	jobText, err := resume.FetchJobText(r.Context(), jobURL)
	if err != nil {
		adminPage(w, "résumé tool", resumeToolBody(true, "Couldn't fetch that URL: "+err.Error()))
		return
	}
	tailored, err := resume.Tailor(r.Context(), s.cfg.OpenAIKey, s.cfg.OpenAIModel, jobText, resume.Data())
	if err != nil {
		adminPage(w, "résumé tool", resumeToolBody(true, "Tailoring failed: "+err.Error()))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, resume.RenderHTML(tailored, "Tailored to: "+jobURL+" — review every line before you send it."))
}

// resumeToolBody renders the legacy tailoring tool form. enabled=false shows a disabled notice; msg
// is an optional status/error line.
func resumeToolBody(enabled bool, msg string) string {
	var b strings.Builder
	b.WriteString(`<div class="top"><h1>résumé tool</h1>` +
		`<form method="post" action="/admin/logout"><button class="ghost">logout</button></form></div>`)
	b.WriteString(`<div class="feeds"><a href="/admin/anime">← anime</a> &nbsp; ` +
		`Canonical résumé: <a href="/resume" target="_blank">/resume</a> (open it, then Save as PDF).</div>`)
	if msg != "" {
		b.WriteString(`<p class="dim">` + html.EscapeString(msg) + `</p>`)
	}
	if !enabled {
		b.WriteString(`<p class="dim">Set the OPENAI_API_KEY environment variable to enable tailoring.</p>`)
		return b.String()
	}
	b.WriteString(`<form method="post" action="/admin/resume/tailor" class="row">` +
		`<input name="url" placeholder="paste a job-posting URL…" autofocus><button>Tailor résumé</button></form>`)
	b.WriteString(`<p class="dim">The tool fetches the posting and re-emphasizes real facts to fit it — ` +
		`it never invents experience. Review the result before sending, then Save as PDF.</p>`)
	return b.String()
}
