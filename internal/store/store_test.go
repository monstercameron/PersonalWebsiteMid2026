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

// TestQOTDPostHistory verifies published posts accumulate (past days stay recorded), come back
// newest first, and that the limit only bounds the feed view, not the stored history.
func TestQOTDPostHistory(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	for day := 1; day <= 3; day++ {
		if err := s.AddQOTDPost(ctx, QOTDPost{Title: "day", Body: "post", PublishedAt: int64(day)}); err != nil {
			t.Fatalf("add day %d: %v", day, err)
		}
	}
	posts, err := s.RecentQOTDPosts(ctx, 2)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(posts) != 2 || posts[0].PublishedAt != 3 || posts[1].PublishedAt != 2 {
		t.Fatalf("want newest 2 of 3 (3,2), got %+v", posts)
	}
	all, err := s.RecentQOTDPosts(ctx, 10)
	if err != nil {
		t.Fatalf("recent all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("history must keep all 3 rows, got %d", len(all))
	}
}

// TestQOTDLegacyMigration verifies the old single-post qotd_published settings blob is moved into
// the qotd_posts table exactly once on open, and the settings row is removed.
func TestQOTDLegacyMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	ctx := context.Background()

	s, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	legacy := `{"title":"Anime discussion — Jul 20, 2026","body":"Old prompt","publishedAt":"2026-07-20T09:00:00Z"}`
	if err := s.SetSetting(ctx, SettingQOTDPublished, legacy); err != nil {
		t.Fatalf("seed legacy setting: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	s, err = Open(path) // migration runs here
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s.Close()
	posts, err := s.RecentQOTDPosts(ctx, 10)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(posts) != 1 || posts[0].Body != "Old prompt" {
		t.Fatalf("legacy post not migrated: %+v", posts)
	}
	if v, _ := s.GetSetting(ctx, SettingQOTDPublished); v != "" {
		t.Fatalf("legacy setting should be deleted after migration, got %q", v)
	}
}

// TestCommandCounters aggregates terminal command runs by name.
func TestCommandCounters(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "cmds.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()
	ctx := context.Background()

	if total, err := s.CommandGrandTotal(ctx); err != nil || total != 0 {
		t.Fatalf("empty total = %d, %v; want 0, nil", total, err)
	}
	for _, name := range []string{"help", "projects", "projects", "projects"} {
		if err := s.RecordCommand(ctx, name); err != nil {
			t.Fatalf("record %s: %v", name, err)
		}
	}
	total, err := s.CommandGrandTotal(ctx)
	if err != nil {
		t.Fatalf("total: %v", err)
	}
	if total != 4 {
		t.Fatalf("total = %d; want 4", total)
	}
	counts, err := s.CommandTotals(ctx, 0)
	if err != nil {
		t.Fatalf("totals: %v", err)
	}
	if len(counts) != 2 || counts[0].Name != "projects" || counts[0].Count != 3 {
		t.Fatalf("totals = %+v; want projects=3 first", counts)
	}
}
