package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type ListFilmsUseCase struct {
	repository FilmRepository
}

func NewListFilmsUseCase(repository FilmRepository) (*ListFilmsUseCase, error) {
	if repository == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	return &ListFilmsUseCase{repository: repository}, nil
}

func (uc *ListFilmsUseCase) Run(ctx context.Context, input ListFilmsInput) (ListFilmsOutput, error) {
	films, total, err := uc.repository.ListFilms(ctx, input.Limit, input.Offset)
	if err != nil {
		return ListFilmsOutput{}, fmt.Errorf("list films: %w", err)
	}

	return NewListFilmsOutput(films, total), nil
}
