package resume

import "testing"

// TestConstrainToBaseBlocksFabrication verifies the prompt-injection guardrail: a model response
// that tries to invent employers, degrees, skills, or extra bullets is stripped back to the
// canonical résumé — only the summary and (bounded, reworded) bullets survive.
func TestConstrainToBaseBlocksFabrication(t *testing.T) {
	base := Data()
	evil := Resume{
		Name:     "Someone Else",
		Email:    "attacker@evil.test",
		Title:    "Fake Title",
		Location: "Nowhere",
		Summary:  "A legitimately tailored summary.",
		Jobs: []Job{{
			Role: "Chief Everything Officer", Org: "FakeCorp", Dates: "1990 — present",
			Bullets: []string{"b1", "b2", "b3", "b4", "b5", "b6", "b7", "b8", "b9", "b10"},
		}, {
			Role: "Invented Second Job", Org: "AlsoFake", Dates: "1980",
		}},
		Skills:   []SkillGroup{{"Quantum", "Time travel"}},
		Projects: []Project{{"Fabricated", "did not happen"}},
		Edu:      []string{"PhD, Fake University"},
	}
	out := constrainToBase(base, evil)

	if out.Name != base.Name || out.Email != base.Email || out.Location != base.Location {
		t.Error("identity fields must come from base")
	}
	if out.Title != base.Title {
		t.Error("title must come from base")
	}
	if len(out.Jobs) != len(base.Jobs) {
		t.Fatalf("job count must match base (%d), got %d", len(base.Jobs), len(out.Jobs))
	}
	if out.Jobs[0].Org != base.Jobs[0].Org || out.Jobs[0].Role != base.Jobs[0].Role || out.Jobs[0].Dates != base.Jobs[0].Dates {
		t.Error("employer/role/dates must come from base")
	}
	if len(out.Jobs[0].Bullets) > len(base.Jobs[0].Bullets) {
		t.Errorf("bullets must be capped at base count %d, got %d", len(base.Jobs[0].Bullets), len(out.Jobs[0].Bullets))
	}
	if len(out.Skills) != len(base.Skills) || out.Skills[0].Label != base.Skills[0].Label {
		t.Error("skills must come from base")
	}
	if len(out.Edu) != len(base.Edu) || out.Edu[0] != base.Edu[0] {
		t.Error("education must come from base")
	}
	if len(out.Projects) != len(base.Projects) || out.Projects[0].Name != base.Projects[0].Name {
		t.Error("projects must come from base")
	}
	if out.Summary != "A legitimately tailored summary." {
		t.Error("a non-empty tailored summary should be applied")
	}
}

// TestConstrainToBaseKeepsRewordedBullets confirms the intended tailoring surface still works:
// reworded/reordered bullets within an existing job are preserved.
func TestConstrainToBaseKeepsRewordedBullets(t *testing.T) {
	base := Data()
	t2 := Data()
	t2.Jobs[0].Bullets = []string{"Reworded bullet emphasizing the job's keywords."}
	out := constrainToBase(base, t2)
	if len(out.Jobs[0].Bullets) != 1 || out.Jobs[0].Bullets[0] != "Reworded bullet emphasizing the job's keywords." {
		t.Error("reworded bullets should be preserved")
	}
}
