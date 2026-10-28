package films

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

const (
	filmColumns = `
		f.id, f.title, f.original_title, f.film_type, f.production_year, f.duration_min, f.age_limit, f.description,
		(SELECT COALESCE(fl.external_url, '/static/' || fl.storage_key)
			FROM film_file AS ff JOIN file AS fl ON fl.id=ff.file_id
			WHERE ff.film_id = f.id AND ff.file_role = 'poster'
			ORDER BY ff.sort_order LIMIT 1
		),
		(SELECT COALESCE(fl.external_url, '/static/' || fl.storage_key)
			FROM film_file AS ff JOIN file AS fl ON fl.id=ff.file_id
			WHERE ff.film_id = f.id AND ff.file_role = 'trailer'
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

	collectionColumns = `
		c.id, c.title, c.slug, c.description,
		(SELECT COALESCE(fl.external_url, '/static/' || fl.storage_key)
		FROM file AS fl WHERE fl.id = c.cover_file_id)
	`
	//одна
	getCollectionQuery = `
		SELECT ` + collectionColumns +
		` FROM collection AS c WHERE c.id = $1 `
	//страница
	listCollectionsQuery = ` SELECT ` + collectionColumns +
		` FROM collection AS c
		  ORDER BY c.id LIMIT $1 OFFSET $2 `
	//сколько
	countCollectionsQuery = `
		SELECT count(*) FROM collection
	`
	//фильмы из одной подборки
	listCollectionFilmsQuery = `
		SELECT ` + filmColumns +
		` FROM collection_film AS cf
		  JOIN film AS f ON f.id = cf.film_id
		  WHERE cf.collection_id = $1
		  ORDER BY cf.position
		  LIMIT $2 OFFSET $3
		`
	//сколько в ней фильмов
	countCollectionFilmsQuery = `
	SELECT count(*) FROM collection_film
	WHERE collection_id = $1
	`
)

type PostgresService struct {
	db *sql.DB
}

func NewPostgresService(db *sql.DB) domain.FilmsRepository {
	return &PostgresService{db: db}
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
	)
	return film, err
}

// метод хранилища, меняет срез, который ей передали, добавляет жанр
func (s *PostgresService) attachGenres(ctx context.Context, films []domain.Film) error {
	if len(films) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(films))                  //номера для $1
	positions := make(map[domain.FilmID]int, len(films)) //номер фльма-индекс в срезе
	for i := range films {
		ids = append(ids, int64(films[i].ID))
		positions[films[i].ID] = i
		films[i].Genres = make([]domain.Genre, 0) //пустой срез чтобы пустой получил не nil
	}
	rows, err := s.db.QueryContext(ctx, filmGenresQuery, ids)
	if err != nil {
		return fmt.Errorf("load genres: %w", err)
	}
	defer rows.Close()
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

func (s *PostgresService) GetFilmByID(ctx context.Context, id domain.FilmID) (domain.Film, error) {
	film, err := scanFilm(s.db.QueryRowContext(ctx, getFilmQuery, id)) //достаем фильм
	//не нашелся
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Film{}, ErrFilmNotFound
	}
	if err != nil {
		return domain.Film{}, fmt.Errorf("get film %d: %w", id, err)
	}
	//добавить жанры
	films := []domain.Film{film}
	if err := s.attachGenres(ctx, films); err != nil {
		return domain.Film{}, err
	}
	return films[0], nil
}

func (s *PostgresService) ListFilms(ctx context.Context, limit, offset int) ([]domain.Film, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, countFilmsQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count films: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, listFilmsQuery, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list films: %w", err)
	}
	defer rows.Close()

	list := make([]domain.Film, 0)
	for rows.Next() {
		film, err := scanFilm(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan film: %w", err)
		}
		list = append(list, film)
	}
	//проверяем чтение
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list films: %w", err)
	}
	//добавляем жанры
	if err := s.attachGenres(ctx, list); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// подборки
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

// страница одной подборки
func (s *PostgresService) GetCollectionByID(ctx context.Context, id domain.CollectionID) (domain.Collection, error) {
	collection, err := scanCollection(s.db.QueryRowContext(ctx, getCollectionQuery, id)) //достаем коллекцию
	//не нашлась
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Collection{}, ErrCollectionNotFound
	}
	if err != nil {
		return domain.Collection{}, fmt.Errorf("get collection %d: %w", id, err)
	}
	return collection, nil
}

// главная страница, список подборок
func (s *PostgresService) ListCollections(ctx context.Context, limit, offset int) ([]domain.Collection, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, countCollectionsQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count collections: %w", err)
	}
	//страница
	rows, err := s.db.QueryContext(ctx, listCollectionsQuery, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list collections: %w", err)
	}
	defer rows.Close()

	list := make([]domain.Collection, 0)
	for rows.Next() {
		collection, err := scanCollection(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan collection: %w", err)
		}
		list = append(list, collection)
	}
	//проверяем чтение
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list collection: %w", err)
	}
	return list, total, nil
}

// фильмы внутри подборки
func (s *PostgresService) ListCollectionFilms(ctx context.Context, collectionID domain.CollectionID, limit, offset int) ([]domain.Film, int, error) {
	//существует ли
	if _, err := s.GetCollectionByID(ctx, collectionID); err != nil {
		return nil, 0, err
	}
	var total int
	if err := s.db.QueryRowContext(ctx, countCollectionFilmsQuery, collectionID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count collection films: %w", err)
	}
	//страница фильмов
	rows, err := s.db.QueryContext(ctx, listCollectionFilmsQuery, collectionID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list collection films: %w", err)
	}
	defer rows.Close()
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
	if err := s.attachGenres(ctx, list); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
