package ratings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const (
	setFilmRatingQuery = `
		INSERT INTO film_rating (account_id, film_id, score)
		VALUES ($1, $2, $3)
		ON CONFLICT (account_id, film_id)
		DO UPDATE SET score = EXCLUDED.score
		RETURNING account_id, film_id, score, created_at, updated_at`

	getFilmRatingSummaryQuery = `
		SELECT COALESCE(ROUND(AVG(score), 1), 0)::double precision, count(*)
		FROM film_rating
		WHERE film_id = $1`

	getFilmRatingSummariesQuery = `
		WITH requested_films AS (
			SELECT film_id, position
			FROM (VALUES %s) AS requested(film_id, position)
		)
		SELECT requested_films.film_id,
		       COALESCE(ROUND(AVG(film_rating.score), 1), 0)::double precision,
		       count(film_rating.score)
		FROM requested_films
		LEFT JOIN film_rating ON film_rating.film_id = requested_films.film_id
		GROUP BY requested_films.film_id, requested_films.position
		ORDER BY requested_films.position`
	getCollectionFilmRatingSummariesQuery = `
		WITH target_collection AS (
			SELECT id FROM collection WHERE id = $1
		),
		collection_total AS (
			SELECT count(*)::bigint AS total
			FROM collection_film
			WHERE collection_id = $1
		),
		page_films AS (
			SELECT cf.film_id, cf.position
			FROM collection_film AS cf
			JOIN target_collection AS c ON c.id = cf.collection_id
			ORDER BY cf.position
			LIMIT $2 OFFSET $3
		)
		SELECT c.id,
		       collection_total.total,
		       page_films.film_id,
		       COALESCE(ROUND(AVG(fr.score), 1), 0)::double precision,
		       count(fr.score)
		FROM target_collection AS c
		CROSS JOIN collection_total
		LEFT JOIN page_films ON TRUE
		LEFT JOIN film_rating AS fr ON fr.film_id = page_films.film_id
		GROUP BY c.id, collection_total.total, page_films.film_id, page_films.position
		ORDER BY page_films.position`
	deleteFilmRatingQuery = `
		DELETE FROM film_rating
		WHERE account_id = $1 AND film_id = $2`
)

type PostgresRepo struct {
	db        *sql.DB
	appLogger logger.Logger
}

func NewPostgresRepo(db *sql.DB, appLogger logger.Logger) *PostgresRepo {
	return &PostgresRepo{db: db, appLogger: appLogger}
}

func (r *PostgresRepo) SetFilmRating(ctx context.Context, accountID domain.UserID, filmID domain.FilmID, score int) (domain.FilmRating, error) {
	fields := map[string]any{
		"account_id": accountID,
		"film_id":    filmID,
		"score":      score,
	}
	r.appLogger.Debug("postgres set film rating started", fields)

	var rating domain.FilmRating
	err := r.db.QueryRowContext(ctx, setFilmRatingQuery, accountID, filmID, score).Scan(
		&rating.AccountID,
		&rating.FilmID,
		&rating.Score,
		&rating.CreatedAt,
		&rating.UpdatedAt,
	)
	if err == nil {
		r.appLogger.Debug("postgres film rating set", map[string]any{
			"account_id": rating.AccountID,
			"film_id":    rating.FilmID,
			"score":      rating.Score,
		})
		return rating, nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == storage.ForeignKeyViolation && pgErr.ConstraintName == "film_rating_film_id_fkey" {
		r.appLogger.Debug("postgres set film rating failed: film not found", map[string]any{
			"film_id": filmID,
			"error":   err,
		})
		return domain.FilmRating{}, domain.ErrFilmNotFound
	}
	if errors.As(err, &pgErr) && pgErr.Code == storage.CheckViolation && pgErr.ConstraintName == "film_rating_score_check" {
		r.appLogger.Debug("postgres set film rating failed: invalid score", map[string]any{
			"film_id": filmID,
			"score":   score,
			"error":   err,
		})
		return domain.FilmRating{}, domain.InvalidFilmRatingLimitsError()
	}
	r.appLogger.Debug("postgres set film rating failed", map[string]any{
		"account_id": accountID,
		"film_id":    filmID,
		"error":      err,
	})
	return domain.FilmRating{}, fmt.Errorf("set film rating: %w", err)
}

func (r *PostgresRepo) GetFilmRatingSummary(ctx context.Context, filmID domain.FilmID) (domain.FilmRatingSummary, error) {
	r.appLogger.Debug("postgres get film rating summary started", map[string]any{"film_id": filmID})
	summary := domain.FilmRatingSummary{FilmID: filmID}
	err := r.db.QueryRowContext(ctx, getFilmRatingSummaryQuery, filmID).Scan(&summary.AverageScore, &summary.Votes)
	if err != nil {
		r.appLogger.Debug("postgres get film rating summary failed", map[string]any{
			"film_id": filmID,
			"error":   err,
		})
		return domain.FilmRatingSummary{}, fmt.Errorf("get film rating summary: %w", err)
	}
	r.appLogger.Debug("postgres film rating summary retrieved", map[string]any{
		"film_id":       summary.FilmID,
		"average_score": summary.AverageScore,
		"votes":         summary.Votes,
	})
	return summary, nil
}

func (r *PostgresRepo) GetFilmRatingsSummaries(ctx context.Context, filmIDs []domain.FilmID) ([]domain.FilmRatingSummary, error) {
	r.appLogger.Debug("postgres get film rating summaries started", map[string]any{"film_count": len(filmIDs)})
	summaries := make([]domain.FilmRatingSummary, 0, len(filmIDs))
	if len(filmIDs) == 0 {
		r.appLogger.Debug("postgres get film rating summaries skipped: no film ids", nil)
		return summaries, nil
	}

	var values strings.Builder
	args := make([]any, len(filmIDs))
	for i, filmID := range filmIDs {
		if i > 0 {
			values.WriteString(", ")
		}
		fmt.Fprintf(&values, "($%d, %d)", i+1, i+1)
		args[i] = int64(filmID)
	}

	query := fmt.Sprintf(getFilmRatingSummariesQuery, values.String())
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.appLogger.Debug("postgres get film rating summaries failed", map[string]any{
			"film_count": len(filmIDs),
			"error":      err,
		})
		return nil, fmt.Errorf("get film rating summaries: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var summary domain.FilmRatingSummary
		if err := rows.Scan(&summary.FilmID, &summary.AverageScore, &summary.Votes); err != nil {
			r.appLogger.Debug("postgres scan film rating summary failed", map[string]any{
				"error": err,
			})
			return nil, fmt.Errorf("scan film rating summary: %w", err)
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		r.appLogger.Debug("postgres iterate film rating summaries failed", map[string]any{
			"error": err,
		})
		return nil, fmt.Errorf("get film rating summaries: %w", err)
	}
	r.appLogger.Debug("postgres film rating summaries retrieved", map[string]any{
		"film_count":   len(filmIDs),
		"result_count": len(summaries),
	})
	return summaries, nil
}

func (r *PostgresRepo) GetCollectionFilmRatingSummaries(ctx context.Context, collectionID domain.CollectionID, limit, offset int) ([]domain.FilmRatingSummary, int, error) {
	fields := map[string]any{
		"collection_id": collectionID,
		"limit":         limit,
		"offset":        offset,
	}
	r.appLogger.Debug("postgres get collection film rating summaries started", fields)
	rows, err := r.db.QueryContext(ctx, getCollectionFilmRatingSummariesQuery, collectionID, limit, offset)
	if err != nil {
		r.appLogger.Debug("postgres get collection film rating summaries failed", map[string]any{
			"collection_id": collectionID,
			"error":         err,
		})
		return nil, 0, fmt.Errorf("get collection film rating summaries: %w", err)
	}
	defer rows.Close()

	summaries := make([]domain.FilmRatingSummary, 0)
	foundCollection := false
	total := 0
	for rows.Next() {
		var rowCollectionID domain.CollectionID
		var rowTotal int
		var filmID sql.NullInt64
		var summary domain.FilmRatingSummary
		if err := rows.Scan(&rowCollectionID, &rowTotal, &filmID, &summary.AverageScore, &summary.Votes); err != nil {
			r.appLogger.Debug("postgres scan collection film rating summary failed", map[string]any{
				"collection_id": collectionID,
				"error":         err,
			})
			return nil, 0, fmt.Errorf("scan collection film rating summary: %w", err)
		}
		foundCollection = true
		total = rowTotal
		if filmID.Valid {
			summary.FilmID = domain.FilmID(filmID.Int64)
			summaries = append(summaries, summary)
		}
	}
	if err := rows.Err(); err != nil {
		r.appLogger.Debug("postgres iterate collection film rating summaries failed", map[string]any{
			"collection_id": collectionID,
			"error":         err,
		})
		return nil, 0, fmt.Errorf("get collection film rating summaries: %w", err)
	}
	if !foundCollection {
		r.appLogger.Debug("postgres collection for rating summaries not found", map[string]any{
			"collection_id": collectionID,
		})
		return nil, 0, domain.ErrCollectionNotFound
	}
	r.appLogger.Debug("postgres collection film rating summaries retrieved", map[string]any{
		"collection_id": collectionID,
		"result_count":  len(summaries),
		"total":         total,
		"limit":         limit,
		"offset":        offset,
	})
	return summaries, total, nil
}

func (r *PostgresRepo) DeleteFilmRating(ctx context.Context, accountID domain.UserID, filmID domain.FilmID) error {
	fields := map[string]any{
		"account_id": accountID,
		"film_id":    filmID,
	}
	r.appLogger.Debug("postgres delete film rating started", fields)
	result, err := r.db.ExecContext(ctx, deleteFilmRatingQuery, accountID, filmID)
	if err != nil {
		r.appLogger.Debug("postgres delete film rating failed", map[string]any{
			"account_id": accountID,
			"film_id":    filmID,
			"error":      err,
		})
		return fmt.Errorf("delete film rating: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		r.appLogger.Debug("postgres get deleted film rating count failed", map[string]any{
			"account_id": accountID,
			"film_id":    filmID,
			"error":      err,
		})
		return fmt.Errorf("get deleted film rating count: %w", err)
	}
	if affected == 0 {
		r.appLogger.Debug("postgres film rating to delete not found", fields)
		return domain.ErrRatingNotFound
	}
	r.appLogger.Debug("postgres film rating deleted", map[string]any{
		"account_id":    accountID,
		"film_id":       filmID,
		"rows_affected": affected,
	})
	return nil
}
