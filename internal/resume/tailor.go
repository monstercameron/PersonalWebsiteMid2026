package resume

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// maxOpenAIBody caps how much of the OpenAI response we read, matching the job-fetch cap.
const maxOpenAIBody = 1 << 20 // 1 MiB

// Tailor asks the model to tailor the base résumé to a job posting and returns a refined Resume.
// Guardrail: the model may only reorder, emphasize, and lightly rephrase EXISTING facts — never
// fabricate. Identity/contact fields are forced back to the originals as a hard safety net.
func Tailor(ctx context.Context, apiKey, model, jobText string, base Resume) (Resume, error) {
	baseJSON, err := json.Marshal(base)
	if err != nil {
		return base, err
	}
	system := "You tailor résumés to a specific job posting. Focus on two things: the professional " +
		"summary and the experience BULLET POINTS — reorder, emphasize, and lightly rephrase them to " +
		"surface what matches the job. HARD RULES you must never break: only rephrase facts that " +
		"ALREADY EXIST in the input; NEVER invent or add experience, skills, employers, titles, dates, " +
		"projects, or education, and never exaggerate seniority or metrics. Do not add bullets — you may " +
		"only reorder, trim, or reword existing ones. Everything else (employers, titles, dates, skills, " +
		"projects, education, contact) is kept exactly as given. Keep the same JSON schema. Return ONLY " +
		"a JSON object."
	user := "RÉSUMÉ JSON (the only facts you may use):\n" + string(baseJSON) + "\n\nJOB POSTING:\n" + jobText

	reqBody, err := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.2,
	})
	if err != nil {
		return base, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return base, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := (&http.Client{Timeout: 45 * time.Second}).Do(req)
	if err != nil {
		return base, err
	}
	defer resp.Body.Close()
	body := io.LimitReader(resp.Body, maxOpenAIBody)
	if resp.StatusCode != http.StatusOK {
		var eb bytes.Buffer
		_, _ = eb.ReadFrom(body)
		return base, fmt.Errorf("openai status %d: %s", resp.StatusCode, truncate(eb.String(), 300))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(body).Decode(&out); err != nil {
		return base, err
	}
	if len(out.Choices) == 0 {
		return base, fmt.Errorf("openai: empty response")
	}
	var tailored Resume
	if err := json.Unmarshal([]byte(out.Choices[0].Message.Content), &tailored); err != nil {
		return base, fmt.Errorf("could not parse tailored résumé: %w", err)
	}
	return constrainToBase(base, tailored), nil
}

// constrainToBase rebuilds the tailored résumé on the canonical skeleton so the model can only
// affect the summary and the wording/selection of experience bullets. Every hard credential —
// employers, titles, dates, skills, projects, education, and contact info — is taken from the base
// résumé, not the model. This is the actual enforcement of the "never invents" guarantee: the
// system prompt is advisory, but a prompt-injecting job posting cannot get fabricated employers or
// degrees past this function.
func constrainToBase(base, t Resume) Resume {
	out := base // struct copy; the untouched slices (Skills/Projects/Edu) stay pinned to base
	if s := strings.TrimSpace(t.Summary); s != "" {
		out.Summary = s
	}
	jobs := make([]Job, len(base.Jobs))
	for i, bj := range base.Jobs {
		jobs[i] = bj // Org/Role/Dates always from the canonical résumé
		if i < len(t.Jobs) {
			if b := cleanBullets(t.Jobs[i].Bullets); len(b) > 0 {
				if len(b) > len(bj.Bullets) {
					b = b[:len(bj.Bullets)] // reorder/trim/reword allowed; padding is not
				}
				jobs[i].Bullets = b
			}
		}
	}
	out.Jobs = jobs
	return out
}

// cleanBullets trims and drops empty bullet strings.
func cleanBullets(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// truncate shortens s to n bytes for error messages.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
