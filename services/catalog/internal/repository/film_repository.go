package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
)

type FilmReader interface {
	GetFilmByID(ctx context.Context, id domain.FilmID) (domain.Film, error)
}

type FilmLister interface {
	ListFilms(ctx context.Context, limit, offset int) ([]domain.Film, int, error)
}

type FilmWriter interface {
	CreateFilm(ctx context.Context, input domain.CreateFilmInput) (domain.FilmID, error)
	UpdateFilm(ctx context.Context, id domain.FilmID, input domain.CreateFilmInput) error
	DeleteFilm(ctx context.Context, id domain.FilmID) error
}
