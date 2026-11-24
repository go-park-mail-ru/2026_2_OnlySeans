package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const (
	maxTopGenresLimit = 100
)

type TopGenresInput struct {
	From  time.Time
	To    time.Time
	Limit int
}

type TopGenresUseCase struct {
	repository domain.AnalyticsRepository
	appLogger  logger.Logger
}

func NewTopGenresUseCase(repository domain.AnalyticsRepository, appLogger logger.Logger) (*TopGenresUseCase, error) {
	if repository == nil {
		return nil, domain.ErrNilAnalyticsRepo
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &TopGenresUseCase{
		repository: repository,
		appLogger:  appLogger,
	}, nil
}

func (uc *TopGenresUseCase) Run(ctx context.Context, input TopGenresInput) ([]domain.GenreOpenStat, error) {
	uc.appLogger.DebugContext(ctx, "top genres started", map[string]any{
		"from":  input.From,
		"to":    input.To,
		"limit": input.Limit,
	})

	if input.From.IsZero() || input.To.IsZero() || !input.From.Before(input.To) {
		return nil, domain.ErrInvalidAnalyticsPeriod
	}

	if input.Limit < 1 || input.Limit > maxTopGenresLimit {
		return nil, domain.ErrInvalidAnalyticsLimit
	}

	stats, err := uc.repository.TopGenresByFilmOpens(ctx, input.From, input.To, input.Limit)

	if err != nil {
		logging.Failure(ctx, uc.appLogger, "top genres failed", err, map[string]any{
			"from":  input.From,
			"to":    input.To,
			"limit": input.Limit,
		})

		return nil, fmt.Errorf("top genres by film opens: %w", err)
	}

	uc.appLogger.InfoContext(ctx, "top genres completed", map[string]any{
		"count": len(stats),
	})

	return stats, nil
}
