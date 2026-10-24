package usecase

import (
	"context"

	entities "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type CollectionRepository interface {
	ListCollections(ctx context.Context, limit, offset int) ([]entities.Collection, int, error)
	GetCollectionByID(ctx context.Context, id entities.CollectionID) (entities.Collection, error)
	ListCollectionFilms(ctx context.Context, collectionID entities.CollectionID, limit, offset int) ([]entities.Film, int, error)
}
