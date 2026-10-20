package usecase

import (
	"context"
	"errors"
	"fmt"
)

type ListCollectionsUseCase struct {
	repository CollectionRepository
}

func NewListCollectionsUseCase(repository CollectionRepository) (*ListCollectionsUseCase, error) {
	if repository == nil {
		return nil, errors.New("collections repository is nil")
	}
	return &ListCollectionsUseCase{repository: repository}, nil
}

func (uc *ListCollectionsUseCase) Run(ctx context.Context, input ListInput) (ListCollectionsOutput, error) {
	collections, total, err := uc.repository.ListCollections(ctx, input.Limit, input.Offset)
	if err != nil {
		return ListCollectionsOutput{}, fmt.Errorf("list collections: %w", err)
	}

	return ListCollectionsOutput{
		Collections: collections,
		Total:       total,
	}, nil
}
