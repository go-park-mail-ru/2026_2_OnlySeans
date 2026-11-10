package ratings_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	ratingsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/ratings"
	ratingsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/ratings"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func testLogger(t *testing.T) logger.Logger {
	t.Helper()
	appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelDebug, "text")
	require.NoError(t, err)
	return appLogger
}

func TestSetFilmRatingUseCase_ReturnsDomainErrors(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		score                     int
		filmErr, repoErr, wantErr error
	}{
		{"invalid score", 0, nil, nil, domain.ErrInvalidFilmRating},
		{"missing film", 8, domain.ErrFilmNotFound, nil, domain.ErrFilmNotFound},
		{"repository rejects score", 8, nil, domain.ErrInvalidFilmRating, domain.ErrInvalidFilmRating},
		{"success", 8, nil, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockFilmRatingRepository(ctrl)
			films := mocks.NewMockFilmRepository(ctrl)
			if tc.score > 0 {
				lookup := films.EXPECT().GetFilmByID(gomock.Any(), domain.FilmID(1)).Return(domain.Film{ID: 1}, tc.filmErr)
				if tc.filmErr == nil {
					repo.EXPECT().SetFilmRating(gomock.Any(), domain.UserID(7), domain.FilmID(1), tc.score).Return(domain.FilmRating{AccountID: 7, FilmID: 1, Score: tc.score}, tc.repoErr).After(lookup)
				}
			}
			uc, err := ratingsusecase.NewSetFilmRatingUseCase(repo, films, testLogger(t))
			require.NoError(t, err)
			result, err := uc.Run(context.Background(), *ratingsusecase.NewSetFilmRatingInput(7, 1, tc.score))
			require.ErrorIs(t, err, tc.wantErr)
			if tc.score == 0 {
				require.EqualError(t, err, "invalid film rating: must be from 1 to 10")
			}
			if err == nil {
				require.Equal(t, tc.score, result.Rating.Score)
			}
		})
	}
}

func TestRatingUseCases_ReturnUnexpectedErrorsUnchanged(t *testing.T) {
	storageErr := errors.New("storage unavailable")
	for _, action := range []string{"set", "get", "missing"} {
		t.Run(action, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockFilmRatingRepository(ctrl)
			films := mocks.NewMockFilmRepository(ctrl)
			filmErr := error(nil)
			if action == "missing" {
				filmErr = domain.ErrFilmNotFound
			}
			lookup := films.EXPECT().GetFilmByID(gomock.Any(), domain.FilmID(1)).Return(domain.Film{ID: 1}, filmErr)
			if action == "set" {
				repo.EXPECT().SetFilmRating(gomock.Any(), domain.UserID(7), domain.FilmID(1), 8).Return(domain.FilmRating{}, storageErr).After(lookup)
				uc, err := ratingsusecase.NewSetFilmRatingUseCase(repo, films, testLogger(t))
				require.NoError(t, err)
				_, err = uc.Run(context.Background(), *ratingsusecase.NewSetFilmRatingInput(7, 1, 8))
				require.Same(t, storageErr, err)
			} else {
				if filmErr == nil {
					repo.EXPECT().GetFilmRatingSummary(gomock.Any(), domain.FilmID(1)).Return(domain.FilmRatingSummary{}, storageErr).After(lookup)
				}
				uc, err := ratingsusecase.NewGetFilmRatingUseCase(repo, films, testLogger(t))
				require.NoError(t, err)
				_, err = uc.Run(context.Background(), *ratingsusecase.NewGetFilmRatingInput(1))
				if filmErr != nil {
					require.Same(t, filmErr, err)
				} else {
					require.Same(t, storageErr, err)
				}
			}
		})
	}
}

func TestGetCollectionRatingsUseCase_ReturnsDomainErrors(t *testing.T) {
	repo := mocks.NewMockCollectionFilmRatingRepository(gomock.NewController(t))
	expected := []domain.FilmRatingSummary{{FilmID: 1}, {FilmID: 7}}
	repo.EXPECT().GetCollectionFilmRatingSummaries(gomock.Any(), domain.CollectionID(1), 20, 0).Return(expected, 2, nil)
	repo.EXPECT().GetCollectionFilmRatingSummaries(gomock.Any(), domain.CollectionID(999), 20, 0).Return(nil, 0, domain.ErrCollectionNotFound)
	uc, err := ratingsusecase.NewGetCollectionRatingsUseCase(repo, testLogger(t))
	require.NoError(t, err)
	result, err := uc.Run(context.Background(), *ratingsusecase.NewGetCollectionRatingsInput(1, 20, 0))
	require.NoError(t, err)
	require.Equal(t, expected, result.Ratings)
	require.Equal(t, 2, result.Total)
	_, err = uc.Run(context.Background(), *ratingsusecase.NewGetCollectionRatingsInput(999, 20, 0))
	require.Same(t, domain.ErrCollectionNotFound, err)
}

func TestNewGetCollectionRatingsUseCase_NilDependencies(t *testing.T) {
	_, err := ratingsusecase.NewGetCollectionRatingsUseCase(nil, testLogger(t))
	require.ErrorIs(t, err, domain.ErrNilCollectionRatingRepo)
}

func TestGetCollectionRatingsUseCase_PaginatesFilms(t *testing.T) {
	ctrl := gomock.NewController(t)
	films := mocks.NewMockCollectionRepository(ctrl)
	ratings := mocks.NewMockFilmRatingRepository(ctrl)
	lookup := films.EXPECT().ListCollectionFilms(gomock.Any(), domain.CollectionID(1), 100, 100).Return([]domain.Film{{ID: 101}}, 101, nil)
	ratings.EXPECT().GetFilmRatingsSummaries(gomock.Any(), []domain.FilmID{101}).Return([]domain.FilmRatingSummary{{FilmID: 101}}, nil).After(lookup)
	collectionRepo, err := ratingsservice.NewInMemoryCollectionRatingRepository(ratings, films)
	require.NoError(t, err)
	uc, err := ratingsusecase.NewGetCollectionRatingsUseCase(collectionRepo, testLogger(t))
	require.NoError(t, err)
	result, err := uc.Run(context.Background(), *ratingsusecase.NewGetCollectionRatingsInput(1, 100, 100))
	require.NoError(t, err)
	require.Equal(t, 101, result.Total)
	require.Len(t, result.Ratings, 1)
	require.Equal(t, domain.FilmID(101), result.Ratings[0].FilmID)
}

func TestInMemoryRatingRepo_Delete(t *testing.T) {
	repo := ratingsservice.NewInMemoryRepo()
	_, err := repo.SetFilmRating(context.Background(), 7, 1, 8)
	require.NoError(t, err)
	_, err = repo.SetFilmRating(context.Background(), 8, 1, 10)
	require.NoError(t, err)

	err = repo.DeleteFilmRating(context.Background(), 9, 1)
	assert.ErrorIs(t, err, domain.ErrRatingNotFound)

	err = repo.DeleteFilmRating(context.Background(), 7, 1)
	require.NoError(t, err)
	summary, err := repo.GetFilmRatingSummary(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, domain.FilmRatingSummary{FilmID: 1, AverageScore: 10, Votes: 1}, summary)

	err = repo.DeleteFilmRating(context.Background(), 7, 1)
	assert.ErrorIs(t, err, domain.ErrRatingNotFound)
}
