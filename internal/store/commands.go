package store

import (
	"context"
	"time"
)

// Terminal command counters.
//
// The point of this table is a single question Cam cannot otherwise answer: does anyone actually
// type anything into the terminal, and if so, what? That decides where the next round of terminal
// work goes. It is answered with the least data that can answer it — a day, a command name, and a
// count — and deliberately nothing that could identify who ran it. See the privacy note on
// CommandEvent in proto/site.proto and on the terminal_commands table in store.go.

// CommandCount is one row of the aggregate: how often a command ran on a given day.
type CommandCount struct {
	Day   string
	Name  string
	Count int64
}

// RecordCommand increments the counter for name on today's date (UTC, so the aggregate does not
// shift with the server's timezone).
func (s *Store) RecordCommand(ctx context.Context, name string) error {
	day := time.Now().UTC().Format("2006-01-02")
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO terminal_commands (day, name, count) VALUES (?, ?, 1)
		 ON CONFLICT(day, name) DO UPDATE SET count = count + 1`, day, name)
	return err
}

// CommandTotals returns per-command totals across the last days days, most-run first. A zero or
// negative days means all of recorded history.
func (s *Store) CommandTotals(ctx context.Context, days int) ([]CommandCount, error) {
	query := `SELECT '', name, SUM(count) AS total FROM terminal_commands
	          GROUP BY name ORDER BY total DESC`
	args := []any{}
	if days > 0 {
		since := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")
		query = `SELECT '', name, SUM(count) AS total FROM terminal_commands
		         WHERE day >= ? GROUP BY name ORDER BY total DESC`
		args = append(args, since)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CommandCount
	for rows.Next() {
		var c CommandCount
		if err := rows.Scan(&c.Day, &c.Name, &c.Count); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CommandGrandTotal returns how many terminal commands have been recorded in total.
func (s *Store) CommandGrandTotal(ctx context.Context) (int64, error) {
	var n *int64
	if err := s.db.QueryRowContext(ctx, `SELECT SUM(count) FROM terminal_commands`).Scan(&n); err != nil {
		return 0, err
	}
	if n == nil { // no rows recorded yet — SUM over an empty set is NULL, not 0
		return 0, nil
	}
	return *n, nil
}
