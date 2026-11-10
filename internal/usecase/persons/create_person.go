package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type CreatePersonUseCase struct {
	persons   domain.PersonRepository
	appLogger logger.Logger
}

func NewCreatePersonUseCase(persons domain.PersonRepository, appLogger logger.Logger) (*CreatePersonUseCase, error) {
	if persons == nil {
		return nil, domain.ErrNilPersonsRepo
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &CreatePersonUseCase{
		persons:   persons,
		appLogger: appLogger,
	}, nil
}

func (uc *CreatePersonUseCase) Run(ctx context.Context, input CreatePersonInput) (*CreatePersonOutput, error) {
	uc.appLogger.DebugContext(ctx, "create person started", nil)

	person, err := preparePerson(input.Person)
	if err != nil {
		logging.Validation(ctx, uc.appLogger, "create person rejected: invalid input", err, nil)
		return nil, err
	}

	id, err := uc.persons.Create(ctx, *person)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "create person failed", err, map[string]any{})

		return nil, fmt.Errorf("create person: %w", err)
	}

	uc.appLogger.InfoContext(ctx, "person created", map[string]any{
		"person_id": id,
	})

	return &CreatePersonOutput{ID: id}, nil
}
