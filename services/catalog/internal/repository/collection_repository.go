package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
)

type CollectionReader interface {
	ListCollections(ctx context.Context, limit, offset int) ([]domain.Collection, int, error)
	GetCollectionByID(ctx context.Context, id domain.CollectionID) (domain.Collection, error)
	ListCollectionFilms(ctx context.Context, collectionID domain.CollectionID, limit, offset int) ([]domain.Film, int, error)
}
