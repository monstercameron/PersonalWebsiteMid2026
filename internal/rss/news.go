package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

// animeNewsFeedURL is Anime News Network's all-news RSS feed, used as the source for the latest
// anime headlines fed into Slack discussion posts.
const animeNewsFeedURL = "https://www.animenewsnetwork.com/all/rss.xml"

// newsFetchTimeout bounds the whole request (dial + read).
const newsFetchTimeout = 15 * time.Second

// newsMaxBody caps how much of the response body is read, so a misbehaving or huge feed can't
// exhaust memory.
const newsMaxBody = 1 << 20 // 1 MiB

// newsTopN is how many of the most recent items FetchAnimeNews returns.
const newsTopN = 5

// NewsItem is one anime-news headline pulled from an external RSS feed.
type NewsItem struct {
	Title   string
	Link    string
	PubDate string
}

// annFeed is the subset of Anime News Network's RSS 2.0 document this package cares about.
type annFeed struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Items []struct {
			Title   string `xml:"title"`
			Link    string `xml:"link"`
			PubDate string `xml:"pubDate"`
		} `xml:"item"`
	} `xml:"channel"`
}

// FetchAnimeNews fetches Anime News Network's all-news RSS feed and returns the top newsTopN
// items in feed order (newest first, per the feed's own ordering).
func FetchAnimeNews(ctx context.Context) ([]NewsItem, error) {
	client := &http.Client{
		Timeout: newsFetchTimeout,
		// Only follow redirects that stay on the known feed host — defense in depth so a compromised
		// upstream can't bounce this server-side fetch at an internal/other address (whose content
		// would then flow into the Slack post).
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("rss: too many redirects")
			}
			if h := req.URL.Hostname(); h != "www.animenewsnetwork.com" && h != "animenewsnetwork.com" {
				return fmt.Errorf("rss: refusing redirect to %q", h)
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, animeNewsFeedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; earlcameron-bot/1.0; +https://earlcameron.com)")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rss: anime news feed status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, newsMaxBody))
	if err != nil {
		return nil, err
	}
	var feed annFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, err
	}
	items := feed.Channel.Items
	if len(items) > newsTopN {
		items = items[:newsTopN]
	}
	out := make([]NewsItem, 0, len(items))
	for _, it := range items {
		out = append(out, NewsItem{Title: it.Title, Link: it.Link, PubDate: it.PubDate})
	}
	return out, nil
}
