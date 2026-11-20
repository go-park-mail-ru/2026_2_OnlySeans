package domain

import "context"

type SearchRepository interface {
	SearchFilms(ctx context.Context, query string, limit int) ([]SearchFilm, error)
	SearchPersons(ctx context.Context, query string, limit int) ([]SearchPerson, error)
}
