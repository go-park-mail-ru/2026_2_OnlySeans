package library

import (
	"context"
	"encoding/json"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type SavePreferenceUseCase struct{ repo domain.LibraryRepository }

func NewSavePreferenceUseCase(repo domain.LibraryRepository) *SavePreferenceUseCase {
	return &SavePreferenceUseCase{repo: repo}
}

func (uc *SavePreferenceUseCase) Run(ctx context.Context, user domain.UserID, key string, value json.RawMessage) error {
	if user <= 0 {
		return domain.ErrUnauthorized
	}

	if !allowedPreference(key, string(value)) {
		return domain.ErrInvalidLibraryData
	}

	return uc.repo.SavePreference(ctx, user, key, value)
}

func allowedPreference(key, value string) bool {
	switch key {
	case "frame:sidebar-collapsed":
		return value == "true" || value == "false"
	case "frame:catalog-view":
		return value == `"grid"` || value == `"list"`
	default:
		return false
	}
}
