package server

import (
	"context"
	"net"
	"testing"
)

// TestFetchJobTextBlocksInternalTargets ensures the SSRF guard refuses loopback, link-local, and
// private addresses (the classic cloud-metadata and internal-network targets).
func TestFetchJobTextBlocksInternalTargets(t *testing.T) {
	for _, raw := range []string{
		"http://127.0.0.1/x",
		"http://localhost/x",
		"http://169.254.169.254/latest/meta-data/", // cloud metadata
		"http://[::1]/x",
		"http://10.0.0.1/x",
		"http://192.168.1.1/x",
	} {
		if _, err := fetchJobText(context.Background(), raw); err == nil {
			t.Errorf("expected %s to be blocked, got nil error", raw)
		}
	}
}

// TestFetchJobTextRejectsBadURLs ensures non-http(s) schemes and embedded credentials are refused
// before any network activity.
func TestFetchJobTextRejectsBadURLs(t *testing.T) {
	for _, raw := range []string{
		"file:///etc/passwd",
		"ftp://example.com/x",
		"http://user:pass@example.com/",
	} {
		if _, err := fetchJobText(context.Background(), raw); err == nil {
			t.Errorf("expected %s to be rejected, got nil error", raw)
		}
	}
}

// TestIsInternalIP spot-checks the address classifier.
func TestIsInternalIP(t *testing.T) {
	blocked := []string{"127.0.0.1", "::1", "10.1.2.3", "172.16.0.1", "192.168.0.1", "169.254.1.1", "0.0.0.0"}
	allowed := []string{"8.8.8.8", "1.1.1.1", "93.184.216.34"}
	for _, s := range blocked {
		if ip := net.ParseIP(s); ip == nil || !isInternalIP(ip) {
			t.Errorf("%s should be internal", s)
		}
	}
	for _, s := range allowed {
		if ip := net.ParseIP(s); ip == nil || isInternalIP(ip) {
			t.Errorf("%s should be public", s)
		}
	}
}
