package files

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
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
	log logger.Logger
	db  *sql.DB
}

func NewPostgresPosterRepo(db *sql.DB, appLogger logger.Logger) (*PostgresPosterRepo, error) {
	if db == nil {
		return nil, errors.New("poster database is nil")
	}

	return &PostgresPosterRepo{log: appLogger, db: db}, nil
}

func (r *PostgresPosterRepo) SavePoster(
	ctx context.Context,
	filmID domain.FilmID,
	userID domain.UserID,
	poster domain.PosterFile,
) (operationErr error) {
	operationLog := storage.StartOperation(ctx, r.log, "postgres", "poster.save_poster", map[string]any{"film_id": filmID, "user_id": userID})
	defer func() { operationLog.Finish(operationErr) }()

	operationLog.Stage = "begin"
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin poster transaction: %w", err)
	}
	defer tx.Rollback()

	var existingFilmID domain.FilmID

	operationLog.Stage = "query"
	err = tx.QueryRowContext(ctx, lockPosterFilmQuery, filmID).Scan(&existingFilmID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrFilmNotFound
	}

	if err != nil {
		return fmt.Errorf("lock poster film %d: %w", filmID, err)
	}

	var fileID int64
	operationLog.Stage = "query"
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

	operationLog.Stage = "query"
	if _, err := tx.ExecContext(ctx, setFilmPosterQuery, filmID, fileID); err != nil {
		return fmt.Errorf("set film %d poster: %w", filmID, err)
	}

	operationLog.Stage = "commit"
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit poster transaction: %w: %w", domain.ErrCommitUnknown, err)
	}

	return nil
}

const getFilmPosterQuery = `
 SELECT COALESCE(fl.storage_key, ''), fl.mime_type, COALESCE(fl.size_bytes, 0), fl.external_url
 FROM film AS f
 LEFT JOIN LATERAL (
  SELECT file.* FROM film_file AS ff JOIN file ON file.id = ff.file_id
  WHERE ff.film_id = f.id AND ff.file_role = 'poster'
  ORDER BY ff.sort_order LIMIT 1
 ) AS fl ON true
 WHERE f.id = $1`

const deleteFilmPosterQuery = `DELETE FROM film_file WHERE film_id = $1 AND file_role = 'poster'`

func (r *PostgresPosterRepo) GetPoster(ctx context.Context, filmID domain.FilmID) (_ *domain.PosterFile, operationErr error) {
	operation := storage.StartOperation(ctx, r.log, "postgres", "poster.get_poster", map[string]any{"film_id": filmID})
	defer func() { operation.Finish(operationErr) }()
	var poster domain.PosterFile
	var mime sql.NullString
	err := r.db.QueryRowContext(ctx, getFilmPosterQuery, filmID).Scan(&poster.Filename, &mime, &poster.Size, &poster.ExternalURL)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrFilmNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get film poster: %w", err)
	}
	if !mime.Valid {
		return nil, domain.ErrFileNotFound
	}
	poster.ContentType = mime.String
	return &poster, nil
}

func (r *PostgresPosterRepo) DeletePoster(ctx context.Context, filmID domain.FilmID) (operationErr error) {
	operation := storage.StartOperation(ctx, r.log, "postgres", "poster.delete_poster", map[string]any{"film_id": filmID})
	defer func() { operation.Finish(operationErr) }()
	operation.Stage = "begin"
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin poster deletion: %w", err)
	}
	defer tx.Rollback()
	var id domain.FilmID
	operation.Stage = "query"
	err = tx.QueryRowContext(ctx, lockPosterFilmQuery, filmID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrFilmNotFound
	}
	if err != nil {
		return fmt.Errorf("lock film for poster deletion: %w", err)
	}
	if _, err = tx.ExecContext(ctx, deleteFilmPosterQuery, filmID); err != nil {
		return fmt.Errorf("detach film poster: %w", err)
	}
	operation.Stage = "commit"
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit poster deletion: %w: %w", domain.ErrCommitUnknown, err)
	}
	return nil
}
