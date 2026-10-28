//хранилище отзывов

package reviews

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	uniqueViolation     = "23505" //у автора уже есть отзыв на этот фильм
	foreignKeyViolation = "23503" //фильма с таким номером нет в базе

	reviewColumns = `id, account_id, film_id, title, content, contains_spoilers, version, created_at, updated_at` //список колонок отзыва

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
		UPDATE review
		SET title = $2, content = $3, contains_spoilers = $4, version = version + 1
		WHERE id = $1 AND version = $5 AND deleted_at IS NULL
		RETURNING ` + reviewColumns

	reviewVersionQuery = `
		SELECT version FROM review
		WHERE id = $1 AND deleted_at IS NULL`

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

// превратить строку из базы в domain.Review
type rowScanner interface {
	Scan(dest ...any) error
}

func scanReview(row rowScanner) (domain.Review, error) {
	var review domain.Review
	err := row.Scan( //разложить колонки по полям
		&review.ID,
		&review.AuthorID,
		&review.FilmID,
		&review.Title,
		&review.Content,
		&review.ContainsSpoilers,
		&review.Version,
		&review.CreatedAt,
		&review.UpdatedAt,
	)
	return review, err
}

func (r *PostgresReviewRepo) GetByID(ctx context.Context, id domain.ReviewID) (*domain.Review, error) {
	review, err := scanReview(r.db.QueryRowContext(ctx, getReviewQuery, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrReviewNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get review %d: %w", id, err)
	}
	return &review, nil
}

func (r *PostgresReviewRepo) Update(ctx context.Context, id domain.ReviewID, changes domain.ReviewUpdate) (*domain.Review, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("update review %d: %w", id, err)
	}
	defer tx.Rollback()

	review, err := scanReview(tx.QueryRowContext(ctx, updateReviewQuery,
		id, changes.Input.Title, changes.Input.Content, changes.Input.ContainsSpoilers, changes.Version))

	if errors.Is(err, sql.ErrNoRows) {
		var current int
		switch err := tx.QueryRowContext(ctx, reviewVersionQuery, id).Scan(&current); {
		case errors.Is(err, sql.ErrNoRows):
			return nil, domain.ErrReviewNotFound
		case err != nil:
			return nil, fmt.Errorf("update review %d: %w", id, err)
		default:
			return nil, fmt.Errorf("%w: expected version %d, actual %d",
				domain.ErrVersionConflict, changes.Version, current)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("update review %d: %w", id, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("update review %d: %w", id, err)
	}
	return &review, nil
}

func (r *PostgresReviewRepo) Delete(ctx context.Context, id domain.ReviewID, deletedBy domain.UserID) error {
	result, err := r.db.ExecContext(ctx, deleteReviewQuery, id, deletedBy)
	if err != nil {
		return fmt.Errorf("delete review %d: %w", id, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete review %d: %w", id, err)
	}
	if affected == 0 {
		return domain.ErrReviewNotFound
	}
	return nil
}

func (r *PostgresReviewRepo) Create(ctx context.Context, authorID domain.UserID, filmID domain.FilmID, input domain.ReviewInput) (*domain.Review, error) {
	review, err := scanReview(r.db.QueryRowContext(ctx, createReviewQuery, authorID, filmID, input.Title, input.Content, input.ContainsSpoilers))
	if err == nil {
		return &review, nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == uniqueViolation {
			return nil, domain.ErrReviewExists
		}
		if pgErr.Code == foreignKeyViolation && pgErr.ConstraintName == "review_film_id_fkey" {
			return nil, domain.ErrFilmNotFound
		}
	}
	return nil, fmt.Errorf("create review: %w", err)
}

func (r *PostgresReviewRepo) ListByFilm(ctx context.Context, filmID domain.FilmID, limit, offset int) ([]domain.Review, int, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, 0, fmt.Errorf("list reviews: %w", err)
	}
	defer tx.Rollback()

	var total int
	if err := tx.QueryRowContext(ctx, countReviewsQuery, filmID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count reviews: %w", err)
	}

	rows, err := tx.QueryContext(ctx, listReviewsQuery, filmID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list reviews: %w", err)
	}
	defer rows.Close() //закрыть курсор и вернуть соединение в пул
	list := make([]domain.Review, 0)
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

	if err := tx.Commit(); err != nil {
		return nil, 0, fmt.Errorf("list reviews: %w", err)
	}

	return list, total, nil
}
