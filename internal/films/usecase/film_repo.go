package usecase

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/entities"
)

type FilmRepository interface {
	ListFilms(ctx context.Context, limit, offset int) ([]entities.Film, int, error)
	GetFilmByID(ctx context.Context, id entities.FilmID) (entities.Film, error)
}
