// Package store is the SQLite-backed persistence layer, using the pure-Go modernc.org/sqlite
// driver (no cgo) so cross-compilation and the single-binary deploy stay clean.
package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Store wraps the application's SQLite database.
type Store struct {
	db *sql.DB
}

// Open opens the database at path, creating the parent directory and applying the schema if
// needed. Use ":memory:" for an ephemeral database.
func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." && path != ":memory:" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }

// migrate creates the schema if it does not already exist. It is idempotent, so it runs safely
// on every startup.
func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS contact_messages (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT    NOT NULL,
			email      TEXT    NOT NULL,
			body       TEXT    NOT NULL,
			created_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS tracked_anime (
			anilist_id  INTEGER PRIMARY KEY,
			title       TEXT    NOT NULL,
			cover_image TEXT    NOT NULL DEFAULT '',
			status      TEXT    NOT NULL DEFAULT '',
			episodes    INTEGER NOT NULL DEFAULT 0,
			format      TEXT    NOT NULL DEFAULT '',
			season_year INTEGER NOT NULL DEFAULT 0,
			site_url    TEXT    NOT NULL DEFAULT '',
			updated_at  INTEGER NOT NULL
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

// ContactMessage is a stored inbound message.
type ContactMessage struct {
	Name      string
	Email     string
	Body      string
	CreatedAt int64
}

// SaveContact inserts a contact message.
func (s *Store) SaveContact(ctx context.Context, m ContactMessage) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO contact_messages (name, email, body, created_at) VALUES (?, ?, ?, ?)`,
		m.Name, m.Email, m.Body, m.CreatedAt)
	return err
}

// CountContacts returns how many messages are stored (used by tests and, later, the admin inbox).
func (s *Store) CountContacts(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM contact_messages`).Scan(&n)
	return n, err
}
