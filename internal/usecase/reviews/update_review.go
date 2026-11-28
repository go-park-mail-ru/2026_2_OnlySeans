package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type UpdateReviewUseCase struct {
	reviews   domain.ReviewRepository
	policy    domain.ReviewPolicyProvider
	appLogger logger.Logger
}

func NewUpdateReviewUseCase(
	reviews domain.ReviewRepository,
	policy domain.ReviewPolicyProvider,
	appLogger logger.Logger,
) (*UpdateReviewUseCase, error) {
	if reviews == nil {
		return nil, domain.ErrNilReviewsRepo
	}

	if policy == nil {
		return nil, domain.ErrNilReviewPolicy
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &UpdateReviewUseCase{
		reviews:   reviews,
		policy:    policy,
		appLogger: appLogger,
	}, nil
}

func (uc *UpdateReviewUseCase) Run(ctx context.Context, input UpdateReviewInput) (*ReviewOutput, error) {
	uc.appLogger.DebugContext(ctx, "update review started", map[string]any{
		"user_id":   input.UserID,
		"review_id": input.ReviewID,
	})

	review, err := uc.reviews.GetByID(ctx, input.ReviewID)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "get review for update failed", err, map[string]any{
			"review_id": input.ReviewID,
		})
		return nil, fmt.Errorf("get review %d: %w", input.ReviewID, err)
	}

	if review.AuthorID != input.UserID {
		uc.appLogger.WarnContext(ctx, "review update forbidden", map[string]any{
			"user_id":   input.UserID,
			"review_id": input.ReviewID,
		})
		return nil, domain.ErrForbidden
	}
	if input.Version <= 0 {
		logging.Validation(
			ctx, uc.appLogger, "update review rejected: invalid version",
			domain.ErrInvalidVersion,
			map[string]any{
				"user_id":   input.UserID,
				"review_id": input.ReviewID,
			},
		)
		return nil, domain.ErrInvalidVersion
	}

	policy := uc.policy.ReviewPolicy(input.UserID)
	if !policy.WriteEnabled {
		logging.Validation(ctx, uc.appLogger, "update review rejected: feature disabled", domain.ErrFeatureDisabled, map[string]any{"user_id": input.UserID, "review_id": input.ReviewID, "revision": policy.Revision})
		return nil, &domain.ReviewFeatureDisabledError{Revision: policy.Revision}
	}

	changes, err := prepareReview(input.Review)
	if err != nil {
		logging.Validation(ctx, uc.appLogger, "update review rejected: invalid input", err, map[string]any{
			"user_id":   input.UserID,
			"review_id": input.ReviewID,
		})

		return nil, err
	}

	if err := validateDynamicContentLength(changes.Content, policy); err != nil {
		logging.Validation(ctx, uc.appLogger, "update review rejected: dynamic content limit", err, map[string]any{"user_id": input.UserID, "review_id": input.ReviewID, "revision": policy.Revision, "max_length": policy.MaxLength})
		return nil, err
	}

	updated, err := uc.reviews.Update(ctx, input.ReviewID, domain.ReviewUpdate{
		Input:   *changes,
		Version: input.Version,
	})
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "update review failed", err, map[string]any{
			"user_id":   input.UserID,
			"review_id": input.ReviewID,
		})
		return nil, fmt.Errorf("update review %d: %w", input.ReviewID, err)
	}

	uc.appLogger.InfoContext(ctx, "review updated", map[string]any{
		"user_id":   input.UserID,
		"review_id": input.ReviewID,
	})

	return NewReviewOutput(updated), nil
}
