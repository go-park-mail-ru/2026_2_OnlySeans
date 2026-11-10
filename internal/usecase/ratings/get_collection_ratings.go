package usecase

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type GetCollectionRatingsUseCase struct {
	repository domain.CollectionFilmRatingRepository
	logger     logger.Logger
}

func NewGetCollectionRatingsUseCase(repository domain.CollectionFilmRatingRepository, appLogger logger.Logger) (*GetCollectionRatingsUseCase, error) {
	if repository == nil {
		return nil, domain.ErrNilCollectionRatingRepo
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}
	return &GetCollectionRatingsUseCase{
		repository: repository,
		logger:     appLogger,
	}, nil
}

func (uc *GetCollectionRatingsUseCase) Run(ctx context.Context, input GetCollectionRatingsInput) (*GetCollectionRatingsOutput, error) {
	fields := map[string]any{
		"collection_id": input.CollectionID,
		"limit":         input.Limit,
		"offset":        input.Offset,
	}
	uc.logger.DebugContext(ctx, "get collection ratings started", fields)
	uc.logger.DebugContext(ctx, "loading collection film rating summaries", fields)
	ratings, total, err := uc.repository.GetCollectionFilmRatingSummaries(ctx, input.CollectionID, input.Limit, input.Offset)
	if err != nil {
		logging.Failure(ctx, uc.logger, "loading collection film rating summaries failed", err, fields)
		return nil, err
	}
	uc.logger.DebugContext(ctx, "collection film rating summaries loaded", map[string]any{
		"collection_id": input.CollectionID,
		"count":         len(ratings),
		"total":         total,
		"limit":         input.Limit,
		"offset":        input.Offset,
	})
	return NewGetCollectionRatingsOutput(ratings, total), nil
}
