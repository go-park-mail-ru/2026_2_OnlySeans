package usecase

import (
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type ListFilmsInput struct {
	Limit  int
	Offset int
}

func NewListFilmsInput(limit, offset int) ListFilmsInput {
	return ListFilmsInput{Limit: limit, Offset: offset}
}

type ListCollectionsOutput struct {
	Collections []domain.Collection
	Total       int
}

func NewListCollectionsOutput(collections []domain.Collection, total int) *ListCollectionsOutput {
	return &ListCollectionsOutput{Collections: collections, Total: total}
}

type ListCollectionsInput struct {
	Limit  int
	Offset int
}

func NewListCollectionsInput(limit, offset int) ListCollectionsInput {
	return ListCollectionsInput{Limit: limit, Offset: offset}
}

type GetCollectionInput struct {
	CollectionID domain.CollectionID
	Limit        int
	Offset       int
}

func NewGetCollectionInput(collectionID domain.CollectionID, limit, offset int) GetCollectionInput {
	return GetCollectionInput{CollectionID: collectionID, Limit: limit, Offset: offset}
}

type GetCollectionOutput struct {
	Collection domain.Collection
	Films      []domain.Film
	Total      int
}

func NewGetCollectionOutput(collection domain.Collection, films []domain.Film, total int) *GetCollectionOutput {
	return &GetCollectionOutput{Collection: collection, Films: films, Total: total}
}

type GetFilmInput struct {
	FilmID domain.FilmID
}

func NewGetFilmInput(filmID domain.FilmID) GetFilmInput {
	return GetFilmInput{FilmID: filmID}
}

type GetFilmOutput struct {
	Film domain.Film
}

func NewGetFilmOutput(film domain.Film) *GetFilmOutput {
	return &GetFilmOutput{Film: film}
}

type ListFilmsOutput struct {
	Films []domain.Film
	Total int
}

func NewListFilmsOutput(films []domain.Film, total int) *ListFilmsOutput {
	return &ListFilmsOutput{Films: films, Total: total}
}
