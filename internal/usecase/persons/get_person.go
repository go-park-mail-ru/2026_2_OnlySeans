package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type GetPersonUseCase struct {
	persons domain.PersonRepository
	logger  logger.Logger
}

func NewGetPersonUseCase(persons domain.PersonRepository, appLogger logger.Logger) (*GetPersonUseCase, error) {
	if persons == nil {
		return nil, domain.ErrNilPersonsRepo
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &GetPersonUseCase{
		persons: persons,
		logger:  appLogger,
	}, nil
}

func (uc *GetPersonUseCase) Run(ctx context.Context, id domain.PersonID) (*PersonOutput, error) {
	uc.logger.Debug("get person started", map[string]any{
		"person_id": id,
	})

	person, err := uc.persons.GetByID(ctx, id)
	if err != nil {
		uc.logger.Error("get person failed", map[string]any{
			"person_id": id,
			"error":     err,
		})

		return nil, fmt.Errorf("get person %d: %w", id, err)
	}

	uc.logger.Info("person retrieved", map[string]any{
		"person_id": id,
	})

	return NewPersonOutput(person), nil
}
