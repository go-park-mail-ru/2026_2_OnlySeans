package files

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

const (
	lockPosterFilmQuery = `
		SELECT id
		FROM film
		WHERE id = $1
		FOR UPDATE
	`

	insertPosterFileQuery = `
		INSERT INTO file (storage_key, mime_type, size_bytes, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	setFilmPosterQuery = `
		INSERT INTO film_file (film_id, file_id, file_role, sort_order)
		VALUES ($1, $2, 'poster', 1)
		ON CONFLICT (film_id, file_role, sort_order)
		DO UPDATE SET file_id = EXCLUDED.file_id, created_at = now()
	`
)

type PostgresPosterRepo struct {
	db *sql.DB
}

func NewPostgresPosterRepo(db *sql.DB) (*PostgresPosterRepo, error) {
	if db == nil {
		return nil, errors.New("poster database is nil")
	}
	return &PostgresPosterRepo{db: db}, nil
}

func (r *PostgresPosterRepo) SavePoster(
	ctx context.Context,
	filmID domain.FilmID,
	userID domain.UserID,
	poster domain.PosterFile,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin poster transaction: %w", err)
	}
	defer tx.Rollback()

	var existingFilmID domain.FilmID
	err = tx.QueryRowContext(ctx, lockPosterFilmQuery, filmID).Scan(&existingFilmID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrFilmNotFound
	}
	if err != nil {
		return fmt.Errorf("lock poster film %d: %w", filmID, err)
	}

	var fileID int64
	err = tx.QueryRowContext(
		ctx,
		insertPosterFileQuery,
		poster.Filename,
		poster.ContentType,
		poster.Size,
		userID,
	).Scan(&fileID)
	if err != nil {
		return fmt.Errorf("insert poster metadata: %w", err)
	}

	if _, err := tx.ExecContext(ctx, setFilmPosterQuery, filmID, fileID); err != nil {
		return fmt.Errorf("set film %d poster: %w", filmID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit poster transaction: %w: %w", domain.ErrPosterCommitUnknown, err)
	}
	return nil
}

var _ domain.PosterRepository = (*PostgresPosterRepo)(nil)
