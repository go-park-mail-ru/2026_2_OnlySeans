package reviews_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/entities"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/reviews"
)

const (
	existingFilm entities.FilmID = 1
	missingFilm  entities.FilmID = 999
)

var errStorage = errors.New("storage is down")

// fakeFilms знает только фильм existingFilm.
type fakeFilms struct{}

func (fakeFilms) GetFilmByID(_ context.Context, id entities.FilmID) (entities.Film, error) {
	if id != existingFilm {
		return entities.Film{}, films.ErrFilmNotFound
	}
	return entities.Film{ID: id, Title: "Тестовый фильм"}, nil
}

// brokenFilms имитирует упавший сервис фильмов.
type brokenFilms struct{}

func (brokenFilms) GetFilmByID(context.Context, entities.FilmID) (entities.Film, error) {
	return entities.Film{}, errStorage
}

func newUsecase(t *testing.T) *reviews.Usecase {
	t.Helper()

	uc, err := reviews.NewUsecase(reviews.NewInMemoryReviewRepo(), fakeFilms{})
	require.NoError(t, err)
	return uc
}

func TestNewUsecase_NilDependencies(t *testing.T) {
	_, err := reviews.NewUsecase(nil, fakeFilms{})
	assert.ErrorIs(t, err, reviews.ErrNilRepo)

	_, err = reviews.NewUsecase(reviews.NewInMemoryReviewRepo(), nil)
	assert.ErrorIs(t, err, reviews.ErrNilFilms)
}

func TestUsecase_Create(t *testing.T) {
	tests := []struct {
		name    string
		film    entities.FilmID
		input   reviews.ReviewInput
		wantErr error
	}{
		{"обычный отзыв", existingFilm, validInput(), nil},
		{"несуществующий фильм", missingFilm, validInput(), films.ErrFilmNotFound},
		{"короткий заголовок", existingFilm, reviews.ReviewInput{Title: "ab", Content: validContent}, reviews.ErrInvalidTitle},
		{"заголовок из одних пробелов", existingFilm, reviews.ReviewInput{Title: "     ", Content: validContent}, reviews.ErrInvalidTitle},
		{"короткий текст", existingFilm, reviews.ReviewInput{Title: validTitle, Content: "мало"}, reviews.ErrInvalidContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newUsecase(t).Create(context.Background(), 7, tt.film, tt.input)
			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestUsecase_CreateTrimsSpaces(t *testing.T) {
	review, err := newUsecase(t).Create(context.Background(), 7, existingFilm, reviews.ReviewInput{
		Title:   "  " + validTitle + "\n",
		Content: "\t" + validContent + "  ",
	})
	require.NoError(t, err)

	assert.Equal(t, validTitle, review.Title)
	assert.Equal(t, validContent, review.Content)
}

func TestUsecase_CreateTwice(t *testing.T) {
	ctx := context.Background()
	uc := newUsecase(t)

	_, err := uc.Create(ctx, 7, existingFilm, validInput())
	require.NoError(t, err)

	_, err = uc.Create(ctx, 7, existingFilm, validInput())
	assert.ErrorIs(t, err, reviews.ErrReviewExists)
}

func TestUsecase_FilmsFailure(t *testing.T) {
	ctx := context.Background()
	uc, err := reviews.NewUsecase(reviews.NewInMemoryReviewRepo(), brokenFilms{})
	require.NoError(t, err)

	_, err = uc.Create(ctx, 7, existingFilm, validInput())
	assert.ErrorIs(t, err, errStorage)

	_, _, err = uc.ListByFilm(ctx, existingFilm, 10, 0)
	assert.ErrorIs(t, err, errStorage)
}

func TestUsecase_Update(t *testing.T) {
	ctx := context.Background()
	changes := reviews.ReviewInput{Title: "  Передумала  ", Content: validContent, ContainsSpoilers: true}

	tests := []struct {
		name    string
		user    auth.UserID
		review  entities.ReviewID
		input   reviews.ReviewInput
		wantErr error
	}{
		{"автор меняет свой отзыв", 7, 1, changes, nil},
		{"чужой отзыв", 8, 1, changes, reviews.ErrForbidden},
		{"несуществующий отзыв", 7, 99, changes, reviews.ErrReviewNotFound},
		{"невалидные данные", 7, 1, reviews.ReviewInput{Title: "ab", Content: validContent}, reviews.ErrInvalidTitle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := newUsecase(t)
			created, err := uc.Create(ctx, 7, existingFilm, validInput())
			require.NoError(t, err)

			updated, err := uc.Update(ctx, tt.user, tt.review, tt.input)

			list, _, listErr := uc.ListByFilm(ctx, existingFilm, 10, 0)
			require.NoError(t, listErr)
			require.Len(t, list, 1)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, created, list[0], "при ошибке отзыв не должен меняться")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "Передумала", updated.Title, "пробелы по краям обрезаются")
			assert.True(t, updated.ContainsSpoilers)
			assert.Equal(t, updated, list[0])
		})
	}
}

func TestUsecase_Delete(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		user      auth.UserID
		review    entities.ReviewID
		wantErr   error
		wantTotal int
	}{
		{"автор удаляет свой отзыв", 7, 1, nil, 0},
		{"чужой отзыв", 8, 1, reviews.ErrForbidden, 1},
		{"несуществующий отзыв", 7, 99, reviews.ErrReviewNotFound, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := newUsecase(t)
			_, err := uc.Create(ctx, 7, existingFilm, validInput())
			require.NoError(t, err)

			err = uc.Delete(ctx, tt.user, tt.review)
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}

			_, total, err := uc.ListByFilm(ctx, existingFilm, 10, 0)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, total)
		})
	}
}

func TestUsecase_ListByFilm(t *testing.T) {
	ctx := context.Background()
	uc := newUsecase(t)

	list, total, err := uc.ListByFilm(ctx, existingFilm, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, list)
	assert.Zero(t, total)

	_, _, err = uc.ListByFilm(ctx, missingFilm, 10, 0)
	assert.ErrorIs(t, err, films.ErrFilmNotFound)
}
