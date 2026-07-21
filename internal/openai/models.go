// Package openai holds small clients for the OpenAI REST API used by the admin tools (listing the
// models available to a key). The chat-completions call for résumé tailoring lives in
// internal/resume; this package is the shared, non-résumé-specific surface.
package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// ListModels returns the chat-capable model IDs available to apiKey, sorted. It calls GET
// /v1/models and filters to families usable with chat completions (what the résumé tailor needs),
// dropping audio/image/embedding/realtime/instruct variants.
func ListModels(ctx context.Context, apiKey string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai models: status %d", resp.StatusCode)
	}
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return nil, err
	}
	var ids []string
	for _, m := range out.Data {
		if isChatModel(m.ID) {
			ids = append(ids, m.ID)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// isChatModel reports whether a model id belongs to a chat-completions-capable family and is not a
// non-chat variant (audio, image, embeddings, etc.).
func isChatModel(id string) bool {
	family := false
	for _, p := range []string{"gpt-", "chatgpt", "o1", "o3", "o4"} {
		if strings.HasPrefix(id, p) {
			family = true
			break
		}
	}
	if !family {
		return false
	}
	for _, bad := range []string{"instruct", "realtime", "audio", "image", "tts", "transcribe", "embedding", "moderation", "search", "dall"} {
		if strings.Contains(id, bad) {
			return false
		}
	}
	return true
}
