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
	ErrNilCollectionsRepo      = errors.New("collections repository is nil")
	ErrNilFilmsRepo            = errors.New("films repository is nil")
	ErrInvalidFilmRating       = errors.New("invalid film rating")
	ErrNilFilmRatingsRepo      = errors.New("film ratings repository is nil")
	ErrNilCollectionRatingRepo = errors.New("collection film ratings repository is nil")
	ErrRatingNotFound          = errors.New("rating not found")
)

func InvalidFilmRatingLimitsError() error {
	return fmt.Errorf("%w: must be from %d to %d", ErrInvalidFilmRating, MinFilmRating, MaxFilmRating)
}
