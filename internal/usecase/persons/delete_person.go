package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type DeletePersonUseCase struct {
	persons   domain.PersonRepository
	appLogger logger.Logger
}

func NewDeletePersonUseCase(persons domain.PersonRepository, appLogger logger.Logger) (*DeletePersonUseCase, error) {
	if persons == nil {
		return nil, domain.ErrNilPersonsRepo
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &DeletePersonUseCase{
		persons:   persons,
		appLogger: appLogger,
	}, nil
}

func (uc *DeletePersonUseCase) Run(ctx context.Context, input DeletePersonInput) error {
	uc.appLogger.DebugContext(ctx, "delete person started", map[string]any{
		"person_id": input.PersonID,
	})

	err := uc.persons.Delete(ctx, input.PersonID)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "delete person failed", err, map[string]any{
			"person_id": input.PersonID,
		})

		return fmt.Errorf("delete person %d: %w", input.PersonID, err)
	}

	uc.appLogger.InfoContext(ctx, "person deleted", map[string]any{
		"person_id": input.PersonID,
	})

	return nil
}
