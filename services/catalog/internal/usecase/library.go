package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository"
)

type LibraryUseCase interface {
	GetLibrary(ctx context.Context, userID domain.UserID) (domain.Library, error)
	SetSaved(ctx context.Context, userID domain.UserID, list domain.List, filmID domain.FilmID, saved bool) error
}

type libraryUseCase struct {
	library repository.LibraryRepository
}

func NewLibraryUseCase(library repository.LibraryRepository) LibraryUseCase {
	return &libraryUseCase{
		library: library,
	}
}

func (u *libraryUseCase) GetLibrary(ctx context.Context, userID domain.UserID) (domain.Library, error) {
	if userID <= 0 {
		return domain.Library{}, domain.ErrUnauthorized
	}

	library, err := u.library.Get(ctx, userID)
	if err != nil {
		return domain.Library{}, fmt.Errorf("get library: %w", err)
	}

	return library, nil
}

func (u *libraryUseCase) SetSaved(ctx context.Context, userID domain.UserID, list domain.List, filmID domain.FilmID, saved bool) error {
	if userID <= 0 {
		return domain.ErrUnauthorized
	}

	if filmID <= 0 {
		return domain.ErrInvalidLibraryData
	}

	switch list {
	case domain.Favorite, domain.WatchLater, domain.Watched:
	default:
		return domain.ErrInvalidLibraryData
	}

	if err := u.library.SetSaved(ctx, userID, list, filmID, saved); err != nil {
		return fmt.Errorf("set saved film: %w", err)
	}

	return nil
}
