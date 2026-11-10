package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type GetFilmUseCase struct {
	repository domain.FilmRepository
	appLogger  logger.Logger
}

func NewGetFilmUseCase(
	repository domain.FilmRepository,
	appLogger logger.Logger,
) (*GetFilmUseCase, error) {
	if repository == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &GetFilmUseCase{
		repository: repository,
		appLogger:  appLogger,
	}, nil
}

func (uc *GetFilmUseCase) Run(ctx context.Context, input GetFilmInput) (*GetFilmOutput, error) {
	uc.appLogger.DebugContext(ctx, "get film started", map[string]any{
		"film_id": input.FilmID,
	})

	film, err := uc.repository.GetFilmByID(ctx, input.FilmID)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "get film failed", err, map[string]any{
			"film_id": input.FilmID,
		})
		return nil, fmt.Errorf("get film %d: %w", input.FilmID, err)
	}

	uc.appLogger.InfoContext(ctx, "film retrieved", map[string]any{
		"film_id": input.FilmID,
	})

	return NewGetFilmOutput(film), nil
}
