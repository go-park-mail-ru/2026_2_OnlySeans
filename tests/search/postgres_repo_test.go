package search_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	searchservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/search"
)

type posterStorageFunc func(context.Context, string) (string, error)

func (f posterStorageFunc) GetURL(ctx context.Context, key string) (string, error) {
	return f(ctx, key)
}

func TestSearchPostgresFilmPosters(t *testing.T) {
	storageErr := errors.New("storage unavailable")

	cases := []struct {
		name string
		err  error
		want string
	}{
		{"signed URL", nil, "https://storage.example/poster?signature=abc"},
		{"missing poster", domain.ErrFileNotFound, ""},
		{"storage failure", storageErr, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}

			defer func() {
				mock.ExpectClose()

				if err := db.Close(); err != nil {
					t.Error(err)
				}
			}()
			mock.ExpectQuery("WITH matches AS").WithArgs("matrix", 3, "matrix%", "%matrix%").
				WillReturnRows(sqlmock.NewRows([]string{"id", "title", "original_title", "production_year", "film_type", "external_url", "storage_key"}).
					AddRow(1, "Матрица", "The Matrix", 1999, "movie", nil, "posters/matrix.jpg")).RowsWillBeClosed()

			repo, err := searchservice.NewPostgresRepository(db, posterStorageFunc(func(_ context.Context, key string) (string, error) {
				if key != "posters/matrix.jpg" {
					t.Fatalf("unexpected storage key: %s", key)
				}

				return tc.want, tc.err
			}))
			if err != nil {
				t.Fatal(err)
			}

			films, err := repo.SearchFilms(context.Background(), "matrix", 3)
			if errors.Is(tc.err, storageErr) {
				if !errors.Is(err, storageErr) {
					t.Fatalf("error = %v, want storage failure", err)
				}
			} else {
				if err != nil || len(films) != 1 {
					t.Fatalf("films/error = %v/%v", films, err)
				}

				if tc.want == "" {
					if films[0].PosterURL != nil {
						t.Fatal("missing object must produce a null poster")
					}
				} else if films[0].PosterURL == nil || *films[0].PosterURL != tc.want {
					t.Fatal("signed URL was not attached")
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSearchPostgresEscapesPatternsAndChecksRowErrors(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		mock.ExpectClose()

		if err := db.Close(); err != nil {
			t.Error(err)
		}
	}()

	repo, err := searchservice.NewPostgresRepository(db, posterStorageFunc(func(context.Context, string) (string, error) {
		t.Fatal("empty result must not access poster storage")
		return "", nil
	}))
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery("WITH matches AS").WithArgs("!%_", 3, "!!!%!_%", "%!!!%!_%").
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "original_title", "production_year", "film_type", "external_url", "storage_key"}))

	films, err := repo.SearchFilms(context.Background(), "!%_", 3)
	if err != nil || films == nil || len(films) != 0 {
		t.Fatalf("empty films/error = %v/%v", films, err)
	}

	rowErr := errors.New("connection lost while reading")
	mock.ExpectQuery("SELECT p.id").WithArgs("Nolan", 3, "Nolan%", "%Nolan%").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "photo_url"}).AddRow(7, "Нолан", nil).RowError(0, rowErr))

	if _, err := repo.SearchPersons(context.Background(), "Nolan", 3); !errors.Is(err, rowErr) {
		t.Fatalf("error = %v, want row error", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
