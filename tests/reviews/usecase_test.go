package reviews_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	existingFilm domain.FilmID = 1
	missingFilm  domain.FilmID = 999
)

var errStorage = errors.New("storage is down")

func testFilms(t *testing.T, failure error) *mocks.MockFilmRepository {
	t.Helper()
	films := mocks.NewMockFilmRepository(gomock.NewController(t))
	films.EXPECT().GetFilmByID(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, id domain.FilmID) (domain.Film, error) {
		if failure != nil {
			return domain.Film{}, failure
		}
		if id != existingFilm {
			return domain.Film{}, domain.ErrFilmNotFound
		}
		return domain.Film{ID: id, Title: "Тестовый фильм"}, nil
	}).AnyTimes()
	return films
}

func TestNewReviewUseCases_NilDependencies(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockReviewRepository(ctrl)
	films := mocks.NewMockFilmRepository(ctrl)
	appLogger := newReviewsTestLogger()
	_, err := usecase.NewListReviewsUseCase(nil, films, appLogger)
	require.ErrorIs(t, err, domain.ErrNilReviewsRepo)
	_, err = usecase.NewListReviewsUseCase(repo, nil, appLogger)
	require.ErrorIs(t, err, domain.ErrNilFilmsRepo)
	_, err = usecase.NewCreateReviewUseCase(nil, films, appLogger)
	require.ErrorIs(t, err, domain.ErrNilReviewsRepo)
	_, err = usecase.NewCreateReviewUseCase(repo, nil, appLogger)
	require.ErrorIs(t, err, domain.ErrNilFilmsRepo)
	_, err = usecase.NewUpdateReviewUseCase(nil, appLogger)
	require.ErrorIs(t, err, domain.ErrNilReviewsRepo)
	_, err = usecase.NewDeleteReviewUseCase(nil, appLogger)
	require.ErrorIs(t, err, domain.ErrNilReviewsRepo)
}

func TestCreateReviewUseCase(t *testing.T) {
	for _, tc := range []struct {
		name    string
		film    domain.FilmID
		input   domain.ReviewInput
		wantErr error
	}{
		{"valid", existingFilm, validInput(), nil},
		{"missing film", missingFilm, validInput(), domain.ErrFilmNotFound},
		{"short title", existingFilm, domain.ReviewInput{Title: "ab", Content: validContent}, domain.ErrInvalidTitle},
		{"blank title", existingFilm, domain.ReviewInput{Title: "     ", Content: validContent}, domain.ErrInvalidTitle},
		{"short content", existingFilm, domain.ReviewInput{Title: validTitle, Content: "мало"}, domain.ErrInvalidContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockReviewRepository(ctrl)
			films := mocks.NewMockFilmRepository(ctrl)
			if tc.wantErr == nil || errors.Is(tc.wantErr, domain.ErrFilmNotFound) {
				lookup := films.EXPECT().GetFilmByID(gomock.Any(), tc.film).Return(domain.Film{ID: tc.film}, tc.wantErr)
				if tc.wantErr == nil {
					repo.EXPECT().Create(gomock.Any(), domain.UserID(7), tc.film, tc.input).Return(&domain.Review{ID: 1}, nil).After(lookup)
				}
			}
			uc, err := usecase.NewCreateReviewUseCase(repo, films, newReviewsTestLogger())
			require.NoError(t, err)
			out, err := uc.Run(t.Context(), usecase.NewCreateReviewInput(7, tc.film, tc.input))
			require.ErrorIs(t, err, tc.wantErr)
			if err == nil {
				require.Equal(t, domain.ReviewID(1), out.ID)
			} else {
				require.Nil(t, out)
			}
		})
	}
}

func TestCreateReviewUseCase_TrimsSpaces(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockReviewRepository(ctrl)
	films := mocks.NewMockFilmRepository(ctrl)
	lookup := films.EXPECT().GetFilmByID(gomock.Any(), existingFilm).Return(domain.Film{ID: existingFilm}, nil)
	repo.EXPECT().Create(gomock.Any(), domain.UserID(7), existingFilm, validInput()).Return(&domain.Review{ID: 1}, nil).After(lookup)
	uc, err := usecase.NewCreateReviewUseCase(repo, films, newReviewsTestLogger())
	require.NoError(t, err)
	_, err = uc.Run(t.Context(), usecase.NewCreateReviewInput(7, existingFilm, domain.ReviewInput{Title: "  " + validTitle + "\n", Content: "\t" + validContent + "  "}))
	require.NoError(t, err)
}

func TestCreateReviewUseCase_Twice(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockReviewRepository(ctrl)
	films := mocks.NewMockFilmRepository(ctrl)
	gomock.InOrder(
		films.EXPECT().GetFilmByID(gomock.Any(), existingFilm).Return(domain.Film{ID: existingFilm}, nil),
		repo.EXPECT().Create(gomock.Any(), domain.UserID(7), existingFilm, validInput()).Return(&domain.Review{ID: 1}, nil),
		films.EXPECT().GetFilmByID(gomock.Any(), existingFilm).Return(domain.Film{ID: existingFilm}, nil),
		repo.EXPECT().Create(gomock.Any(), domain.UserID(7), existingFilm, validInput()).Return(nil, domain.ErrReviewExists),
	)
	uc, err := usecase.NewCreateReviewUseCase(repo, films, newReviewsTestLogger())
	require.NoError(t, err)
	_, err = uc.Run(t.Context(), usecase.NewCreateReviewInput(7, existingFilm, validInput()))
	require.NoError(t, err)
	_, err = uc.Run(t.Context(), usecase.NewCreateReviewInput(7, existingFilm, validInput()))
	require.ErrorIs(t, err, domain.ErrReviewExists)
}

func TestReviewUseCases_FilmsFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockReviewRepository(ctrl)
	films := mocks.NewMockFilmRepository(ctrl)
	films.EXPECT().GetFilmByID(gomock.Any(), existingFilm).Return(domain.Film{}, errStorage).Times(2)
	create, err := usecase.NewCreateReviewUseCase(repo, films, newReviewsTestLogger())
	require.NoError(t, err)
	_, err = create.Run(t.Context(), usecase.NewCreateReviewInput(7, existingFilm, validInput()))
	require.ErrorIs(t, err, errStorage)
	list, err := usecase.NewListReviewsUseCase(repo, films, newReviewsTestLogger())
	require.NoError(t, err)
	_, err = list.Run(t.Context(), usecase.NewListReviewsInput(existingFilm, 10, 0))
	require.ErrorIs(t, err, errStorage)
}

func TestUpdateReviewUseCase(t *testing.T) {
	for _, tc := range []struct {
		name    string
		user    domain.UserID
		id      domain.ReviewID
		input   domain.ReviewInput
		wantErr error
	}{
		{"author", 7, 1, domain.ReviewInput{Title: "  Передумала  ", Content: validContent, ContainsSpoilers: true}, nil},
		{"other author", 8, 1, validInput(), domain.ErrForbidden},
		{"missing review", 7, 99, validInput(), domain.ErrReviewNotFound},
		{"invalid input", 7, 1, domain.ReviewInput{Title: "ab", Content: validContent}, domain.ErrInvalidTitle},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewMockReviewRepository(gomock.NewController(t))
			normalized := tc.input
			normalized.Title = strings.TrimSpace(normalized.Title)
			normalized.Content = strings.TrimSpace(normalized.Content)
			if !errors.Is(tc.wantErr, domain.ErrInvalidTitle) {
				getErr := error(nil)
				if tc.id == 99 {
					getErr = domain.ErrReviewNotFound
				}
				lookup := repo.EXPECT().GetByID(gomock.Any(), tc.id).Return(&domain.Review{ID: tc.id, AuthorID: 7}, getErr)
				if tc.wantErr == nil {
					repo.EXPECT().Update(gomock.Any(), tc.id, domain.ReviewUpdate{Input: normalized, Version: 1}).Return(&domain.Review{ID: tc.id, AuthorID: 7, Title: normalized.Title, Content: normalized.Content, ContainsSpoilers: normalized.ContainsSpoilers}, nil).After(lookup)
				}
			}
			uc, err := usecase.NewUpdateReviewUseCase(repo, newReviewsTestLogger())
			require.NoError(t, err)
			out, err := uc.Run(t.Context(), usecase.NewUpdateReviewInput(tc.user, tc.id, tc.input, 1))
			require.ErrorIs(t, err, tc.wantErr)
			if err == nil {
				require.Equal(t, normalized.Title, out.Review.Title)
				require.Equal(t, normalized.ContainsSpoilers, out.Review.ContainsSpoilers)
			} else {
				require.Nil(t, out)
			}
		})
	}
}

func TestDeleteReviewUseCase(t *testing.T) {
	for _, tc := range []struct {
		name    string
		user    domain.UserID
		id      domain.ReviewID
		wantErr error
	}{
		{"author", 7, 1, nil}, {"other author", 8, 1, domain.ErrForbidden}, {"missing", 7, 99, domain.ErrReviewNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewMockReviewRepository(gomock.NewController(t))
			getErr := error(nil)
			if tc.id == 99 {
				getErr = domain.ErrReviewNotFound
			}
			lookup := repo.EXPECT().GetByID(gomock.Any(), tc.id).Return(&domain.Review{ID: tc.id, AuthorID: 7}, getErr)
			if tc.wantErr == nil {
				repo.EXPECT().Delete(gomock.Any(), tc.id, tc.user).Return(nil).After(lookup)
			}
			uc, err := usecase.NewDeleteReviewUseCase(repo, newReviewsTestLogger())
			require.NoError(t, err)
			require.ErrorIs(t, uc.Run(t.Context(), usecase.NewDeleteReviewInput(tc.user, tc.id)), tc.wantErr)
		})
	}
}

func TestListReviewsUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockReviewRepository(ctrl)
	films := mocks.NewMockFilmRepository(ctrl)
	lookup := films.EXPECT().GetFilmByID(gomock.Any(), existingFilm).Return(domain.Film{ID: existingFilm}, nil)
	repo.EXPECT().ListByFilm(gomock.Any(), existingFilm, 10, 0).Return([]domain.Review{}, 0, nil).After(lookup)
	films.EXPECT().GetFilmByID(gomock.Any(), missingFilm).Return(domain.Film{}, domain.ErrFilmNotFound)
	uc, err := usecase.NewListReviewsUseCase(repo, films, newReviewsTestLogger())
	require.NoError(t, err)
	out, err := uc.Run(t.Context(), usecase.NewListReviewsInput(existingFilm, 10, 0))
	require.NoError(t, err)
	require.Empty(t, out.Reviews)
	require.Zero(t, out.Total)
	_, err = uc.Run(t.Context(), usecase.NewListReviewsInput(missingFilm, 10, 0))
	require.ErrorIs(t, err, domain.ErrFilmNotFound)
}
