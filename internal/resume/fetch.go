package resume

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// jobFetchClient fetches job postings with SSRF protection: its dialer resolves each host and
// refuses to connect to loopback/private/link-local/unspecified addresses, and only ports 80/443.
// Because the dialer re-runs for every redirect hop too, redirect-based SSRF is covered by the same
// check; CheckRedirect only bounds the hop count. Dialing the resolved IP directly (rather than the
// hostname) also closes the DNS-rebinding window between validation and connection.
var jobFetchClient = &http.Client{
	Timeout:   15 * time.Second,
	Transport: &http.Transport{DialContext: safeDialContext},
	CheckRedirect: func(_ *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("stopped after 5 redirects")
		}
		return nil
	},
}

// safeDialContext dials addr only if its port is 80/443 and none of its resolved IPs are internal.
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if port != "80" && port != "443" {
		return nil, fmt.Errorf("port %s not allowed", port)
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	for _, ip := range ips {
		if isInternalIP(ip) {
			return nil, fmt.Errorf("refusing to connect to internal address %s", ip)
		}
	}
	// Dial the validated IP directly so no second (rebindable) lookup happens.
	return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
}

// isInternalIP reports whether ip is in a range we must never fetch from (loopback, RFC1918/ULA
// private, link-local, unspecified, or multicast).
func isInternalIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}

// FetchJobText GETs a job-posting URL and returns its visible text (scripts/styles/tags stripped,
// whitespace collapsed, capped) for the tailoring prompt. It rejects non-http(s) schemes and URLs
// carrying credentials; SSRF filtering happens in the client's dialer.
func FetchJobText(ctx context.Context, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("URL must be http or https")
	}
	if u.User != nil {
		return "", errors.New("URL must not contain credentials")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; earlcameron-resume-tool)")
	resp, err := jobFetchClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		return "", err
	}
	text := htmlToText(string(body))
	if len(text) < 40 {
		return "", fmt.Errorf("no readable text found at that URL")
	}
	return text, nil
}

var (
	reScriptStyle = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	reTag         = regexp.MustCompile(`(?s)<[^>]+>`)
	reWhitespace  = regexp.MustCompile(`\s+`)
)

// htmlToText reduces an HTML document to collapsed visible text, capped at 6000 chars.
func htmlToText(doc string) string {
	doc = reScriptStyle.ReplaceAllString(doc, " ")
	doc = reTag.ReplaceAllString(doc, " ")
	doc = html.UnescapeString(doc)
	doc = strings.TrimSpace(reWhitespace.ReplaceAllString(doc, " "))
	if len(doc) > 6000 {
		doc = doc[:6000]
	}
	return doc
}
