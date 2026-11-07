package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type ListCollectionsUseCase struct {
	repository domain.CollectionRepository
	logger     logger.Logger
}

func NewListCollectionsUseCase(
	repository domain.CollectionRepository,
	appLogger logger.Logger,
) (*ListCollectionsUseCase, error) {
	if repository == nil {
		return nil, domain.ErrNilCollectionsRepo
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &ListCollectionsUseCase{
		repository: repository,
		logger:     appLogger,
	}, nil
}

func (uc *ListCollectionsUseCase) Run(ctx context.Context, input ListCollectionsInput) (*ListCollectionsOutput, error) {
	uc.logger.Debug("list collections started", map[string]any{
		"limit":  input.Limit,
		"offset": input.Offset,
	})

	collections, total, err := uc.repository.ListCollections(ctx, input.Limit, input.Offset)
	if err != nil {
		logging.Failure(uc.logger, "list collections failed", err, map[string]any{
			"limit":  input.Limit,
			"offset": input.Offset,
		})
		return nil, fmt.Errorf("list collections: %w", err)
	}

	uc.logger.Info("collections listed", map[string]any{
		"limit":  input.Limit,
		"offset": input.Offset,
		"total":  total,
	})

	return NewListCollectionsOutput(collections, total), nil
}
