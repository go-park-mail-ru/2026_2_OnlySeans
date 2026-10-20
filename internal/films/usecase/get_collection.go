package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/entities"
)

type GetCollectionUseCase struct {
	repository CollectionRepository
}

func NewGetCollectionUseCase(repository CollectionRepository) (*GetCollectionUseCase, error) {
	if repository == nil {
		return nil, errors.New("collections repository is nil")
	}
	return &GetCollectionUseCase{repository: repository}, nil
}

func (uc *GetCollectionUseCase) Run(ctx context.Context, id entities.CollectionID, input ListInput) (GetCollectionOutput, error) {
	collection, err := uc.repository.GetCollectionByID(ctx, id)
	if err != nil {
		return GetCollectionOutput{}, fmt.Errorf("get collection %d: %w", id, err)
	}

	films, total, err := uc.repository.ListCollectionFilms(ctx, id, input.Limit, input.Offset)
	if err != nil {
		return GetCollectionOutput{}, fmt.Errorf("list films of collection %d: %w", id, err)
	}
	return GetCollectionOutput{
		Collection: collection,
		Films:      films,
		Total:      total,
	}, nil
}
