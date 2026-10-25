package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type GetCollectionUseCase struct {
	repository domain.CollectionRepository
}

func NewGetCollectionUseCase(repository domain.CollectionRepository) (*GetCollectionUseCase, error) {
	if repository == nil {
		return nil, errors.New("collections repository is nil")
	}
	return &GetCollectionUseCase{repository: repository}, nil
}

func (uc *GetCollectionUseCase) Run(ctx context.Context, input GetCollectionInput) (*GetCollectionOutput, error) {
	collection, err := uc.repository.GetCollectionByID(ctx, input.CollectionID)
	if err != nil {
		return nil, fmt.Errorf("get collection %d: %w", input.CollectionID, err)
	}

	films, total, err := uc.repository.ListCollectionFilms(ctx, input.CollectionID, input.Limit, input.Offset)
	if err != nil {
		return nil, fmt.Errorf("list films of collection %d: %w", input.CollectionID, err)
	}
	return NewGetCollectionOutput(collection, films, total), nil
}
