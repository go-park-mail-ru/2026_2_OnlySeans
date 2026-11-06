package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

// обновляет данные существующего фильма
type UpdateFilmUseCase struct {
	repository domain.FilmUpdater
}

func NewUpdateFilmUseCase(repository domain.FilmUpdater) (*UpdateFilmUseCase, error) {
	if repository == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	return &UpdateFilmUseCase{repository: repository}, nil
}

func (uc *UpdateFilmUseCase) Run(ctx context.Context, id domain.FilmID, input domain.CreateFilmInput) error {
	input.Title = strings.TrimSpace(input.Title)
	err := input.Validate()
	if err != nil {
		return fmt.Errorf("validate film input: %w", err)
	}
	err = uc.repository.UpdateFilm(ctx, id, input)
	if err != nil {
		return fmt.Errorf("update film: %w", err)
	}
	return nil
}
