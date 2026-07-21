package store

import "context"

// TrackedAnime is one anime the tracker follows (mirrors an AniList Media entry).
type TrackedAnime struct {
	AniListID  int
	Title      string
	CoverImage string
	Status     string
	Episodes   int
	Format     string
	SeasonYear int
	SiteURL    string
	UpdatedAt  int64
}

// ListTrackedAnime returns all tracked anime, most-recently-updated first.
func (s *Store) ListTrackedAnime(ctx context.Context) ([]TrackedAnime, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT anilist_id, title, cover_image, status, episodes, format, season_year, site_url, updated_at
		FROM tracked_anime ORDER BY updated_at DESC, anilist_id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TrackedAnime
	for rows.Next() {
		var a TrackedAnime
		if err := rows.Scan(&a.AniListID, &a.Title, &a.CoverImage, &a.Status, &a.Episodes, &a.Format, &a.SeasonYear, &a.SiteURL, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// UpsertTrackedAnime inserts or updates a tracked anime by AniList id.
func (s *Store) UpsertTrackedAnime(ctx context.Context, a TrackedAnime) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO tracked_anime
		(anilist_id, title, cover_image, status, episodes, format, season_year, site_url, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(anilist_id) DO UPDATE SET
			title=excluded.title, cover_image=excluded.cover_image, status=excluded.status,
			episodes=excluded.episodes, format=excluded.format, season_year=excluded.season_year,
			site_url=excluded.site_url, updated_at=excluded.updated_at`,
		a.AniListID, a.Title, a.CoverImage, a.Status, a.Episodes, a.Format, a.SeasonYear, a.SiteURL, a.UpdatedAt)
	return err
}

// DeleteTrackedAnime removes a tracked anime by AniList id.
func (s *Store) DeleteTrackedAnime(ctx context.Context, aniListID int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM tracked_anime WHERE anilist_id = ?`, aniListID)
	return err
}
