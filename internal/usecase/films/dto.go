package usecase

import (
	entities "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type ListFilmsInput struct {
	Limit  int
	Offset int
}

func NewListFilmsInput(limit, offset int) ListFilmsInput {
	return ListFilmsInput{Limit: limit, Offset: offset}
}

type ListCollectionsOutput struct {
	Collections []entities.Collection
	Total       int
}

func NewListCollectionsOutput(collections []entities.Collection, total int) ListCollectionsOutput {
	return ListCollectionsOutput{Collections: collections, Total: total}
}

type ListCollectionsInput struct {
	Limit  int
	Offset int
}

func NewListCollectionsInput(limit, offset int) ListCollectionsInput {
	return ListCollectionsInput{Limit: limit, Offset: offset}
}

type GetCollectionInput struct {
	CollectionID entities.CollectionID
	Limit        int
	Offset       int
}

func NewGetCollectionInput(collectionID entities.CollectionID, limit, offset int) GetCollectionInput {
	return GetCollectionInput{CollectionID: collectionID, Limit: limit, Offset: offset}
}

type GetCollectionOutput struct {
	Collection entities.Collection
	Films      []entities.Film
	Total      int
}

func NewGetCollectionOutput(collection entities.Collection, films []entities.Film, total int) GetCollectionOutput {
	return GetCollectionOutput{Collection: collection, Films: films, Total: total}
}

type GetFilmInput struct {
	FilmID entities.FilmID
}

func NewGetFilmInput(filmID entities.FilmID) GetFilmInput {
	return GetFilmInput{FilmID: filmID}
}

type GetFilmOutput struct {
	Film entities.Film
}

func NewGetFilmOutput(film entities.Film) GetFilmOutput {
	return GetFilmOutput{Film: film}
}

type ListFilmsOutput struct {
	Films []entities.Film
	Total int
}

func NewListFilmsOutput(films []entities.Film, total int) ListFilmsOutput {
	return ListFilmsOutput{Films: films, Total: total}
}
