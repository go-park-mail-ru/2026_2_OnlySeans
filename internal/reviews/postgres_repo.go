//хранилище отзывов

package reviews

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/entities"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/films"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	uniqueViolation     = "23505" //у автора уже есть отзыв на этот фильм
	foreignKeyViolation = "23503" //фильма с таким номером нет в базе

	reviewColumns = `id, account_id, film_id, title, content, contains_spoilers, created_at, updated_at` //список колонок отзыва

	createReviewQuery = `
		INSERT INTO review (account_id, film_id, title, content, contains_spoilers)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + reviewColumns

	getReviewQuery = `
		SELECT ` + reviewColumns +
		` FROM review
		WHERE id = $1 AND deleted_at IS NULL`

	listReviewsQuery = `
		SELECT ` + reviewColumns +
		`
			FROM review
			WHERE film_id = $1 AND deleted_at IS NULL
			ORDER BY id DESC
			LIMIT $2 OFFSET $3`

	countReviewsQuery = `
		SELECT count(*) FROM review
		WHERE film_id = $1 AND deleted_at IS NULL`

	updateReviewQuery = `
		UPDATE review SET title = $2, content= $3, contains_spoilers=$4
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING ` + reviewColumns

	deleteReviewQuery = `
		UPDATE review SET deleted_at = now(), deleted_by = $2
		WHERE id = $1 AND deleted_at IS NULL
	`
)

// пул соединений
type PostgresReviewRepo struct {
	db *sql.DB
}

func NewPostgresReviewRepo(db *sql.DB) *PostgresReviewRepo {
	return &PostgresReviewRepo{db: db}
}

var _ ReviewRepository = (*PostgresReviewRepo)(nil)

// превратить строку из базы в entities.Review
type rowScanner interface {
	Scan(dest ...any) error
}

func scanReview(row rowScanner) (entities.Review, error) {
	var review entities.Review
	err := row.Scan( //разложить колонки по 8 полям
		&review.ID,
		&review.AuthorID,
		&review.FilmID,
		&review.Title,
		&review.Content,
		&review.ContainsSpoilers,
		&review.CreatedAt,
		&review.UpdatedAt,
	)
	return review, err
}

func (r *PostgresReviewRepo) GetByID(ctx context.Context, id entities.ReviewID) (entities.Review, error) {
	review, err := scanReview(r.db.QueryRowContext(ctx, getReviewQuery, id))
	if errors.Is(err, sql.ErrNoRows) {
		return entities.Review{}, ErrReviewNotFound
	}
	if err != nil {
		return entities.Review{}, fmt.Errorf("get review %d: %w", id, err)
	}
	return review, nil
}

func (r *PostgresReviewRepo) Update(ctx context.Context, id entities.ReviewID, input ReviewInput) (entities.Review, error) {
	review, err := scanReview(r.db.QueryRowContext(ctx, updateReviewQuery, id, input.Title, input.Content, input.ContainsSpoilers))
	if errors.Is(err, sql.ErrNoRows) {
		return entities.Review{}, ErrReviewNotFound
	}
	if err != nil {
		return entities.Review{}, fmt.Errorf("update review %d: %w", id, err)
	}
	return review, nil
}

func (r *PostgresReviewRepo) Delete(ctx context.Context, id entities.ReviewID, deletedBy auth.UserID) error {
	result, err := r.db.ExecContext(ctx, deleteReviewQuery, id, deletedBy)
	if err != nil {
		return fmt.Errorf("delete review %d: %w", id, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete review %d: %w", id, err)
	}
	if affected == 0 {
		return ErrReviewNotFound
	}
	return nil
}

func (r *PostgresReviewRepo) Create(ctx context.Context, authorID auth.UserID, filmID entities.FilmID, input ReviewInput) (entities.Review, error) {
	review, err := scanReview(r.db.QueryRowContext(ctx, createReviewQuery, authorID, filmID, input.Title, input.Content, input.ContainsSpoilers))
	if err == nil {
		return review, nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == uniqueViolation {
			return entities.Review{}, ErrReviewExists
		}
		if pgErr.Code == foreignKeyViolation && pgErr.ConstraintName == "review_film_id_fkey" {
			return entities.Review{}, films.ErrFilmNotFound
		}
	}
	return entities.Review{}, fmt.Errorf("create review: %w", err)
}

func (r *PostgresReviewRepo) ListByFilm(ctx context.Context, filmID entities.FilmID, limit, offset int) ([]entities.Review, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, countReviewsQuery, filmID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count reviews: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, listReviewsQuery, filmID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list reviews: %w", err)
	}
	defer rows.Close() //закрыть курсор и вернуть соединение в пул
	list := make([]entities.Review, 0)
	for rows.Next() {
		review, err := scanReview(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan review: %w", err)
		}
		list = append(list, review)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list reviews: %w", err)
	}

	return list, total, nil
}
