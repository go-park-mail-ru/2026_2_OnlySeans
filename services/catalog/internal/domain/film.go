package domain

import (
	"fmt"
	"strings"
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
	ID   GenreID
	Name string
	Slug string
}

type Film struct {
	ID               FilmID
	Title            string
	OriginalTitle    *string
	FilmType         FilmType
	ReleaseYear      int16
	DurationMin      *int16
	AgeLimit         int16
	Description      *string
	PosterURL        *string
	PosterStorageKey *string
	TrailerURL       *string
	Genres           []Genre
}

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
	titleLen := utf8.RuneCountInString(strings.TrimSpace(input.Title))
	if titleLen < minFilmTitleLen || titleLen > maxFilmTitleLen {
		return ErrInvalidFilmInput
	}

	if input.ReleaseYear < minFilmProductionYear || input.ReleaseYear > maxFilmProductionYear {
		return ErrInvalidFilmInput
	}

	if input.FilmType != FilmTypeMovie && input.FilmType != FilmTypeSeries {
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
