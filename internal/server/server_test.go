package server

import (
	"net/http"
	"testing"
)

// req builds a minimal request with the given Origin header and Host.
func req(origin, host string) *http.Request {
	r := &http.Request{Host: host, Header: http.Header{}}
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	return r
}

// TestOriginChecker verifies the CSWSH guard: same-origin and allow-listed origins pass, a
// browser page on another site is rejected, and a missing Origin (non-browser) is allowed.
func TestOriginChecker(t *testing.T) {
	check := originChecker([]string{"https://earlcameron.com"})
	cases := []struct {
		name, origin, host string
		want               bool
	}{
		{"same-origin", "http://127.0.0.1:8095", "127.0.0.1:8095", true},
		{"no-origin non-browser", "", "127.0.0.1:8095", true},
		{"allow-listed", "https://earlcameron.com", "internal-host", true},
		{"cross-site rejected", "https://evil.example", "127.0.0.1:8095", false},
	}
	for _, c := range cases {
		if got := check(req(c.origin, c.host)); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
