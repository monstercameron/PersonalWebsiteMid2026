package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/earlcameron/internal/adminui"
	"github.com/monstercameron/earlcameron/internal/anime"
	"github.com/monstercameron/earlcameron/internal/store"
)

// loginLimiter throttles admin login attempts to blunt password brute-forcing. It locks a client
// out for `window` after `max` consecutive failures. Behind a reverse proxy every request shares
// one RemoteAddr, so this degrades to a global lockout — acceptable (even desirable) for a
// single-owner admin gate.
type loginLimiter struct {
	mu     sync.Mutex
	fails  map[string]*failRecord
	max    int
	window time.Duration
}

// failRecord tracks one client's recent failed attempts.
type failRecord struct {
	count int       // consecutive failures in the current run
	until time.Time // lockout expiry (zero = not locked)
	seen  time.Time // last activity, for pruning
}

// newLoginLimiter builds a limiter: 5 failures → 15-minute lockout.
func newLoginLimiter() *loginLimiter {
	return &loginLimiter{fails: map[string]*failRecord{}, max: 5, window: 15 * time.Minute}
}

// allowed reports whether ip may attempt a login at time now.
func (l *loginLimiter) allowed(ip string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	r := l.fails[ip]
	return r == nil || now.After(r.until)
}

// fail records a failed attempt for ip and starts a lockout once max is reached.
func (l *loginLimiter) fail(ip string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prune(now)
	r := l.fails[ip]
	if r == nil {
		r = &failRecord{}
		l.fails[ip] = r
	}
	r.count++
	r.seen = now
	if r.count >= l.max {
		r.until = now.Add(l.window)
		r.count = 0
	}
}

// success clears ip's failure record after a correct password.
func (l *loginLimiter) success(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.fails, ip)
}

// prune drops stale records; callers must hold the lock.
func (l *loginLimiter) prune(now time.Time) {
	for k, r := range l.fails {
		if now.Sub(r.seen) > l.window && now.After(r.until) {
			delete(l.fails, k)
		}
	}
}

// clientIP returns the request's remote host (without port) for rate-limit keying.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// registerAdminRoutes wires the public anime RSS feeds and the password-gated config page.
func (s *Server) registerAdminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/anime.xml", s.animeFeed)      // public RSS: tracked releases
	mux.HandleFunc("/anime/qotd.xml", s.qotdFeed)  // public RSS: daily prompts
	mux.HandleFunc("/admin", s.adminHome)          // login
	mux.HandleFunc("/admin/login", s.adminLogin)   // POST login
	mux.HandleFunc("/admin/logout", s.adminLogout) // POST logout
	mux.HandleFunc("/admin/anime", s.adminAnime)   // gated config page
	mux.HandleFunc("/admin/anime/track", s.adminTrack)
	mux.HandleFunc("/admin/anime/untrack", s.adminUntrack)
	mux.HandleFunc("/admin/anime/check", s.adminCheck)
	mux.HandleFunc("/admin/settings", s.adminSettings)          // gated settings
	mux.HandleFunc("/admin/settings/save", s.adminSettingsSave) // POST save
}

// animeFeed serves the tracked-releases RSS feed.
func (s *Server) animeFeed(w http.ResponseWriter, r *http.Request) {
	list, err := s.anime.List(r.Context())
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	_, _ = fmt.Fprint(w, s.anime.TrackedFeedXML(list, s.cfg.BaseURL))
}

// qotdFeed serves the daily-prompts RSS feed.
func (s *Server) qotdFeed(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	_, _ = fmt.Fprint(w, anime.QuestionFeedXML(s.cfg.BaseURL, time.Now()))
}

// requireAdmin enforces the session; it redirects to login (or 503 if admin is unconfigured).
func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if !s.sessions.Enabled() {
		http.Error(w, "admin is disabled — set ADMIN_PASSWORD", http.StatusServiceUnavailable)
		return false
	}
	if !s.sessions.Authed(r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return false
	}
	return true
}

// adminHome renders the login page (or redirects in if already authed).
func (s *Server) adminHome(w http.ResponseWriter, r *http.Request) {
	if s.sessions.Enabled() && s.sessions.Authed(r) {
		http.Redirect(w, r, "/admin/anime", http.StatusSeeOther)
		return
	}
	page, err := adminui.LoginPage(s.sessions.Enabled(), "")
	writeHTML(w, page, err)
}

// adminLogin checks the password and issues a session.
func (s *Server) adminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	ip, now := clientIP(r), time.Now()
	if !s.login.allowed(ip, now) {
		w.Header().Set("Retry-After", "900")
		http.Error(w, "too many attempts — try again later", http.StatusTooManyRequests)
		return
	}
	if s.sessions.CheckCredentials(r.FormValue("username"), r.FormValue("password")) {
		s.login.success(ip)
		s.sessions.Issue(w)
		http.Redirect(w, r, "/admin/anime", http.StatusSeeOther)
		return
	}
	s.login.fail(ip, now)
	page, err := adminui.LoginPage(s.sessions.Enabled(), "wrong username or password")
	writeHTML(w, page, err)
}

// adminLogout clears the session.
func (s *Server) adminLogout(w http.ResponseWriter, r *http.Request) {
	s.sessions.Clear(w)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// adminAnime renders the config page: search AniList, view/remove tracked anime, feed links.
func (s *Server) adminAnime(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	list, _ := s.anime.List(r.Context())
	tracked := make(map[int]bool, len(list))
	for _, a := range list {
		tracked[a.AniListID] = true
	}
	var results []anime.Media
	if q != "" {
		results, _ = s.anime.Search(r.Context(), q)
	}
	page, err := adminui.AnimeConsole(q, mediaCards(results, tracked), trackedCards(list))
	writeHTML(w, page, err)
}

// adminSettings renders the settings page (OpenAI key/model), reflecting the effective config.
func (s *Server) adminSettings(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	key, model := s.openAI(r.Context())
	msg := ""
	if r.URL.Query().Get("saved") == "1" {
		msg = "Settings saved."
	}
	page, err := adminui.SettingsPage(key != "", model, msg)
	writeHTML(w, page, err)
}

// adminSettingsSave persists submitted settings. A blank field leaves the existing value untouched
// (so saving the form doesn't wipe a key you didn't retype). POST only.
func (s *Server) adminSettingsSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAdmin(w, r) {
		return
	}
	ctx := r.Context()
	if k := strings.TrimSpace(r.FormValue("openai_api_key")); k != "" {
		_ = s.store.SetSetting(ctx, store.SettingOpenAIKey, k)
	}
	if m := strings.TrimSpace(r.FormValue("openai_model")); m != "" {
		_ = s.store.SetSetting(ctx, store.SettingOpenAIModel, m)
	}
	http.Redirect(w, r, "/admin/settings?saved=1", http.StatusSeeOther)
}

// adminTrack tracks an anime by AniList id.
func (s *Server) adminTrack(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if id, _ := strconv.Atoi(r.FormValue("id")); id > 0 {
		_ = s.anime.Track(r.Context(), id)
	}
	http.Redirect(w, r, "/admin/anime", http.StatusSeeOther)
}

// adminUntrack removes a tracked anime.
func (s *Server) adminUntrack(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if id, _ := strconv.Atoi(r.FormValue("id")); id > 0 {
		_ = s.anime.Untrack(r.Context(), id)
	}
	http.Redirect(w, r, "/admin/anime", http.StatusSeeOther)
}

// adminCheck kicks off a release-check pass in the background.
func (s *Server) adminCheck(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	go func() { _, _ = s.anime.RunReleaseCheck(context.Background()) }()
	http.Redirect(w, r, "/admin/anime", http.StatusSeeOther)
}

// --- rendering bridge to the GWC admin UI (internal/adminui) ---

// writeHTML writes a rendered admin page, or a 500 if rendering failed.
func writeHTML(w http.ResponseWriter, page string, err error) {
	if err != nil {
		http.Error(w, "admin render error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, page)
}

// mediaCards maps AniList search results to console view-models, flagging already-tracked shows.
func mediaCards(results []anime.Media, tracked map[int]bool) []adminui.AnimeCard {
	out := make([]adminui.AnimeCard, 0, len(results))
	for _, m := range results {
		out = append(out, adminui.AnimeCard{
			ID: m.ID, Title: m.DisplayTitle(), Meta: animeMeta(m.Format, m.Status, m.Episodes, m.SeasonYear),
			Cover: m.CoverImage.Large, Tracked: tracked[m.ID],
		})
	}
	return out
}

// trackedCards maps stored tracked shows to console view-models.
func trackedCards(list []store.TrackedAnime) []adminui.AnimeCard {
	out := make([]adminui.AnimeCard, 0, len(list))
	for _, a := range list {
		out = append(out, adminui.AnimeCard{
			ID: a.AniListID, Title: a.Title, Meta: animeMeta(a.Format, a.Status, a.Episodes, a.SeasonYear),
			Cover: a.CoverImage, Tracked: true,
		})
	}
	return out
}

// animeMeta formats the "format · status · N eps · year" line for a card.
func animeMeta(format, status string, episodes, year int) string {
	meta := fmt.Sprintf("%s · %s · %d eps", format, status, episodes)
	if year > 0 {
		meta += fmt.Sprintf(" · %d", year)
	}
	return meta
}
