package admin

import "testing"

// TestJWTMintVerify checks the JWT round-trip and that tampered, wrong-secret, and wrong-subject
// tokens are rejected.
func TestJWTMintVerify(t *testing.T) {
	s := NewSessions("cam", "pw", "sekret")

	tok := s.Mint()
	if tok == "" || !s.Verify(tok) {
		t.Fatal("a freshly minted token should verify")
	}
	if s.Verify(tok + "x") {
		t.Fatal("a tampered token must fail")
	}
	if s.Verify("not-a-jwt") {
		t.Fatal("garbage must fail")
	}
	// A token signed with a different secret must not verify.
	if NewSessions("cam", "pw", "different-secret").Verify(tok) {
		t.Fatal("token from a different secret must fail")
	}
	// A validly-signed token for a different subject must not verify against this owner.
	otherSubject := NewSessions("eve", "pw", "sekret").Mint()
	if s.Verify(otherSubject) {
		t.Fatal("token for a different subject must fail")
	}
}

// TestCheckCredentials verifies both username and password must match.
func TestCheckCredentials(t *testing.T) {
	s := NewSessions("cam", "pw", "sekret")
	if !s.CheckCredentials("cam", "pw") {
		t.Fatal("correct credentials should pass")
	}
	if s.CheckCredentials("cam", "wrong") || s.CheckCredentials("eve", "pw") || s.CheckCredentials("", "") {
		t.Fatal("bad credentials must fail")
	}
	// A disabled gate (no password) rejects everything.
	if NewSessions("cam", "", "sekret").CheckCredentials("cam", "") {
		t.Fatal("disabled gate must reject")
	}
}
