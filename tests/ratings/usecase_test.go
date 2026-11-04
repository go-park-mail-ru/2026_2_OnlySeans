package ratings_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filmsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/films"
	ratingsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/ratings"
	ratingsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/ratings"
)

type ratingTestFilms struct {
	err error
}

type manyFilmsCollectionRepo struct {
	films []domain.Film
}

func (r manyFilmsCollectionRepo) ListCollections(context.Context, int, int) ([]domain.Collection, int, error) {
	return nil, 0, nil
}

func (r manyFilmsCollectionRepo) GetCollectionByID(context.Context, domain.CollectionID) (domain.Collection, error) {
	return domain.Collection{ID: 1}, nil
}

func (r manyFilmsCollectionRepo) ListCollectionFilms(_ context.Context, _ domain.CollectionID, limit, offset int) ([]domain.Film, int, error) {
	start := min(offset, len(r.films))
	end := min(start+limit, len(r.films))
	return r.films[start:end], len(r.films), nil
}

func (f ratingTestFilms) GetFilmByID(context.Context, domain.FilmID) (domain.Film, error) {
	if f.err != nil {
		return domain.Film{}, f.err
	}
	return domain.Film{ID: 1}, nil
}

func (ratingTestFilms) ListFilms(context.Context, int, int) ([]domain.Film, int, error) {
	return nil, 0, nil
}

type ratingTestRepo struct {
	setErr       error
	summaryErr   error
	summariesErr error
	setCalls     int
	deleteErr    error
}

func (r *ratingTestRepo) Set(_ context.Context, accountID domain.UserID, filmID domain.FilmID, score int16) (domain.FilmRating, error) {
	r.setCalls++
	if r.setErr != nil {
		return domain.FilmRating{}, r.setErr
	}
	return domain.FilmRating{AccountID: accountID, FilmID: filmID, Score: score}, nil
}

func (r *ratingTestRepo) GetSummary(_ context.Context, filmID domain.FilmID) (domain.FilmRatingSummary, error) {
	if r.summaryErr != nil {
		return domain.FilmRatingSummary{}, r.summaryErr
	}
	return domain.FilmRatingSummary{FilmID: filmID}, nil
}

func (r *ratingTestRepo) GetSummaries(_ context.Context, filmIDs []domain.FilmID) ([]domain.FilmRatingSummary, error) {
	if r.summariesErr != nil {
		return nil, r.summariesErr
	}
	summaries := make([]domain.FilmRatingSummary, len(filmIDs))
	for i, filmID := range filmIDs {
		summaries[i] = domain.FilmRatingSummary{FilmID: filmID}
	}
	return summaries, nil
}

func (r *ratingTestRepo) Delete(context.Context, domain.UserID, domain.FilmID) error {
	return r.deleteErr
}

func TestSetFilmRatingUseCase_ReturnsDomainErrors(t *testing.T) {
	repo := &ratingTestRepo{}
	uc, err := ratingsusecase.NewSetFilmRatingUseCase(repo, ratingTestFilms{})
	require.NoError(t, err)

	_, err = uc.Run(context.Background(), ratingsusecase.NewSetFilmRatingInput(7, 1, 0))
	assert.ErrorIs(t, err, domain.ErrInvalidFilmRating)
	assert.Zero(t, repo.setCalls, "invalid score must not reach repository")

	uc, err = ratingsusecase.NewSetFilmRatingUseCase(repo, ratingTestFilms{err: domain.ErrFilmNotFound})
	require.NoError(t, err)
	_, err = uc.Run(context.Background(), ratingsusecase.NewSetFilmRatingInput(7, 1, 8))
	assert.Same(t, domain.ErrFilmNotFound, err)

	uc, err = ratingsusecase.NewSetFilmRatingUseCase(&ratingTestRepo{setErr: domain.ErrInvalidFilmRating}, ratingTestFilms{})
	require.NoError(t, err)
	_, err = uc.Run(context.Background(), ratingsusecase.NewSetFilmRatingInput(7, 1, 8))
	assert.Same(t, domain.ErrInvalidFilmRating, err)
}

func TestRatingUseCases_ReturnUnexpectedErrorsUnchanged(t *testing.T) {
	storageErr := errors.New("storage unavailable")

	set, err := ratingsusecase.NewSetFilmRatingUseCase(&ratingTestRepo{setErr: storageErr}, ratingTestFilms{})
	require.NoError(t, err)
	_, err = set.Run(context.Background(), ratingsusecase.NewSetFilmRatingInput(7, 1, 8))
	assert.Same(t, storageErr, err)

	get, err := ratingsusecase.NewGetFilmRatingUseCase(&ratingTestRepo{summaryErr: storageErr}, ratingTestFilms{})
	require.NoError(t, err)
	_, err = get.Run(context.Background(), ratingsusecase.NewGetFilmRatingInput(1))
	assert.Same(t, storageErr, err)

	get, err = ratingsusecase.NewGetFilmRatingUseCase(&ratingTestRepo{}, ratingTestFilms{err: domain.ErrFilmNotFound})
	require.NoError(t, err)
	_, err = get.Run(context.Background(), ratingsusecase.NewGetFilmRatingInput(1))
	assert.Same(t, domain.ErrFilmNotFound, err)
}

func TestGetCollectionRatingsUseCase_ReturnsDomainErrors(t *testing.T) {
	repo := &ratingTestRepo{}
	collections, err := filmsservice.NewSeededService()
	require.NoError(t, err)
	uc, err := ratingsusecase.NewGetCollectionRatingsUseCase(repo, collections)
	require.NoError(t, err)

	result, err := uc.Run(context.Background(), ratingsusecase.NewGetCollectionRatingsInput(1))
	require.NoError(t, err)
	require.Len(t, result.Ratings, 6)
	require.Equal(t, domain.FilmID(1), result.Ratings[0].FilmID)
	require.Equal(t, domain.FilmID(7), result.Ratings[5].FilmID)

	_, err = uc.Run(context.Background(), ratingsusecase.NewGetCollectionRatingsInput(999))
	assert.Same(t, domain.ErrCollectionNotFound, err)
}

func TestNewGetCollectionRatingsUseCase_NilDependencies(t *testing.T) {
	collections, err := filmsservice.NewSeededService()
	require.NoError(t, err)
	_, err = ratingsusecase.NewGetCollectionRatingsUseCase(nil, collections)
	assert.ErrorIs(t, err, domain.ErrNilFilmRatingsRepo)
	_, err = ratingsusecase.NewGetCollectionRatingsUseCase(&ratingTestRepo{}, nil)
	assert.ErrorIs(t, err, domain.ErrNilCollectionsRepo)
}

func TestGetCollectionRatingsUseCase_PaginatesAllFilms(t *testing.T) {
	films := make([]domain.Film, 101)
	for i := range films {
		films[i].ID = domain.FilmID(i + 1)
	}

	uc, err := ratingsusecase.NewGetCollectionRatingsUseCase(
		&ratingTestRepo{},
		manyFilmsCollectionRepo{films: films},
	)
	require.NoError(t, err)

	result, err := uc.Run(context.Background(), ratingsusecase.NewGetCollectionRatingsInput(1))
	require.NoError(t, err)
	require.Len(t, result.Ratings, len(films))
	require.Equal(t, domain.FilmID(1), result.Ratings[0].FilmID)
	require.Equal(t, domain.FilmID(101), result.Ratings[100].FilmID)
}

func TestInMemoryRatingRepo_Delete(t *testing.T) {
	repo := ratingsservice.NewInMemoryRepo()
	_, err := repo.Set(context.Background(), 7, 1, 8)
	require.NoError(t, err)
	_, err = repo.Set(context.Background(), 8, 1, 10)
	require.NoError(t, err)

	err = repo.Delete(context.Background(), 9, 1)
	assert.ErrorIs(t, err, domain.ErrRatingNotFound)

	err = repo.Delete(context.Background(), 7, 1)
	require.NoError(t, err)
	summary, err := repo.GetSummary(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, domain.FilmRatingSummary{FilmID: 1, AverageScore: 10, Votes: 1}, summary)

	err = repo.Delete(context.Background(), 7, 1)
	assert.ErrorIs(t, err, domain.ErrRatingNotFound)
}
