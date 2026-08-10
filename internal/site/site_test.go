package site

import (
	"strings"
	"testing"

	"github.com/monstercameron/earlcameron/proto/sitepb"
)

// TestHomepageCashFluxLinksUsePublicDemo verifies every CashFlux anchor rendered by the homepage
// points to one canonical target, and that the target is the PUBLIC build.
//
// This reverses an earlier decision, on purpose. The links used to point at
// budget.earlcameron.com because it was the "canonical standalone deployment" — but that instance
// is Cam's own, password-gated, and holds real financial data. A portfolio read by recruiters and
// crawlers should not advertise it. The retired list below is therefore the personal instance and
// the old in-site /budget/ mount; the canonical target is the GitHub Pages build, which starts
// empty and keeps everything in the visitor's own browser.
func TestHomepageCashFluxLinksUsePublicDemo(t *testing.T) {
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

	// Two visitor-facing links: the project card's demo and the ~/elsewhere card.
	if got, want := strings.Count(html, `href="`+cashFluxURL+`"`), 2; got != want {
		t.Errorf("public CashFlux link count = %d, want %d", got, want)
	}
	// Exactly one owner link, in the top navigation. More than one would mean a visitor-facing
	// surface had drifted onto the private instance, which is the thing this test exists to stop.
	if got, want := strings.Count(html, `href="`+cashFluxOwnerURL+`"`), 1; got != want {
		t.Errorf("owner CashFlux link count = %d, want %d", got, want)
	}
	if strings.Contains(html, `href="/budget/"`) {
		t.Error(`homepage still contains the retired in-site /budget/ mount`)
	}
}
