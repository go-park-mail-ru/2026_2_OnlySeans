package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type CreateReviewUseCase struct {
	reviews   domain.ReviewRepository
	films     domain.FilmRepository
	policy    domain.ReviewPolicyProvider
	appLogger logger.Logger
}

func NewCreateReviewUseCase(
	reviews domain.ReviewRepository,
	films domain.FilmRepository,
	policy domain.ReviewPolicyProvider,
	appLogger logger.Logger,
) (*CreateReviewUseCase, error) {
	if reviews == nil {
		return nil, domain.ErrNilReviewsRepo
	}
	if films == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	if policy == nil {
		return nil, domain.ErrNilReviewPolicy
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &CreateReviewUseCase{
		reviews:   reviews,
		films:     films,
		policy:    policy,
		appLogger: appLogger,
	}, nil
}

func (uc *CreateReviewUseCase) Run(ctx context.Context, input CreateReviewInput) (*CreateReviewOutput, error) {
	uc.appLogger.DebugContext(ctx, "create review started", map[string]any{
		"user_id": input.AuthorID,
		"film_id": input.FilmID,
	})

	policy := uc.policy.ReviewPolicy(input.AuthorID)
	if !policy.WriteEnabled {
		logging.Validation(ctx, uc.appLogger, "create review rejected: feature disabled", domain.ErrFeatureDisabled, map[string]any{"user_id": input.AuthorID, "revision": policy.Revision})
		return nil, domain.ErrFeatureDisabled
	}

	review, err := prepareReview(input.Review)
	if err != nil {
		logging.Validation(ctx, uc.appLogger, "create review rejected: invalid input", err, map[string]any{"user_id": input.AuthorID, "film_id": input.FilmID})
		return nil, err
	}
	if err := validateDynamicContentLength(review.Content, policy); err != nil {
		logging.Validation(ctx, uc.appLogger, "create review rejected: dynamic content limit", err, map[string]any{"user_id": input.AuthorID, "revision": policy.Revision, "max_length": policy.MaxLength})
		return nil, err
	}

	if _, err := uc.films.GetFilmByID(ctx, input.FilmID); err != nil {
		logging.Failure(ctx, uc.appLogger, "get film for review failed", err, map[string]any{
			"film_id": input.FilmID,
		})
		if errors.Is(err, domain.ErrFilmNotFound) {
			return nil, domain.ErrFilmNotFound
		}
		return nil, fmt.Errorf("get film %d: %w", input.FilmID, err)
	}

	created, err := uc.reviews.Create(ctx, input.AuthorID, input.FilmID, *review)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "create review failed", err, map[string]any{
			"user_id": input.AuthorID,
			"film_id": input.FilmID,
		})
		return nil, fmt.Errorf("create review: %w", err)
	}

	uc.appLogger.InfoContext(ctx, "review created", map[string]any{
		"user_id":   input.AuthorID,
		"film_id":   input.FilmID,
		"review_id": created.ID,
	})

	return &CreateReviewOutput{ID: created.ID}, nil
}
