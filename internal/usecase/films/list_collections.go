package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type ListCollectionsUseCase struct {
	repository domain.CollectionRepository
}

func NewListCollectionsUseCase(repository domain.CollectionRepository) (*ListCollectionsUseCase, error) {
	if repository == nil {
		return nil, domain.ErrNilCollectionsRepo
	}
	return &ListCollectionsUseCase{repository: repository}, nil
}

func (uc *ListCollectionsUseCase) Run(ctx context.Context, input ListCollectionsInput) (*ListCollectionsOutput, error) {
	collections, total, err := uc.repository.ListCollections(ctx, input.Limit, input.Offset)
	if err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}

	return NewListCollectionsOutput(collections, total), nil
}
