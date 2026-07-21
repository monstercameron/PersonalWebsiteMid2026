// Package admin holds the owner-gated control plane: password sessions (session.go), the gRPC
// auth interceptor (interceptor.go), and the AdminService implementation (this file) that backs
// the anime tracker and the résumé tailoring tool.
package admin

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/monstercameron/earlcameron/internal/anime"
	"github.com/monstercameron/earlcameron/internal/openai"
	"github.com/monstercameron/earlcameron/internal/resume"
	"github.com/monstercameron/earlcameron/internal/store"
	"github.com/monstercameron/earlcameron/proto/sitepb"
)

// Service implements sitepb.AdminServiceServer — the anime tracker and résumé tailoring data plane.
// Auth is enforced upstream by Sessions.UnaryAuthInterceptor (every method but Login), so the
// methods here assume the caller is already authenticated.
type Service struct {
	sitepb.UnimplementedAdminServiceServer
	anime    *anime.Service
	sessions *Sessions
	store    *store.Store
	// openAI resolves the effective OpenAI key + model at call time (a DB setting overrides the env
	// default), so a key added via the settings page takes effect without a restart. Empty key
	// disables résumé tailoring.
	openAI func(context.Context) (key, model string)
}

// NewService builds the admin gRPC service over the anime service, session manager, store (for
// web-editable settings), and an OpenAI config resolver.
func NewService(a *anime.Service, s *Sessions, st *store.Store, openAI func(context.Context) (string, string)) *Service {
	return &Service{anime: a, sessions: s, store: st, openAI: openAI}
}

// GetSettings returns the current settings — the API key itself is never returned, only whether one
// is configured (key_set) plus the effective model.
func (s *Service) GetSettings(ctx context.Context, _ *sitepb.Empty) (*sitepb.Settings, error) {
	key, model := s.openAI(ctx)
	return &sitepb.Settings{OpenaiModel: model, KeySet: key != ""}, nil
}

// SaveSettings persists submitted settings. A blank openai_api_key leaves the stored key unchanged.
func (s *Service) SaveSettings(ctx context.Context, req *sitepb.Settings) (*sitepb.Ack, error) {
	if k := strings.TrimSpace(req.GetOpenaiApiKey()); k != "" {
		if err := s.store.SetSetting(ctx, store.SettingOpenAIKey, k); err != nil {
			return &sitepb.Ack{Ok: false, Message: err.Error()}, nil
		}
	}
	if m := strings.TrimSpace(req.GetOpenaiModel()); m != "" {
		if err := s.store.SetSetting(ctx, store.SettingOpenAIModel, m); err != nil {
			return &sitepb.Ack{Ok: false, Message: err.Error()}, nil
		}
	}
	return &sitepb.Ack{Ok: true}, nil
}

// ListModels returns the chat models available to the stored OpenAI key (empty if no key / on error).
func (s *Service) ListModels(ctx context.Context, _ *sitepb.Empty) (*sitepb.ModelList, error) {
	key, _ := s.openAI(ctx)
	if key == "" {
		return &sitepb.ModelList{}, nil
	}
	models, err := openai.ListModels(ctx, key)
	if err != nil {
		return &sitepb.ModelList{}, nil // best-effort; the client falls back to a text field
	}
	return &sitepb.ModelList{Models: models}, nil
}

// Login exchanges the admin password for a signed session token used by every other method.
func (s *Service) Login(ctx context.Context, req *sitepb.LoginRequest) (*sitepb.LoginReply, error) {
	if !s.sessions.CheckCredentials(req.GetUsername(), req.GetPassword()) {
		// Throttle brute-forcing: a fixed delay on failure caps attempts to ~1/sec per connection
		// without a lockout that a griefer could use to lock the owner out.
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
		}
		return &sitepb.LoginReply{Ok: false}, nil
	}
	return &sitepb.LoginReply{Ok: true, Token: s.sessions.Mint()}, nil
}

// SearchAnime queries AniList and flags results that are already tracked.
func (s *Service) SearchAnime(ctx context.Context, req *sitepb.SearchRequest) (*sitepb.AnimeList, error) {
	results, err := s.anime.Search(ctx, req.GetQuery())
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "anime search failed: %v", err)
	}
	tracked, _ := s.anime.List(ctx)
	isTracked := make(map[int]bool, len(tracked))
	for _, a := range tracked {
		isTracked[a.AniListID] = true
	}
	out := &sitepb.AnimeList{}
	for _, m := range results {
		out.Items = append(out.Items, &sitepb.Anime{
			AnilistId: int32(m.ID), Title: m.DisplayTitle(), Format: m.Format, Status: m.Status,
			Episodes: int32(m.Episodes), SeasonYear: int32(m.SeasonYear), CoverImage: m.CoverImage.Large,
			Tracked: isTracked[m.ID],
		})
	}
	return out, nil
}

// ListTracked returns the currently tracked shows.
func (s *Service) ListTracked(ctx context.Context, _ *sitepb.Empty) (*sitepb.AnimeList, error) {
	list, err := s.anime.List(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list failed: %v", err)
	}
	out := &sitepb.AnimeList{}
	for _, a := range list {
		out.Items = append(out.Items, trackedToProto(a))
	}
	return out, nil
}

// TrackAnime begins tracking an AniList id.
func (s *Service) TrackAnime(ctx context.Context, req *sitepb.AnimeId) (*sitepb.Ack, error) {
	if err := s.anime.Track(ctx, int(req.GetAnilistId())); err != nil {
		return &sitepb.Ack{Ok: false, Message: err.Error()}, nil
	}
	return &sitepb.Ack{Ok: true}, nil
}

// UntrackAnime stops tracking an AniList id.
func (s *Service) UntrackAnime(ctx context.Context, req *sitepb.AnimeId) (*sitepb.Ack, error) {
	if err := s.anime.Untrack(ctx, int(req.GetAnilistId())); err != nil {
		return &sitepb.Ack{Ok: false, Message: err.Error()}, nil
	}
	return &sitepb.Ack{Ok: true}, nil
}

// RunReleaseCheck refreshes tracked shows against AniList and reports how many changed.
func (s *Service) RunReleaseCheck(ctx context.Context, _ *sitepb.Empty) (*sitepb.CheckReply, error) {
	n, err := s.anime.RunReleaseCheck(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "release check failed: %v", err)
	}
	return &sitepb.CheckReply{Updated: int32(n)}, nil
}

// GetResume returns the active résumé: the applied override if one is stored, else the canonical.
func (s *Service) GetResume(ctx context.Context, _ *sitepb.Empty) (*sitepb.Resume, error) {
	if v, _ := s.store.GetSetting(ctx, store.SettingActiveResume); v != "" {
		var dom resume.Resume
		if json.Unmarshal([]byte(v), &dom) == nil && dom.Name != "" {
			return resumeToProto(dom), nil
		}
	}
	return resumeToProto(resume.Data()), nil
}

// ApplyResume saves the given résumé as the active one (served by GetResume + the /resume page).
func (s *Service) ApplyResume(ctx context.Context, req *sitepb.Resume) (*sitepb.Ack, error) {
	data, err := json.Marshal(protoToResume(req))
	if err != nil {
		return &sitepb.Ack{Ok: false, Message: err.Error()}, nil
	}
	if err := s.store.SetSetting(ctx, store.SettingActiveResume, string(data)); err != nil {
		return &sitepb.Ack{Ok: false, Message: err.Error()}, nil
	}
	return &sitepb.Ack{Ok: true}, nil
}

// TailorResume fetches the job posting and returns a résumé re-emphasized to fit it. The result is
// constrained to the canonical résumé (resume.Tailor), so the model cannot fabricate credentials.
func (s *Service) TailorResume(ctx context.Context, req *sitepb.TailorRequest) (*sitepb.TailorResult, error) {
	key, model := s.openAI(ctx)
	if key == "" {
		return nil, status.Error(codes.FailedPrecondition, "tailoring disabled — add an OpenAI API key in settings")
	}
	jobText, err := resume.FetchJobText(ctx, req.GetJobUrl())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "couldn't fetch that URL: %v", err)
	}
	res, err := resume.Tailor(ctx, key, model, jobText, resume.Data())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "tailoring failed: %v", err)
	}
	out := &sitepb.TailorResult{
		Resume: resumeToProto(res.Resume),
		Job: &sitepb.JobAnalysis{
			Title: res.Job.Title, Company: res.Job.Company,
			Keywords: res.Job.Keywords, Requirements: res.Job.Requirements,
		},
	}
	for _, r := range res.Rationales {
		out.Rationales = append(out.Rationales, &sitepb.Rationale{Focus: r.Focus, Reason: r.Reason})
	}
	// Persist the variant (it cost an OpenAI call) with glanceable metadata so it survives restarts,
	// shows in the list, and can be re-opened.
	if data, err := protojson.Marshal(out); err == nil {
		_, _ = s.store.SaveTailoring(ctx, req.GetJobUrl(), res.Job.Title, res.Job.Company, string(data), time.Now().Unix())
	}
	return out, nil
}

// GetLastTailoring returns the most recent saved tailoring (an empty result if none exists).
func (s *Service) GetLastTailoring(ctx context.Context, _ *sitepb.Empty) (*sitepb.TailorResult, error) {
	t, ok, err := s.store.LatestTailoring(ctx)
	if err != nil || !ok {
		return &sitepb.TailorResult{}, nil
	}
	return unmarshalResult(t.Result), nil
}

// GetBaseResume returns the permanent canonical résumé (the diff baseline; never overwritten).
func (s *Service) GetBaseResume(_ context.Context, _ *sitepb.Empty) (*sitepb.Resume, error) {
	return resumeToProto(resume.Data()), nil
}

// ListTailorings returns saved variants (metadata only), newest first.
func (s *Service) ListTailorings(ctx context.Context, _ *sitepb.Empty) (*sitepb.TailoringList, error) {
	list, err := s.store.ListTailorings(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list failed: %v", err)
	}
	out := &sitepb.TailoringList{}
	for _, t := range list {
		out.Items = append(out.Items, &sitepb.TailoringMeta{
			Id: t.ID, JobUrl: t.JobURL, Title: t.Title, Company: t.Company, CreatedAt: t.CreatedAt,
		})
	}
	return out, nil
}

// GetTailoring loads one saved variant's full result by id.
func (s *Service) GetTailoring(ctx context.Context, req *sitepb.TailoringId) (*sitepb.TailorResult, error) {
	t, ok, err := s.store.GetTailoring(ctx, req.GetId())
	if err != nil || !ok {
		return &sitepb.TailorResult{}, nil
	}
	return unmarshalResult(t.Result), nil
}

// DeleteTailoring removes a saved variant by id.
func (s *Service) DeleteTailoring(ctx context.Context, req *sitepb.TailoringId) (*sitepb.Ack, error) {
	if err := s.store.DeleteTailoring(ctx, req.GetId()); err != nil {
		return &sitepb.Ack{Ok: false, Message: err.Error()}, nil
	}
	return &sitepb.Ack{Ok: true}, nil
}

// unmarshalResult decodes a stored TailorResult (protojson), returning an empty result on error.
func unmarshalResult(data string) *sitepb.TailorResult {
	var out sitepb.TailorResult
	if err := protojson.Unmarshal([]byte(data), &out); err != nil {
		return &sitepb.TailorResult{}
	}
	return &out
}

// trackedToProto maps a stored tracked show to the wire DTO.
func trackedToProto(a store.TrackedAnime) *sitepb.Anime {
	return &sitepb.Anime{
		AnilistId: int32(a.AniListID), Title: a.Title, Format: a.Format, Status: a.Status,
		Episodes: int32(a.Episodes), SeasonYear: int32(a.SeasonYear), CoverImage: a.CoverImage, Tracked: true,
	}
}

// resumeToProto maps a domain résumé to the wire DTO.
func resumeToProto(r resume.Resume) *sitepb.Resume {
	out := &sitepb.Resume{
		Name: r.Name, Title: r.Title, Location: r.Location, Email: r.Email,
		Github: r.GitHub, Linkedin: r.LinkedIn, Summary: r.Summary, Education: r.Edu,
	}
	for _, j := range r.Jobs {
		out.Jobs = append(out.Jobs, &sitepb.ResumeJob{Role: j.Role, Org: j.Org, Dates: j.Dates, Bullets: j.Bullets})
	}
	for _, sk := range r.Skills {
		out.Skills = append(out.Skills, &sitepb.ResumeSkill{Label: sk.Label, Items: sk.Items})
	}
	for _, p := range r.Projects {
		out.Projects = append(out.Projects, &sitepb.ResumeProject{Name: p.Name, Desc: p.Desc})
	}
	return out
}

// protoToResume maps a wire résumé DTO back to the domain type (inverse of resumeToProto).
func protoToResume(p *sitepb.Resume) resume.Resume {
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
	return out
}
