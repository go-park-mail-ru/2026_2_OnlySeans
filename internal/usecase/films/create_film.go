package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type CreateFilmUseCase struct {
	repository domain.FilmCreator
}

func NewCreateFilmUseCase(repository domain.FilmCreator) (*CreateFilmUseCase, error) {
	if repository == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	return &CreateFilmUseCase{repository: repository}, nil
}

func (uc *CreateFilmUseCase) Run(ctx context.Context, input domain.CreateFilmInput) (domain.FilmID, error) {
	input.Title = strings.TrimSpace(input.Title)

	if err := input.Validate(); err != nil {
		return 0, fmt.Errorf("validate film input: %w", err)
	}

	id, err := uc.repository.CreateFilm(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("create film: %w", err)
	}

	return id, nil
}
