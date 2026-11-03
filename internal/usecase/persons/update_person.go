package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type UpdatePersonUseCase struct {
	persons domain.PersonRepository
	logger  logger.Logger
}

func NewUpdatePersonUseCase(persons domain.PersonRepository, appLogger logger.Logger) (*UpdatePersonUseCase, error) {
	if persons == nil {
		return nil, domain.ErrNilPersonsRepo
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &UpdatePersonUseCase{
		persons: persons,
		logger:  appLogger,
	}, nil
}

func (uc *UpdatePersonUseCase) Run(ctx context.Context, input UpdatePersonInput) (*PersonOutput, error) {
	person, err := preparePerson(input.Person)
	if err != nil {
		return nil, err
	}

	updated, err := uc.persons.Update(
		ctx,
		input.PersonID,
		*person,
	)
	if err != nil {
		uc.logger.Error("update person failed", map[string]any{
			"person_id": input.PersonID,
			"error":     err,
		})

		return nil, fmt.Errorf("update person %d: %w", input.PersonID, err)
	}

	uc.logger.Info("person updated", map[string]any{
		"person_id": updated.ID,
	})

	return NewPersonOutput(updated), nil
}
