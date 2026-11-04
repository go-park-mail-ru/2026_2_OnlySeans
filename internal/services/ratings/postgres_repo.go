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
	deleteFilmRatingQuery = `
		DELETE FROM film_rating
		WHERE account_id = $1 AND film_id = $2`
)

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) domain.FilmRatingRepository {
	return &PostgresRepo{db: db}
}

func (r *PostgresRepo) Set(ctx context.Context, accountID domain.UserID, filmID domain.FilmID, score int16) (domain.FilmRating, error) {
	var rating domain.FilmRating
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
		return domain.FilmRating{}, domain.ErrInvalidFilmRating
	}
	return domain.FilmRating{}, fmt.Errorf("set film rating: %w", err)
}

func (r *PostgresRepo) GetSummary(ctx context.Context, filmID domain.FilmID) (domain.FilmRatingSummary, error) {
	summary := domain.FilmRatingSummary{FilmID: filmID}
	err := r.db.QueryRowContext(ctx, getFilmRatingSummaryQuery, filmID).Scan(&summary.AverageScore, &summary.Votes)
	if err != nil {
		return domain.FilmRatingSummary{}, fmt.Errorf("get film rating summary: %w", err)
	}
	return summary, nil
}

func (r *PostgresRepo) GetSummaries(ctx context.Context, filmIDs []domain.FilmID) ([]domain.FilmRatingSummary, error) {
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

func (r *PostgresRepo) Delete(ctx context.Context, accountID domain.UserID, filmID domain.FilmID) error {
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
