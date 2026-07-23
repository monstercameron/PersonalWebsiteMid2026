package rss

import (
	"fmt"
	"strings"
	"time"

	"github.com/monstercameron/earlcameron/internal/store"
)

// DefaultQOTDPrompt is the starter generation instruction shown when no prompt has been saved. It is
// an instruction to a model (not a literal question): the dry-run / publish paths feed it the latest
// anime headline and it returns a short discussion prompt.
const DefaultQOTDPrompt = "You are the host of a friendly anime watercooler channel in a company Slack. " +
	"Using the latest anime news headline provided below, write ONE short, punchy discussion or debate " +
	"prompt (1–2 sentences) that gets coworkers talking. Reference the news naturally, keep it inclusive " +
	"and spoiler-free, and end with a question. Output only the prompt text — no preamble, no quotes."

// PostTitle returns a dated title for a post generated/published at t.
func PostTitle(t time.Time) string {
	return "Anime discussion — " + t.Format("Jan 2, 2006")
}

// PublishedFeedXML renders the QOTD feed from the stored post history (newest first, as given —
// callers pass store.RecentQOTDPosts output). An empty history renders an empty but valid feed.
// GUIDs come from the immutable DB row id, so they are unique per publish (even two in the same
// minute) and stable across requests — readers key/de-dupe on guid.
func PublishedFeedXML(posts []store.QOTDPost, baseURL string, now time.Time) (string, error) {
	self := baseURL + "/anime/qotd.xml"
	items := make([]FeedItem, 0, len(posts))
	for _, p := range posts {
		if strings.TrimSpace(p.Body) == "" {
			continue
		}
		items = append(items, postItem(p.Title, p.Body, baseURL, fmt.Sprintf("anime-qotd-%d", p.ID), time.Unix(p.PublishedAt, 0)))
	}
	return buildFeed("Anime Release Radar — Daily Prompt", "AI-generated anime discussion prompts for engagement.", baseURL, self, items, now)
}

// PreviewFeedXML renders a one-item feed for a dry-run: exactly what publishing title+body would
// produce, but never served publicly (so its fixed preview guid is harmless).
func PreviewFeedXML(title, body, baseURL string, now time.Time) (string, error) {
	self := baseURL + "/anime/qotd.xml"
	items := []FeedItem{postItem(title, body, baseURL, "anime-qotd-preview", now)}
	return buildFeed("Anime Release Radar — Daily Prompt (preview)", "Preview of a generated anime discussion prompt.", baseURL, self, items, now)
}

// postItem builds one RSS item for a generated post. The item links to the site's anime section
// (items should link to a web page, not the feed itself).
func postItem(title, body, baseURL, guid string, at time.Time) FeedItem {
	return FeedItem{
		Title:       title,
		Link:        baseURL + "/#anime",
		Description: body,
		GUID:        guid,
		PubDate:     at.UTC(),
	}
}
