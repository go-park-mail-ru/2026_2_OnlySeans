package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/storage"
)

const (
	collectionColumns = `
		c.id, c.title, c.slug, c.description,
		(SELECT COALESCE(fl.external_url, '/static/' || fl.storage_key)
		FROM file AS fl WHERE fl.id = c.cover_file_id)
	`

	getCollectionQuery = `
		SELECT ` + collectionColumns +
		` FROM collection AS c WHERE c.id = $1 `

	listCollectionsQuery = ` SELECT ` + collectionColumns +
		` FROM collection AS c
		  ORDER BY c.id LIMIT $1 OFFSET $2 `

	countCollectionsQuery = `
		SELECT count(*) FROM collection
	`

	listCollectionFilmsQuery = `
		SELECT ` + filmColumns +
		` FROM collection_film AS cf
		  JOIN film AS f ON f.id = cf.film_id
		  WHERE cf.collection_id = $1
		  ORDER BY cf.position
		  LIMIT $2 OFFSET $3
		`

	countCollectionFilmsQuery = `
	SELECT count(*) FROM collection_film
	WHERE collection_id = $1
	`
)

func scanCollection(row rowScanner) (domain.Collection, error) {
	var collection domain.Collection
	err := row.Scan(
		&collection.ID,
		&collection.Title,
		&collection.Slug,
		&collection.Description,
		&collection.CoverURL,
	)

	return collection, err
}

func (s *FilmRepo) GetCollectionByID(ctx context.Context, id domain.CollectionID) (_ domain.Collection, operationErr error) {
	operationLog := storage.StartOperation(ctx, s.appLogger, "postgres", "film.get_collection_by_id", map[string]any{
		"collection_id": id,
	})
	defer func() { operationLog.Finish(operationErr) }()

	operationLog.Stage = "query"
	collection, err := scanCollection(s.db.QueryRowContext(ctx, getCollectionQuery, id))

	if errors.Is(err, sql.ErrNoRows) {
		return domain.Collection{}, domain.ErrCollectionNotFound
	}

	if err != nil {
		return domain.Collection{}, fmt.Errorf("get collection %d: %w", id, err)
	}

	return collection, nil
}

func (s *FilmRepo) ListCollections(ctx context.Context, limit, offset int) (_ []domain.Collection, _ int, operationErr error) {
	operationLog := storage.StartOperation(ctx, s.appLogger, "postgres", "film.list_collections", map[string]any{
		"limit":  limit,
		"offset": offset,
	})
	defer func() { operationLog.Finish(operationErr) }()

	operationLog.Stage = "begin"

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list collections: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	var total int
	operationLog.Stage = "query"

	if err := tx.QueryRowContext(ctx, countCollectionsQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count collections: %w", err)
	}

	operationLog.Stage = "query"

	rows, err := tx.QueryContext(ctx, listCollectionsQuery, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list collections: %w", err)
	}

	defer func() { _ = rows.Close() }()

	list := make([]domain.Collection, 0)

	for rows.Next() {
		collection, err := scanCollection(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan collection: %w", err)
		}

		list = append(list, collection)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list collection: %w", err)
	}

	operationLog.Stage = "commit"

	if err := tx.Commit(); err != nil {
		return nil, 0, fmt.Errorf("list collections: %w", err)
	}

	return list, total, nil
}

func (s *FilmRepo) ListCollectionFilms(ctx context.Context, collectionID domain.CollectionID, limit, offset int) (_ []domain.Film, _ int, operationErr error) {
	operationLog := storage.StartOperation(ctx, s.appLogger, "postgres", "film.list_collection_films", map[string]any{
		"collection_id": collectionID,
		"limit":         limit,
		"offset":        offset,
	})
	defer func() { operationLog.Finish(operationErr) }()

	if _, err := s.GetCollectionByID(ctx, collectionID); err != nil {
		return nil, 0, err
	}

	operationLog.Stage = "begin"

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list collection films: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	var total int
	operationLog.Stage = "query"

	if err := tx.QueryRowContext(ctx, countCollectionFilmsQuery, collectionID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count collection films: %w", err)
	}

	operationLog.Stage = "query"

	rows, err := tx.QueryContext(ctx, listCollectionFilmsQuery, collectionID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list collection films: %w", err)
	}

	defer func() { _ = rows.Close() }()

	list := make([]domain.Film, 0)

	for rows.Next() {
		film, err := scanFilm(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan film: %w", err)
		}

		list = append(list, film)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list collection films: %w", err)
	}

	if err := attachGenresTx(ctx, tx, list, s.appLogger); err != nil {
		return nil, 0, err
	}

	operationLog.Stage = "commit"

	if err := tx.Commit(); err != nil {
		return nil, 0, fmt.Errorf("list collection films: %w", err)
	}

	return list, total, nil
}
