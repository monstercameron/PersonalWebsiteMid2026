package server

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/monstercameron/earlcameron/internal/anime"
	"github.com/monstercameron/earlcameron/internal/store"
)

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
	adminPage(w, "sign in", loginBody(s.sessions.Enabled(), ""))
}

// adminLogin checks the password and issues a session.
func (s *Server) adminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	if s.sessions.CheckPassword(r.FormValue("password")) {
		s.sessions.Issue(w)
		http.Redirect(w, r, "/admin/anime", http.StatusSeeOther)
		return
	}
	adminPage(w, "sign in", loginBody(s.sessions.Enabled(), "wrong password"))
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
	var results []anime.Media
	if q != "" {
		results, _ = s.anime.Search(r.Context(), q)
	}
	list, _ := s.anime.List(r.Context())
	adminPage(w, "anime tracker", animeBody(q, results, list))
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

// --- HTML rendering (admin tool — self-contained inline styling) ---

const adminCSS = `*{box-sizing:border-box}body{margin:0;background:#17040f;color:#f3e9e6;` +
	`font-family:ui-monospace,"SF Mono",Menlo,monospace}a{color:#be7be6}` +
	`.wrap{max-width:1000px;margin:0 auto;padding:32px 24px}` +
	`h1{font-size:24px;margin:0}h2{font-size:15px;color:#a98ba0;margin:28px 0 12px;text-transform:uppercase;letter-spacing:.15em}` +
	`.dim{color:#a98ba0;font-size:13px}.top{display:flex;justify-content:space-between;align-items:center}` +
	`.feeds{margin:14px 0;color:#a98ba0;font-size:13px}` +
	`input{background:#210a19;border:1px solid #3a1b2e;color:#f3e9e6;padding:10px 12px;border-radius:8px;font:inherit}` +
	`button{background:#e95420;border:0;color:#fff;padding:10px 16px;border-radius:8px;font:inherit;font-weight:600;cursor:pointer}` +
	`button.ghost{background:transparent;border:1px solid #3a1b2e;color:#f3e9e6;font-weight:400}` +
	`form.row{display:flex;gap:10px;margin:8px 0}form.row input{flex:1}form.col{display:flex;flex-direction:column;gap:12px;max-width:320px}` +
	`.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(260px,1fr));gap:14px}` +
	`.card{display:flex;gap:12px;background:#210a19;border:1px solid #3a1b2e;border-radius:12px;padding:12px}` +
	`.card img{width:56px;height:80px;object-fit:cover;border-radius:6px;background:#3a1b2e}` +
	`.card b{font-size:14px}.card form{margin-top:8px}`

// adminPage writes a full admin HTML document.
func adminPage(w http.ResponseWriter, title, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<!doctype html><html lang="en"><head><meta charset="utf-8">`+
		`<meta name="viewport" content="width=device-width,initial-scale=1"><title>%s · admin</title>`+
		`<style>%s</style></head><body><div class="wrap">%s</div></body></html>`,
		html.EscapeString(title), adminCSS, body)
}

// loginBody renders the sign-in form (or a disabled notice).
func loginBody(enabled bool, errMsg string) string {
	if !enabled {
		return `<h1>admin</h1><p class="dim">Admin is disabled — set the ADMIN_PASSWORD environment variable.</p>`
	}
	e := ""
	if errMsg != "" {
		e = `<p style="color:#ef5350;font-size:13px">` + html.EscapeString(errMsg) + `</p>`
	}
	return `<h1>admin</h1>` + e + `<form method="post" action="/admin/login" class="col">` +
		`<input type="password" name="password" placeholder="password" autofocus>` +
		`<button type="submit">Sign in</button></form>`
}

// animeBody renders the anime config page.
func animeBody(q string, results []anime.Media, list []store.TrackedAnime) string {
	var b strings.Builder
	b.WriteString(`<div class="top"><h1>anime tracker</h1>` +
		`<form method="post" action="/admin/logout"><button class="ghost">logout</button></form></div>`)
	b.WriteString(`<div class="feeds">Feeds: <a href="/anime.xml">/anime.xml</a> · ` +
		`<a href="/anime/qotd.xml">/anime/qotd.xml</a> &nbsp; ` +
		`<form method="post" action="/admin/anime/check" style="display:inline"><button class="ghost">run release check</button></form></div>`)
	fmt.Fprintf(&b, `<form method="get" action="/admin/anime" class="row">`+
		`<input name="q" value="%s" placeholder="search AniList…" autofocus><button>Search</button></form>`, html.EscapeString(q))

	if len(results) > 0 {
		b.WriteString(`<h2>results</h2><div class="grid">`)
		for _, m := range results {
			b.WriteString(card(m.CoverImage.Large, m.DisplayTitle(), m.Format, m.Status, m.Episodes, 0, m.ID, "track", "/admin/anime/track", false))
		}
		b.WriteString(`</div>`)
	}

	fmt.Fprintf(&b, `<h2>tracked (%d)</h2>`, len(list))
	if len(list) == 0 {
		b.WriteString(`<p class="dim">Nothing tracked yet — search above and hit “track”.</p>`)
	} else {
		b.WriteString(`<div class="grid">`)
		for _, a := range list {
			b.WriteString(card(a.CoverImage, a.Title, a.Format, a.Status, a.Episodes, a.SeasonYear, a.AniListID, "remove", "/admin/anime/untrack", true))
		}
		b.WriteString(`</div>`)
	}
	return b.String()
}

// card renders one anime card with a track/remove action.
func card(cover, title, format, status string, episodes, year, id int, action, endpoint string, ghost bool) string {
	meta := fmt.Sprintf("%s · %s · %d eps", format, status, episodes)
	if year > 0 {
		meta += fmt.Sprintf(" · %d", year)
	}
	cls := ""
	if ghost {
		cls = ` class="ghost"`
	}
	return fmt.Sprintf(`<div class="card"><img src="%s" alt=""><div><b>%s</b><div class="dim">%s</div>`+
		`<form method="post" action="%s"><input type="hidden" name="id" value="%d"><button%s>%s</button></form></div></div>`,
		html.EscapeString(cover), html.EscapeString(title), html.EscapeString(meta), endpoint, id, cls, action)
}
