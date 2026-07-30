package site

import (
	"strings"
	"testing"

	"github.com/monstercameron/earlcameron/proto/sitepb"
)

// TestHomepageCashFluxLinksUseDedicatedService verifies every CashFlux anchor rendered by the
// homepage points to the canonical standalone deployment.
func TestHomepageCashFluxLinksUseDedicatedService(t *testing.T) {
	projects := []*sitepb.Project{{
		Id:   "cashflux",
		Name: "CashFlux",
		Repo: "https://github.com/monstercameron/CashFlux",
		Demo: cashFluxURL,
	}}
	html, err := RenderHTML(&sitepb.About{}, projects, "https://www.earlcameron.com")
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}

	if got, want := strings.Count(html, `href="`+cashFluxURL+`"`), 3; got != want {
		t.Errorf("CashFlux link count = %d, want %d", got, want)
	}
	for _, retired := range []string{`href="/budget/"`, "https://monstercameron.github.io/CashFlux/"} {
		if strings.Contains(html, retired) {
			t.Errorf("homepage still contains retired CashFlux target %q", retired)
		}
	}
}
