//go:build integration

package films_test

import (
	"context"
	"database/sql"
	"io"
	"os"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filmsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

type integrationPosterSigner struct{}

func (integrationPosterSigner) GetURL(_ context.Context, key string) (string, error) {
	return "https://media.example.org/posters/" + key + "?signed=test", nil
}

func TestPostgresPosterURLSelection(t *testing.T) {
	dsn := os.Getenv("POSTERS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set POSTERS_TEST_DATABASE_URL for the PostgreSQL integration test")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	for _, query := range []string{
		`CREATE TEMP TABLE film (id bigint PRIMARY KEY, title text, original_title text, film_type text, production_year smallint, duration_min smallint, age_limit smallint, description text)`,
		`CREATE TEMP TABLE file (id bigint PRIMARY KEY, storage_key text, external_url text)`,
		`CREATE TEMP TABLE film_file (film_id bigint, file_id bigint, file_role text, sort_order smallint)`,
		`CREATE TEMP TABLE genre (id bigint PRIMARY KEY, name text, slug text)`,
		`CREATE TEMP TABLE film_genre (film_id bigint, genre_id bigint)`,
		`CREATE TEMP TABLE collection_film (collection_id bigint, film_id bigint, position smallint)`,
		`INSERT INTO film VALUES (9, 'Film', NULL, 'movie', 2020, NULL, 0, NULL)`,
		`INSERT INTO collection_film VALUES (5, 9, 1)`,
	} {
		_, err := db.ExecContext(ctx, query)
		require.NoError(t, err)
	}
	appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelDebug, "json")
	require.NoError(t, err)
	repo, err := filmsservice.NewRepositoryWithPosterURLs(filmsservice.NewPostgresService(db), integrationPosterSigner{}, appLogger)
	require.NoError(t, err)
	tests := []struct {
		name  string
		setup string
		want  *string
	}{
		{
			name: "no poster",
		},
		{
			name:  "main S3 poster chosen over backup",
			setup: `INSERT INTO file VALUES (1, 'main.png', NULL), (2, 'backup.png', NULL); INSERT INTO film_file VALUES (9, 2, 'poster', 2), (9, 1, 'poster', 1)`,
			want:  ptr("https://media.example.org/posters/main.png?signed=test"),
		},
		{
			name:  "external URL takes priority over its storage key",
			setup: `UPDATE file SET external_url = 'https://images.example.org/main.png' WHERE id = 1`,
			want:  ptr("https://images.example.org/main.png"),
		},
		{
			name:  "replacement is visible on next read",
			setup: `INSERT INTO file VALUES (3, 'replacement.png', NULL); UPDATE film_file SET file_id = 3 WHERE file_role = 'poster' AND sort_order = 1`,
			want:  ptr("https://media.example.org/posters/replacement.png?signed=test"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != "" {
				_, err := db.ExecContext(ctx, tt.setup)
				require.NoError(t, err)
			}
			film, err := repo.GetFilmByID(ctx, domain.FilmID(9))
			require.NoError(t, err)
			require.Equal(t, tt.want, film.PosterURL)
			list, total, err := repo.ListFilms(ctx, 20, 0)
			require.NoError(t, err)
			require.Equal(t, 1, total)
			require.Equal(t, tt.want, list[0].PosterURL)
			collection, total, err := repo.ListCollectionFilms(ctx, 5, 20, 0)
			require.NoError(t, err)
			require.Equal(t, 1, total)
			require.Equal(t, tt.want, collection[0].PosterURL)
		})
	}
}
