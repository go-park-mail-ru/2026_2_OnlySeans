package library

import (
	"context"
	"encoding/json"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type SavePreferenceUseCase struct{ repo domain.LibraryRepository }

func NewSavePreferenceUseCase(repo domain.LibraryRepository) (*SavePreferenceUseCase, error) {
	if repo == nil {
		return nil, domain.ErrNilRepo
	}

	return &SavePreferenceUseCase{repo: repo}, nil
}

func (uc *SavePreferenceUseCase) Run(ctx context.Context, user domain.UserID, key string, value json.RawMessage) error {
	if user <= 0 {
		return domain.ErrUnauthorized
	}

	if !allowedPreference(domain.PreferenceKey(key), domain.PreferenceValue(value)) {
		return domain.ErrInvalidLibraryData
	}

	return uc.repo.SavePreference(ctx, user, key, value)
}

func allowedPreference(key domain.PreferenceKey, value domain.PreferenceValue) bool {
	switch key {
	case domain.PreferenceSidebarCollapsed:
		return value == domain.PreferenceEnabled || value == domain.PreferenceDisabled
	case domain.PreferenceCatalogView:
		return value == domain.PreferenceGrid || value == domain.PreferenceList
	default:
		return false
	}
}
