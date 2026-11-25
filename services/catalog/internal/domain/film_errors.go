package domain

import (
	"errors"
	"fmt"
)

var (
	ErrFilmNotFound            = errors.New("film not found")
	ErrFilmExists              = errors.New("film already exists")
	ErrCollectionNotFound      = errors.New("collection not found")
	ErrCollectionExists        = errors.New("collection already exists")
	ErrDuplicateCollectionFilm = errors.New("film is already in collection")
	ErrInvalidFilmInput        = errors.New("invalid film input")
	ErrInvalidFilmRating       = errors.New("invalid film rating")
	ErrRatingNotFound          = errors.New("rating not found")
)

func InvalidFilmRatingLimitsError() error {
	return fmt.Errorf("%w: must be from %d to %d", ErrInvalidFilmRating, MinFilmRating, MaxFilmRating)
}
