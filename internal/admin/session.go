package admin

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// sessionCookie is the admin session cookie name.
const sessionCookie = "ec_admin"

// Sessions issues and verifies password-based admin sessions via a signed cookie. It's a
// pragmatic single-password gate (env ADMIN_PASSWORD) — the WebAuthn console is a later phase.
type Sessions struct {
	password string
	secret   []byte
	ttl      time.Duration
}

// NewSessions builds a session manager. An empty password disables admin login entirely.
func NewSessions(password, secret string) *Sessions {
	if secret == "" {
		secret = "dev-insecure-secret-change-me"
	}
	return &Sessions{password: password, secret: []byte(secret), ttl: 12 * time.Hour}
}

// Enabled reports whether admin login is configured (a password is set).
func (s *Sessions) Enabled() bool { return s.password != "" }

// CheckPassword reports whether pw matches the configured password (constant-time).
func (s *Sessions) CheckPassword(pw string) bool {
	if s.password == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(pw), []byte(s.password)) == 1
}

// Issue writes a signed, HttpOnly session cookie scoped to /admin.
func (s *Sessions) Issue(w http.ResponseWriter) {
	exp := time.Now().Add(s.ttl).Unix()
	payload := "admin." + strconv.FormatInt(exp, 10)
	val := base64.RawURLEncoding.EncodeToString([]byte(payload + "." + s.sign(payload)))
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: val, Path: "/admin", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Expires: time.Unix(exp, 0),
	})
}

// Clear removes the session cookie (logout).
func (s *Sessions) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/admin", HttpOnly: true, MaxAge: -1})
}

// Authed reports whether the request carries a valid, unexpired admin session.
func (s *Sessions) Authed(r *http.Request) bool {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		return false
	}
	parts := strings.Split(string(raw), ".")
	if len(parts) != 3 {
		return false
	}
	payload := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(parts[2]), []byte(s.sign(payload))) {
		return false
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	return parts[0] == "admin"
}

// sign returns the base64 HMAC-SHA256 of payload.
func (s *Sessions) sign(payload string) string {
	m := hmac.New(sha256.New, s.secret)
	m.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
