package library

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type SetSavedUseCase struct{ repo domain.LibraryRepository }

func NewSetSavedUseCase(repo domain.LibraryRepository) *SetSavedUseCase {
	return &SetSavedUseCase{repo: repo}
}

func (uc *SetSavedUseCase) Run(ctx context.Context, user domain.UserID, list domain.List, film int64, saved bool) error {
	if user <= 0 {
		return domain.ErrUnauthorized
	}

	if film <= 0 {
		return domain.ErrInvalidLibraryData
	}

	switch list {
	case domain.Favorite, domain.WatchLater, domain.Watched:
	default:
		return domain.ErrInvalidLibraryData
	}

	return uc.repo.SetSaved(ctx, user, list, film, saved)
}
