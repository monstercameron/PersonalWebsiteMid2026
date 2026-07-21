package admin

import (
	"crypto/rand"
	"crypto/subtle"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Sessions authenticates the owner by username + password and mints/verifies HS256 JWTs. The token
// gates the gRPC AdminService: Login returns it and the client sends it back in "authorization"
// metadata on every other call (see interceptor.go). It's a single-owner gate — one username
// (ADMIN_USERNAME) and one password (ADMIN_PASSWORD).
type Sessions struct {
	username string
	password string
	secret   []byte
	ttl      time.Duration
}

// NewSessions builds a session manager. An empty password disables admin login entirely. When
// secret is empty a random 32-byte secret is generated per process — deliberate: a fixed fallback
// secret would let anyone forge a valid JWT, so we never ship a known default (set ADMIN_SECRET to
// keep tokens valid across restarts).
func NewSessions(username, password, secret string) *Sessions {
	sec := []byte(secret)
	if len(sec) == 0 {
		sec = make([]byte, 32)
		if _, err := rand.Read(sec); err != nil {
			panic("admin: cannot generate a session secret: " + err.Error())
		}
	}
	return &Sessions{username: username, password: password, secret: sec, ttl: 12 * time.Hour}
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

// Mint returns a fresh signed HS256 JWT for the owner (subject = username, with iat + exp), sent by
// the gRPC admin client in "authorization" metadata.
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
// pins the algorithm to HS256 (no alg-confusion / "none") and requires an expiry.
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
