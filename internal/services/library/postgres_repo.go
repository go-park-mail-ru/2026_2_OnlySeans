package library

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/jackc/pgx/v5/pgconn"
)

type folder struct {
	folderType string
	title      string
}

var folders = map[domain.List]folder{
	domain.Favorite:   {folderType: "favorite", title: "Избранное"},
	domain.WatchLater: {folderType: "watch_later", title: "Буду смотреть"},
	domain.Watched:    {folderType: "watched", title: "Просмотрено"},
}

const (
	libraryQuery = `
		SELECT f.folder_type, i.film_id
		FROM folder AS f
		JOIN folder_item AS i ON i.folder_id = f.id
		WHERE f.account_id = $1 AND f.folder_type IN ('favorite', 'watch_later', 'watched') AND i.film_id IS NOT NULL
		ORDER BY i.created_at DESC, i.film_id DESC`

	lockAccountQuery = `SELECT id FROM account WHERE id = $1 FOR UPDATE`

	folderQuery = `
		INSERT INTO folder (account_id, title, folder_type) VALUES ($1, $2, $3)
		ON CONFLICT (account_id, folder_type) WHERE folder_type <> 'custom'
		DO UPDATE SET folder_type = EXCLUDED.folder_type
		RETURNING id`

	addFilmQuery = `INSERT INTO folder_item (folder_id, film_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`

	removeFilmQuery = `DELETE FROM folder_item WHERE folder_id = $1 AND film_id = $2`

	removeFromWatchLaterQuery = `
		DELETE FROM folder_item
		WHERE film_id = $2 AND folder_id IN (
			SELECT id FROM folder WHERE account_id = $1 AND folder_type = 'watch_later'
		)`
)

type PostgresRepo struct {
	db  *sql.DB
	log logger.Logger
}

func NewPostgresRepo(db *sql.DB, appLogger logger.Logger) *PostgresRepo {
	return &PostgresRepo{log: appLogger, db: db}
}

func (r *PostgresRepo) Get(ctx context.Context, user domain.UserID) (_ domain.Library, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.log, "postgres", "library.get", map[string]any{"user_id": user})
	defer func() { operationLog.Finish(operationErr) }()

	data := domain.Library{Favorite: []int64{}, WatchLater: []int64{}, Watched: []int64{}}
	operationLog.Stage = "query"
	rows, err := r.db.QueryContext(ctx, libraryQuery, user)
	if err != nil {
		return data, err
	}
	defer rows.Close()

	for rows.Next() {
		var folderType string
		var film int64

		if err := rows.Scan(&folderType, &film); err != nil {
			return data, err
		}

		switch folderType {
		case folders[domain.Favorite].folderType:
			data.Favorite = append(data.Favorite, film)
		case folders[domain.WatchLater].folderType:
			data.WatchLater = append(data.WatchLater, film)
		case folders[domain.Watched].folderType:
			data.Watched = append(data.Watched, film)
		}
	}

	return data, rows.Err()
}

func (r *PostgresRepo) SetSaved(ctx context.Context, user domain.UserID, list domain.List, film int64, saved bool) (err error) {
	operationLog := storage.StartOperation(ctx, r.log, "postgres", "library.set_saved", map[string]any{"user_id": user, "list": list, "film_id": film, "saved": saved})
	defer func() { operationLog.Finish(err) }()

	defer func() {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" && pgErr.ConstraintName == "folder_item_film_id_fkey" {
			err = domain.ErrFilmNotFound
		}
	}()

	target, ok := folders[list]
	if !ok || film < 1 {
		return domain.ErrInvalidLibraryData
	}

	operationLog.Stage = "begin"
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	operationLog.Stage = "query"
	if _, err = tx.ExecContext(ctx, lockAccountQuery, user); err != nil {
		return err
	}

	var folderID int64

	operationLog.Stage = "query"
	if err = tx.QueryRowContext(ctx, folderQuery, user, target.title, target.folderType).Scan(&folderID); err != nil {
		return err
	}

	if !saved {
		operationLog.Stage = "query"
		_, err = tx.ExecContext(ctx, removeFilmQuery, folderID, film)
	} else if _, err = tx.ExecContext(ctx, addFilmQuery, folderID, film); err == nil && list == domain.Watched {
		operationLog.Stage = "query"
		_, err = tx.ExecContext(ctx, removeFromWatchLaterQuery, user, film)
	}

	if err != nil {
		return err
	}

	operationLog.Stage = "commit"
	return tx.Commit()
}
