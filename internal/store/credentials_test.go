package store

import (
	"context"
	"path/filepath"
	"testing"
)

// TestInsertCredentialAtomic verifies InsertCredential creates the owner row once and never
// overwrites it — the guard first-run setup relies on to defeat a concurrent-setup race.
func TestInsertCredentialAtomic(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()
	ctx := context.Background()

	created, err := st.InsertCredential(ctx, OwnerCredential{Username: "owner", PasswordHash: "h1", CreatedAt: 1})
	if err != nil || !created {
		t.Fatalf("first insert should create: created=%v err=%v", created, err)
	}
	// A second insert must NOT overwrite and must report created=false.
	created, err = st.InsertCredential(ctx, OwnerCredential{Username: "attacker", PasswordHash: "h2", CreatedAt: 2})
	if err != nil {
		t.Fatalf("second insert err: %v", err)
	}
	if created {
		t.Fatal("second insert must not create/overwrite an existing account")
	}
	got, ok, err := st.GetCredential(ctx)
	if err != nil || !ok {
		t.Fatalf("GetCredential: ok=%v err=%v", ok, err)
	}
	if got.Username != "owner" || got.PasswordHash != "h1" {
		t.Fatalf("existing account was overwritten: %+v", got)
	}
}
