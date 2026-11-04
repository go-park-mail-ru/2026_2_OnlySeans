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
	MinFilmRating int16 = 1
	MaxFilmRating int16 = 10
)

type FilmRating struct {
	AccountID UserID
	FilmID    FilmID
	Score     int16
	CreatedAt time.Time
	UpdatedAt time.Time
}

type FilmRatingSummary struct {
	FilmID       FilmID
	AverageScore float64
	Votes        int64
}

type FilmRatingRepository interface {
	Set(ctx context.Context, accountID UserID, filmID FilmID, score int16) (FilmRating, error)
	GetSummary(ctx context.Context, filmID FilmID) (FilmRatingSummary, error)
	GetSummaries(ctx context.Context, filmIDs []FilmID) ([]FilmRatingSummary, error)
	Delete(ctx context.Context, accountID UserID, filmID FilmID) error
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
