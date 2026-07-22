package rss

import (
	"strings"
	"time"
)

// DefaultQOTDPrompt is the starter generation instruction shown when no prompt has been saved. It is
// an instruction to a model (not a literal question): the dry-run / publish paths feed it the latest
// anime headline and it returns a short discussion prompt.
const DefaultQOTDPrompt = "You are the host of a friendly anime watercooler channel in a company Slack. " +
	"Using the latest anime news headline provided below, write ONE short, punchy discussion or debate " +
	"prompt (1–2 sentences) that gets coworkers talking. Reference the news naturally, keep it inclusive " +
	"and spoiler-free, and end with a question. Output only the prompt text — no preamble, no quotes."

// PublishedPost is a generated anime discussion post that has been published (served by the QOTD
// feed). Title defaults to a dated label; Body is the generated prompt text.
type PublishedPost struct {
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"publishedAt"`
}

// PostTitle returns a dated title for a post generated/published at t.
func PostTitle(t time.Time) string {
	return "Anime discussion — " + t.Format("Jan 2, 2006")
}

// PublishedFeedXML renders the QOTD feed from the single last-published post (or an empty but valid
// feed when nothing has been published yet).
func PublishedFeedXML(post *PublishedPost, baseURL string, now time.Time) (string, error) {
	self := baseURL + "/anime/qotd.xml"
	var items []FeedItem
	if post != nil && strings.TrimSpace(post.Body) != "" {
		items = append(items, postItem(post.Title, post.Body, self, post.PublishedAt))
	}
	return buildFeed("Anime Release Radar — Daily Prompt", "AI-generated anime discussion prompts for engagement.", self, items, now)
}

// PreviewFeedXML renders a one-item feed for a dry-run: exactly what publishing title+body would
// produce, but never served publicly.
func PreviewFeedXML(title, body, baseURL string, now time.Time) (string, error) {
	self := baseURL + "/anime/qotd.xml"
	items := []FeedItem{postItem(title, body, self, now)}
	return buildFeed("Anime Release Radar — Daily Prompt (preview)", "Preview of a generated anime discussion prompt.", self, items, now)
}

// postItem builds the single RSS item for a generated post. The guid is minute-stamped so each
// publish is a distinct entry.
func postItem(title, body, self string, at time.Time) FeedItem {
	return FeedItem{
		Title:       title,
		Link:        self,
		Description: body,
		GUID:        "anime-qotd-" + at.UTC().Format("2006-01-02-1504"),
		PubDate:     at.UTC(),
	}
}
