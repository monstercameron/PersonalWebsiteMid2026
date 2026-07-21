package resume

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Tailor asks the model to tailor the base résumé to a job posting and returns a refined Resume.
// Guardrail: the model may only reorder, emphasize, and lightly rephrase EXISTING facts — never
// fabricate. Identity/contact fields are forced back to the originals as a hard safety net.
func Tailor(ctx context.Context, apiKey, model, jobText string, base Resume) (Resume, error) {
	baseJSON, err := json.Marshal(base)
	if err != nil {
		return base, err
	}
	system := "You tailor résumés to a specific job posting. HARD RULES you must never break: " +
		"only reorder, emphasize, and lightly rephrase facts that ALREADY EXIST in the input. NEVER " +
		"invent or add experience, skills, employers, titles, dates, projects, or education. Do not " +
		"exaggerate seniority or metrics. You may trim to the most relevant items and re-order bullets " +
		"and skills to surface what matches the job. Keep the exact same JSON schema as the input. " +
		"Return ONLY a JSON object."
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
	if resp.StatusCode != http.StatusOK {
		var eb bytes.Buffer
		_, _ = eb.ReadFrom(resp.Body)
		return base, fmt.Errorf("openai status %d: %s", resp.StatusCode, truncate(eb.String(), 300))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return base, err
	}
	if len(out.Choices) == 0 {
		return base, fmt.Errorf("openai: empty response")
	}
	var tailored Resume
	if err := json.Unmarshal([]byte(out.Choices[0].Message.Content), &tailored); err != nil {
		return base, fmt.Errorf("could not parse tailored résumé: %w", err)
	}
	// Hard safety net: never let the model change identity/contact fields.
	tailored.Name = base.Name
	tailored.Email = base.Email
	tailored.GitHub = base.GitHub
	tailored.LinkedIn = base.LinkedIn
	tailored.Location = base.Location
	if len(tailored.Jobs) == 0 { // if the model returned something empty, fall back
		return base, fmt.Errorf("tailored résumé came back empty")
	}
	return tailored, nil
}

// truncate shortens s to n runes for error messages.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
