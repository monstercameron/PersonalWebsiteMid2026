package admin

import (
	"crypto/rand"
	"crypto/subtle"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// sessionCookie is the admin session cookie name (carries the JWT for the legacy HTTP path).
const sessionCookie = "ec_admin"

// Sessions authenticates the owner by username + password and mints/verifies HS256 JWTs. The same
// token gates both the HTTP admin pages (via cookie) and the gRPC AdminService (via metadata). It's
// a single-owner gate: one username (ADMIN_USERNAME) and one password (ADMIN_PASSWORD).
type Sessions struct {
	username string
	password string
	secret   []byte
	ttl      time.Duration
	secure   bool
}

// NewSessions builds a session manager. An empty password disables admin login entirely. When
// secret is empty a random 32-byte secret is generated per process — deliberate: a fixed fallback
// secret would let anyone forge a valid JWT, so we never ship a known default (set ADMIN_SECRET to
// keep sessions valid across restarts). secure marks the cookie Secure (HTTPS-only) in production.
func NewSessions(username, password, secret string, secure bool) *Sessions {
	sec := []byte(secret)
	if len(sec) == 0 {
		sec = make([]byte, 32)
		if _, err := rand.Read(sec); err != nil {
			panic("admin: cannot generate a session secret: " + err.Error())
		}
	}
	return &Sessions{username: username, password: password, secret: sec, ttl: 12 * time.Hour, secure: secure}
}

// Enabled reports whether admin login is configured (a password is set).
func (s *Sessions) Enabled() bool { return s.password != "" }

// CheckCredentials reports whether username+password match the configured owner (constant-time on
// both fields; both comparisons always run so timing doesn't reveal which one failed).
func (s *Sessions) CheckCredentials(username, password string) bool {
	if s.password == "" {
		return false
	}
	okUser := subtle.ConstantTimeCompare([]byte(username), []byte(s.username)) == 1
	okPass := subtle.ConstantTimeCompare([]byte(password), []byte(s.password)) == 1
	return okUser && okPass
}

// Mint returns a fresh signed HS256 JWT for the owner (subject = username, with iat + exp). This is
// the value carried by the cookie and, for the gRPC admin plane, by the "authorization" metadata.
func (s *Sessions) Mint() string {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   s.username,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return ""
	}
	return signed
}

// Verify reports whether token is a valid, unexpired JWT signed by us for the configured owner. It
// pins the algorithm to HS256 (no alg-confusion / "none") and requires an expiry. Shared core
// behind both the cookie check (Authed) and the gRPC metadata check (tokenFromContext).
func (s *Sessions) Verify(token string) bool {
	if token == "" {
		return false
	}
	parsed, err := jwt.Parse(token, func(*jwt.Token) (any, error) { return s.secret, nil },
		jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil || !parsed.Valid {
		return false
	}
	sub, err := parsed.Claims.GetSubject()
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(sub), []byte(s.username)) == 1
}

// Issue writes the JWT as a signed, HttpOnly session cookie scoped to /admin.
func (s *Sessions) Issue(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: s.Mint(), Path: "/admin", HttpOnly: true, Secure: s.secure,
		SameSite: http.SameSiteLaxMode, Expires: time.Now().Add(s.ttl),
	})
}

// Clear removes the session cookie (logout).
func (s *Sessions) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/admin", HttpOnly: true, MaxAge: -1})
}

// Authed reports whether the request carries a valid, unexpired admin session cookie.
func (s *Sessions) Authed(r *http.Request) bool {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	return s.Verify(c.Value)
}
