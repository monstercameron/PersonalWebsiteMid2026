// Package anime is a Go port of the site's anime tracker + QOTD RSS generator. It follows anime
// on AniList (public GraphQL API), stores them, refreshes episode/status on a release-check
// pass, and emits two RSS 2.0 feeds: tracked releases and a daily discussion prompt (QOTD).
package anime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/monstercameron/earlcameron/internal/store"
)

// aniListEndpoint is AniList's public GraphQL API (no auth needed for reads).
const aniListEndpoint = "https://graphql.anilist.co"

// Service tracks anime and builds feeds.
type Service struct {
	store *store.Store
	http  *http.Client
}

// New returns an anime Service backed by the store.
func New(s *store.Store) *Service {
	return &Service{store: s, http: &http.Client{Timeout: 12 * time.Second}}
}

// Media is an AniList anime entry.
type Media struct {
	ID    int `json:"id"`
	Title struct {
		Romaji  string `json:"romaji"`
		English string `json:"english"`
	} `json:"title"`
	CoverImage struct {
		Large string `json:"large"`
	} `json:"coverImage"`
	Status     string `json:"status"`
	Episodes   int    `json:"episodes"`
	Format     string `json:"format"`
	SeasonYear int    `json:"seasonYear"`
	SiteURL    string `json:"siteUrl"`
}

// DisplayTitle prefers the English title, falling back to romaji.
func (m Media) DisplayTitle() string {
	if strings.TrimSpace(m.Title.English) != "" {
		return m.Title.English
	}
	return m.Title.Romaji
}

// graphql runs a GraphQL query against AniList and decodes the result into out.
func (s *Service) graphql(ctx context.Context, query string, vars map[string]any, out any) error {
	payload, err := json.Marshal(map[string]any{"query": query, "variables": vars})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, aniListEndpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("anilist: status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

const mediaFields = `id title { romaji english } coverImage { large } status episodes format seasonYear siteUrl`

// Search returns up to a handful of anime matching the query.
func (s *Service) Search(ctx context.Context, query string) ([]Media, error) {
	q := `query ($search: String) { Page(perPage: 8) { media(search: $search, type: ANIME, sort: SEARCH_MATCH) { ` + mediaFields + ` } } }`
	var out struct {
		Data struct {
			Page struct {
				Media []Media `json:"media"`
			} `json:"Page"`
		} `json:"data"`
	}
	if err := s.graphql(ctx, q, map[string]any{"search": query}, &out); err != nil {
		return nil, err
	}
	return out.Data.Page.Media, nil
}

// fetchByID fetches one anime by AniList id.
func (s *Service) fetchByID(ctx context.Context, id int) (*Media, error) {
	q := `query ($id: Int) { Media(id: $id, type: ANIME) { ` + mediaFields + ` } }`
	var out struct {
		Data struct {
			Media Media `json:"Media"`
		} `json:"data"`
	}
	if err := s.graphql(ctx, q, map[string]any{"id": id}, &out); err != nil {
		return nil, err
	}
	return &out.Data.Media, nil
}

// Track fetches an anime by id and stores it.
func (s *Service) Track(ctx context.Context, id int) error {
	m, err := s.fetchByID(ctx, id)
	if err != nil {
		return err
	}
	return s.store.UpsertTrackedAnime(ctx, toTracked(*m))
}

// Untrack removes a tracked anime.
func (s *Service) Untrack(ctx context.Context, id int) error {
	return s.store.DeleteTrackedAnime(ctx, id)
}

// List returns all tracked anime.
func (s *Service) List(ctx context.Context) ([]store.TrackedAnime, error) {
	return s.store.ListTrackedAnime(ctx)
}

// RunReleaseCheck refreshes every tracked anime from AniList and reports how many changed
// (episode count or status). Sleeps briefly between calls to respect AniList's rate limit.
func (s *Service) RunReleaseCheck(ctx context.Context) (int, error) {
	list, err := s.store.ListTrackedAnime(ctx)
	if err != nil {
		return 0, err
	}
	changed := 0
	for _, a := range list {
		m, err := s.fetchByID(ctx, a.AniListID)
		if err != nil {
			continue
		}
		if m.Episodes != a.Episodes || m.Status != a.Status {
			changed++
		}
		_ = s.store.UpsertTrackedAnime(ctx, toTracked(*m))
		time.Sleep(700 * time.Millisecond)
	}
	return changed, nil
}

// toTracked converts an AniList Media into a stored record stamped now.
func toTracked(m Media) store.TrackedAnime {
	return store.TrackedAnime{
		AniListID: m.ID, Title: m.DisplayTitle(), CoverImage: m.CoverImage.Large,
		Status: m.Status, Episodes: m.Episodes, Format: m.Format, SeasonYear: m.SeasonYear,
		SiteURL: m.SiteURL, UpdatedAt: time.Now().Unix(),
	}
}

// --- RSS 2.0 feeds ---

type rssItem struct{ title, link, guid, desc, pubDate string }

// rssFeed renders an RSS 2.0 document.
func rssFeed(title, desc, self string, items []rssItem) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<rss version="2.0"><channel>`)
	fmt.Fprintf(&b, "<title>%s</title><description>%s</description><link>%s</link><language>en-us</language>", esc(title), esc(desc), esc(self))
	for _, it := range items {
		fmt.Fprintf(&b, `<item><title>%s</title><link>%s</link><guid isPermaLink="false">%s</guid><pubDate>%s</pubDate><description>%s</description></item>`,
			esc(it.title), esc(it.link), esc(it.guid), esc(it.pubDate), esc(it.desc))
	}
	b.WriteString("</channel></rss>")
	return b.String()
}

func esc(s string) string { return html.EscapeString(s) }

// TrackedFeedXML renders the "Release Radar — Tracked Releases" feed.
func (s *Service) TrackedFeedXML(list []store.TrackedAnime, baseURL string) string {
	items := make([]rssItem, 0, len(list))
	for _, a := range list {
		desc := fmt.Sprintf("%s · %s · %d episodes · %d", a.Format, a.Status, a.Episodes, a.SeasonYear)
		items = append(items, rssItem{
			title:   a.Title,
			link:    a.SiteURL,
			guid:    fmt.Sprintf("anime-tracked-%d-%d", a.AniListID, a.Episodes),
			desc:    desc,
			pubDate: time.Unix(a.UpdatedAt, 0).UTC().Format(time.RFC1123Z),
		})
	}
	return rssFeed("Anime Release Radar — Tracked Releases", "Tracked show release updates and status snapshots.", baseURL+"/anime.xml", items)
}

// QuestionFeedXML renders the daily anime discussion prompt (QOTD) feed for the last week.
func QuestionFeedXML(baseURL string, now time.Time) string {
	items := make([]rssItem, 0, 7)
	for i := 0; i < 7; i++ {
		day := now.AddDate(0, 0, -i)
		q := animePrompts[day.YearDay()%len(animePrompts)]
		items = append(items, rssItem{
			title:   "Anime prompt — " + day.Format("Jan 2"),
			link:    baseURL + "/anime/qotd.xml",
			guid:    "anime-qotd-" + day.Format("2006-01-02"),
			desc:    q,
			pubDate: day.UTC().Format(time.RFC1123Z),
		})
	}
	return rssFeed("Anime Release Radar — Daily Prompts", "Daily anime discussion prompts for engagement.", baseURL+"/anime/qotd.xml", items)
}

// animePrompts rotates one discussion question per day.
var animePrompts = []string{
	"Which anime got the ending it deserved — and which one didn't?",
	"Name an anime that's better on a rewatch than the first time through.",
	"What's the most underrated anime of the last five years?",
	"Sub or dub — and does it depend on the show?",
	"Which anime has the best opening sequence, no skips?",
	"What anime world would you actually want to live in?",
	"Which side character deserved to be the main character?",
	"Best fight scene, purely on animation?",
	"An anime you'd recommend to someone who 'doesn't watch anime'?",
	"Which studio is on the best run right now?",
	"What's a trope you're tired of, and one you'll never get tired of?",
	"Best redemption arc — did it earn it?",
	"Which manga adaptation actually improved on the source?",
	"An anime soundtrack you listen to outside the show?",
	"Which anime aged the best visually?",
	"Best 'monster of the week' show that's secretly deep?",
	"Which anime made you cry and you're not ashamed to admit it?",
	"What's the strongest first episode you've ever watched?",
	"Which power system is the most internally consistent?",
	"An anime you dropped that you should probably give another shot?",
}
