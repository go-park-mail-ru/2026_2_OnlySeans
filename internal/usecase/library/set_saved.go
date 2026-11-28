package library

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type SetSavedUseCase struct {
	repo   domain.LibraryRepository
	canAdd func(domain.UserID) bool
}

func NewSetSavedUseCase(repo domain.LibraryRepository) (*SetSavedUseCase, error) {
	return NewSetSavedWithPolicy(repo, nil)
}

func NewSetSavedWithPolicy(repo domain.LibraryRepository, canAdd func(domain.UserID) bool) (*SetSavedUseCase, error) {
	if repo == nil {
		return nil, domain.ErrNilRepo
	}

	return &SetSavedUseCase{repo: repo, canAdd: canAdd}, nil
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

	if saved && uc.canAdd != nil && !uc.canAdd(user) {
		return domain.ErrLibraryAddDisabled
	}
	return uc.repo.SetSaved(ctx, user, list, film, saved)
}
