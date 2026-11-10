package films_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresServiceCreateFilmOptionalFieldsAndSQLPayload(t *testing.T) {
	repo, mock := newPostgresService(t)
	input := domain.CreateFilmInput{
		Title:         "'; DROP TABLE film; --",
		OriginalTitle: ptr("Alien"), FilmType: domain.FilmTypeMovie,
		ReleaseYear: 1979, DurationMin: ptr[int16](117), AgeLimit: 16,
		Description: ptr("Science fiction"),
	}
	mock.ExpectQuery(`INSERT INTO film \(title, original_title, film_type, production_year, duration_min, age_limit, description\)\s+VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\)\s+RETURNING id`).
		WithArgs(input.Title, *input.OriginalTitle, string(input.FilmType), input.ReleaseYear,
			*input.DurationMin, input.AgeLimit, *input.Description).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))

	id, err := repo.CreateFilm(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, domain.FilmID(42), id)
}

func TestPostgresServiceCreateFilmScanError(t *testing.T) {
	repo, mock := newPostgresService(t)
	mock.ExpectQuery("INSERT INTO film").
		WithArgs("Alien", nil, "movie", int16(1979), nil, int16(16), nil).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("not-an-integer"))

	id, err := repo.CreateFilm(context.Background(), domain.CreateFilmInput{
		Title: "Alien", FilmType: domain.FilmTypeMovie, ReleaseYear: 1979, AgeLimit: 16,
	})

	require.Error(t, err)
	assert.Equal(t, domain.FilmID(0), id)
	assert.Contains(t, err.Error(), "create film:")
}
