package search

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type PosterURLStorage interface {
	GetURL(context.Context, string) (string, error)
}

type PostgresRepository struct {
	db      *sql.DB
	posters PosterURLStorage
}

var _ domain.SearchRepository = (*PostgresRepository)(nil)

func NewPostgresRepository(db *sql.DB, posters PosterURLStorage) (*PostgresRepository, error) {
	if db == nil {
		return nil, errors.New("search database is nil")
	}

	if posters == nil {
		return nil, errors.New("search poster storage is nil")
	}

	return &PostgresRepository{db: db, posters: posters}, nil
}

const searchFilmsQuery = `
WITH matches AS (
    SELECT f.id, f.title, f.original_title, f.production_year, f.film_type,
        CASE
            WHEN lower(f.title) = lower($1) OR lower(f.original_title) = lower($1) THEN 0
            WHEN lower(f.title) LIKE lower($3) ESCAPE '!' OR lower(f.original_title) LIKE lower($3) ESCAPE '!' THEN 1
            WHEN lower(f.title) LIKE lower($4) ESCAPE '!' OR lower(f.original_title) LIKE lower($4) ESCAPE '!' THEN 2
            ELSE 3
        END AS match_kind,
        GREATEST(word_similarity(lower($1), lower(f.title)),
                 word_similarity(lower($1), lower(COALESCE(f.original_title, '')))) AS score
    FROM film AS f
    WHERE lower(f.title) LIKE lower($4) ESCAPE '!'
       OR lower(f.original_title) LIKE lower($4) ESCAPE '!'
       OR lower(f.title) % lower($1) OR lower(f.original_title) % lower($1)
       OR lower($1) <% lower(f.title) OR lower($1) <% lower(f.original_title)
    ORDER BY match_kind, score DESC, f.id
    LIMIT $2
)
SELECT m.id, m.title, m.original_title, m.production_year, m.film_type,
       poster.external_url, poster.storage_key
FROM matches AS m
LEFT JOIN LATERAL (
    SELECT fl.external_url, fl.storage_key
    FROM film_file AS ff JOIN file AS fl ON fl.id = ff.file_id
    WHERE ff.film_id = m.id AND ff.file_role = 'poster'
    ORDER BY ff.sort_order, ff.file_id LIMIT 1
) AS poster ON true
ORDER BY m.match_kind, m.score DESC, m.id`

const searchPersonsQuery = `
SELECT p.id, btrim(p.first_name || ' ' || p.last_name),
       COALESCE(photo.external_url, '/static/' || photo.storage_key)
FROM person AS p
LEFT JOIN file AS photo ON photo.id = p.photo_file_id
WHERE (
       lower(p.first_name || ' ' || p.last_name) LIKE lower($4) ESCAPE '!'
    OR lower(p.original_name) LIKE lower($4) ESCAPE '!'
    OR lower(p.first_name || ' ' || p.last_name) % lower($1)
    OR lower(p.original_name) % lower($1)
    OR lower($1) <% lower(p.first_name || ' ' || p.last_name)
    OR lower($1) <% lower(p.original_name)
)
AND EXISTS (
    SELECT 1 FROM film_person AS fp
    JOIN role_type AS rt ON rt.id = fp.role_type_id
    WHERE fp.person_id = p.id AND rt.code IN ('actor', 'director')
)
ORDER BY
    CASE
        WHEN lower(p.first_name || ' ' || p.last_name) = lower($1) OR lower(p.original_name) = lower($1) THEN 0
        WHEN lower(p.first_name || ' ' || p.last_name) LIKE lower($3) ESCAPE '!'
          OR lower(p.original_name) LIKE lower($3) ESCAPE '!' THEN 1
        WHEN lower(p.first_name || ' ' || p.last_name) LIKE lower($4) ESCAPE '!'
          OR lower(p.original_name) LIKE lower($4) ESCAPE '!' THEN 2
        ELSE 3
    END,
    GREATEST(word_similarity(lower($1), lower(p.first_name || ' ' || p.last_name)),
             word_similarity(lower($1), lower(COALESCE(p.original_name, '')))) DESC,
    p.id
LIMIT $2`

// Escape LIKE metacharacters so user input is always searched literally.
func searchPatterns(query string) (prefix, contains string) {
	escaped := strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(query)
	return escaped + "%", "%" + escaped + "%"
}

func (r *PostgresRepository) SearchFilms(ctx context.Context, query string, limit int) ([]domain.SearchFilm, error) {
	prefix, contains := searchPatterns(query)
	films := make([]domain.SearchFilm, 0)
	// Close the database cursor before requesting signed URLs from object storage.
	if err := func() (operationErr error) {
		rows, err := r.db.QueryContext(ctx, searchFilmsQuery, query, limit, prefix, contains)
		if err != nil {
			return err
		}
		defer func() { operationErr = errors.Join(operationErr, rows.Close()) }()
		for rows.Next() {
			var film domain.SearchFilm
			if err := rows.Scan(&film.ID, &film.Title, &film.OriginalTitle, &film.ReleaseYear,
				&film.FilmType, &film.PosterURL, &film.PosterStorageKey); err != nil {
				return err
			}
			films = append(films, film)
		}
		return rows.Err()
	}(); err != nil {
		return nil, fmt.Errorf("search films: %w", err)
	}

	for i := range films {
		film := &films[i]
		if film.PosterURL != nil || film.PosterStorageKey == nil {
			continue
		}

		url, err := r.posters.GetURL(ctx, *film.PosterStorageKey)
		if errors.Is(err, domain.ErrFileNotFound) {
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("get search film poster: %w", err)
		}

		film.PosterURL = &url
	}

	return films, nil
}

func (r *PostgresRepository) SearchPersons(ctx context.Context, query string, limit int) (_ []domain.SearchPerson, operationErr error) {
	prefix, contains := searchPatterns(query)

	rows, err := r.db.QueryContext(ctx, searchPersonsQuery, query, limit, prefix, contains)
	if err != nil {
		return nil, fmt.Errorf("search persons: %w", err)
	}

	defer func() { operationErr = errors.Join(operationErr, rows.Close()) }()

	persons := make([]domain.SearchPerson, 0)

	for rows.Next() {
		var person domain.SearchPerson
		if err := rows.Scan(&person.ID, &person.Name, &person.PhotoURL); err != nil {
			return nil, fmt.Errorf("scan search person: %w", err)
		}

		persons = append(persons, person)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read search persons: %w", err)
	}

	return persons, nil
}
