package domain

import "errors"

var (
	ErrInvalidFilmID     = errors.New("film ID is invalid")
	ErrInvalidPersonID   = errors.New("person ID is invalid")
	ErrInvalidFirstName  = errors.New("first_name is invalid")
	ErrInvalidLastName   = errors.New("last_name is invalid")
	ErrInvalidBirthDate  = errors.New("birth_date is invalid")
	ErrPersonNotFound    = errors.New("person not found")
	ErrNilPersonsRepo    = errors.New("persons repository is nil")
	ErrInvalidPagination = errors.New("invalid pagination")
)
