package store

import (
	"context"
	"database/sql"
	"errors"
)

// Setting keys used by the admin console. Kept as constants so callers don't hardcode strings.
const (
	SettingOpenAIKey    = "openai_api_key"
	SettingOpenAIModel  = "openai_model"
	SettingActiveResume = "active_resume"     // JSON of the applied résumé (overrides the canonical)
	SettingSlackWebhook = "slack_webhook_url" // Slack incoming-webhook URL for QOTD/news posts
	SettingSlackEnabled = "slack_enabled"     // "1"/"true" to enable scheduled Slack posting
	SettingQOTDPrompt   = "qotd_prompt"       // the single generation instruction for the anime discussion post
	SettingQOTDPublished = "qotd_published"   // JSON of the last generated-and-published post (served by the QOTD feed)
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
