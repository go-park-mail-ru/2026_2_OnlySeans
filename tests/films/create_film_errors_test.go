package films_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filmsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateFilmUseCaseRunRepositoryError(t *testing.T) {
	storageErr := errors.New("storage unavailable")
	repo := mocks.NewMockFilmRepository(gomock.NewController(t))
	repo.EXPECT().CreateFilm(gomock.Any(), domain.CreateFilmInput{Title: "Alien", FilmType: domain.FilmTypeMovie, ReleaseYear: 1979, AgeLimit: 16}).Return(domain.FilmID(0), storageErr)
	uc, err := filmsusecase.NewCreateFilmUseCase(repo, newFilmsTestLogger())
	require.NoError(t, err)

	id, err := uc.Run(context.Background(), domain.CreateFilmInput{
		Title: "Alien", FilmType: domain.FilmTypeMovie, ReleaseYear: 1979, AgeLimit: 16,
	})

	assert.Equal(t, domain.FilmID(0), id)
	assert.ErrorIs(t, err, storageErr)
}

func TestNewCreateFilmUseCaseNilRepository(t *testing.T) {
	uc, err := filmsusecase.NewCreateFilmUseCase(nil, newFilmsTestLogger())
	assert.Nil(t, uc)
	assert.ErrorIs(t, err, domain.ErrNilFilmsRepo)
}

func TestCreateFilmUseCasePassesContextAndOptionalFields(t *testing.T) {
	repo := mocks.NewMockFilmRepository(gomock.NewController(t))
	uc, err := filmsusecase.NewCreateFilmUseCase(repo, newFilmsTestLogger())
	require.NoError(t, err)
	original, description := "Alien", "Science fiction"
	duration := int16(117)
	input := domain.CreateFilmInput{
		Title: "  Alien  ", OriginalTitle: &original, FilmType: domain.FilmTypeMovie,
		ReleaseYear: 1979, DurationMin: &duration, AgeLimit: 16, Description: &description,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	expected := input
	expected.Title = "Alien"
	repo.EXPECT().CreateFilm(ctx, expected).Return(domain.FilmID(42), nil)
	id, err := uc.Run(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, domain.FilmID(42), id)
	assert.Equal(t, "  Alien  ", input.Title, "the caller's input value is preserved")
}
