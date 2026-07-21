package store

import (
	"context"
	"database/sql"
	"errors"
)

// Tailoring is a persisted résumé-tailoring variant: metadata for the list plus the serialized
// TailorResult (protojson) that can be re-opened.
type Tailoring struct {
	ID        int64
	JobURL    string
	Title     string
	Company   string
	Result    string
	CreatedAt int64
}

// SaveTailoring persists a tailoring result (with glanceable metadata) and returns its new id.
func (s *Store) SaveTailoring(ctx context.Context, jobURL, title, company, result string, createdAt int64) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO tailorings (job_url, title, company, result, created_at) VALUES (?, ?, ?, ?, ?)`,
		jobURL, title, company, result, createdAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListTailorings returns saved variants (newest first) — metadata only, no result payload.
func (s *Store) ListTailorings(ctx context.Context) ([]Tailoring, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, job_url, title, company, created_at FROM tailorings ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tailoring
	for rows.Next() {
		var t Tailoring
		if err := rows.Scan(&t.ID, &t.JobURL, &t.Title, &t.Company, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetTailoring returns one variant's full result by id, or ok=false if it doesn't exist.
func (s *Store) GetTailoring(ctx context.Context, id int64) (Tailoring, bool, error) {
	var t Tailoring
	err := s.db.QueryRowContext(ctx,
		`SELECT id, job_url, title, company, result, created_at FROM tailorings WHERE id = ?`, id).
		Scan(&t.ID, &t.JobURL, &t.Title, &t.Company, &t.Result, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Tailoring{}, false, nil
	}
	if err != nil {
		return Tailoring{}, false, err
	}
	return t, true, nil
}

// LatestTailoring returns the most recent variant's full result, or ok=false if none exist.
func (s *Store) LatestTailoring(ctx context.Context) (Tailoring, bool, error) {
	var t Tailoring
	err := s.db.QueryRowContext(ctx,
		`SELECT id, job_url, title, company, result, created_at FROM tailorings ORDER BY id DESC LIMIT 1`).
		Scan(&t.ID, &t.JobURL, &t.Title, &t.Company, &t.Result, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Tailoring{}, false, nil
	}
	if err != nil {
		return Tailoring{}, false, err
	}
	return t, true, nil
}

// DeleteTailoring removes a variant by id.
func (s *Store) DeleteTailoring(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM tailorings WHERE id = ?`, id)
	return err
}
