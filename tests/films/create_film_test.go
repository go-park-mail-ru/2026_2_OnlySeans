package films_test

import (
	"context"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filmsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/films"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type filmCreatorStub struct {
	id     domain.FilmID
	err    error
	called bool
	input  domain.CreateFilmInput
}

func (s *filmCreatorStub) CreateFilm(ctx context.Context, input domain.CreateFilmInput) (domain.FilmID, error) {
	s.called = true
	s.input = input
	return s.id, s.err
}

func TestCreateFilmUseCaseRun(t *testing.T) {
	repo := &filmCreatorStub{
		id: 42,
	}
	uc, err := filmsusecase.NewCreateFilmUseCase(repo, newFilmsTestLogger())
	require.NoError(t, err)

	input := domain.CreateFilmInput{
		Title:       "  Alien  ",
		FilmType:    domain.FilmTypeMovie,
		ReleaseYear: 1979,
		AgeLimit:    16,
	}

	id, err := uc.Run(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, domain.FilmID(42), id)
	assert.True(t, repo.called)
	assert.Equal(t, "Alien", repo.input.Title)
}

// неверная карточка останавливается до обращения к репозиторию
func TestCreateFilmUseCaseRunInvalidInput(t *testing.T) {
	repo := &filmCreatorStub{
		id: 42,
	}
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
	assert.False(t, repo.called)
}
