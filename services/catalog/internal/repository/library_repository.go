package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
)

type LibraryRepository interface {
	Get(ctx context.Context, user domain.UserID) (domain.Library, error)
	SetSaved(ctx context.Context, user domain.UserID, list domain.List, film domain.FilmID, saved bool) error
}
