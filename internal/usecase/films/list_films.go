package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
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
	films, total, err := uc.repository.ListFilms(ctx, input.Limit, input.Offset)
	if err != nil {
		uc.logger.Error("list films failed", map[string]any{
			"limit":  input.Limit,
			"offset": input.Offset,
			"error":  err,
		})
		return nil, fmt.Errorf("list films: %w", err)
	}

	uc.logger.Info("films listed", map[string]any{
		"limit":  input.Limit,
		"offset": input.Offset,
		"total":  total,
	})

	return NewListFilmsOutput(films, total), nil
}
