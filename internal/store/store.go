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
		`CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS tailorings (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			job_url    TEXT    NOT NULL DEFAULT '',
			title      TEXT    NOT NULL DEFAULT '',
			company    TEXT    NOT NULL DEFAULT '',
			result     TEXT    NOT NULL,
			created_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS qotd_prompts (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			prompt     TEXT    NOT NULL,
			created_at INTEGER NOT NULL
		)`,
		// Unique prompt text: makes seeding and adds idempotent (INSERT OR IGNORE), so a concurrent
		// double-seed or a duplicate add can never duplicate a prompt.
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_qotd_prompt ON qotd_prompts(prompt)`,
		// The single owner account for the deployed site (first-run setup writes row id=1). The CHECK
		// pins it to one row: there is exactly one owner. Password and recovery phrase are stored only
		// as bcrypt hashes; the hint is plaintext (it is a memory jog the owner chooses, shown on the
		// reset screen). password_changed_at invalidates sessions minted before the last change.
		`CREATE TABLE IF NOT EXISTS owner_credentials (
			id                  INTEGER PRIMARY KEY CHECK (id = 1),
			username            TEXT    NOT NULL,
			password_hash       TEXT    NOT NULL,
			recovery_hash       TEXT    NOT NULL DEFAULT '',
			recovery_hint       TEXT    NOT NULL DEFAULT '',
			password_changed_at INTEGER NOT NULL DEFAULT 0,
			created_at          INTEGER NOT NULL
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	// Best-effort column additions for databases created before these columns existed (SQLite has
	// no ADD COLUMN IF NOT EXISTS, so a duplicate-column error here is expected and ignored).
	for _, alter := range []string{
		`ALTER TABLE tailorings ADD COLUMN title TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tailorings ADD COLUMN company TEXT NOT NULL DEFAULT ''`,
	} {
		_, _ = db.Exec(alter)
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
