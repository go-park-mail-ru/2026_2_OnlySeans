package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/storage"
)

const (
	filmColumns = `
		f.id, f.title, f.original_title, f.film_type, f.production_year, f.duration_min, f.age_limit, f.description,
		(SELECT fl.external_url
			FROM film_file AS ff JOIN file AS fl ON fl.id=ff.file_id
			WHERE ff.film_id = f.id AND ff.file_role = 'poster'
			ORDER BY ff.sort_order LIMIT 1
		),
		(SELECT COALESCE(fl.external_url, '/static/' || fl.storage_key)
			FROM film_file AS ff JOIN file AS fl ON fl.id=ff.file_id
			WHERE ff.film_id = f.id AND ff.file_role = 'trailer'
			ORDER BY ff.sort_order LIMIT 1
		),
		(SELECT CASE WHEN fl.external_url IS NULL THEN fl.storage_key END
			FROM film_file AS ff JOIN file AS fl ON fl.id=ff.file_id
			WHERE ff.film_id = f.id AND ff.file_role = 'poster'
			ORDER BY ff.sort_order LIMIT 1
		)`

	getFilmQuery = `
		SELECT ` + filmColumns +
		` FROM film AS f 
		 WHERE f.id = $1`

	listFilmsQuery = `
		SELECT ` + filmColumns +
		` FROM film AS f 
		ORDER BY f.id
		LIMIT $1 OFFSET $2`

	countFilmsQuery = `
		SELECT count(*) FROM film`

	filmGenresQuery = `
		SELECT fg.film_id, g.id, g.name, g.slug
		FROM film_genre AS fg
		JOIN genre AS g ON g.id = fg.genre_id
		WHERE fg.film_id = ANY($1)
		ORDER BY fg.film_id, g.id
	`

	createFilmQuery = `
		INSERT INTO film (title, original_title, film_type, production_year, duration_min, age_limit, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	updateFilmQuery = `
		UPDATE film SET title = $2, original_title = $3, film_type = $4, production_year = $5, duration_min = $6, age_limit = $7, description = $8
		WHERE id = $1
	`

	deleteFilmQuery = `
		DELETE FROM film
		WHERE id = $1
	`
)

type FilmRepo struct {
	operations storage.OperationRecorder
	db         *sql.DB
}

func NewFilmRepo(db *sql.DB, appLogger logger.Logger, observers ...storage.ExternalObserver) *FilmRepo {
	return &FilmRepo{
		operations: storage.NewOperationRecorder(appLogger, observers...),
		db:         db,
	}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanFilm(row rowScanner) (domain.Film, error) {
	var film domain.Film
	err := row.Scan(
		&film.ID, &film.Title,
		&film.OriginalTitle,
		&film.FilmType,
		&film.ReleaseYear,
		&film.DurationMin,
		&film.AgeLimit,
		&film.Description,
		&film.PosterURL,
		&film.TrailerURL,
		&film.PosterStorageKey,
	)

	return film, err
}

type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func attachGenresTx(ctx context.Context, q querier, films []domain.Film, operations storage.OperationRecorder) (operationErr error) {
	if len(films) == 0 {
		return nil
	}

	operationLog := operations.Start(ctx, "postgres", "film.load_genres", map[string]any{
		"film_count": len(films),
	})
	defer func() { operationLog.Finish(operationErr) }()

	operationLog.Stage = "query"

	ids := make([]int64, 0, len(films))
	positions := make(map[domain.FilmID]int, len(films))

	for i := range films {
		ids = append(ids, int64(films[i].ID))
		positions[films[i].ID] = i
		films[i].Genres = make([]domain.Genre, 0)
	}

	rows, err := q.QueryContext(ctx, filmGenresQuery, ids)
	if err != nil {
		return fmt.Errorf("load genres: %w", err)
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var filmID domain.FilmID

		var genre domain.Genre
		if err := rows.Scan(&filmID, &genre.ID, &genre.Name, &genre.Slug); err != nil {
			return fmt.Errorf("scan genre: %w", err)
		}

		i := positions[filmID]
		films[i].Genres = append(films[i].Genres, genre)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("load genres: %w", err)
	}

	return nil
}

func (s *FilmRepo) GetFilmByID(ctx context.Context, id domain.FilmID) (_ domain.Film, operationErr error) {
	operationLog := s.operations.Start(ctx, "postgres", "film.get_film_by_id", map[string]any{
		"film_id": id,
	})
	defer func() { operationLog.Finish(operationErr) }()

	operationLog.Stage = "query"
	film, err := scanFilm(s.db.QueryRowContext(ctx, getFilmQuery, id))

	if errors.Is(err, sql.ErrNoRows) {
		operationLog.NotFound = true
		return domain.Film{}, domain.ErrFilmNotFound
	}

	if err != nil {
		return domain.Film{}, fmt.Errorf("get film %d: %w", id, err)
	}

	films := []domain.Film{film}
	if err := attachGenresTx(ctx, s.db, films, s.operations); err != nil {
		return domain.Film{}, err
	}

	return films[0], nil
}

func (s *FilmRepo) CreateFilm(ctx context.Context, input domain.CreateFilmInput) (_ domain.FilmID, operationErr error) {
	operationLog := s.operations.Start(ctx, "postgres", "film.create_film", map[string]any{})
	defer func() { operationLog.Finish(operationErr) }()

	var id domain.FilmID
	operationLog.Stage = "query"

	err := s.db.QueryRowContext(ctx, createFilmQuery,
		input.Title,
		input.OriginalTitle,
		input.FilmType,
		input.ReleaseYear,
		input.DurationMin,
		input.AgeLimit,
		input.Description,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create film: %w", err)
	}

	return id, nil
}

func (s *FilmRepo) UpdateFilm(ctx context.Context, id domain.FilmID, input domain.CreateFilmInput) (operationErr error) {
	operationLog := s.operations.Start(ctx, "postgres", "film.update_film", map[string]any{
		"film_id": id,
	})
	defer func() { operationLog.Finish(operationErr) }()

	operationLog.Stage = "query"

	result, err := s.db.ExecContext(ctx, updateFilmQuery, id, input.Title, input.OriginalTitle, input.FilmType, input.ReleaseYear, input.DurationMin, input.AgeLimit, input.Description)
	if err != nil {
		return fmt.Errorf("update film %d: %w", id, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update film %d: %w", id, err)
	}

	if affected == 0 {
		operationLog.NotFound = true
		return domain.ErrFilmNotFound
	}

	return nil
}

func (s *FilmRepo) DeleteFilm(ctx context.Context, id domain.FilmID) (operationErr error) {
	operationLog := s.operations.Start(ctx, "postgres", "film.delete_film", map[string]any{
		"film_id": id,
	})
	defer func() { operationLog.Finish(operationErr) }()

	operationLog.Stage = "query"

	result, err := s.db.ExecContext(ctx, deleteFilmQuery, id)
	if err != nil {
		return fmt.Errorf("delete film %d: %w", id, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete film %d: %w", id, err)
	}

	if affected == 0 {
		operationLog.NotFound = true
		return domain.ErrFilmNotFound
	}

	return nil
}

func (s *FilmRepo) ListFilms(ctx context.Context, limit, offset int) (_ []domain.Film, _ int, operationErr error) {
	operationLog := s.operations.Start(ctx, "postgres", "film.list_films", map[string]any{
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
		return nil, 0, fmt.Errorf("list films: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	var total int
	operationLog.Stage = "query"

	if err := tx.QueryRowContext(ctx, countFilmsQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count films: %w", err)
	}

	operationLog.Stage = "query"

	rows, err := tx.QueryContext(ctx, listFilmsQuery, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list films: %w", err)
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
		return nil, 0, fmt.Errorf("list films: %w", err)
	}

	if err := attachGenresTx(ctx, tx, list, s.operations); err != nil {
		return nil, 0, err
	}

	operationLog.Stage = "commit"

	if err := tx.Commit(); err != nil {
		return nil, 0, fmt.Errorf("list films: %w", err)
	}

	return list, total, nil
}
