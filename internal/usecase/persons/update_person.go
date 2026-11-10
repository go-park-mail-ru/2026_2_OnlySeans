package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type UpdatePersonUseCase struct {
	persons   domain.PersonRepository
	appLogger logger.Logger
}

func NewUpdatePersonUseCase(persons domain.PersonRepository, appLogger logger.Logger) (*UpdatePersonUseCase, error) {
	if persons == nil {
		return nil, domain.ErrNilPersonsRepo
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &UpdatePersonUseCase{
		persons:   persons,
		appLogger: appLogger,
	}, nil
}

func (uc *UpdatePersonUseCase) Run(ctx context.Context, input UpdatePersonInput) (*PersonOutput, error) {
	uc.appLogger.DebugContext(ctx, "update person started", map[string]any{
		"person_id": input.PersonID,
	})

	person, err := preparePerson(input.Person)
	if err != nil {
		logging.Validation(ctx, uc.appLogger, "update person rejected: invalid input", err, map[string]any{"person_id": input.PersonID})
		return nil, err
	}

	updated, err := uc.persons.Update(
		ctx,
		input.PersonID,
		*person,
	)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "update person failed", err, map[string]any{
			"person_id": input.PersonID,
		})

		return nil, fmt.Errorf("update person %d: %w", input.PersonID, err)
	}

	uc.appLogger.InfoContext(ctx, "person updated", map[string]any{
		"person_id": updated.ID,
	})

	return NewPersonOutput(updated), nil
}
