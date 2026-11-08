package films_test

import (
	"context"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filmsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/films"
	"github.com/stretchr/testify/require"
)

func TestInMemoryServiceCreateFilm(t *testing.T) {
	service := filmsservice.NewInMemoryService()
	firstID, err := service.CreateFilm(context.Background(), domain.CreateFilmInput{
		Title:       "Alien",
		FilmType:    domain.FilmTypeMovie,
		ReleaseYear: 1979,
		AgeLimit:    16,
	})
	require.NoError(t, err)

	secondID, err := service.CreateFilm(context.Background(), domain.CreateFilmInput{
		Title:       "Blade Runner",
		FilmType:    domain.FilmTypeMovie,
		ReleaseYear: 1982,
		AgeLimit:    16,
	})
	require.NoError(t, err)

	require.Equal(t, domain.FilmID(1), firstID)
	require.Equal(t, domain.FilmID(2), secondID)

	firstFilm, err := service.GetFilmByID(context.Background(), firstID)
	require.NoError(t, err)
	require.Equal(t, "Alien", firstFilm.Title)

	films, total, err := service.ListFilms(context.Background(), 20, 0)
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, films, 2)
}
