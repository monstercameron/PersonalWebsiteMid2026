package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// maxGenerateBody caps how much of an OpenAI response we read, so a hostile/broken endpoint can't
// stream unbounded data into memory.
const maxGenerateBody = 1 << 20 // 1 MiB

// BaseURL is the OpenAI API origin. It is a variable only so tests can point Generate at a local
// httptest server; production code must never change it.
var BaseURL = "https://api.openai.com"

// Generate calls the OpenAI Responses API (/v1/responses) for free-form text: instructions is the
// system role, input is the user role. It returns the aggregated output text. temperature is
// intentionally omitted — newer models only accept the default and reject an explicit value.
func Generate(ctx context.Context, apiKey, model, instructions, input string) (string, error) {
	reqBody, err := json.Marshal(map[string]any{
		"model": model,
		"input": []map[string]string{
			{"role": "system", "content": instructions},
			{"role": "user", "content": input},
		},
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, BaseURL+"/v1/responses", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body := io.LimitReader(resp.Body, maxGenerateBody)
	if resp.StatusCode != http.StatusOK {
		var eb bytes.Buffer
		_, _ = eb.ReadFrom(body)
		return "", fmt.Errorf("openai status %d: %s", resp.StatusCode, truncateBody(eb.String(), 300))
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
		return "", err
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
		return "", fmt.Errorf("openai: empty response")
	}
	return content, nil
}

// truncateBody shortens s to at most n runes, appending an ellipsis when cut.
func truncateBody(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
