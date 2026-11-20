//go:build integration

package search_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"

	searchservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/search"
)

func TestSearchPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("SEARCH_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set SEARCH_TEST_DATABASE_URL to a disposable PostgreSQL database with pg_trgm available")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)

	defer func() { require.NoError(t, db.Close()) }()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	// Session-local tables isolate fixtures from any application tables.
	_, err = db.ExecContext(ctx, `
CREATE TEMP TABLE film (id bigint PRIMARY KEY, title text NOT NULL, original_title text, production_year smallint, film_type text);
CREATE TEMP TABLE file (id bigint PRIMARY KEY, external_url text, storage_key text);
CREATE TEMP TABLE film_file (film_id bigint, file_id bigint, file_role text, sort_order smallint);
CREATE TEMP TABLE person (id bigint PRIMARY KEY, first_name text NOT NULL, last_name text NOT NULL, photo_file_id bigint);
CREATE TEMP TABLE role_type (id bigint PRIMARY KEY, code text);
CREATE TEMP TABLE film_person (person_id bigint, role_type_id bigint);
`)
	require.NoError(t, err)
	migration, err := os.ReadFile("../../db/migrations/000016_search.up.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(migration))
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
INSERT INTO film VALUES
 (1, 'Матрица', 'The Matrix', 1999, 'movie'),
 (2, 'Матрица: Перезагрузка', 'The Matrix Reloaded', 2003, 'movie'),
 (3, 'Начало', 'Inception', 2010, 'movie'),
 (4, 'Иван Васильевич меняет профессию', NULL, 1973, 'movie');
INSERT INTO file VALUES (1, 'https://example.com/poster.jpg', NULL), (2, 'https://example.com/person.jpg', NULL);
INSERT INTO film_file VALUES (1, 1, 'poster', 1);
INSERT INTO person VALUES (7, 'Кристофер', 'Нолан', 2, 'Christopher Nolan'), (8, 'Тестовый', 'Сценарист', NULL, 'Only Writer');
INSERT INTO role_type VALUES (1, 'actor'), (2, 'director'), (3, 'writer');
INSERT INTO film_person VALUES (7, 1), (7, 2), (7, 2), (8, 3);
`)
	require.NoError(t, err)
	repo, err := searchservice.NewPostgresRepository(db, posterStorageFunc(func(context.Context, string) (string, error) {
		t.Fatal("external posters must not use object storage")
		return "", nil
	}))
	require.NoError(t, err)

	for _, query := range []string{"МАТРИЦА", "матреца", "The Matrix"} {
		t.Run(query, func(t *testing.T) {
			films, err := repo.SearchFilms(ctx, query, 1)
			require.NoError(t, err)
			require.Len(t, films, 1)
			require.EqualValues(t, 1, films[0].ID)
			require.NotNil(t, films[0].PosterURL)
		})
	}

	films, err := repo.SearchFilms(ctx, "Васильевич", 3)
	require.NoError(t, err)
	require.Len(t, films, 1)
	require.EqualValues(t, 4, films[0].ID)
	require.Nil(t, films[0].OriginalTitle)
	require.Nil(t, films[0].PosterURL)
	films, err = repo.SearchFilms(ctx, "zzzzzz", 3)
	require.NoError(t, err)
	require.Empty(t, films)
	films, err = repo.SearchFilms(ctx, "%%", 3)
	require.NoError(t, err)
	require.Empty(t, films, "LIKE metacharacters must be literal")

	for _, query := range []string{"Нолан", "Nolan", "Nolann"} {
		persons, err := repo.SearchPersons(ctx, query, 5)
		require.NoError(t, err)
		require.Len(t, persons, 1, "multiple credits must not duplicate a person")
		require.EqualValues(t, 7, persons[0].ID)
	}

	persons, err := repo.SearchPersons(ctx, "Only Writer", 5)
	require.NoError(t, err)
	require.Empty(t, persons, "only actors and directors belong in suggestions")

	down, err := os.ReadFile("../../db/migrations/000016_search.down.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(down))
	require.NoError(t, err)
}
