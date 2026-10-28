package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type UpdateReviewUseCase struct {
	reviews domain.ReviewRepository
}

func NewUpdateReviewUseCase(reviews domain.ReviewRepository) (*UpdateReviewUseCase, error) {
	if reviews == nil {
		return nil, domain.ErrNilReviewsRepo
	}

	return &UpdateReviewUseCase{reviews: reviews}, nil
}

func (uc *UpdateReviewUseCase) Run(ctx context.Context, input UpdateReviewInput) (*ReviewOutput, error) {
	changes, err := prepareReview(input.Review)
	if err != nil {
		return nil, err
	}

	review, err := uc.reviews.GetByID(ctx, input.ReviewID)
	if err != nil {
		return nil, fmt.Errorf("get review %d: %w", input.ReviewID, err)
	}
	if review.AuthorID != input.UserID {
		return nil, domain.ErrForbidden
	}
	if input.Version <= 0 {
		return nil, domain.ErrInvalidVersion
	}

	updated, err := uc.reviews.Update(ctx, input.ReviewID, domain.ReviewUpdate{
		Input:   *changes,
		Version: input.Version,
	})
	if err != nil {
		return nil, fmt.Errorf("update review %d: %w", input.ReviewID, err)
	}
	return NewReviewOutput(updated), nil
}
