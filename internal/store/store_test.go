package store

import (
	"context"
	"path/filepath"
	"testing"
)

// TestSaveAndCount verifies a message round-trips through a fresh on-disk database.
func TestSaveAndCount(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	if err := s.SaveContact(ctx, ContactMessage{Name: "Ada", Email: "ada@example.com", Body: "hi", CreatedAt: 1}); err != nil {
		t.Fatalf("save: %v", err)
	}
	n, err := s.CountContacts(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("count = %d, want 1", n)
	}
}
