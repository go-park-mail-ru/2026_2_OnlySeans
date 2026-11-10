package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type DeleteFilmUseCase struct {
	repository domain.FilmRepository
	logger     logger.Logger
}

func NewDeleteFilmUseCase(repository domain.FilmRepository, appLogger logger.Logger) (*DeleteFilmUseCase, error) {
	if repository == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}
	return &DeleteFilmUseCase{repository: repository, logger: appLogger}, nil
}

func (uc *DeleteFilmUseCase) Run(ctx context.Context, id domain.FilmID) error {
	uc.logger.DebugContext(ctx, "delete film started", map[string]any{"film_id": id})
	if err := uc.repository.DeleteFilm(ctx, id); err != nil {
		logging.Failure(ctx, uc.logger, "delete film failed", err, map[string]any{"film_id": id})
		return fmt.Errorf("delete film: %w", err)
	}
	uc.logger.InfoContext(ctx, "film deleted", map[string]any{"film_id": id})
	return nil
}
