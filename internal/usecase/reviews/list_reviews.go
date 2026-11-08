package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type ListReviewsUseCase struct {
	reviews domain.ReviewRepository
	films   domain.FilmRepository
	logger  logger.Logger
}

func NewListReviewsUseCase(
	reviews domain.ReviewRepository,
	films domain.FilmRepository,
	appLogger logger.Logger,
) (*ListReviewsUseCase, error) {
	if reviews == nil {
		return nil, domain.ErrNilReviewsRepo
	}
	if films == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &ListReviewsUseCase{
		reviews: reviews,
		films:   films,
		logger:  appLogger,
	}, nil
}

func (uc *ListReviewsUseCase) Run(ctx context.Context, input ListReviewsInput) (*ListReviewsOutput, error) {
	uc.logger.Debug("list reviews started", map[string]any{
		"film_id": input.FilmID,
		"limit":   input.Limit,
		"offset":  input.Offset,
	})

	if _, err := uc.films.GetFilmByID(ctx, input.FilmID); err != nil {
		logging.Failure(uc.logger, "get film for reviews failed", err, map[string]any{
			"film_id": input.FilmID,
		})
		return nil, fmt.Errorf("get film %d: %w", input.FilmID, err)
	}

	reviews, total, err := uc.reviews.ListByFilm(ctx, input.FilmID, input.Limit, input.Offset)
	if err != nil {
		logging.Failure(uc.logger, "list reviews failed", err, map[string]any{
			"film_id": input.FilmID,
			"limit":   input.Limit,
			"offset":  input.Offset,
		})
		return nil, fmt.Errorf("list reviews of film %d: %w", input.FilmID, err)
	}

	uc.logger.Info("reviews listed", map[string]any{
		"film_id": input.FilmID,
		"limit":   input.Limit,
		"offset":  input.Offset,
		"total":   total,
	})

	return NewListReviewsOutput(reviews, total), nil
}
