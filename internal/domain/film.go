package domain

import (
	"context"
	"time"
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

const (
	MinFilmRating int = 1
	MaxFilmRating int = 10
)

type FilmRating struct {
	AccountID UserID
	FilmID    FilmID
	Score     int
	CreatedAt time.Time
	UpdatedAt time.Time
}

type FilmRatingSummary struct {
	FilmID       FilmID
	AverageScore float64
	Votes        int64
}

type FilmRatingRepository interface {
	SetFilmRating(ctx context.Context, accountID UserID, filmID FilmID, score int) (FilmRating, error)
	GetFilmRatingSummary(ctx context.Context, filmID FilmID) (FilmRatingSummary, error)
	GetFilmRatingsSummaries(ctx context.Context, filmIDs []FilmID) ([]FilmRatingSummary, error)
	DeleteFilmRating(ctx context.Context, accountID UserID, filmID FilmID) error
}

type CollectionFilmRatingRepository interface {
	GetCollectionFilmRatingSummaries(ctx context.Context, collectionID CollectionID, limit, offset int) ([]FilmRatingSummary, int, error)
}

type FilmRatingCollectionRepository interface {
	FilmRatingRepository
	CollectionFilmRatingRepository
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

// фильмы и подборки вместе
type FilmsRepository interface {
	FilmRepository
	CollectionRepository
}
