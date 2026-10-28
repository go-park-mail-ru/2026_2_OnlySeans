package reviews_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	reviewsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/reviews"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/reviews"
)

const (
	existingFilm domain.FilmID = 1
	missingFilm  domain.FilmID = 999
)

var errStorage = errors.New("storage is down")

// fakeFilms знает только фильм existingFilm.
type fakeFilms struct{}

func (fakeFilms) GetFilmByID(_ context.Context, id domain.FilmID) (domain.Film, error) {
	if id != existingFilm {
		return domain.Film{}, domain.ErrFilmNotFound
	}
	return domain.Film{ID: id, Title: "Тестовый фильм"}, nil
}

func (fakeFilms) ListFilms(context.Context, int, int) ([]domain.Film, int, error) {
	return nil, 0, nil
}

// brokenFilms имитирует упавшее хранилище фильмов.
type brokenFilms struct{}

func (brokenFilms) GetFilmByID(context.Context, domain.FilmID) (domain.Film, error) {
	return domain.Film{}, errStorage
}

func (brokenFilms) ListFilms(context.Context, int, int) ([]domain.Film, int, error) {
	return nil, 0, errStorage
}

// reviewCases — четыре сценария отзывов над одним хранилищем.
type reviewCases struct {
	list   *usecase.ListReviewsUseCase
	create *usecase.CreateReviewUseCase
	update *usecase.UpdateReviewUseCase
	delete *usecase.DeleteReviewUseCase
}

func newReviewCases(t *testing.T, films domain.FilmRepository) reviewCases {
	t.Helper()

	repo := reviewsservice.NewInMemoryReviewRepo()
	list, err := usecase.NewListReviewsUseCase(repo, films)
	require.NoError(t, err)
	create, err := usecase.NewCreateReviewUseCase(repo, films)
	require.NoError(t, err)
	update, err := usecase.NewUpdateReviewUseCase(repo)
	require.NoError(t, err)
	del, err := usecase.NewDeleteReviewUseCase(repo)
	require.NoError(t, err)

	return reviewCases{list: list, create: create, update: update, delete: del}
}

func (c reviewCases) createReview(author domain.UserID, film domain.FilmID, in domain.ReviewInput) (domain.Review, error) {
	out, err := c.create.Run(context.Background(), usecase.NewCreateReviewInput(author, film, in))
	if err != nil {
		return domain.Review{}, err
	}
	return out.Review, nil
}

func (c reviewCases) updateReview(user domain.UserID, id domain.ReviewID, in domain.ReviewInput) (domain.Review, error) {
	out, err := c.update.Run(context.Background(), usecase.NewUpdateReviewInput(user, id, in, 1))
	if err != nil {
		return domain.Review{}, err
	}
	return out.Review, nil
}

func (c reviewCases) listReviews(film domain.FilmID) ([]domain.Review, int, error) {
	out, err := c.list.Run(context.Background(), usecase.NewListReviewsInput(film, 10, 0))
	if err != nil {
		return nil, 0, err
	}
	return out.Reviews, out.Total, nil
}

func TestNewReviewUseCases_NilDependencies(t *testing.T) {
	repo := reviewsservice.NewInMemoryReviewRepo()

	_, err := usecase.NewListReviewsUseCase(nil, fakeFilms{})
	assert.ErrorIs(t, err, domain.ErrNilReviewsRepo)
	_, err = usecase.NewListReviewsUseCase(repo, nil)
	assert.ErrorIs(t, err, domain.ErrNilFilmsRepo)

	_, err = usecase.NewCreateReviewUseCase(nil, fakeFilms{})
	assert.ErrorIs(t, err, domain.ErrNilReviewsRepo)
	_, err = usecase.NewCreateReviewUseCase(repo, nil)
	assert.ErrorIs(t, err, domain.ErrNilFilmsRepo)

	_, err = usecase.NewUpdateReviewUseCase(nil)
	assert.ErrorIs(t, err, domain.ErrNilReviewsRepo)

	_, err = usecase.NewDeleteReviewUseCase(nil)
	assert.ErrorIs(t, err, domain.ErrNilReviewsRepo)
}

func TestCreateReviewUseCase(t *testing.T) {
	tests := []struct {
		name    string
		film    domain.FilmID
		input   domain.ReviewInput
		wantErr error
	}{
		{"обычный отзыв", existingFilm, validInput(), nil},
		{"несуществующий фильм", missingFilm, validInput(), domain.ErrFilmNotFound},
		{"короткий заголовок", existingFilm, domain.ReviewInput{Title: "ab", Content: validContent}, domain.ErrInvalidTitle},
		{"заголовок из одних пробелов", existingFilm, domain.ReviewInput{Title: "     ", Content: validContent}, domain.ErrInvalidTitle},
		{"короткий текст", existingFilm, domain.ReviewInput{Title: validTitle, Content: "мало"}, domain.ErrInvalidContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newReviewCases(t, fakeFilms{}).createReview(7, tt.film, tt.input)
			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestCreateReviewUseCase_TrimsSpaces(t *testing.T) {
	review, err := newReviewCases(t, fakeFilms{}).createReview(7, existingFilm, domain.ReviewInput{
		Title:   "  " + validTitle + "\n",
		Content: "\t" + validContent + "  ",
	})
	require.NoError(t, err)

	assert.Equal(t, validTitle, review.Title)
	assert.Equal(t, validContent, review.Content)
	assert.Equal(t, domain.UserID(7), review.AuthorID)
}

func TestCreateReviewUseCase_Twice(t *testing.T) {
	cases := newReviewCases(t, fakeFilms{})

	_, err := cases.createReview(7, existingFilm, validInput())
	require.NoError(t, err)

	_, err = cases.createReview(7, existingFilm, validInput())
	assert.ErrorIs(t, err, domain.ErrReviewExists)
}

func TestReviewUseCases_FilmsFailure(t *testing.T) {
	cases := newReviewCases(t, brokenFilms{})

	_, err := cases.createReview(7, existingFilm, validInput())
	assert.ErrorIs(t, err, errStorage)

	_, _, err = cases.listReviews(existingFilm)
	assert.ErrorIs(t, err, errStorage)
}

func TestUpdateReviewUseCase(t *testing.T) {
	changes := domain.ReviewInput{Title: "  Передумала  ", Content: validContent, ContainsSpoilers: true}

	tests := []struct {
		name    string
		user    domain.UserID
		review  domain.ReviewID
		input   domain.ReviewInput
		wantErr error
	}{
		{"автор меняет свой отзыв", 7, 1, changes, nil},
		{"чужой отзыв", 8, 1, changes, domain.ErrForbidden},
		{"несуществующий отзыв", 7, 99, changes, domain.ErrReviewNotFound},
		{"невалидные данные", 7, 1, domain.ReviewInput{Title: "ab", Content: validContent}, domain.ErrInvalidTitle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cases := newReviewCases(t, fakeFilms{})
			created, err := cases.createReview(7, existingFilm, validInput())
			require.NoError(t, err)

			updated, err := cases.updateReview(tt.user, tt.review, tt.input)

			list, _, listErr := cases.listReviews(existingFilm)
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

func TestDeleteReviewUseCase(t *testing.T) {
	tests := []struct {
		name      string
		user      domain.UserID
		review    domain.ReviewID
		wantErr   error
		wantTotal int
	}{
		{"автор удаляет свой отзыв", 7, 1, nil, 0},
		{"чужой отзыв", 8, 1, domain.ErrForbidden, 1},
		{"несуществующий отзыв", 7, 99, domain.ErrReviewNotFound, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cases := newReviewCases(t, fakeFilms{})
			_, err := cases.createReview(7, existingFilm, validInput())
			require.NoError(t, err)

			err = cases.delete.Run(context.Background(), usecase.NewDeleteReviewInput(tt.user, tt.review))
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}

			_, total, err := cases.listReviews(existingFilm)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, total)
		})
	}
}

func TestListReviewsUseCase(t *testing.T) {
	cases := newReviewCases(t, fakeFilms{})

	list, total, err := cases.listReviews(existingFilm)
	require.NoError(t, err)
	assert.Empty(t, list)
	assert.Zero(t, total)

	_, _, err = cases.listReviews(missingFilm)
	assert.ErrorIs(t, err, domain.ErrFilmNotFound)
}
