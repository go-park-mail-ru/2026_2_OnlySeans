package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type UpdateReviewUseCase struct {
	reviews domain.ReviewRepository
	logger  logger.Logger
}

func NewUpdateReviewUseCase(
	reviews domain.ReviewRepository,
	appLogger logger.Logger,
) (*UpdateReviewUseCase, error) {
	if reviews == nil {
		return nil, domain.ErrNilReviewsRepo
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &UpdateReviewUseCase{
		reviews: reviews,
		logger:  appLogger,
	}, nil
}

func (uc *UpdateReviewUseCase) Run(ctx context.Context, input UpdateReviewInput) (*ReviewOutput, error) {
	uc.logger.Debug("update review started", map[string]any{
		"user_id":   input.UserID,
		"review_id": input.ReviewID,
	})

	changes, err := prepareReview(input.Review)
	if err != nil {
		logging.Validation(uc.logger, "update review rejected: invalid input", err, map[string]any{"user_id": input.UserID, "review_id": input.ReviewID})
		return nil, err
	}

	review, err := uc.reviews.GetByID(ctx, input.ReviewID)
	if err != nil {
		logging.Failure(uc.logger, "get review for update failed", err, map[string]any{
			"review_id": input.ReviewID,
		})
		return nil, fmt.Errorf("get review %d: %w", input.ReviewID, err)
	}

	if review.AuthorID != input.UserID {
		uc.logger.Warn("review update forbidden", map[string]any{
			"user_id":   input.UserID,
			"review_id": input.ReviewID,
		})
		return nil, domain.ErrForbidden
	}

	updated, err := uc.reviews.Update(ctx, input.ReviewID, *changes)
	if err != nil {
		logging.Failure(uc.logger, "update review failed", err, map[string]any{
			"user_id":   input.UserID,
			"review_id": input.ReviewID,
		})
		return nil, fmt.Errorf("update review %d: %w", input.ReviewID, err)
	}

	uc.logger.Info("review updated", map[string]any{
		"user_id":   input.UserID,
		"review_id": input.ReviewID,
	})

	return NewReviewOutput(updated), nil
}
