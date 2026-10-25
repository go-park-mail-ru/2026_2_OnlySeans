package domain

import (
	"context"
)

type FilmID int64

type GenreID int64

type FilmType string

const (
	FilmTypeMovie  FilmType = "movie"
	FilmTypeSeries FilmType = "series"
)

type Genre struct {
	ID   GenreID `db:"id"`
	Name string  `db:"name"`
	Slug string  `db:"slug"`
}

type Film struct {
	ID            FilmID   `db:"id"`
	Title         string   `db:"title"`
	OriginalTitle *string  `db:"original_title"`
	FilmType      FilmType `db:"film_type"`
	ReleaseYear   int16    `db:"release_year"`
	DurationMin   *int16   `db:"duration_min"`
	AgeLimit      int16    `db:"age_limit"`
	Description   *string  `db:"description"`
	PosterURL     *string  `db:"poster_url"`
	TrailerURL    *string  `db:"trailer_url"`
	Genres        []Genre
}

type FilmRepository interface {
	ListFilms(ctx context.Context, limit, offset int) ([]Film, int, error)
	GetFilmByID(ctx context.Context, id FilmID) (Film, error)
}

type CollectionRepository interface {
	ListCollections(ctx context.Context, limit, offset int) ([]Collection, int, error)
	GetCollectionByID(ctx context.Context, id CollectionID) (Collection, error)
	ListCollectionFilms(ctx context.Context, collectionID CollectionID, limit, offset int) ([]Film, int, error)
}
