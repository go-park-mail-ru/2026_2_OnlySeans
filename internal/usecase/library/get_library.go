package library

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type GetLibraryUseCase struct{ repo domain.LibraryRepository }

func NewGetLibraryUseCase(repo domain.LibraryRepository) (*GetLibraryUseCase, error) {
	if repo == nil {
		return nil, domain.ErrNilRepo
	}

	return &GetLibraryUseCase{repo: repo}, nil
}

func (uc *GetLibraryUseCase) Run(ctx context.Context, user domain.UserID) (domain.Library, error) {
	if user <= 0 {
		return domain.Library{}, domain.ErrUnauthorized
	}

	return uc.repo.Get(ctx, user)
}
