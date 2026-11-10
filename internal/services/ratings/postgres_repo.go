package ratings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/jackc/pgx/v5/pgconn"
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

func (r *PostgresRepo) SetFilmRating(ctx context.Context, accountID domain.UserID, filmID domain.FilmID, score int) (_ domain.FilmRating, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "rating.set_film_rating", map[string]any{"user_id": accountID, "film_id": filmID, "score": score})
	defer func() { operationLog.Finish(operationErr) }()

	var rating domain.FilmRating
	operationLog.Stage = "query"
	err := r.db.QueryRowContext(ctx, setFilmRatingQuery, accountID, filmID, score).Scan(
		&rating.AccountID,
		&rating.FilmID,
		&rating.Score,
		&rating.CreatedAt,
		&rating.UpdatedAt,
	)
	if err == nil {
		return rating, nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == storage.ForeignKeyViolation && pgErr.ConstraintName == "film_rating_film_id_fkey" {
		return domain.FilmRating{}, domain.ErrFilmNotFound
	}
	if errors.As(err, &pgErr) && pgErr.Code == storage.CheckViolation && pgErr.ConstraintName == "film_rating_score_check" {
		return domain.FilmRating{}, domain.InvalidFilmRatingLimitsError()
	}
	return domain.FilmRating{}, fmt.Errorf("set film rating: %w", err)
}

func (r *PostgresRepo) GetFilmRatingSummary(ctx context.Context, filmID domain.FilmID) (_ domain.FilmRatingSummary, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "rating.get_film_rating_summary", map[string]any{"film_id": filmID})
	defer func() { operationLog.Finish(operationErr) }()

	summary := domain.FilmRatingSummary{FilmID: filmID}
	operationLog.Stage = "query"
	err := r.db.QueryRowContext(ctx, getFilmRatingSummaryQuery, filmID).Scan(&summary.AverageScore, &summary.Votes)
	if err != nil {
		return domain.FilmRatingSummary{}, fmt.Errorf("get film rating summary: %w", err)
	}
	return summary, nil
}

func (r *PostgresRepo) GetFilmRatingsSummaries(ctx context.Context, filmIDs []domain.FilmID) (_ []domain.FilmRatingSummary, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "rating.get_film_ratings_summaries", map[string]any{})
	defer func() { operationLog.Finish(operationErr) }()

	summaries := make([]domain.FilmRatingSummary, 0, len(filmIDs))
	if len(filmIDs) == 0 {
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
	operationLog.Stage = "query"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get film rating summaries: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var summary domain.FilmRatingSummary
		if err := rows.Scan(&summary.FilmID, &summary.AverageScore, &summary.Votes); err != nil {
			return nil, fmt.Errorf("scan film rating summary: %w", err)
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get film rating summaries: %w", err)
	}
	return summaries, nil
}

func (r *PostgresRepo) GetCollectionFilmRatingSummaries(ctx context.Context, collectionID domain.CollectionID, limit, offset int) (_ []domain.FilmRatingSummary, _ int, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "rating.get_collection_film_rating_summaries", map[string]any{"collection_id": collectionID, "limit": limit, "offset": offset})
	defer func() { operationLog.Finish(operationErr) }()

	operationLog.Stage = "query"
	rows, err := r.db.QueryContext(ctx, getCollectionFilmRatingSummariesQuery, collectionID, limit, offset)
	if err != nil {
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
		return nil, 0, fmt.Errorf("get collection film rating summaries: %w", err)
	}
	if !foundCollection {
		return nil, 0, domain.ErrCollectionNotFound
	}
	return summaries, total, nil
}

func (r *PostgresRepo) DeleteFilmRating(ctx context.Context, accountID domain.UserID, filmID domain.FilmID) (operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "rating.delete_film_rating", map[string]any{"user_id": accountID, "film_id": filmID})
	defer func() { operationLog.Finish(operationErr) }()

	operationLog.Stage = "query"
	result, err := r.db.ExecContext(ctx, deleteFilmRatingQuery, accountID, filmID)
	if err != nil {
		return fmt.Errorf("delete film rating: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get deleted film rating count: %w", err)
	}
	if affected == 0 {
		return domain.ErrRatingNotFound
	}
	return nil
}
