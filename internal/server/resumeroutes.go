package server

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

	"github.com/monstercameron/earlcameron/internal/resume"
)

// registerResumeRoutes wires the public résumé page and the owner-only tailoring tool.
func (s *Server) registerResumeRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/resume", s.resumePage)                    // public: print-to-PDF résumé
	mux.HandleFunc("/admin/resume", s.adminResume)             // gated: tailoring tool
	mux.HandleFunc("/admin/resume/tailor", s.adminResumeTailor) // gated: POST job URL → tailored résumé
}

// resumePage serves the canonical résumé as a clean, print-optimized HTML document.
func (s *Server) resumePage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, resume.RenderHTML(resume.Data(), ""))
}

// adminResume renders the résumé tailoring tool (owner-gated).
func (s *Server) adminResume(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	adminPage(w, "résumé tool", resumeToolBody(s.cfg.OpenAIKey != "", ""))
}

// adminResumeTailor fetches a job posting URL, tailors the résumé to it via the model, and renders
// the tailored résumé (still print-to-PDF friendly). Owner-gated; the model may only re-emphasize
// existing facts (see resume.Tailor).
func (s *Server) adminResumeTailor(w http.ResponseWriter, r *http.Request) {
	// POST only: a GET with ?url= would let a SameSite=Lax top-level navigation (a link Cam clicks
	// while logged in) trigger an authenticated server-side fetch — a CSRF-driven SSRF.
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAdmin(w, r) {
		return
	}
	if s.cfg.OpenAIKey == "" {
		adminPage(w, "résumé tool", resumeToolBody(false, "Tailoring is disabled — set OPENAI_API_KEY."))
		return
	}
	jobURL := strings.TrimSpace(r.FormValue("url"))
	if !strings.HasPrefix(jobURL, "http://") && !strings.HasPrefix(jobURL, "https://") {
		adminPage(w, "résumé tool", resumeToolBody(true, "Enter a full job-posting URL (http/https)."))
		return
	}
	jobText, err := fetchJobText(r.Context(), jobURL)
	if err != nil {
		adminPage(w, "résumé tool", resumeToolBody(true, "Couldn't fetch that URL: "+err.Error()))
		return
	}
	tailored, err := resume.Tailor(r.Context(), s.cfg.OpenAIKey, s.cfg.OpenAIModel, jobText, resume.Data())
	if err != nil {
		adminPage(w, "résumé tool", resumeToolBody(true, "Tailoring failed: "+err.Error()))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, resume.RenderHTML(tailored, "Tailored to: "+jobURL+" — review every line before you send it."))
}

// resumeToolBody renders the tailoring tool form. enabled=false shows a disabled notice; msg is an
// optional status/error line.
func resumeToolBody(enabled bool, msg string) string {
	var b strings.Builder
	b.WriteString(`<div class="top"><h1>résumé tool</h1>` +
		`<form method="post" action="/admin/logout"><button class="ghost">logout</button></form></div>`)
	b.WriteString(`<div class="feeds"><a href="/admin/anime">← anime</a> &nbsp; ` +
		`Canonical résumé: <a href="/resume" target="_blank">/resume</a> (open it, then Save as PDF).</div>`)
	if msg != "" {
		b.WriteString(`<p class="dim">` + html.EscapeString(msg) + `</p>`)
	}
	if !enabled {
		b.WriteString(`<p class="dim">Set the OPENAI_API_KEY environment variable to enable tailoring.</p>`)
		return b.String()
	}
	b.WriteString(`<form method="post" action="/admin/resume/tailor" class="row">` +
		`<input name="url" placeholder="paste a job-posting URL…" autofocus><button>Tailor résumé</button></form>`)
	b.WriteString(`<p class="dim">The tool fetches the posting and re-emphasizes real facts to fit it — ` +
		`it never invents experience. Review the result before sending, then Save as PDF.</p>`)
	return b.String()
}

var (
	reScriptStyle = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	reTag         = regexp.MustCompile(`(?s)<[^>]+>`)
	reWhitespace  = regexp.MustCompile(`\s+`)
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

// fetchJobText GETs a job-posting URL and returns its visible text (scripts/styles/tags stripped,
// whitespace collapsed, capped) for the tailoring prompt. It rejects non-http(s) schemes and URLs
// carrying credentials; SSRF filtering happens in the client's dialer.
func fetchJobText(ctx context.Context, rawURL string) (string, error) {
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
