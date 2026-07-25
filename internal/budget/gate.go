package budget

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

// cookieName is the gate session cookie. It is scoped to the /budget/ path and carries the session
// mode ("full" or "guest") signed so it cannot be forged into a higher mode.
const cookieName = "cf_gate"

// enterPath is the (prefix-stripped) path the gate form posts to. Under the "/budget/" mount this is
// reached as POST /budget/__enter.
const enterPath = "/__enter"

// Session modes. "full" is granted by the password (the owner's synced budget); "guest" is the
// password-bypass (a local-only session). Both open the same app — the sync difference is enforced
// by the sync access token, which a guest is never given.
const (
	modeFull  = "full"
	modeGuest = "guest"
)

// Gate is a password door in front of the CashFlux app. Entering the password grants a full session;
// a guest bypass grants a local-only session. It signs the session cookie with an HMAC secret so a
// guest cannot forge a full session.
type Gate struct {
	password string
	secret   []byte
	ttl      time.Duration
	// activationValid, when set, reports whether an activation code is one the
	// embedded CashFlux server minted and has not yet consumed. A visitor arriving
	// with a live code came from an authenticated admin console that minted it for
	// them, which is at least as strong a claim as knowing the gate password — so
	// the gate lets them in on that basis instead of asking for a second secret.
	// Nil when CashFlux embedding isn't configured, in which case nothing changes.
	activationValid func(code string) (bool, error)
}

// ActivationParam is the query parameter carrying a handoff activation code. The
// gate only PEEKS at it; the client redeems it, once, over the normal RPC.
const ActivationParam = "activate"

// SetActivationChecker wires the non-consuming code check. Kept separate from
// NewGate so the gate has no compile-time dependency on CashFlux: a deployment
// without the embedded sync engine simply never calls this.
func (g *Gate) SetActivationChecker(fn func(code string) (bool, error)) {
	if g != nil {
		g.activationValid = fn
	}
}

// arrivingWithLiveActivationCode reports whether this request carries a code the
// embedded server minted and has not consumed. Errors are treated as "no": a
// lookup failure must fall back to asking for the password, never to opening the
// door.
func (g *Gate) arrivingWithLiveActivationCode(r *http.Request) bool {
	if g == nil || g.activationValid == nil {
		return false
	}
	code := strings.TrimSpace(r.URL.Query().Get(ActivationParam))
	if code == "" {
		return false
	}
	ok, err := g.activationValid(code)
	return err == nil && ok
}

// NewGate builds a gate for the given password and cookie-signing secret. An empty password disables
// the gate (see Enabled). An empty secret generates a random 32-byte secret per process — deliberate,
// matching the admin session manager: a fixed fallback would let anyone forge a full session, so we
// never ship a known default (set ADMIN_SECRET to keep sessions valid across restarts).
func NewGate(password, secret string) *Gate {
	var sec []byte
	if secret != "" {
		// Domain-separate from any other use of the same secret (the admin JWT signing key is the same
		// ADMIN_SECRET): derive a gate-specific sub-key so the two signing schemes never share raw key
		// material and a token from one can't be replayed against the other.
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte("cashflux-budget-gate-v1"))
		sec = mac.Sum(nil)
	} else {
		sec = make([]byte, 32)
		if _, err := rand.Read(sec); err != nil {
			panic("budget: cannot generate a gate secret: " + err.Error())
		}
	}
	return &Gate{password: password, secret: sec, ttl: 30 * 24 * time.Hour}
}

// Enabled reports whether the gate is active (a password is configured). When disabled the app is
// served open, with no gate page.
func (g *Gate) Enabled() bool { return g.password != "" }

// checkPassword reports whether pw matches the configured password, in constant time.
func (g *Gate) checkPassword(pw string) bool {
	if g.password == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(pw), []byte(g.password)) == 1
}

// sign returns the HMAC-SHA256 of msg as hex, keyed by the gate secret.
func (g *Gate) sign(msg string) string {
	mac := hmac.New(sha256.New, g.secret)
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

// issue mints a signed cookie value for mode: "<mode>.<expUnix>.<hexHMAC>".
func (g *Gate) issue(mode string) string {
	exp := strconv.FormatInt(time.Now().Add(g.ttl).Unix(), 10)
	msg := mode + "." + exp
	return msg + "." + g.sign(msg)
}

// session returns the valid, unexpired mode carried by value, or ("", false) if it is malformed,
// tampered with, or expired.
func (g *Gate) session(value string) (string, bool) {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return "", false
	}
	mode, exp, sig := parts[0], parts[1], parts[2]
	if mode != modeFull && mode != modeGuest {
		return "", false
	}
	if subtle.ConstantTimeCompare([]byte(sig), []byte(g.sign(mode+"."+exp))) != 1 {
		return "", false
	}
	sec, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || time.Now().Unix() >= sec {
		return "", false
	}
	return mode, true
}

// modeFromRequest returns the valid session mode carried by the request's gate cookie, or ("", false).
func (g *Gate) modeFromRequest(r *http.Request) (string, bool) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return "", false
	}
	return g.session(c.Value)
}

// setCookie writes the signed gate cookie for mode, scoped to the /budget/ path. It is HttpOnly and
// SameSite=Lax, and Secure whenever the request arrived over HTTPS — including behind a TLS-
// terminating reverse proxy (Nginx), where the Go server sees plaintext and r.TLS is nil, so the
// proxy's X-Forwarded-Proto is trusted for the Secure flag.
func (g *Gate) setCookie(w http.ResponseWriter, r *http.Request, mode string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    g.issue(mode),
		Path:     "/budget/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsHTTPS(r),
		Expires:  time.Now().Add(g.ttl),
	})
}

// requestIsHTTPS reports whether the original client request was HTTPS: either a direct TLS
// connection, or a plaintext hop from a reverse proxy that terminated TLS and set X-Forwarded-Proto.
func requestIsHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// sameOrigin reports whether the request is same-origin (CSRF defense for the state-changing POST):
// it allows a missing Origin header (non-browser clients / same-origin navigations that omit it) and
// requires any present Origin's host to match the request Host. This mirrors the WebSocket tunnel's
// origin check and stops a hostile third-party page from silently forcing a session (e.g. downgrading
// a full session to guest) by auto-submitting the form.
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

// Wrap puts the gate in front of the app handler. When the gate is disabled it returns app unchanged.
// Otherwise: a valid session cookie passes through to app; POST /__enter processes the password or
// guest bypass and redirects back to the app; any other request without a valid session renders the
// gate page.
func (g *Gate) Wrap(app http.Handler) http.Handler {
	if !g.Enabled() {
		return app
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Normalize the path: this handler is mounted behind http.StripPrefix("/budget/"), which strips
		// the leading slash too, so a POST to /budget/__enter arrives here as "__enter". Cleaning with a
		// leading slash makes the match robust to that (and to any "//"/".." noise).
		if r.Method == http.MethodPost && path.Clean("/"+r.URL.Path) == enterPath {
			g.handleEnter(w, r)
			return
		}
		if _, ok := g.modeFromRequest(r); ok {
			app.ServeHTTP(w, r)
			return
		}
		// Arriving from the admin console with a live activation code: the operator
		// has already authenticated to mint it, so making them type a second,
		// unrelated password here proves nothing. Grant the session and continue —
		// the code itself is still redeemed by the client, once, over the RPC.
		if g.arrivingWithLiveActivationCode(r) {
			g.setCookie(w, r, modeFull)
			app.ServeHTTP(w, r)
			return
		}
		// No valid session. Answer the lock page ONLY to a document navigation; every
		// other unauthenticated request gets a bodyless 401.
		//
		// The old assumption here — "the SPA never requests its assets until index.html
		// has run, which requires a session" — is false, and the way it fails is nasty.
		// A service worker fetches "./bin/main.wasm.gz" on install, in the background,
		// carrying whatever cookies it has: none, on a first visit or after the cookie
		// expires. It would receive this HTML with a 200, cache it under the WASM's URL,
		// and the next boot would try to instantiate a lock page as WebAssembly and hang
		// with no error anyone could act on. The SW's cache key is precached and
		// long-lived, so the poisoning outlives any number of reloads.
		//
		// A 401 is both correct and self-healing: the SW's install cache-put is skipped
		// for a non-ok response, so nothing is poisoned, and the asset is fetched
		// properly once a session exists.
		if !isDocumentNavigation(r) {
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(renderGatePage("")))
	})
}

// isDocumentNavigation reports whether this request is a browser navigating to a
// page, as opposed to a subresource or script-initiated fetch.
//
// Sec-Fetch-Mode is the reliable signal and every browser that ships service
// workers sends it. The Accept header is the fallback for anything that does not:
// a navigation asks for text/html, while a service worker fetching a .wasm or a
// script fetching JSON does not. Both are advisory headers a client could lie
// about — which is fine, because getting this wrong only decides whether an
// unauthenticated caller sees a login page or an empty 401. It gates no data.
func isDocumentNavigation(r *http.Request) bool {
	if mode := r.Header.Get("Sec-Fetch-Mode"); mode != "" {
		return mode == "navigate"
	}
	return strings.Contains(r.Header.Get("Accept"), "text/html")
}

// handleEnter processes the gate form: a "guest" bypass grants a local-only session; otherwise the
// password field is checked and, on success, grants a full session. Both redirect back to the app; a
// wrong password re-renders the gate with an error (401).
func (g *Gate) handleEnter(w http.ResponseWriter, r *http.Request) {
	// CSRF: reject cross-origin POSTs so a hostile page can't silently force a session (e.g. downgrade
	// a full session to guest) via an auto-submitting form.
	if !sameOrigin(r) {
		http.Error(w, "bad origin", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if r.FormValue("mode") == modeGuest {
		g.setCookie(w, r, modeGuest)
		http.Redirect(w, r, "/budget/", http.StatusSeeOther)
		return
	}
	if g.checkPassword(r.FormValue("password")) {
		g.setCookie(w, r, modeFull)
		http.Redirect(w, r, "/budget/", http.StatusSeeOther)
		return
	}
	// Throttle brute-forcing: a fixed delay on failure caps attempts to ~1/sec per connection without a
	// lockout a griefer could use to lock the owner out (matches the admin Login throttle).
	select {
	case <-time.After(time.Second):
	case <-r.Context().Done():
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(renderGatePage("That password didn't match. Try again, or continue as a guest.")))
}
