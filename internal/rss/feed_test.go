package rss

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/earlcameron/internal/store"
)

// parsedFeed mirrors just enough of the RSS 2.0 shape to assert on required elements after
// round-tripping through encoding/xml. It deliberately omits a Channel.Link field: encoding/xml
// matches unqualified element-name tags by local name only, ignoring namespaces (a documented Go
// quirk), so a bare `xml:"link"` field would also (wrongly) bind to the namespaced atom:link
// sibling and get clobbered by its empty content. Channel-level <link> presence is instead
// asserted directly on the raw XML string in the test.
type parsedFeed struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel struct {
		Title         string `xml:"title"`
		Description   string `xml:"description"`
		Language      string `xml:"language"`
		LastBuildDate string `xml:"lastBuildDate"`
		Items         []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
			PubDate     string `xml:"pubDate"`
			GUID        struct {
				IsPermaLink string `xml:"isPermaLink,attr"`
				Value       string `xml:",chardata"`
			} `xml:"guid"`
		} `xml:"item"`
	} `xml:"channel"`
}

// TestQuestionFeedXMLSpecCompliance parses the question feed and asserts every RSS-2.0-required
// element and the atom:self link are present and well-formed.
func TestQuestionFeedXMLSpecCompliance(t *testing.T) {
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	prompts := []string{"Prompt A", "Prompt B & C <escaped>"}

	xmlStr, err := QuestionFeedXML(prompts, "https://example.com", now)
	if err != nil {
		t.Fatalf("QuestionFeedXML: %v", err)
	}

	if !strings.Contains(xmlStr, `xmlns:atom="http://www.w3.org/2005/Atom"`) {
		t.Errorf("missing atom namespace declaration:\n%s", xmlStr)
	}
	if !strings.Contains(xmlStr, `<atom:link href="https://example.com/anime/qotd.xml" rel="self" type="application/rss+xml">`) {
		t.Errorf("missing atom:self link:\n%s", xmlStr)
	}
	if !strings.Contains(xmlStr, "Prompt B &amp; C &lt;escaped&gt;") {
		t.Errorf("description not XML-escaped:\n%s", xmlStr)
	}

	channelSection := xmlStr
	if i := strings.Index(channelSection, "<item>"); i >= 0 {
		channelSection = channelSection[:i]
	}
	if !strings.Contains(channelSection, "<link>https://example.com/anime/qotd.xml</link>") {
		t.Errorf("missing channel-level link:\n%s", channelSection)
	}

	var parsed parsedFeed
	if err := xml.Unmarshal([]byte(xmlStr), &parsed); err != nil {
		t.Fatalf("output does not parse as XML: %v\n%s", err, xmlStr)
	}
	if parsed.Version != "2.0" {
		t.Errorf("version = %q, want 2.0", parsed.Version)
	}
	if parsed.Channel.Title == "" || parsed.Channel.Description == "" {
		t.Errorf("missing required channel elements: %+v", parsed.Channel)
	}
	if parsed.Channel.Language != "en-us" {
		t.Errorf("language = %q, want en-us", parsed.Channel.Language)
	}
	if _, err := time.Parse(time.RFC1123Z, parsed.Channel.LastBuildDate); err != nil {
		t.Errorf("lastBuildDate %q not RFC1123Z: %v", parsed.Channel.LastBuildDate, err)
	}
	if len(parsed.Channel.Items) != 7 {
		t.Fatalf("got %d items, want 7", len(parsed.Channel.Items))
	}
	for i, it := range parsed.Channel.Items {
		if it.Title == "" || it.Link == "" || it.Description == "" {
			t.Errorf("item %d missing required fields: %+v", i, it)
		}
		if it.GUID.IsPermaLink != "false" {
			t.Errorf("item %d guid isPermaLink = %q, want false", i, it.GUID.IsPermaLink)
		}
		if it.GUID.Value == "" {
			t.Errorf("item %d guid value empty", i)
		}
		if _, err := time.Parse(time.RFC1123Z, it.PubDate); err != nil {
			t.Errorf("item %d pubDate %q not RFC1123Z: %v", i, it.PubDate, err)
		}
	}
}

// TestTrackedFeedXML verifies the tracked-anime feed builds one item per input anime with the
// expected description shape and a valid, escaped guid/link.
func TestTrackedFeedXML(t *testing.T) {
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	list := []store.TrackedAnime{
		{AniListID: 1, Title: "Show <One>", Status: "RELEASING", Episodes: 5, Format: "TV", SeasonYear: 2026, SiteURL: "https://anilist.co/anime/1", UpdatedAt: now.Unix()},
	}

	xmlStr, err := TrackedFeedXML(list, "https://example.com", now)
	if err != nil {
		t.Fatalf("TrackedFeedXML: %v", err)
	}
	var parsed parsedFeed
	if err := xml.Unmarshal([]byte(xmlStr), &parsed); err != nil {
		t.Fatalf("output does not parse as XML: %v\n%s", err, xmlStr)
	}
	if len(parsed.Channel.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(parsed.Channel.Items))
	}
	item := parsed.Channel.Items[0]
	if item.Title != "Show <One>" {
		t.Errorf("title = %q, want %q (unescaped after parse)", item.Title, "Show <One>")
	}
	if !strings.Contains(item.Description, "TV") || !strings.Contains(item.Description, "RELEASING") {
		t.Errorf("description missing expected fields: %q", item.Description)
	}
	if item.GUID.IsPermaLink != "false" {
		t.Errorf("guid isPermaLink = %q, want false", item.GUID.IsPermaLink)
	}
}
