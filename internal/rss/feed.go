package rss

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/monstercameron/earlcameron/internal/store"
)

// atomNS is the Atom namespace declared on the <rss> root so a self-referencing <atom:link> is
// well-formed per the RSS Best Practices recommendation.
const atomNS = "http://www.w3.org/2005/Atom"

// rssDateFormat is the wire format for lastBuildDate/pubDate: RFC 822 with a numeric zone
// (RFC1123Z), as required by the RSS 2.0 spec.
const rssDateFormat = time.RFC1123Z

// feedDoc is the XML document root: <rss version="2.0" xmlns:atom="...">.
type feedDoc struct {
	XMLName   xml.Name   `xml:"rss"`
	Version   string     `xml:"version,attr"`
	AtomXMLNS string     `xml:"xmlns:atom,attr"`
	Channel   channelDoc `xml:"channel"`
}

// channelDoc is the required RSS 2.0 <channel> element set, plus a self-referencing atom:link.
type channelDoc struct {
	Title         string      `xml:"title"`
	Link          string      `xml:"link"`
	Description   string      `xml:"description"`
	Language      string      `xml:"language"`
	LastBuildDate string      `xml:"lastBuildDate"`
	AtomLink      atomLinkDoc `xml:"atom:link"`
	Items         []itemDoc   `xml:"item"`
}

// atomLinkDoc is the atom:link self-reference: <atom:link rel="self" type="application/rss+xml">.
type atomLinkDoc struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

// itemDoc is one RSS <item>: title, link, description, a non-permalink guid, and pubDate.
type itemDoc struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	Description string  `xml:"description"`
	GUID        guidDoc `xml:"guid"`
	PubDate     string  `xml:"pubDate"`
}

// guidDoc is <guid isPermaLink="false">value</guid>; encoding/xml escapes the chardata value.
type guidDoc struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

// FeedItem is one entry to render into an RSS feed. GUID should be a stable, unique string (the
// feed always marks it isPermaLink="false" since none of our guids are dereferenceable URLs).
type FeedItem struct {
	Title       string
	Link        string
	Description string
	GUID        string
	PubDate     time.Time
}

// buildFeed renders a fully spec-compliant RSS 2.0 document: title/link/description/language on
// the channel, a self-referencing atom:link, lastBuildDate stamped at now, and one <item> per
// entry in items. siteURL is the HTML website the channel <link> points at (per the RSS 2.0 spec —
// not the feed itself); selfURL is the feed's own public URL, used for the atom:link self href.
func buildFeed(title, description, siteURL, selfURL string, items []FeedItem, now time.Time) (string, error) {
	doc := feedDoc{
		Version:   "2.0",
		AtomXMLNS: atomNS,
		Channel: channelDoc{
			Title:         title,
			Link:          siteURL,
			Description:   description,
			Language:      "en-us",
			LastBuildDate: now.UTC().Format(rssDateFormat),
			AtomLink: atomLinkDoc{
				Href: selfURL,
				Rel:  "self",
				Type: "application/rss+xml",
			},
			Items: make([]itemDoc, 0, len(items)),
		},
	}
	for _, it := range items {
		doc.Channel.Items = append(doc.Channel.Items, itemDoc{
			Title:       it.Title,
			Link:        it.Link,
			Description: it.Description,
			GUID:        guidDoc{IsPermaLink: "false", Value: it.GUID},
			PubDate:     it.PubDate.UTC().Format(rssDateFormat),
		})
	}
	body, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	return xml.Header + string(body), nil
}

// TrackedFeedXML renders the "Anime Release Radar — Tracked Releases" feed: one item per tracked
// anime, describing its format/status/episode count/season, linking to its AniList page.
func TrackedFeedXML(list []store.TrackedAnime, baseURL string, now time.Time) (string, error) {
	self := baseURL + "/anime.xml"
	items := make([]FeedItem, 0, len(list))
	for _, a := range list {
		items = append(items, FeedItem{
			Title:       a.Title,
			Link:        a.SiteURL,
			Description: fmt.Sprintf("%s · %s · %d episodes · %d", a.Format, a.Status, a.Episodes, a.SeasonYear),
			// GUID is stable per show (no episode count) so a tracked show isn't re-shown as "new"
			// by readers each week its episode count ticks up; the episode count lives in the title/desc.
			GUID:    fmt.Sprintf("anime-tracked-%d", a.AniListID),
			PubDate: time.Unix(a.UpdatedAt, 0).UTC(),
		})
	}
	return buildFeed("Anime Release Radar — Tracked Releases", "Tracked show release updates and status snapshots.", baseURL, self, items, now)
}
