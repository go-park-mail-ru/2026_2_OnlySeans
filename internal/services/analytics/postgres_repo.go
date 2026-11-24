package analytics

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const topGenresByFilmOpensQuery = `
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

type PostgresRepo struct {
	db        *sql.DB
	appLogger logger.Logger
}

func NewPostgresRepo(db *sql.DB, appLogger logger.Logger) *PostgresRepo {
	return &PostgresRepo{db: db, appLogger: appLogger}
}

func (r *PostgresRepo) TopGenresByFilmOpens(ctx context.Context, from, to time.Time, limit int) (_ []domain.GenreOpenStat, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "analytics.top_genres_by_film_opens", map[string]any{"from": from, "to": to, "limit": limit})

	defer func() { operationLog.Finish(operationErr) }()

	stats := make([]domain.GenreOpenStat, 0)
	operationLog.Stage = "query"

	rows, err := r.db.QueryContext(ctx, topGenresByFilmOpensQuery, from, to, limit)
	if err != nil {
		return stats, err
	}

	defer rows.Close()

	for rows.Next() {
		var stat domain.GenreOpenStat

		if err := rows.Scan(
			&stat.GenreID,
			&stat.Name,
			&stat.Slug,
			&stat.OpenCount,
		); err != nil {
			return stats, err
		}

		stats = append(stats, stat)
	}

	return stats, rows.Err()
}
