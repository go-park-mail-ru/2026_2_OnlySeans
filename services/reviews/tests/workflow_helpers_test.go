package reviews_test

import (
	"context"
	"testing"

	reviewsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	reviewsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/services/reviews/internal/repository"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/services/reviews/internal/usecase"
	"github.com/stretchr/testify/require"
)

type reviewCases struct {
	repo   domain.ReviewRepository
	list   *usecase.ListReviewsUseCase
	create *usecase.CreateReviewUseCase
	update *usecase.UpdateReviewUseCase
	delete *usecase.DeleteReviewUseCase
}

func newReviewCases(t *testing.T, films domain.FilmRepository) reviewCases {
	t.Helper()
	repo := reviewsservice.NewInMemoryReviewRepo()
	appLogger := newReviewsTestLogger()
	list, err := usecase.NewListReviewsUseCase(repo, films, appLogger)
	require.NoError(t, err)
	create, err := usecase.NewCreateReviewUseCase(repo, films, appLogger)
	require.NoError(t, err)
	update, err := usecase.NewUpdateReviewUseCase(repo, appLogger)
	require.NoError(t, err)
	remove, err := usecase.NewDeleteReviewUseCase(repo, appLogger)
	require.NoError(t, err)
	return reviewCases{repo: repo, list: list, create: create, update: update, delete: remove}
}

func (c reviewCases) createReview(author domain.UserID, film domain.FilmID, input domain.ReviewInput) (domain.Review, error) {
	out, err := c.create.Run(context.Background(), usecase.NewCreateReviewInput(author, film, input))
	if err != nil {
		return domain.Review{}, err
	}
	review, err := c.repo.GetByID(context.Background(), out.ID)
	if err != nil {
		return domain.Review{}, err
	}
	return *review, nil
}

func (c reviewCases) operations() reviewsdelivery.Operations {
	return reviewsdelivery.Operations{
		List: func(ctx context.Context, filmID domain.FilmID, limit, offset int) ([]domain.Review, int, error) {
			out, err := c.list.Run(ctx, usecase.NewListReviewsInput(filmID, limit, offset))
			if err != nil {
				return nil, 0, err
			}
			return out.Reviews, out.Total, nil
		},
		Create: func(ctx context.Context, userID domain.UserID, filmID domain.FilmID, input domain.ReviewInput) (domain.ReviewID, error) {
			out, err := c.create.Run(ctx, usecase.NewCreateReviewInput(userID, filmID, input))
			if err != nil {
				return 0, err
			}
			return out.ID, nil
		},
		Update: func(ctx context.Context, userID domain.UserID, reviewID domain.ReviewID, input domain.ReviewInput, version int) (domain.Review, error) {
			out, err := c.update.Run(ctx, usecase.NewUpdateReviewInput(userID, reviewID, input, version))
			if err != nil {
				return domain.Review{}, err
			}
			return out.Review, nil
		},
		Delete: func(ctx context.Context, userID domain.UserID, reviewID domain.ReviewID) error {
			return c.delete.Run(ctx, usecase.NewDeleteReviewInput(userID, reviewID))
		},
	}
}
