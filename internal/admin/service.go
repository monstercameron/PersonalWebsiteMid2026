// Package admin holds the owner-gated control plane: password sessions (session.go), the gRPC
// auth interceptor (interceptor.go), and the AdminService implementation (this file) that backs
// the anime tracker and the résumé tailoring tool.
package admin

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/monstercameron/earlcameron/internal/anime"
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
	// openAI resolves the effective OpenAI key + model at call time (a DB setting overrides the env
	// default), so a key added via the settings page takes effect without a restart. Empty key
	// disables résumé tailoring.
	openAI func(context.Context) (key, model string)
}

// NewService builds the admin gRPC service over the anime service, session manager, and an OpenAI
// config resolver.
func NewService(a *anime.Service, s *Sessions, openAI func(context.Context) (string, string)) *Service {
	return &Service{anime: a, sessions: s, openAI: openAI}
}

// Login exchanges the admin password for a signed session token used by every other method.
func (s *Service) Login(_ context.Context, req *sitepb.LoginRequest) (*sitepb.LoginReply, error) {
	if !s.sessions.CheckCredentials(req.GetUsername(), req.GetPassword()) {
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

// GetResume returns the canonical résumé data.
func (s *Service) GetResume(_ context.Context, _ *sitepb.Empty) (*sitepb.Resume, error) {
	return resumeToProto(resume.Data()), nil
}

// TailorResume fetches the job posting and returns a résumé re-emphasized to fit it. The result is
// constrained to the canonical résumé (resume.Tailor), so the model cannot fabricate credentials.
func (s *Service) TailorResume(ctx context.Context, req *sitepb.TailorRequest) (*sitepb.Resume, error) {
	key, model := s.openAI(ctx)
	if key == "" {
		return nil, status.Error(codes.FailedPrecondition, "tailoring disabled — add an OpenAI API key in settings")
	}
	jobText, err := resume.FetchJobText(ctx, req.GetJobUrl())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "couldn't fetch that URL: %v", err)
	}
	tailored, err := resume.Tailor(ctx, key, model, jobText, resume.Data())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "tailoring failed: %v", err)
	}
	return resumeToProto(tailored), nil
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
