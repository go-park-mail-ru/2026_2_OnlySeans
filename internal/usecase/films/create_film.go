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
	repository domain.FilmRepository
	appLogger  logger.Logger
}

func NewCreateFilmUseCase(repository domain.FilmRepository, appLogger logger.Logger) (*CreateFilmUseCase, error) {
	if repository == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}
	return &CreateFilmUseCase{repository: repository, appLogger: appLogger}, nil
}

func (uc *CreateFilmUseCase) Run(ctx context.Context, input domain.CreateFilmInput) (domain.FilmID, error) {
	uc.appLogger.DebugContext(ctx, "create film started", nil)
	input.Title = strings.TrimSpace(input.Title)

	if err := input.Validate(); err != nil {
		uc.appLogger.WarnContext(ctx, "create film rejected: invalid input", map[string]any{"error": err})
		return 0, fmt.Errorf("validate film input: %w", err)
	}

	id, err := uc.repository.CreateFilm(ctx, input)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "create film failed", err, nil)
		return 0, fmt.Errorf("create film: %w", err)
	}

	uc.appLogger.InfoContext(ctx, "film created", map[string]any{"film_id": id})
	return id, nil
}
