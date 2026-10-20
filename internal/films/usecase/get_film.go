package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/entities"
)

type GetFilmUseCase struct {
	repository FilmRepository
}

func NewGetFilmUseCase(repository FilmRepository) (*GetFilmUseCase, error) {
	if repository == nil {
		return nil, errors.New("films repository is nil")
	}
	return &GetFilmUseCase{repository: repository}, nil
}

func (uc *GetFilmUseCase) Run(ctx context.Context, id entities.FilmID) (entities.Film, error) {
	film, err := uc.repository.GetFilmByID(ctx, id)
	if err != nil {
		return entities.Film{}, fmt.Errorf("get film %d: %w", id, err)
	}
	return film, nil
}
