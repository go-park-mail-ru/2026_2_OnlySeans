package library

import (
	"context"
	"encoding/json"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type GetPreferencesUseCase struct{ repo domain.LibraryRepository }

func NewGetPreferencesUseCase(repo domain.LibraryRepository) *GetPreferencesUseCase {
	return &GetPreferencesUseCase{repo: repo}
}

func (uc *GetPreferencesUseCase) Run(ctx context.Context, user domain.UserID) (json.RawMessage, error) {
	if user <= 0 {
		return nil, domain.ErrUnauthorized
	}

	return uc.repo.Preferences(ctx, user)
}
