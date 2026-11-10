package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type ListFilmsUseCase struct {
	repository domain.FilmRepository
	logger     logger.Logger
}

func NewListFilmsUseCase(
	repository domain.FilmRepository,
	appLogger logger.Logger,
) (*ListFilmsUseCase, error) {
	if repository == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &ListFilmsUseCase{
		repository: repository,
		logger:     appLogger,
	}, nil
}

func (uc *ListFilmsUseCase) Run(ctx context.Context, input ListFilmsInput) (*ListFilmsOutput, error) {
	uc.logger.DebugContext(ctx, "list films started", map[string]any{
		"limit":  input.Limit,
		"offset": input.Offset,
	})

	films, total, err := uc.repository.ListFilms(ctx, input.Limit, input.Offset)
	if err != nil {
		logging.Failure(ctx, uc.logger, "list films failed", err, map[string]any{
			"limit":  input.Limit,
			"offset": input.Offset,
		})
		return nil, fmt.Errorf("list films: %w", err)
	}

	uc.logger.InfoContext(ctx, "films listed", map[string]any{
		"limit":  input.Limit,
		"offset": input.Offset,
		"total":  total,
	})

	return NewListFilmsOutput(films, total), nil
}
