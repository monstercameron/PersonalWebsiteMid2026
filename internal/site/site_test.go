package site

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/monstercameron/earlcameron/internal/content"
	"github.com/monstercameron/earlcameron/internal/theme"
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

// TestFluxArtAssetsExist holds every project's Art flag to the filesystem.
//
// The pages reference brand files by slug, so a project marked Art:true without its four files
// renders broken images on its own page, in the sibling strip on all five others, and in the card
// a link unfurler builds — none of which fails a build or a unit test. This is the check that
// turns "did you remember the assets?" into a red test.
func TestFluxArtAssetsExist(t *testing.T) {
	for _, p := range content.FluxProjects() {
		if !p.Art {
			continue
		}
		for _, suffix := range []string{"-mark.webp", "-logo.webp", "-poster.webp", "-og.jpg"} {
			path := filepath.Join("..", "..", "web", "static", "brand", p.Slug+suffix)
			info, err := os.Stat(path)
			if err != nil {
				t.Errorf("%s declares Art:true but %s is missing: %v", p.Slug, p.Slug+suffix, err)
				continue
			}
			if info.Size() == 0 {
				t.Errorf("%s: %s is empty", p.Slug, p.Slug+suffix)
			}
		}
	}
}

// TestEveryFluxProjectHasABrandPalette catches the other half of the same drift: a page whose slug
// has no theme.Brands entry renders with a zero-value colour, which is not an error anywhere.
func TestEveryFluxProjectHasABrandPalette(t *testing.T) {
	for _, p := range content.FluxProjects() {
		if _, ok := theme.Brands[p.Slug]; !ok {
			t.Errorf("%s has no theme.Brands entry", p.Slug)
		}
	}
}

// TestFluxPagesRenderWithoutBrokenAssetRefs renders every project page and asserts that each
// brand asset it points at exists on disk.
func TestFluxPagesRenderWithoutBrokenAssetRefs(t *testing.T) {
	all := content.FluxProjects()
	ref := regexp.MustCompile(`/static/brand/([a-z0-9-]+\.(?:webp|jpg|png))`)
	for _, p := range all {
		html, err := RenderProjectHTML(p, theme.Brands[p.Slug], all, "https://example.com")
		if err != nil {
			t.Fatalf("%s: render: %v", p.Slug, err)
		}
		for _, m := range ref.FindAllStringSubmatch(html, -1) {
			if _, err := os.Stat(filepath.Join("..", "..", "web", "static", "brand", m[1])); err != nil {
				t.Errorf("%s page references %s, which does not exist", p.Slug, m[1])
			}
		}
	}
}

// TestSpecOnlyProjectDoesNotClaimShippedCode guards the one sentence on these pages that would be
// a lie for a project that is still a specification.
func TestSpecOnlyProjectDoesNotClaimShippedCode(t *testing.T) {
	p, ok := content.FluxProjectBySlug("animefeedflux")
	if !ok {
		t.Fatal("animefeedflux is missing")
	}
	html, err := RenderProjectHTML(p, theme.Brands[p.Slug], content.FluxProjects(), "https://example.com")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(html, "The product is real and the code is linked above") {
		t.Error("a specification-stage project must not claim its product is real")
	}
}
