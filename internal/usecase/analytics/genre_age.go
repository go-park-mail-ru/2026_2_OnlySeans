package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type GenreAgeInput struct {
	GenreSlug string
	From      time.Time
	To        time.Time
}

type GenreAgeUseCase struct {
	repository domain.AnalyticsRepository
	appLogger  logger.Logger
}

func NewGenreAgeUseCase(repository domain.AnalyticsRepository, appLogger logger.Logger) (*GenreAgeUseCase, error) {
	if repository == nil {
		return nil, domain.ErrNilAnalyticsRepo
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &GenreAgeUseCase{
		repository: repository,
		appLogger:  appLogger,
	}, nil
}

func (uc *GenreAgeUseCase) Run(ctx context.Context, input GenreAgeInput) ([]domain.AgeGroupOpenStat, error) {
	genreSlug := strings.TrimSpace(input.GenreSlug)

	uc.appLogger.DebugContext(ctx, "genre age analytics started", map[string]any{
		"genre_slug": genreSlug,
		"from":       input.From,
		"to":         input.To,
	})

	if genreSlug == "" {
		return nil, domain.ErrInvalidAnalyticsGenreSlug
	}

	if input.From.IsZero() || input.To.IsZero() || !input.From.Before(input.To) {
		return nil, domain.ErrInvalidAnalyticsPeriod
	}

	stats, err := uc.repository.GenreOpensByAge(ctx, genreSlug, input.From, input.To)

	if err != nil {
		logging.Failure(ctx, uc.appLogger, "genre age analytics failed", err, map[string]any{
			"genre_slug": genreSlug,
			"from":       input.From,
			"to":         input.To,
		})

		return nil, fmt.Errorf("genre opens by age: %w", err)
	}

	uc.appLogger.InfoContext(ctx, "genre age analytics completed", map[string]any{
		"count": len(stats),
	})

	return stats, nil
}
