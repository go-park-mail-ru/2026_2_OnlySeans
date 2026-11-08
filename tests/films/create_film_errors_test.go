package films_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filmsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/films"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateFilmUseCaseRunRepositoryError(t *testing.T) {
	storageErr := errors.New("storage unavailable")
	repo := &filmCreatorStub{err: storageErr}
	uc, err := filmsusecase.NewCreateFilmUseCase(repo, newFilmsTestLogger())
	require.NoError(t, err)

	id, err := uc.Run(context.Background(), domain.CreateFilmInput{
		Title: "Alien", FilmType: domain.FilmTypeMovie, ReleaseYear: 1979, AgeLimit: 16,
	})

	assert.Equal(t, domain.FilmID(0), id)
	assert.ErrorIs(t, err, storageErr)
	assert.True(t, repo.called)
}

func TestNewCreateFilmUseCaseNilRepository(t *testing.T) {
	uc, err := filmsusecase.NewCreateFilmUseCase(nil, newFilmsTestLogger())
	assert.Nil(t, uc)
	assert.ErrorIs(t, err, domain.ErrNilFilmsRepo)
}

type contextFilmCreator struct {
	filmCreatorStub
	ctx context.Context
}

func (s *contextFilmCreator) CreateFilm(ctx context.Context, input domain.CreateFilmInput) (domain.FilmID, error) {
	s.ctx = ctx
	return s.filmCreatorStub.CreateFilm(ctx, input)
}

func TestCreateFilmUseCasePassesContextAndOptionalFields(t *testing.T) {
	repo := &contextFilmCreator{filmCreatorStub: filmCreatorStub{id: 42}}
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

	id, err := uc.Run(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, domain.FilmID(42), id)
	assert.Same(t, ctx, repo.ctx)
	expected := input
	expected.Title = "Alien"
	assert.Equal(t, expected, repo.input)
	assert.Equal(t, "  Alien  ", input.Title, "the caller's input value is preserved")
}
