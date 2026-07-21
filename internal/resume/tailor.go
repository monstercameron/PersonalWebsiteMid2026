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

// TailorResult is the full output of a tailoring pass: the tailored résumé (constrained to the
// canonical one), the signals extracted from the posting, and the rationale for each choice.
type TailorResult struct {
	Resume     Resume
	Job        JobAnalysis
	Rationales []Rationale
}

// JobAnalysis is what the tool extracted from the job posting.
type JobAnalysis struct {
	Title        string
	Company      string
	Keywords     []string
	Requirements []string
}

// Rationale explains one tailoring decision.
type Rationale struct {
	Focus  string
	Reason string
}

// Tailor asks the model to (1) extract the key signals from the job posting, (2) tailor the base
// résumé to fit — only reordering/rephrasing existing summary + bullets, never fabricating — and
// (3) explain each choice. The résumé is passed through constrainToBase as a hard guarantee against
// fabrication regardless of what the model returns.
func Tailor(ctx context.Context, apiKey, model, jobText string, base Resume) (TailorResult, error) {
	baseJSON, err := json.Marshal(base)
	if err != nil {
		return TailorResult{}, err
	}
	system := "You tailor résumés to a job posting AND explain your reasoning. Return ONE JSON object " +
		"with exactly three keys:\n" +
		"\"job\": what you extracted from the posting — {title, company, keywords (key skills/terms the " +
		"posting emphasizes), requirements (key responsibilities/must-haves)}.\n" +
		"\"resume\": the tailored résumé in the SAME schema as the input. You may ONLY reorder, emphasize, " +
		"and lightly reword the professional summary and the existing experience BULLET POINTS. NEVER " +
		"invent or add experience, skills, employers, titles, dates, projects, or education; never " +
		"exaggerate. Everything except summary + bullets is kept exactly as given.\n" +
		"\"rationales\": a list of {focus, reason} objects — for each meaningful choice, focus names what " +
		"you emphasized/reworded and reason ties it to a specific job signal. Be concrete and honest."
	user := "RÉSUMÉ JSON (the only facts you may use):\n" + string(baseJSON) + "\n\nJOB POSTING:\n" + jobText

	// Uses the OpenAI Responses API (/v1/responses): `input` (not `messages`) and `text.format` for
	// the JSON-object output. temperature is intentionally omitted — newer models only accept the
	// default and reject any explicit value.
	reqBody, err := json.Marshal(map[string]any{
		"model": model,
		"input": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"text": map[string]any{
			"format": map[string]string{"type": "json_object"},
		},
	})
	if err != nil {
		return TailorResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/responses", bytes.NewReader(reqBody))
	if err != nil {
		return TailorResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return TailorResult{}, err
	}
	defer resp.Body.Close()
	body := io.LimitReader(resp.Body, maxOpenAIBody)
	if resp.StatusCode != http.StatusOK {
		var eb bytes.Buffer
		_, _ = eb.ReadFrom(body)
		return TailorResult{}, fmt.Errorf("openai status %d: %s", resp.StatusCode, truncate(eb.String(), 300))
	}
	// Responses API: prefer the aggregated output_text, else concatenate the message items' text.
	var out struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.NewDecoder(body).Decode(&out); err != nil {
		return TailorResult{}, err
	}
	content := out.OutputText
	if content == "" {
		for _, item := range out.Output {
			if item.Type != "message" {
				continue
			}
			for _, c := range item.Content {
				if c.Type == "output_text" {
					content += c.Text
				}
			}
		}
	}
	if content == "" {
		return TailorResult{}, fmt.Errorf("openai: empty response")
	}

	var parsed struct {
		Job struct {
			Title        string   `json:"title"`
			Company      string   `json:"company"`
			Keywords     []string `json:"keywords"`
			Requirements []string `json:"requirements"`
		} `json:"job"`
		Resume     Resume `json:"resume"`
		Rationales []struct {
			Focus  string `json:"focus"`
			Reason string `json:"reason"`
		} `json:"rationales"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return TailorResult{}, fmt.Errorf("could not parse tailoring result: %w", err)
	}

	result := TailorResult{
		Resume: constrainToBase(base, parsed.Resume),
		Job: JobAnalysis{
			Title: parsed.Job.Title, Company: parsed.Job.Company,
			Keywords: parsed.Job.Keywords, Requirements: parsed.Job.Requirements,
		},
	}
	for _, r := range parsed.Rationales {
		if strings.TrimSpace(r.Focus) == "" && strings.TrimSpace(r.Reason) == "" {
			continue
		}
		result.Rationales = append(result.Rationales, Rationale{Focus: r.Focus, Reason: r.Reason})
	}
	return result, nil
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
