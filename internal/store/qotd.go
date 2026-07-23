package store

import "context"

// QOTDPost is one generated-and-published anime discussion post — a row of the QOTD feed's
// durable history. PublishedAt is a Unix timestamp (seconds).
type QOTDPost struct {
	ID          int64
	Title       string
	Body        string
	PublishedAt int64
}

// AddQOTDPost appends a published post to the history.
func (s *Store) AddQOTDPost(ctx context.Context, p QOTDPost) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO qotd_posts (title, body, published_at) VALUES (?, ?, ?)`,
		p.Title, p.Body, p.PublishedAt)
	return err
}

// RecentQOTDPosts returns the newest n published posts, newest first. Older rows stay recorded;
// they just fall off the feed.
func (s *Store) RecentQOTDPosts(ctx context.Context, n int) ([]QOTDPost, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, title, body, published_at FROM qotd_posts ORDER BY published_at DESC, id DESC LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []QOTDPost
	for rows.Next() {
		var p QOTDPost
		if err := rows.Scan(&p.ID, &p.Title, &p.Body, &p.PublishedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
