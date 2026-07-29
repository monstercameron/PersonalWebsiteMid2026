package server

import (
	"net/http/httptest"
	"testing"
)

// TestFeedBaseURL pins the two things this function has to get right at once: a feed served through
// the real domain must advertise that domain, and a client-supplied Host must never be able to move
// the advertised origin somewhere else.
func TestFeedBaseURL(t *testing.T) {
	const configured = "http://167.99.232.99:8080"

	cases := []struct {
		name       string
		host       string
		fwdHost    string
		fwdProto   string
		configured string
		want       string
	}{
		{
			name: "domain behind nginx wins over a configured IP",
			host: "www.earlcameron.com", fwdHost: "www.earlcameron.com", fwdProto: "https",
			configured: configured, want: "https://www.earlcameron.com",
		},
		{
			name: "bare IP request falls back to the configured value",
			host: "167.99.232.99:8080", configured: configured, want: configured,
		},
		{
			name: "localhost is not a public name, so it falls back",
			host: "localhost:8096", configured: configured, want: configured,
		},
		{
			name: "IPv6 literal falls back",
			host: "[2001:db8::1]:8080", configured: configured, want: configured,
		},
		{
			name: "no proxy headers and no TLS means http on the request host",
			host: "feeds.example.org", configured: configured, want: "http://feeds.example.org",
		},
		{
			name: "the first entry of a multi-proxy chain is the client-facing one",
			host: "internal.local", fwdHost: "www.earlcameron.com, inner.svc", fwdProto: "https, http",
			configured: configured, want: "https://www.earlcameron.com",
		},
		{
			name: "a configured value keeps no trailing slash",
			host: "10.0.0.5", configured: "https://example.com/", want: "https://example.com",
		},
		{
			// The guard that matters: a forged Host carrying a path or scheme must not be able to
			// smuggle a different origin into a feed that readers will follow for months.
			name: "a Host with a path separator is rejected",
			host: "evil.example.com/../x", configured: configured, want: configured,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/anime.xml", nil)
			r.Host = tc.host
			if tc.fwdHost != "" {
				r.Header.Set("X-Forwarded-Host", tc.fwdHost)
			}
			if tc.fwdProto != "" {
				r.Header.Set("X-Forwarded-Proto", tc.fwdProto)
			}
			if got := feedBaseURL(r, tc.configured); got != tc.want {
				t.Errorf("feedBaseURL() = %q, want %q", got, tc.want)
			}
		})
	}
}
