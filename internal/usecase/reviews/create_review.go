package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type CreateReviewUseCase struct {
	reviews domain.ReviewRepository
	films   domain.FilmRepository
	logger  logger.Logger
}

func NewCreateReviewUseCase(
	reviews domain.ReviewRepository,
	films domain.FilmRepository,
	appLogger logger.Logger,
) (*CreateReviewUseCase, error) {
	if reviews == nil {
		return nil, domain.ErrNilReviewsRepo
	}
	if films == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &CreateReviewUseCase{
		reviews: reviews,
		films:   films,
		logger:  appLogger,
	}, nil
}

// Повторный отзыв того же автора на тот же фильм отклоняет хранилище, без отдельной проверки заранее.
func (uc *CreateReviewUseCase) Run(ctx context.Context, input CreateReviewInput) (*ReviewOutput, error) {
	uc.logger.Debug("create review started", map[string]any{
		"user_id": input.AuthorID,
		"film_id": input.FilmID,
	})

	review, err := prepareReview(input.Review)
	if err != nil {
		return nil, err
	}

	if _, err := uc.films.GetFilmByID(ctx, input.FilmID); err != nil {
		uc.logger.Error("get film for review failed", map[string]any{
			"film_id": input.FilmID,
			"error":   err,
		})
		return nil, fmt.Errorf("get film %d: %w", input.FilmID, err)
	}

	created, err := uc.reviews.Create(ctx, input.AuthorID, input.FilmID, *review)
	if err != nil {
		uc.logger.Error("create review failed", map[string]any{
			"user_id": input.AuthorID,
			"film_id": input.FilmID,
			"error":   err,
		})
		return nil, fmt.Errorf("create review: %w", err)
	}

	uc.logger.Info("review created", map[string]any{
		"user_id":   input.AuthorID,
		"film_id":   input.FilmID,
		"review_id": created.ID,
	})

	return NewReviewOutput(created), nil
}
