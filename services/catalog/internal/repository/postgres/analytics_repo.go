package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/storage"
)

const (
	topGenresByFilmOpensQuery = `
		SELECT g.id, g.name, g.slug, COUNT(*) AS open_count
		FROM analytic_event AS ae
		JOIN event_type AS et ON et.id = ae.event_type_id
		JOIN film_genre AS fg ON fg.film_id = ae.film_id
		JOIN genre AS g ON g.id = fg.genre_id
		WHERE et.code = 'film_open'
			AND ae.occurred_at >= $1
			AND ae.occurred_at < $2
		GROUP BY g.id, g.name, g.slug
		ORDER BY open_count DESC, g.id ASC
		LIMIT $3`

	genreOpensByAgeQuery = `
		WITH filtered_events AS (
			SELECT ae.id AS event_id,
				ae.film_id,
				ae.occurred_at,
				a.birth_date,
				EXTRACT(YEAR FROM AGE(ae.occurred_at::date, a.birth_date)) AS age_years,
				EXISTS (
					SELECT 1
					FROM film_genre AS fg
					JOIN genre AS g ON g.id = fg.genre_id
					WHERE fg.film_id = ae.film_id AND g.slug = $3
				) AS is_selected_genre
			FROM analytic_event AS ae
			JOIN event_type AS et ON ae.event_type_id = et.id
			JOIN visit AS v ON v.id = ae.visit_id
			LEFT JOIN account AS a ON a.id = v.account_id
			WHERE et.code = 'film_open'
				AND ae.occurred_at >= $1
				AND ae.occurred_at < $2
		), grouped_events AS (
			SELECT *, CASE
				WHEN age_years IS NULL THEN 'unknown'
				WHEN age_years < 18 THEN 'under_18'
				WHEN age_years < 25 THEN '18_24'
				WHEN age_years < 35 THEN '25_34'
				WHEN age_years < 45 THEN '35_44'
				WHEN age_years < 55 THEN '45_54'
				ELSE '55_plus'
			END AS age_group
			FROM filtered_events
		), age_totals AS (
			SELECT age_group,
				COUNT(*) FILTER (WHERE is_selected_genre) AS genre_open_count,
				COUNT(*) AS total_open_count,
				(COUNT(*) FILTER (WHERE is_selected_genre))::double precision / NULLIF(COUNT(*), 0) AS share
			FROM grouped_events
			GROUP BY age_group
		)
		SELECT age_group, genre_open_count, total_open_count, share
		FROM age_totals
		ORDER BY CASE age_group
			WHEN 'under_18' THEN 1
			WHEN '18_24' THEN 2
			WHEN '25_34' THEN 3
			WHEN '35_44' THEN 4
			WHEN '45_54' THEN 5
			WHEN '55_plus' THEN 6
			WHEN 'unknown' THEN 7
		END`

	recordFilmOpenQuery = `
		WITH upserted_visitor AS (
			INSERT INTO visitor (public_id)
			VALUES ($1)
			ON CONFLICT (public_id) DO UPDATE
			SET updated_at = EXCLUDED.updated_at
			RETURNING id
		), existing_visit AS (
			SELECT v.id
			FROM visit AS v
			JOIN upserted_visitor AS uv ON uv.id = v.visitor_id
			WHERE v.finished_at IS NULL
				AND v.started_at >= $4 - INTERVAL '30 minutes'
				AND v.account_id IS NOT DISTINCT FROM $2
			ORDER BY v.started_at DESC
			LIMIT 1
		), new_visit AS (
			INSERT INTO visit (visitor_id, account_id, started_at)
			SELECT uv.id, $2, $4
			FROM upserted_visitor AS uv
			WHERE NOT EXISTS (SELECT 1 FROM existing_visit)
			RETURNING id
		), selected_visit AS (
			SELECT id FROM existing_visit
			UNION ALL
			SELECT id FROM new_visit
			LIMIT 1
		)
		INSERT INTO analytic_event (visit_id, event_type_id, film_id, occurred_at)
		SELECT sv.id, et.id, $3, $4
		FROM selected_visit AS sv
		JOIN event_type AS et ON et.code = 'film_open'
		RETURNING id`
)

type AnalyticsRepo struct {
	db        *sql.DB
	appLogger logger.Logger
}

func NewAnalyticsRepo(db *sql.DB, appLogger logger.Logger) *AnalyticsRepo {
	return &AnalyticsRepo{db: db, appLogger: appLogger}
}

func (r *AnalyticsRepo) TopGenresByFilmOpens(ctx context.Context, from, to time.Time, limit int) (_ []domain.GenreOpenStat, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "analytics.top_genres", map[string]any{"from": from, "to": to, "limit": limit})
	defer func() { operationLog.Finish(operationErr) }()

	rows, err := r.db.QueryContext(ctx, topGenresByFilmOpensQuery, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("query top genres: %w", err)
	}
	defer func() { _ = rows.Close() }()

	stats := make([]domain.GenreOpenStat, 0)
	for rows.Next() {
		var stat domain.GenreOpenStat
		if err := rows.Scan(&stat.GenreID, &stat.Name, &stat.Slug, &stat.OpenCount); err != nil {
			return nil, fmt.Errorf("scan top genre: %w", err)
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read top genres: %w", err)
	}

	return stats, nil
}

func (r *AnalyticsRepo) GenreOpensByAge(ctx context.Context, genreSlug string, from, to time.Time) (_ []domain.AgeGroupOpenStat, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "analytics.genre_age", map[string]any{"genre_slug": genreSlug, "from": from, "to": to})
	defer func() { operationLog.Finish(operationErr) }()

	rows, err := r.db.QueryContext(ctx, genreOpensByAgeQuery, from, to, genreSlug)
	if err != nil {
		return nil, fmt.Errorf("query genre age: %w", err)
	}
	defer func() { _ = rows.Close() }()

	stats := make([]domain.AgeGroupOpenStat, 0)
	for rows.Next() {
		var stat domain.AgeGroupOpenStat
		if err := rows.Scan(&stat.AgeGroup, &stat.GenreOpenCount, &stat.TotalOpenCount, &stat.Share); err != nil {
			return nil, fmt.Errorf("scan genre age: %w", err)
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read genre age: %w", err)
	}

	return stats, nil
}

func (r *AnalyticsRepo) RecordFilmOpen(ctx context.Context, event domain.FilmOpenEvent) (operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "analytics.record_film_open", map[string]any{"film_id": event.FilmID, "account_id": event.AccountID})
	defer func() { operationLog.Finish(operationErr) }()

	var eventID int64
	if err := r.db.QueryRowContext(ctx, recordFilmOpenQuery, event.VisitorPublicID, event.AccountID, event.FilmID, event.OccurredAt).Scan(&eventID); err != nil {
		return fmt.Errorf("record film open: %w", err)
	}

	return nil
}
