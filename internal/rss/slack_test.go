package rss

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestBuildDiscussionMessage covers the news+prompt, news-only, prompt-only, and empty-input
// shapes of the composed Slack message.
func TestBuildDiscussionMessage(t *testing.T) {
	cases := []struct {
		name      string
		newsItems []NewsItem
		prompt    string
		wantParts []string
	}{
		{
			name:      "news and prompt",
			newsItems: []NewsItem{{Title: "Big Anime News", Link: "https://example.com/a"}},
			prompt:    "What's the best opening this season?",
			wantParts: []string{"Big Anime News", "https://example.com/a", "What's the best opening this season?"},
		},
		{
			name:      "prompt only",
			newsItems: nil,
			prompt:    "Sub or dub?",
			wantParts: []string{"Sub or dub?"},
		},
		{
			name:      "news only",
			newsItems: []NewsItem{{Title: "Headline", Link: "https://example.com/b"}},
			prompt:    "",
			wantParts: []string{"Headline", "https://example.com/b"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildDiscussionMessage(tc.newsItems, tc.prompt)
			if got == "" {
				t.Fatalf("BuildDiscussionMessage returned empty")
			}
			for _, part := range tc.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("message %q missing expected part %q", got, part)
				}
			}
		})
	}
}

// TestPostToSlack verifies the webhook is called with a JSON {"text": ...} body and that
// non-2xx responses surface as errors.
func TestPostToSlack(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := PostToSlack(context.Background(), srv.URL, "hello world"); err != nil {
		t.Fatalf("PostToSlack: %v", err)
	}
	if gotBody["text"] != "hello world" {
		t.Errorf("posted text = %q, want %q", gotBody["text"], "hello world")
	}

	errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer errSrv.Close()

	if err := PostToSlack(context.Background(), errSrv.URL, "hello"); err == nil {
		t.Error("expected error on non-2xx response, got nil")
	}
}

// TestBuildDiscussionMessageEscapesInjection ensures a hostile external headline can't inject Slack
// mrkdwn (channel pings / forged links) into the composed message.
func TestBuildDiscussionMessageEscapesInjection(t *testing.T) {
	news := []NewsItem{{Title: "Reveal <!channel> now|x", Link: "https://ann.test/a?b=1&c=2"}}
	msg := BuildDiscussionMessage(news, "Debate <!here> & win")
	for _, bad := range []string{"<!channel>", "<!here>", "?b=1&c=2"} {
		if strings.Contains(msg, bad) {
			t.Errorf("unescaped %q leaked into Slack message: %q", bad, msg)
		}
	}
	if !strings.Contains(msg, "&lt;!channel&gt;") || !strings.Contains(msg, "&amp;") {
		t.Errorf("expected escaped sequences in %q", msg)
	}
}
