package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type GetFilmUseCase struct {
	repository FilmRepository
}

func NewGetFilmUseCase(repository FilmRepository) (*GetFilmUseCase, error) {
	if repository == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	return &GetFilmUseCase{repository: repository}, nil
}

func (uc *GetFilmUseCase) Run(ctx context.Context, input GetFilmInput) (GetFilmOutput, error) {
	film, err := uc.repository.GetFilmByID(ctx, input.FilmID)
	if err != nil {
		return GetFilmOutput{}, fmt.Errorf("get film %d: %w", input.FilmID, err)
	}
	return NewGetFilmOutput(film), nil
}
