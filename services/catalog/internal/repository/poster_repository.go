package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
)

type PosterRepository interface {
	GetPoster(ctx context.Context, filmID domain.FilmID) (*domain.PosterFile, error)
	DeletePoster(ctx context.Context, filmID domain.FilmID) error
	SavePoster(ctx context.Context, filmID domain.FilmID, userID domain.UserID, poster domain.PosterFile) error
}
