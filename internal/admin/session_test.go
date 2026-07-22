package admin

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/monstercameron/earlcameron/internal/store"
)

// newTestStore opens a fresh temp-file store (in-memory :memory: is unreliable under the sql pool).
func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// TestJWTMintVerifyEnv checks the JWT round-trip on the env-bootstrap path and that tampered,
// wrong-secret, and wrong-subject tokens are rejected.
func TestJWTMintVerifyEnv(t *testing.T) {
	ctx := context.Background()
	s := NewSessions(newTestStore(t), "cam", "pw12345678", "sekret", "", "")

	tok := s.Mint(ctx)
	if tok == "" || !s.Verify(ctx, tok) {
		t.Fatal("a freshly minted token should verify")
	}
	if s.Verify(ctx, tok+"x") {
		t.Fatal("a tampered token must fail")
	}
	if s.Verify(ctx, "not-a-jwt") {
		t.Fatal("garbage must fail")
	}
	if NewSessions(newTestStore(t), "cam", "pw12345678", "different-secret", "", "").Verify(ctx, tok) {
		t.Fatal("token from a different secret must fail")
	}
	other := NewSessions(newTestStore(t), "eve", "pw12345678", "sekret", "", "").Mint(ctx)
	if s.Verify(ctx, other) {
		t.Fatal("token for a different subject must fail")
	}
}

// TestCheckCredentialsEnv verifies the env-bootstrap path requires both fields and that a disabled
// gate (no stored account, no env password) rejects everything.
func TestCheckCredentialsEnv(t *testing.T) {
	ctx := context.Background()
	s := NewSessions(newTestStore(t), "cam", "pw12345678", "sekret", "", "")
	if !s.CheckCredentials(ctx, "cam", "pw12345678") {
		t.Fatal("correct credentials should pass")
	}
	if s.CheckCredentials(ctx, "cam", "wrong") || s.CheckCredentials(ctx, "eve", "pw12345678") || s.CheckCredentials(ctx, "", "") {
		t.Fatal("bad credentials must fail")
	}
	if NewSessions(newTestStore(t), "cam", "", "sekret", "", "").CheckCredentials(ctx, "cam", "") {
		t.Fatal("disabled gate must reject")
	}
}

// TestNeedsSetup covers the first-run detection: setup needed only when there's no stored account and
// no env password.
func TestNeedsSetup(t *testing.T) {
	ctx := context.Background()
	// Fresh deploy, no env password → needs setup.
	fresh := NewSessions(newTestStore(t), "cam", "", "sekret", "", "")
	if !fresh.NeedsSetup(ctx) {
		t.Fatal("a fresh unconfigured deploy should need setup")
	}
	// Env password configured → no setup screen.
	env := NewSessions(newTestStore(t), "cam", "pw12345678", "sekret", "", "")
	if env.NeedsSetup(ctx) {
		t.Fatal("an env-configured deploy should not need setup")
	}
}

// TestSetupAndLogin covers first-run setup, the returned recovery phrase, DB-backed login, and the
// guards that close setup afterward.
func TestSetupAndLogin(t *testing.T) {
	ctx := context.Background()
	s := NewSessions(newTestStore(t), "cam", "", "sekret", "", "")

	phrase, err := s.Setup(ctx, "owner", "hunter2hunter2", "my first pet", "")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if len(phrase) == 0 {
		t.Fatal("setup should return a recovery phrase")
	}
	if s.NeedsSetup(ctx) {
		t.Fatal("after setup the site no longer needs setup")
	}
	if s.RecoveryHint(ctx) != "my first pet" {
		t.Fatalf("hint not stored, got %q", s.RecoveryHint(ctx))
	}
	// DB-backed login works; the old env identity ("cam") does not.
	if !s.CheckCredentials(ctx, "owner", "hunter2hunter2") {
		t.Fatal("stored credentials should authenticate")
	}
	if s.CheckCredentials(ctx, "owner", "wrong") {
		t.Fatal("wrong password must fail against the stored account")
	}
	// Setup is closed once an account exists.
	if _, err := s.Setup(ctx, "attacker", "password1234", "", ""); err == nil {
		t.Fatal("setup must be closed after an account exists")
	}
}

// TestSetupDisabledUnderEnv confirms a stranger can't seize an env-configured deployment via Setup.
func TestSetupDisabledUnderEnv(t *testing.T) {
	ctx := context.Background()
	s := NewSessions(newTestStore(t), "cam", "envpassword", "sekret", "", "")
	if _, err := s.Setup(ctx, "attacker", "password1234", "", ""); err == nil {
		t.Fatal("setup must be refused when env credentials manage auth")
	}
}

// TestSetupToken confirms first-run setup requires the configured setup token.
func TestSetupToken(t *testing.T) {
	ctx := context.Background()
	s := NewSessions(newTestStore(t), "cam", "", "sekret", "", "let-me-in")
	if _, err := s.Setup(ctx, "owner", "password1234", "", "wrong-token"); err == nil {
		t.Fatal("setup must require the correct setup token")
	}
	if _, err := s.Setup(ctx, "owner", "password1234", "", "let-me-in"); err != nil {
		t.Fatalf("setup with the correct token should succeed: %v", err)
	}
}

// TestWeakPasswordRejected confirms the minimum-length policy.
func TestWeakPasswordRejected(t *testing.T) {
	ctx := context.Background()
	s := NewSessions(newTestStore(t), "cam", "", "sekret", "", "")
	if _, err := s.Setup(ctx, "owner", "short", "", ""); err == nil {
		t.Fatal("a too-short password must be rejected")
	}
}

// TestResetAndTokenInvalidation covers the recovery-phrase reset, phrase rotation, and that changing
// the password invalidates previously minted tokens.
func TestResetAndTokenInvalidation(t *testing.T) {
	ctx := context.Background()
	s := NewSessions(newTestStore(t), "cam", "", "sekret", "", "")
	phrase, err := s.Setup(ctx, "owner", "originalpass", "hint", "")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	// A token minted now should stop verifying once the password changes.
	oldTok := s.Mint(ctx)
	if !s.Verify(ctx, oldTok) {
		t.Fatal("token should verify before reset")
	}
	// Wrong phrase fails.
	if _, err := s.ResetPassword(ctx, "totally wrong phrase words here", "brandnewpass"); err == nil {
		t.Fatal("reset with a wrong phrase must fail")
	}
	// Correct phrase resets and rotates.
	newPhrase, err := s.ResetPassword(ctx, phrase, "brandnewpass")
	if err != nil {
		t.Fatalf("reset with the correct phrase: %v", err)
	}
	if newPhrase == "" || newPhrase == phrase {
		t.Fatal("reset should rotate to a new recovery phrase")
	}
	// New password works; old one doesn't; the old token is now invalid.
	if !s.CheckCredentials(ctx, "owner", "brandnewpass") {
		t.Fatal("the new password should authenticate")
	}
	if s.CheckCredentials(ctx, "owner", "originalpass") {
		t.Fatal("the old password must no longer authenticate")
	}
	if s.Verify(ctx, oldTok) {
		t.Fatal("a token minted before the password change must be invalidated")
	}
	// The old phrase no longer works; the new one does.
	if _, err := s.ResetPassword(ctx, phrase, "anotherpass1"); err == nil {
		t.Fatal("the rotated-out phrase must not work")
	}
	if _, err := s.ResetPassword(ctx, newPhrase, "anotherpass1"); err != nil {
		t.Fatalf("the rotated-in phrase should work: %v", err)
	}
}

// TestResetBreakGlass confirms the env break-glass token can reset when the phrase is lost.
func TestResetBreakGlass(t *testing.T) {
	ctx := context.Background()
	s := NewSessions(newTestStore(t), "cam", "", "sekret", "break-glass-token", "")
	if _, err := s.Setup(ctx, "owner", "originalpass", "hint", ""); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := s.ResetPassword(ctx, "break-glass-token", "recoveredpass"); err != nil {
		t.Fatalf("break-glass reset should work: %v", err)
	}
	if !s.CheckCredentials(ctx, "owner", "recoveredpass") {
		t.Fatal("password should be reset via break-glass")
	}
}

// TestResetNoAccount confirms reset fails cleanly when nothing is set up.
func TestResetNoAccount(t *testing.T) {
	ctx := context.Background()
	s := NewSessions(newTestStore(t), "cam", "", "sekret", "", "")
	if _, err := s.ResetPassword(ctx, "anything", "newpassword1"); err == nil {
		t.Fatal("reset with no account must fail")
	}
}
