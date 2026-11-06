package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type GetFilmUseCase struct {
	repository domain.FilmRepository
	logger     logger.Logger
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
		logger:     appLogger,
	}, nil
}

func (uc *GetFilmUseCase) Run(ctx context.Context, input GetFilmInput) (*GetFilmOutput, error) {
	uc.logger.Debug("get film started", map[string]any{
		"film_id": input.FilmID,
	})

	film, err := uc.repository.GetFilmByID(ctx, input.FilmID)
	if err != nil {
		uc.logger.Error("get film failed", map[string]any{
			"film_id": input.FilmID,
			"error":   err,
		})
		return nil, fmt.Errorf("get film %d: %w", input.FilmID, err)
	}

	uc.logger.Info("film retrieved", map[string]any{
		"film_id": input.FilmID,
	})

	return NewGetFilmOutput(film), nil
}
