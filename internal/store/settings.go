package store

import (
	"context"
	"database/sql"
	"errors"
)

// Setting keys used by the admin console. Kept as constants so callers don't hardcode strings.
const (
	SettingOpenAIKey   = "openai_api_key"
	SettingOpenAIModel = "openai_model"
	SettingActiveResume = "active_resume" // JSON of the applied résumé (overrides the canonical)
)

// GetSetting returns the stored value for key, or "" if it has never been set.
func (s *Store) GetSetting(ctx context.Context, key string) (string, error) {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

// SetSetting upserts a key/value setting.
func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}
