package usecase

import (
	"errors"
	"fmt"
)

const (
	DefaultLimit  = 20
	DefaultOffset = 0
	MinLimit      = 1
	MaxLimit      = 100
)

var (
	ErrInvalidLimit        = errors.New("invalid limit")
	ErrInvalidOffset       = errors.New("offset must be a non-negative integer")
	ErrInvalidFilmID       = errors.New("film id must be a positive integer")
	ErrInvalidCollectionID = errors.New("collection id must be a positive integer")
)

func validatePagination(limit, offset int) error {
	if limit < MinLimit || limit > MaxLimit {
		return fmt.Errorf("%w: must be an integer from %d to %d", ErrInvalidLimit, MinLimit, MaxLimit)
	}

	if offset < 0 {
		return ErrInvalidOffset
	}

	return nil
}
