package films_test

import (
	"context"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filmsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateFilmUseCaseRun(t *testing.T) {
	repo := mocks.NewMockFilmRepository(gomock.NewController(t))
	uc, err := filmsusecase.NewCreateFilmUseCase(repo, newFilmsTestLogger())
	require.NoError(t, err)

	input := domain.CreateFilmInput{
		Title:       "  Alien  ",
		FilmType:    domain.FilmTypeMovie,
		ReleaseYear: 1979,
		AgeLimit:    16,
	}

	expected := input
	expected.Title = "Alien"
	repo.EXPECT().CreateFilm(gomock.Any(), expected).Return(domain.FilmID(42), nil)
	id, err := uc.Run(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, domain.FilmID(42), id)
}

func TestCreateFilmUseCaseRunInvalidInput(t *testing.T) {
	repo := mocks.NewMockFilmRepository(gomock.NewController(t))
	uc, err := filmsusecase.NewCreateFilmUseCase(repo, newFilmsTestLogger())
	require.NoError(t, err)

	input := domain.CreateFilmInput{
		Title:       "   ",
		FilmType:    domain.FilmTypeMovie,
		ReleaseYear: 1979,
		AgeLimit:    16,
	}

	id, err := uc.Run(context.Background(), input)

	assert.Equal(t, domain.FilmID(0), id)
	assert.ErrorIs(t, err, domain.ErrInvalidFilmInput)
}
