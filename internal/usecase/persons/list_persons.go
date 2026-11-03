package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type ListPersonsUseCase struct {
	persons domain.PersonRepository
	logger  logger.Logger
}

func NewListPersonsUseCase(persons domain.PersonRepository, appLogger logger.Logger) (*ListPersonsUseCase, error) {
	if persons == nil {
		return nil, domain.ErrNilPersonsRepo
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &ListPersonsUseCase{
		persons: persons,
		logger:  appLogger,
	}, nil
}

func (uc *ListPersonsUseCase) Run(ctx context.Context, input ListPersonsInput) (*ListPersonsOutput, error) {
	persons, total, err := uc.persons.List(
		ctx,
		input.Limit,
		input.Offset,
	)
	if err != nil {
		uc.logger.Error("list persons failed", map[string]any{
			"limit":  input.Limit,
			"offset": input.Offset,
			"error":  err,
		})

		return nil, fmt.Errorf("list persons: %w", err)
	}

	return NewListPersonsOutput(persons, total), nil
}
