package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository"
)

type CollectionUseCase interface {
	ListCollections(ctx context.Context, limit, offset int) ([]domain.Collection, int, error)
	GetCollection(ctx context.Context, id domain.CollectionID, limit, offset int) (CollectionDetails, error)
}

type CollectionDetails struct {
	Collection domain.Collection
	Films      []domain.Film
	Total      int
}

type collectionUseCase struct {
	collections repository.CollectionReader
}

func NewCollectionUseCase(collections repository.CollectionReader) CollectionUseCase {
	return &collectionUseCase{
		collections: collections,
	}
}

func (u *collectionUseCase) ListCollections(ctx context.Context, limit, offset int) ([]domain.Collection, int, error) {
	if err := validatePagination(limit, offset); err != nil {
		return nil, 0, err
	}

	collections, total, err := u.collections.ListCollections(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list collections: %w", err)
	}

	return collections, total, nil
}

func (u *collectionUseCase) GetCollection(ctx context.Context, id domain.CollectionID, limit, offset int) (CollectionDetails, error) {
	if id <= 0 {
		return CollectionDetails{}, ErrInvalidCollectionID
	}

	if err := validatePagination(limit, offset); err != nil {
		return CollectionDetails{}, err
	}

	collection, err := u.collections.GetCollectionByID(ctx, id)
	if err != nil {
		return CollectionDetails{}, fmt.Errorf("get collection: %w", err)
	}

	films, total, err := u.collections.ListCollectionFilms(ctx, id, limit, offset)
	if err != nil {
		return CollectionDetails{}, fmt.Errorf("list collection films: %w", err)
	}

	return CollectionDetails{
		Collection: collection,
		Films:      films,
		Total:      total,
	}, nil
}
