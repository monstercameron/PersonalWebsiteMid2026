package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/monstercameron/earlcameron/internal/resume"
	"github.com/monstercameron/earlcameron/internal/store"
	"github.com/monstercameron/earlcameron/proto/sitepb"
)

// registerResumeRoutes wires the public résumé document (print-to-PDF). Résumé tailoring is an admin
// action and lives on AdminService (gRPC) — consumed by the wasm admin console, not an HTTP form.
func (s *Server) registerResumeRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/resume", s.resumePage)
}

// resumePage serves a résumé as a clean, print-optimized HTML document (Save as PDF). With
// ?variant=<id> it renders that saved tailoring variant; otherwise the active override if one is
// applied, else the permanent base résumé.
func (s *Server) resumePage(w http.ResponseWriter, r *http.Request) {
	doc := resume.Data()
	note := ""
	if idStr := r.URL.Query().Get("variant"); idStr != "" {
		if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			if t, ok, _ := s.store.GetTailoring(r.Context(), id); ok {
				if d, ok := resumeFromTailoring(t.Result); ok {
					doc, note = d, "Tailored variant"
				}
			}
		}
	} else if v, _ := s.store.GetSetting(r.Context(), store.SettingActiveResume); v != "" {
		var dom resume.Resume
		if json.Unmarshal([]byte(v), &dom) == nil && dom.Name != "" {
			doc = dom
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, resume.RenderHTML(doc, note))
}

// resumeFromTailoring extracts the domain résumé from a stored TailorResult (protojson).
func resumeFromTailoring(result string) (resume.Resume, bool) {
	var tr sitepb.TailorResult
	if err := protojson.Unmarshal([]byte(result), &tr); err != nil || tr.GetResume() == nil {
		return resume.Resume{}, false
	}
	p := tr.GetResume()
	if p.GetName() == "" {
		return resume.Resume{}, false
	}
	out := resume.Resume{
		Name: p.GetName(), Title: p.GetTitle(), Location: p.GetLocation(), Email: p.GetEmail(),
		GitHub: p.GetGithub(), LinkedIn: p.GetLinkedin(), Summary: p.GetSummary(), Edu: p.GetEducation(),
	}
	for _, j := range p.GetJobs() {
		out.Jobs = append(out.Jobs, resume.Job{Role: j.GetRole(), Org: j.GetOrg(), Dates: j.GetDates(), Bullets: j.GetBullets()})
	}
	for _, sk := range p.GetSkills() {
		out.Skills = append(out.Skills, resume.SkillGroup{Label: sk.GetLabel(), Items: sk.GetItems()})
	}
	for _, pr := range p.GetProjects() {
		out.Projects = append(out.Projects, resume.Project{Name: pr.GetName(), Desc: pr.GetDesc()})
	}
	return out, true
}
