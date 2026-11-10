package reviews_test

import (
	"context"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	reviewsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/reviews"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/reviews"
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
