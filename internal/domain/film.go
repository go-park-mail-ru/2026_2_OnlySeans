package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

type FilmID int64

type GenreID int64

type FilmType string

const (
	FilmTypeMovie  FilmType = "movie"
	FilmTypeSeries FilmType = "series"
)

const (
	minFilmTitleLen       = 1
	maxFilmTitleLen       = 255
	minFilmProductionYear = 1888
	maxFilmProductionYear = 2200
	minFilmDurationMin    = 1
	maxFilmDurationMin    = 6000
	maxFilmDescriptionLen = 10000
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

// данные для одной карточки
type CreateFilmInput struct {
	Title         string
	OriginalTitle *string
	FilmType      FilmType
	ReleaseYear   int16
	DurationMin   *int16
	AgeLimit      int16
	Description   *string
}

func (input CreateFilmInput) Validate() error {
	trimmedTitle := strings.TrimSpace(input.Title)
	titleLen := utf8.RuneCountInString(trimmedTitle)
	if titleLen < minFilmTitleLen || titleLen > maxFilmTitleLen {
		return ErrInvalidFilmInput
	}

	if input.ReleaseYear < minFilmProductionYear || input.ReleaseYear > maxFilmProductionYear {
		return ErrInvalidFilmInput
	}

	if FilmType(input.FilmType) != FilmTypeMovie && FilmType(input.FilmType) != FilmTypeSeries {
		return ErrInvalidFilmInput
	}

	switch input.AgeLimit {
	case 0, 6, 12, 16, 18:
	default:
		return fmt.Errorf("invalid age limit: %w", ErrInvalidFilmInput)
	}

	if input.DurationMin != nil {
		if input.FilmType == FilmTypeSeries {
			return fmt.Errorf("series cannot have duration: %w", ErrInvalidFilmInput)
		}
		duration := *input.DurationMin
		if duration < minFilmDurationMin || duration > maxFilmDurationMin {
			return fmt.Errorf("invalid duration: %w", ErrInvalidFilmInput)
		}
	}

	if input.OriginalTitle != nil {
		originalTitleLen := utf8.RuneCountInString(*input.OriginalTitle)
		if originalTitleLen < minFilmTitleLen || originalTitleLen > maxFilmTitleLen {
			return fmt.Errorf("invalid original title: %w", ErrInvalidFilmInput)
		}
	}

	if input.Description != nil {
		descriptionLen := utf8.RuneCountInString(*input.Description)
		if descriptionLen > maxFilmDescriptionLen {
			return fmt.Errorf("description is too long: %w", ErrInvalidFilmInput)
		}
	}

	return nil
}

// создание фильма в репозитории
type FilmCreator interface {
	CreateFilm(ctx context.Context, input CreateFilmInput) (FilmID, error)
}

type FilmUpdater interface {
	UpdateFilm(ctx context.Context, id FilmID, input CreateFilmInput) error
}

type FilmDeleter interface {
	DeleteFilm(ctx context.Context, id FilmID) error
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
	FilmCreator
	FilmUpdater
	FilmDeleter
}
