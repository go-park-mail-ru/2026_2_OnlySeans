package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type DeleteReviewUseCase struct {
	reviews   domain.ReviewRepository
	appLogger logger.Logger
}

func NewDeleteReviewUseCase(
	reviews domain.ReviewRepository,
	appLogger logger.Logger,
) (*DeleteReviewUseCase, error) {
	if reviews == nil {
		return nil, domain.ErrNilReviewsRepo
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &DeleteReviewUseCase{
		reviews:   reviews,
		appLogger: appLogger,
	}, nil
}

func (uc *DeleteReviewUseCase) Run(ctx context.Context, input DeleteReviewInput) error {
	uc.appLogger.DebugContext(ctx, "delete review started", map[string]any{
		"user_id":   input.UserID,
		"review_id": input.ReviewID,
	})

	review, err := uc.reviews.GetByID(ctx, input.ReviewID)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "get review for deletion failed", err, map[string]any{
			"review_id": input.ReviewID,
		})
		return fmt.Errorf("get review %d: %w", input.ReviewID, err)
	}

	if review.AuthorID != input.UserID {
		uc.appLogger.WarnContext(ctx, "review deletion forbidden", map[string]any{
			"user_id":   input.UserID,
			"review_id": input.ReviewID,
		})
		return domain.ErrForbidden
	}

	if err := uc.reviews.Delete(ctx, input.ReviewID, input.UserID); err != nil {
		logging.Failure(ctx, uc.appLogger, "delete review failed", err, map[string]any{
			"user_id":   input.UserID,
			"review_id": input.ReviewID,
		})
		return fmt.Errorf("delete review %d: %w", input.ReviewID, err)
	}

	uc.appLogger.InfoContext(ctx, "review deleted", map[string]any{
		"user_id":   input.UserID,
		"review_id": input.ReviewID,
	})

	return nil
}
