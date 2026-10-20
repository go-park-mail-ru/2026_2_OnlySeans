package usecase

import (
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/entities"
)

type ListInput struct {
	Limit  int
	Offset int
}

type ListCollectionsOutput struct {
	Collections []entities.Collection
	Total       int
}

type GetCollectionOutput struct {
	Collection entities.Collection
	Films      []entities.Film
	Total      int
}

type ListFilmsOutput struct {
	Films []entities.Film
	Total int
}
