package rss

import (
	"encoding/xml"
	"testing"
)

// annFixture is a trimmed real-shaped Anime News Network RSS fixture, used so parsing tests never
// touch the network.
const annFixture = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Anime News Network</title>
    <link>https://www.animenewsnetwork.com/</link>
    <description>Anime News Network's news feed</description>
    <item>
      <title>Show One Gets Season 2</title>
      <link>https://www.animenewsnetwork.com/news/2026-07-20/show-one-season-2</link>
      <pubDate>Mon, 20 Jul 2026 12:00:00 GMT</pubDate>
    </item>
    <item>
      <title>Show Two &amp; Three Crossover Announced</title>
      <link>https://www.animenewsnetwork.com/news/2026-07-19/crossover</link>
      <pubDate>Sun, 19 Jul 2026 09:00:00 GMT</pubDate>
    </item>
    <item>
      <title>Item 3</title>
      <link>https://example.com/3</link>
      <pubDate>Sat, 18 Jul 2026 09:00:00 GMT</pubDate>
    </item>
    <item>
      <title>Item 4</title>
      <link>https://example.com/4</link>
      <pubDate>Fri, 17 Jul 2026 09:00:00 GMT</pubDate>
    </item>
    <item>
      <title>Item 5</title>
      <link>https://example.com/5</link>
      <pubDate>Thu, 16 Jul 2026 09:00:00 GMT</pubDate>
    </item>
    <item>
      <title>Item 6 (should be truncated)</title>
      <link>https://example.com/6</link>
      <pubDate>Wed, 15 Jul 2026 09:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`

// TestParseAnimeNewsFixture verifies the annFeed struct correctly unmarshals a realistic RSS
// fixture and that FetchAnimeNews's own top-N truncation logic (mirrored here since the network
// call itself isn't exercised in tests) keeps only the first 5 items.
func TestParseAnimeNewsFixture(t *testing.T) {
	var feed annFeed
	if err := xml.Unmarshal([]byte(annFixture), &feed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(feed.Channel.Items) != 6 {
		t.Fatalf("fixture parsed %d items, want 6", len(feed.Channel.Items))
	}

	items := feed.Channel.Items
	if len(items) > newsTopN {
		items = items[:newsTopN]
	}
	if len(items) != newsTopN {
		t.Fatalf("truncated to %d items, want %d", len(items), newsTopN)
	}
	if items[0].Title != "Show One Gets Season 2" {
		t.Errorf("first item title = %q", items[0].Title)
	}
	if items[1].Title != "Show Two & Three Crossover Announced" {
		t.Errorf("second item title (unescaped) = %q", items[1].Title)
	}
	if items[0].Link != "https://www.animenewsnetwork.com/news/2026-07-20/show-one-season-2" {
		t.Errorf("first item link = %q", items[0].Link)
	}
	if items[0].PubDate != "Mon, 20 Jul 2026 12:00:00 GMT" {
		t.Errorf("first item pubDate = %q", items[0].PubDate)
	}
}
