package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

// обновляет данные существующего фильма
type UpdateFilmUseCase struct {
	repository domain.FilmUpdater
	logger     logger.Logger
}

func NewUpdateFilmUseCase(repository domain.FilmUpdater, appLogger logger.Logger) (*UpdateFilmUseCase, error) {
	if repository == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}
	return &UpdateFilmUseCase{repository: repository, logger: appLogger}, nil
}

func (uc *UpdateFilmUseCase) Run(ctx context.Context, id domain.FilmID, input domain.CreateFilmInput) error {
	uc.logger.DebugContext(ctx, "update film started", map[string]any{"film_id": id})
	input.Title = strings.TrimSpace(input.Title)
	err := input.Validate()
	if err != nil {
		uc.logger.WarnContext(ctx, "update film rejected: invalid input", map[string]any{"error": err, "film_id": id})
		return fmt.Errorf("validate film input: %w", err)
	}
	err = uc.repository.UpdateFilm(ctx, id, input)
	if err != nil {
		logging.Failure(ctx, uc.logger, "update film failed", err, map[string]any{"film_id": id})
		return fmt.Errorf("update film: %w", err)
	}
	uc.logger.InfoContext(ctx, "film updated", map[string]any{"film_id": id})
	return nil
}
