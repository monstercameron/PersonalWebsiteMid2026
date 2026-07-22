package store

import (
	"context"
	"database/sql"
	"errors"
)

// OwnerCredential is the single owner account for the deployed site. Password and RecoveryPhrase are
// held only as bcrypt hashes (never plaintext); RecoveryHint is a plaintext memory jog the owner
// chose. PasswordChangedAt (unix seconds) is stamped on every password change so sessions minted
// before it can be rejected.
type OwnerCredential struct {
	Username          string
	PasswordHash      string
	RecoveryHash      string
	RecoveryHint      string
	PasswordChangedAt int64
	CreatedAt         int64
}

// GetCredential returns the owner account, or ok=false when none has been set up yet.
func (s *Store) GetCredential(ctx context.Context) (OwnerCredential, bool, error) {
	var c OwnerCredential
	err := s.db.QueryRowContext(ctx,
		`SELECT username, password_hash, recovery_hash, recovery_hint, password_changed_at, created_at
		 FROM owner_credentials WHERE id = 1`).
		Scan(&c.Username, &c.PasswordHash, &c.RecoveryHash, &c.RecoveryHint, &c.PasswordChangedAt, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return OwnerCredential{}, false, nil
	}
	if err != nil {
		return OwnerCredential{}, false, err
	}
	return c, true, nil
}

// HasCredential reports whether an owner account has been set up.
func (s *Store) HasCredential(ctx context.Context) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM owner_credentials WHERE id = 1`).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// InsertCredential atomically creates the owner row only if it does not already exist, returning
// created=false when a row is already present. Unlike PutCredential it never overwrites, so first-run
// setup can't be raced (two concurrent setups) or replayed into overwriting an established account —
// the "does it exist" check and the write are a single atomic statement.
func (s *Store) InsertCredential(ctx context.Context, c OwnerCredential) (created bool, err error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO owner_credentials
		   (id, username, password_hash, recovery_hash, recovery_hint, password_changed_at, created_at)
		 VALUES (1, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO NOTHING`,
		c.Username, c.PasswordHash, c.RecoveryHash, c.RecoveryHint, c.PasswordChangedAt, c.CreatedAt)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// PutCredential upserts the single owner row (id = 1).
func (s *Store) PutCredential(ctx context.Context, c OwnerCredential) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO owner_credentials
		   (id, username, password_hash, recovery_hash, recovery_hint, password_changed_at, created_at)
		 VALUES (1, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   username            = excluded.username,
		   password_hash       = excluded.password_hash,
		   recovery_hash       = excluded.recovery_hash,
		   recovery_hint       = excluded.recovery_hint,
		   password_changed_at = excluded.password_changed_at`,
		c.Username, c.PasswordHash, c.RecoveryHash, c.RecoveryHint, c.PasswordChangedAt, c.CreatedAt)
	return err
}
