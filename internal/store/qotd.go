package store

import "context"

// QOTDPrompt is one stored question-of-the-day discussion prompt.
type QOTDPrompt struct {
	ID        int64
	Prompt    string
	CreatedAt int64
}

// AddPrompt inserts a new QOTD prompt stamped with createdAt (unix seconds) and returns its id.
func (s *Store) AddPrompt(ctx context.Context, prompt string, createdAt int64) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO qotd_prompts (prompt, created_at) VALUES (?, ?)`, prompt, createdAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListPrompts returns every stored QOTD prompt, oldest first.
func (s *Store) ListPrompts(ctx context.Context) ([]QOTDPrompt, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, prompt, created_at FROM qotd_prompts ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []QOTDPrompt
	for rows.Next() {
		var p QOTDPrompt
		if err := rows.Scan(&p.ID, &p.Prompt, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// DeletePrompt removes a QOTD prompt by id.
func (s *Store) DeletePrompt(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM qotd_prompts WHERE id = ?`, id)
	return err
}

// CountPrompts returns how many QOTD prompts are stored.
func (s *Store) CountPrompts(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM qotd_prompts`).Scan(&n)
	return n, err
}
