//go:build integration

package files_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filesservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/files"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

func TestPostgresPosterRepoLifecycle(t *testing.T) {
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
		`CREATE TEMP TABLE film (id bigint PRIMARY KEY)`,
		`CREATE TEMP TABLE account (id bigint PRIMARY KEY)`,
		`CREATE TEMP TABLE file (
			id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			storage_key text UNIQUE NOT NULL,
			mime_type text NOT NULL,
			size_bytes bigint NOT NULL CHECK (size_bytes > 0),
			created_by bigint REFERENCES account(id)
		)`,
		`CREATE TEMP TABLE film_file (
			film_id bigint REFERENCES film(id),
			file_id bigint REFERENCES file(id),
			file_role text NOT NULL,
			sort_order smallint NOT NULL CHECK (sort_order > 0),
			created_at timestamptz NOT NULL DEFAULT now(),
			PRIMARY KEY (film_id, file_id),
			UNIQUE (film_id, file_role, sort_order)
		)`,
		`INSERT INTO film VALUES (9)`,
		`INSERT INTO account VALUES (7)`,
		`INSERT INTO file (storage_key, mime_type, size_bytes, created_by) VALUES ('backup.png', 'image/png', 123, 7)`,
		`INSERT INTO film_file SELECT 9, id, 'poster', 2, now() FROM file WHERE storage_key = 'backup.png'`,
	} {
		_, err := db.ExecContext(ctx, query)
		require.NoError(t, err)
	}
	repo, err := filesservice.NewPostgresPosterRepo(db, testFilesLogger(t))
	require.NoError(t, err)
	tests := []struct {
		name           string
		filmID         domain.FilmID
		userID         domain.UserID
		filename       string
		wantMain       string
		wantFiles      int
		wantErr        error
		storageErr     bool
		bindingFailure bool
	}{
		{
			name:      "first main poster preserves backup",
			filmID:    9,
			userID:    7,
			filename:  "first.png",
			wantMain:  "first.png",
			wantFiles: 2,
		},
		{
			name:      "replacement changes main binding without deleting old metadata",
			filmID:    9,
			userID:    7,
			filename:  "second.png",
			wantMain:  "second.png",
			wantFiles: 3,
		},
		{
			name:      "missing film creates no metadata",
			filmID:    999,
			userID:    7,
			filename:  "missing.png",
			wantMain:  "second.png",
			wantFiles: 3,
			wantErr:   domain.ErrFilmNotFound,
		},
		{
			name:       "foreign key failure preserves previous poster",
			filmID:     9,
			userID:     999,
			filename:   "invalid-author.png",
			wantMain:   "second.png",
			wantFiles:  3,
			storageErr: true,
		},
		{
			name:       "duplicate key preserves previous poster",
			filmID:     9,
			userID:     7,
			filename:   "first.png",
			wantMain:   "second.png",
			wantFiles:  3,
			storageErr: true,
		},
		{
			name:           "binding failure rolls back newly inserted metadata",
			filmID:         9,
			userID:         7,
			filename:       "binding-failure.png",
			wantMain:       "second.png",
			wantFiles:      3,
			storageErr:     true,
			bindingFailure: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.bindingFailure {
				var maxFileID int64
				require.NoError(t, db.QueryRowContext(ctx, `SELECT max(id) FROM file`).Scan(&maxFileID))
				_, err := db.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE film_file ADD CONSTRAINT reject_new_poster CHECK (file_id <= %d)`, maxFileID))
				require.NoError(t, err)
				t.Cleanup(func() {
					_, err := db.ExecContext(ctx, `ALTER TABLE film_file DROP CONSTRAINT reject_new_poster`)
					require.NoError(t, err)
				})
			}
			err := repo.SavePoster(ctx, tt.filmID, tt.userID, domain.PosterFile{Filename: tt.filename, ContentType: "image/png", Size: 123})
			switch {
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
			case tt.storageErr:
				require.Error(t, err)
			default:
				require.NoError(t, err)
			}
			var main, backup string
			var fileCount, mainCount int
			err = db.QueryRowContext(ctx, `SELECT f.storage_key FROM film_file ff JOIN file f ON f.id=ff.file_id WHERE ff.film_id=9 AND ff.file_role='poster' AND ff.sort_order=1`).Scan(&main)
			require.NoError(t, err)
			require.Equal(t, tt.wantMain, main)
			err = db.QueryRowContext(ctx, `SELECT f.storage_key FROM film_file ff JOIN file f ON f.id=ff.file_id WHERE ff.film_id=9 AND ff.file_role='poster' AND ff.sort_order=2`).Scan(&backup)
			require.NoError(t, err)
			require.Equal(t, "backup.png", backup)
			require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM file`).Scan(&fileCount))
			require.Equal(t, tt.wantFiles, fileCount)
			require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM film_file WHERE film_id=9 AND file_role='poster' AND sort_order=1`).Scan(&mainCount))
			require.Equal(t, 1, mainCount)
		})
	}
}
