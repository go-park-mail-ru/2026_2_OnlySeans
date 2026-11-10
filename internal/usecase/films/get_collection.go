package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type GetCollectionUseCase struct {
	repository domain.CollectionRepository
	appLogger  logger.Logger
}

func NewGetCollectionUseCase(
	repository domain.CollectionRepository,
	appLogger logger.Logger,
) (*GetCollectionUseCase, error) {
	if repository == nil {
		return nil, errors.New("collections repository is nil")
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &GetCollectionUseCase{
		repository: repository,
		appLogger:  appLogger,
	}, nil
}

func (uc *GetCollectionUseCase) Run(ctx context.Context, input GetCollectionInput) (*GetCollectionOutput, error) {
	uc.appLogger.DebugContext(ctx, "get collection started", map[string]any{
		"collection_id": input.CollectionID,
		"limit":         input.Limit,
		"offset":        input.Offset,
	})

	collection, err := uc.repository.GetCollectionByID(ctx, input.CollectionID)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "get collection failed", err, map[string]any{
			"collection_id": input.CollectionID,
		})
		return nil, fmt.Errorf("get collection %d: %w", input.CollectionID, err)
	}

	films, total, err := uc.repository.ListCollectionFilms(ctx, input.CollectionID, input.Limit, input.Offset)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "list films of collection failed", err, map[string]any{
			"collection_id": input.CollectionID,
		})
		return nil, fmt.Errorf("list films of collection %d: %w", input.CollectionID, err)
	}

	uc.appLogger.InfoContext(ctx, "collection retrieved", map[string]any{
		"collection_id": input.CollectionID,
		"limit":         input.Limit,
		"offset":        input.Offset,
		"total":         total,
	})

	return NewGetCollectionOutput(collection, films, total), nil
}
