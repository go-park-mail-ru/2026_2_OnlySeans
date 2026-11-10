package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type ListPersonsUseCase struct {
	persons   domain.PersonRepository
	appLogger logger.Logger
}

func NewListPersonsUseCase(persons domain.PersonRepository, appLogger logger.Logger) (*ListPersonsUseCase, error) {
	if persons == nil {
		return nil, domain.ErrNilPersonsRepo
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &ListPersonsUseCase{
		persons:   persons,
		appLogger: appLogger,
	}, nil
}

func (uc *ListPersonsUseCase) Run(ctx context.Context, input ListPersonsInput) (*ListPersonsOutput, error) {
	uc.appLogger.DebugContext(ctx, "list persons started", map[string]any{
		"limit":  input.Limit,
		"offset": input.Offset,
	})

	persons, total, err := uc.persons.List(
		ctx,
		input.Limit,
		input.Offset,
	)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "list persons failed", err, map[string]any{
			"limit":  input.Limit,
			"offset": input.Offset,
		})

		return nil, fmt.Errorf("list persons: %w", err)
	}

	uc.appLogger.InfoContext(ctx, "persons listed", map[string]any{
		"limit":  input.Limit,
		"offset": input.Offset,
		"total":  total,
	})

	return NewListPersonsOutput(persons, total), nil
}
