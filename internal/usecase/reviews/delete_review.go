package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type DeleteReviewUseCase struct {
	reviews domain.ReviewRepository
}

func NewDeleteReviewUseCase(reviews domain.ReviewRepository) (*DeleteReviewUseCase, error) {
	if reviews == nil {
		return nil, domain.ErrNilReviewsRepo
	}
	return &DeleteReviewUseCase{reviews: reviews}, nil
}

// Run удаляет отзыв. Удалять можно только свой отзыв.
func (uc *DeleteReviewUseCase) Run(ctx context.Context, input DeleteReviewInput) error {
	review, err := uc.reviews.GetByID(ctx, input.ReviewID)
	if err != nil {
		return fmt.Errorf("get review %d: %w", input.ReviewID, err)
	}
	if review.AuthorID != input.UserID {
		return domain.ErrForbidden
	}

	if err := uc.reviews.Delete(ctx, input.ReviewID, input.UserID); err != nil {
		return fmt.Errorf("delete review %d: %w", input.ReviewID, err)
	}
	return nil
}
