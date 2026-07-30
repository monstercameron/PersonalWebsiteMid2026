package content

import (
	"context"
	"testing"

	"github.com/monstercameron/earlcameron/proto/sitepb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestListProjects verifies the featured set is non-empty and every project has the fields the
// UI depends on (id, name, repo).
func TestListProjects(t *testing.T) {
	list, err := New().ListProjects(context.Background(), &sitepb.LocaleRequest{})
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(list.GetProjects()) == 0 {
		t.Fatal("no projects returned")
	}
	for _, p := range list.GetProjects() {
		if p.GetId() == "" || p.GetName() == "" || p.GetRepo() == "" {
			t.Errorf("project missing id/name/repo: %+v", p)
		}
	}
}

// TestGetProjectFound returns a known project by id.
func TestGetProjectFound(t *testing.T) {
	p, err := New().GetProject(context.Background(), &sitepb.ProjectRequest{Id: "gwc"})
	if err != nil {
		t.Fatalf("GetProject gwc: %v", err)
	}
	if p.GetName() != "GoWebComponents" {
		t.Errorf("got name %q, want GoWebComponents", p.GetName())
	}
}

// TestCashFluxDemoUsesDedicatedService guards the homepage project card against falling back to
// the retired GitHub Pages demo or the portfolio's legacy /budget/ mount.
func TestCashFluxDemoUsesDedicatedService(t *testing.T) {
	p, err := New().GetProject(context.Background(), &sitepb.ProjectRequest{Id: "cashflux"})
	if err != nil {
		t.Fatalf("GetProject cashflux: %v", err)
	}
	if got, want := p.GetDemo(), "https://budget.earlcameron.com"; got != want {
		t.Errorf("CashFlux demo = %q, want %q", got, want)
	}
}

// TestGetProjectNotFound maps an unknown id to a NotFound status.
func TestGetProjectNotFound(t *testing.T) {
	_, err := New().GetProject(context.Background(), &sitepb.ProjectRequest{Id: "does-not-exist"})
	if status.Code(err) != codes.NotFound {
		t.Errorf("want NotFound, got %v", err)
	}
}
