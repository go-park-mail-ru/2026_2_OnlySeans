package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type CreateFilmUseCase struct {
	repository domain.FilmCreator
	logger     logger.Logger
}

func NewCreateFilmUseCase(repository domain.FilmCreator, appLogger logger.Logger) (*CreateFilmUseCase, error) {
	if repository == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}
	return &CreateFilmUseCase{repository: repository, logger: appLogger}, nil
}

func (uc *CreateFilmUseCase) Run(ctx context.Context, input domain.CreateFilmInput) (domain.FilmID, error) {
	uc.logger.DebugContext(ctx, "create film started", nil)
	input.Title = strings.TrimSpace(input.Title)

	if err := input.Validate(); err != nil {
		uc.logger.WarnContext(ctx, "create film rejected: invalid input", map[string]any{"error": err})
		return 0, fmt.Errorf("validate film input: %w", err)
	}

	id, err := uc.repository.CreateFilm(ctx, input)
	if err != nil {
		logging.Failure(ctx, uc.logger, "create film failed", err, nil)
		return 0, fmt.Errorf("create film: %w", err)
	}

	uc.logger.InfoContext(ctx, "film created", map[string]any{"film_id": id})
	return id, nil
}
