package admin

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/monstercameron/earlcameron/internal/store"
)

// Sessions authenticates the single site owner and mints/verifies HS256 JWTs that gate the gRPC
// AdminService. Credentials live in the store as bcrypt hashes once the owner completes first-run
// setup; before that (a fresh deploy) it falls back to the ADMIN_USERNAME/ADMIN_PASSWORD environment
// bootstrap, and if neither exists the site reports NeedsSetup so the client shows the setup screen.
//
// Password resets use a recovery phrase (generated at setup, shown once, stored only as a bcrypt
// hash) with an owner-chosen hint, plus an ADMIN_RECOVERY_TOKEN environment break-glass for total
// lockout. Every password change stamps PasswordChangedAt, and minted tokens carry it, so changing
// the password invalidates every previously issued session.
type Sessions struct {
	store       *store.Store
	envUser     string // ADMIN_USERNAME (bootstrap identity + default)
	envPass     string // ADMIN_PASSWORD (bootstrap password; used only until a stored account exists)
	recoveryEnv string // ADMIN_RECOVERY_TOKEN break-glass for reset when the phrase is lost
	setupToken  string // ADMIN_SETUP_TOKEN: when set, first-run setup requires it (closes the land-grab window)
	secret      []byte
	ttl         time.Duration
	now         func() time.Time
}

// ownerClaims is the JWT payload: the standard registered claims plus the password-change stamp the
// token was minted against, so a token becomes invalid the moment the password changes.
type ownerClaims struct {
	jwt.RegisteredClaims
	PwdAt int64 `json:"pwa,omitempty"`
}

// Minimum password length for stored credentials.
const minPasswordLen = 8

// maxPasswordLen is bcrypt's input limit: it silently truncates at 72 bytes, so we reject longer
// passwords rather than let extra characters be ignored (a false sense of added entropy).
const maxPasswordLen = 72

// maxHintLen caps the recovery hint so a pathological value can't bloat the row / reset page.
const maxHintLen = 200

// Errors surfaced to the service layer (which maps them to gRPC statuses / throttled replies).
var (
	errSetupClosed = errors.New("setup already complete")
	errSetupEnv    = errors.New("setup disabled: credentials are configured via environment")
	errSetupToken  = errors.New("invalid setup token")
	errNoAccount   = errors.New("no account to reset")
	errRecovery    = errors.New("recovery phrase did not match")
	errWeakPass    = errors.New("password must be at least 8 characters")
	errLongPass    = errors.New("password must be at most 72 bytes")
	errNoUsername  = errors.New("username is required")
)

// NewSessions builds a session manager. envUser/envPass are the environment bootstrap credentials;
// recoveryToken is the break-glass reset secret; setupToken, when non-empty, is required to complete
// first-run setup. When secret is empty a random 32-byte signing secret is generated per process —
// deliberate: a fixed fallback would let anyone forge a valid session, so we never ship a known
// default (set ADMIN_SECRET to keep tokens valid across restarts).
func NewSessions(st *store.Store, envUser, envPass, secret, recoveryToken, setupToken string) *Sessions {
	sec := []byte(secret)
	if len(sec) == 0 {
		sec = make([]byte, 32)
		if _, err := rand.Read(sec); err != nil {
			panic("admin: cannot generate a session secret: " + err.Error())
		}
	}
	return &Sessions{
		store: st, envUser: envUser, envPass: envPass, recoveryEnv: recoveryToken,
		setupToken: setupToken, secret: sec, ttl: 12 * time.Hour, now: time.Now,
	}
}

// effective resolves the active identity: the stored account when one exists, else the environment
// bootstrap. It returns the username, the password-change stamp (0 for the env account), and whether
// a stored account exists.
func (s *Sessions) effective(ctx context.Context) (username string, pwdAt int64, stored bool) {
	if c, ok, _ := s.store.GetCredential(ctx); ok {
		return c.Username, c.PasswordChangedAt, true
	}
	return s.envUser, 0, false
}

// Enabled reports whether login is possible at all (a stored account or an env password exists).
func (s *Sessions) Enabled(ctx context.Context) bool {
	if has, _ := s.store.HasCredential(ctx); has {
		return true
	}
	return s.envPass != ""
}

// NeedsSetup reports whether the deployed site has no owner yet and no env bootstrap — i.e. the
// client should show the first-run setup screen.
func (s *Sessions) NeedsSetup(ctx context.Context) bool {
	if has, _ := s.store.HasCredential(ctx); has {
		return false
	}
	return s.envPass == ""
}

// RecoveryHint returns the owner-chosen reset hint (empty when there is no stored account).
func (s *Sessions) RecoveryHint(ctx context.Context) string {
	if c, ok, _ := s.store.GetCredential(ctx); ok {
		return c.RecoveryHint
	}
	return ""
}

// CheckCredentials reports whether username+password match the active account. It compares in
// constant time: the stored path via bcrypt, the env path via subtle.ConstantTimeCompare (both
// comparisons always run so timing doesn't reveal which field failed).
func (s *Sessions) CheckCredentials(ctx context.Context, username, password string) bool {
	if c, ok, _ := s.store.GetCredential(ctx); ok {
		okUser := subtle.ConstantTimeCompare([]byte(username), []byte(c.Username)) == 1
		okPass := matchSecret(c.PasswordHash, password)
		return okUser && okPass
	}
	if s.envPass != "" {
		okUser := subtle.ConstantTimeCompare([]byte(username), []byte(s.envUser)) == 1
		okPass := subtle.ConstantTimeCompare([]byte(password), []byte(s.envPass)) == 1
		return okUser && okPass
	}
	return false
}

// Setup creates the owner account on first run and returns the one-time recovery phrase. It is
// closed once an account exists or when env credentials are managing auth (so a stranger can't seize
// an env-configured deployment), and — when ADMIN_SETUP_TOKEN is set — requires the matching token.
func (s *Sessions) Setup(ctx context.Context, username, password, hint, setupToken string) (string, error) {
	if has, err := s.store.HasCredential(ctx); err != nil {
		return "", err
	} else if has {
		return "", errSetupClosed
	}
	if s.envPass != "" {
		return "", errSetupEnv
	}
	if s.setupToken != "" && subtle.ConstantTimeCompare([]byte(setupToken), []byte(s.setupToken)) != 1 {
		return "", errSetupToken
	}
	if err := validateCredential(username, password); err != nil {
		return "", err
	}
	phrase := generatePhrase()
	passHash, err := hashSecret(password)
	if err != nil {
		return "", err
	}
	recHash, err := hashSecret(normalizePhrase(phrase))
	if err != nil {
		return "", err
	}
	now := s.now()
	// InsertCredential (not upsert) makes creation atomic: if a row already exists — because another
	// setup won a concurrent race between the HasCredential check above and here — created is false and
	// we refuse, rather than silently overwriting the established account.
	created, err := s.store.InsertCredential(ctx, store.OwnerCredential{
		Username: username, PasswordHash: passHash, RecoveryHash: recHash,
		// PasswordChangedAt is nanosecond-resolution so a change invalidates a token minted moments
		// earlier (a coarser second-resolution stamp could collide with the token's own mint time).
		RecoveryHint: clampHint(hint), PasswordChangedAt: now.UnixNano(), CreatedAt: now.Unix(),
	})
	if err != nil {
		return "", err
	}
	if !created {
		return "", errSetupClosed
	}
	return phrase, nil
}

// ResetPassword verifies the recovery phrase (or the ADMIN_RECOVERY_TOKEN break-glass) and sets a new
// password, rotating the recovery phrase (the new one is returned, shown once) and bumping
// PasswordChangedAt so every prior session is invalidated. A wrong phrase returns errRecovery; the
// caller throttles.
func (s *Sessions) ResetPassword(ctx context.Context, phrase, newPassword string) (string, error) {
	c, ok, err := s.store.GetCredential(ctx)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errNoAccount
	}
	matched := matchSecret(c.RecoveryHash, normalizePhrase(phrase))
	if !matched && s.recoveryEnv != "" &&
		subtle.ConstantTimeCompare([]byte(strings.TrimSpace(phrase)), []byte(s.recoveryEnv)) == 1 {
		matched = true
	}
	if !matched {
		return "", errRecovery
	}
	if err := validatePassword(newPassword); err != nil {
		return "", err
	}
	newPhrase := generatePhrase()
	passHash, err := hashSecret(newPassword)
	if err != nil {
		return "", err
	}
	recHash, err := hashSecret(normalizePhrase(newPhrase))
	if err != nil {
		return "", err
	}
	c.PasswordHash = passHash
	c.RecoveryHash = recHash
	c.PasswordChangedAt = s.now().UnixNano()
	if err := s.store.PutCredential(ctx, c); err != nil {
		return "", err
	}
	return newPhrase, nil
}

// Mint returns a fresh signed session token for the active account, stamped with its current
// password-change time so it can be invalidated by a later password change.
func (s *Sessions) Mint(ctx context.Context) string {
	username, pwdAt, _ := s.effective(ctx)
	now := s.now()
	claims := ownerClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
		PwdAt: pwdAt,
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return ""
	}
	return signed
}

// Verify reports whether token is a valid, unexpired session for the active owner. It pins HS256 (no
// alg-confusion / "none"), requires an expiry, checks the subject, and rejects a token whose
// password-change stamp is older than the account's current one (i.e. minted before a password
// change).
func (s *Sessions) Verify(ctx context.Context, token string) bool {
	if token == "" {
		return false
	}
	var claims ownerClaims
	parsed, err := jwt.ParseWithClaims(token, &claims, func(*jwt.Token) (any, error) { return s.secret, nil },
		jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil || !parsed.Valid {
		return false
	}
	username, pwdAt, _ := s.effective(ctx)
	if subtle.ConstantTimeCompare([]byte(claims.Subject), []byte(username)) != 1 {
		return false
	}
	return claims.PwdAt >= pwdAt
}

// validateCredential checks a new username+password pair for first-run setup.
func validateCredential(username, password string) error {
	if strings.TrimSpace(username) == "" {
		return errNoUsername
	}
	return validatePassword(password)
}

// validatePassword enforces the password length bounds (minimum for strength, maximum because bcrypt
// silently truncates past 72 bytes).
func validatePassword(password string) error {
	if len(password) < minPasswordLen {
		return errWeakPass
	}
	if len(password) > maxPasswordLen {
		return errLongPass
	}
	return nil
}

// clampHint trims and length-caps the recovery hint, cutting on a rune boundary so a multibyte
// character straddling the limit isn't stored as mangled UTF-8.
func clampHint(hint string) string {
	hint = strings.TrimSpace(hint)
	if r := []rune(hint); len(r) > maxHintLen {
		return string(r[:maxHintLen])
	}
	return hint
}
