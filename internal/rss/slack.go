package rss

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// slackPostTimeout bounds a single webhook POST.
const slackPostTimeout = 10 * time.Second

// slackPayload is the body Slack's incoming-webhook API expects: a plain text message.
type slackPayload struct {
	Text string `json:"text"`
}

// PostToSlack posts text to a Slack incoming webhook. It returns an error if the request can't
// be built/sent or the webhook responds with a non-2xx status.
func PostToSlack(ctx context.Context, webhookURL, text string) error {
	body, err := json.Marshal(slackPayload{Text: text})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: slackPostTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack: webhook status %d", resp.StatusCode)
	}
	return nil
}

// BuildDiscussionMessage composes a short Slack message that pairs the latest anime headline
// (as a debate-starter) with the day's QOTD prompt. If newsItems is empty, it falls back to just
// the prompt. prompt may also be "" (e.g. no prompts configured yet), in which case only the
// headline is included.
func BuildDiscussionMessage(newsItems []NewsItem, prompt string) string {
	var b strings.Builder
	b.WriteString("*Anime Discussion Time!* 🎌\n")
	if len(newsItems) > 0 {
		top := newsItems[0]
		// Escape both the URL and the label: the title/link come from an external RSS feed (Anime
		// News Network), so an untrusted headline containing `|`, `<`, `>`, or a literal
		// `<!channel>`/`<!here>` must not break out of the <url|label> link and become live Slack
		// markup (which would mass-ping the channel or forge a link).
		fmt.Fprintf(&b, "Debate topic: <%s|%s>\n", slackEscape(top.Link), slackEscape(top.Title))
	}
	if prompt != "" {
		fmt.Fprintf(&b, "Today's question: %s", slackEscape(prompt))
	}
	return strings.TrimRight(b.String(), "\n")
}

// slackEscape escapes the three characters Slack treats specially in message text, in the required
// order (& first), so untrusted content can't inject mrkdwn links or channel-mention control
// sequences. Per Slack's "Escaping text" rules.
func slackEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
