package store

import (
	"context"
	"database/sql"
	"errors"
)

// Tailoring is a persisted résumé-tailoring result (the serialized TailorResult plus its job URL).
type Tailoring struct {
	JobURL    string
	Result    string // serialized proto (protojson)
	CreatedAt int64
}

// SaveTailoring persists a tailoring result so it survives restarts and can be re-opened.
func (s *Store) SaveTailoring(ctx context.Context, jobURL, result string, createdAt int64) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO tailorings (job_url, result, created_at) VALUES (?, ?, ?)`,
		jobURL, result, createdAt)
	return err
}

// LatestTailoring returns the most recent tailoring, or ok=false if none has been saved.
func (s *Store) LatestTailoring(ctx context.Context) (Tailoring, bool, error) {
	var t Tailoring
	err := s.db.QueryRowContext(ctx,
		`SELECT job_url, result, created_at FROM tailorings ORDER BY id DESC LIMIT 1`).
		Scan(&t.JobURL, &t.Result, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Tailoring{}, false, nil
	}
	if err != nil {
		return Tailoring{}, false, err
	}
	return t, true, nil
}
