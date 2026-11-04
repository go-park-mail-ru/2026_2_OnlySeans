package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type CreatePersonUseCase struct {
	persons domain.PersonRepository
	logger  logger.Logger
}

func NewCreatePersonUseCase(persons domain.PersonRepository, appLogger logger.Logger) (*CreatePersonUseCase, error) {
	if persons == nil {
		return nil, domain.ErrNilPersonsRepo
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &CreatePersonUseCase{
		persons: persons,
		logger:  appLogger,
	}, nil
}

func (uc *CreatePersonUseCase) Run(ctx context.Context, input CreatePersonInput) (*PersonOutput, error) {
	uc.logger.Debug("create person started", nil)

	person, err := preparePerson(input.Person)
	if err != nil {
		return nil, err
	}

	created, err := uc.persons.Create(ctx, *person)
	if err != nil {
		uc.logger.Error("create person failed", map[string]any{
			"error": err,
		})

		return nil, fmt.Errorf("create person: %w", err)
	}

	uc.logger.Info("person created", map[string]any{
		"person_id": created.ID,
	})

	return NewPersonOutput(created), nil
}
