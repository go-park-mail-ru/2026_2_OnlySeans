package films_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	filmsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filmsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
	"github.com/stretchr/testify/require"
)

var errListGenres = errors.New("list genres failed")

type genreRepositoryStub struct {
	genres []domain.Genre
	err    error
}

func (stub genreRepositoryStub) ListGenres(_ context.Context) ([]domain.Genre, error) {
	return stub.genres, stub.err
}

func TestGenresHandler_ListGenres(t *testing.T) {
	tests := []struct {
		name     string
		genres   []domain.Genre
		wantBody string
	}{
		{
			name:     "genres",
			genres:   []domain.Genre{{ID: 1, Name: "драма", Slug: "drama"}},
			wantBody: `{"genres":[{"id":1,"name":"драма","slug":"drama"}]}`,
		},
		{
			name:     "empty",
			genres:   []domain.Genre{},
			wantBody: `{"genres":[]}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := newGenresTestHandler(t, genreRepositoryStub{genres: test.genres})
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/genres", nil)

			handler.ListGenres(response, request)

			require.Equal(t, http.StatusOK, response.Code)
			require.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
			require.JSONEq(t, test.wantBody, response.Body.String())
		})
	}
}

func TestGenresHandler_ListGenresError(t *testing.T) {
	handler := newGenresTestHandler(t, genreRepositoryStub{err: errListGenres})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/genres", nil)

	handler.ListGenres(response, request)

	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.JSONEq(t, `{"error":"internal server error"}`, response.Body.String())
}

func TestNewGenresHandlerRejectsNilDependencies(t *testing.T) {
	appLogger := testutil.DiscardLogger()
	listGenres, err := filmsusecase.NewListGenresUseCase(genreRepositoryStub{}, appLogger)
	require.NoError(t, err)

	handler, err := filmsdelivery.NewGenresHandler(nil, appLogger)
	require.Nil(t, handler)
	require.EqualError(t, err, "list genres use case is nil")

	handler, err = filmsdelivery.NewGenresHandler(listGenres, nil)
	require.Nil(t, handler)
	require.EqualError(t, err, "logger is nil")
}

func TestNewListGenresUseCaseRejectsNilDependencies(t *testing.T) {
	listGenres, err := filmsusecase.NewListGenresUseCase(nil, testutil.DiscardLogger())
	require.Nil(t, listGenres)
	require.ErrorIs(t, err, domain.ErrNilGenresRepo)

	listGenres, err = filmsusecase.NewListGenresUseCase(genreRepositoryStub{}, nil)
	require.Nil(t, listGenres)
	require.ErrorIs(t, err, domain.ErrInvalidAppLogger)
}

func newGenresTestHandler(t *testing.T, repository domain.GenreRepository) *filmsdelivery.GenresHandler {
	t.Helper()
	appLogger := testutil.DiscardLogger()
	listGenres, err := filmsusecase.NewListGenresUseCase(repository, appLogger)
	require.NoError(t, err)
	handler, err := filmsdelivery.NewGenresHandler(listGenres, appLogger)
	require.NoError(t, err)
	return handler
}
