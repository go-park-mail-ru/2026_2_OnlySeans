package usecase

import (
	"context"
	"errors"
	"fmt"
)

type ListFilmsUseCase struct {
	repository FilmRepository
}

func NewListFilmsUseCase(repository FilmRepository) (*ListFilmsUseCase, error) {
	if repository == nil {
		return nil, errors.New("films repository is nil")
	}
	return &ListFilmsUseCase{repository: repository}, nil
}

func (uc *ListFilmsUseCase) Run(ctx context.Context, input ListInput) (ListFilmsOutput, error) {
	films, total, err := uc.repository.ListFilms(ctx, input.Limit, input.Offset)
	if err != nil {
		return ListFilmsOutput{}, fmt.Errorf("list films: %w", err)
	}

	return ListFilmsOutput{
		Films: films,
		Total: total,
	}, nil
}
