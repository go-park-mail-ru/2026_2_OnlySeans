package domain

import "errors"

var (
	ErrFilmNotFound            = errors.New("film not found")
	ErrFilmExists              = errors.New("film already exists")
	ErrCollectionNotFound      = errors.New("collection not found")
	ErrCollectionExists        = errors.New("collection already exists")
	ErrDuplicateCollectionFilm = errors.New("film is already in collection")
	ErrNilCollectionsRepo      = errors.New("collections repository is nil")
	ErrNilFilmsRepo            = errors.New("films repository is nil")
	ErrInvalidFilmRating       = errors.New("film rating must be from 1 to 10")
	ErrNilFilmRatingsRepo      = errors.New("film ratings repository is nil")
	ErrRatingNotFound          = errors.New("rating not found")
)
