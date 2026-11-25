package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository"
)

type PreferencesUseCase interface {
	GetPreferences(ctx context.Context, userID domain.UserID) (domain.Preferences, error)
	SavePreference(ctx context.Context, userID domain.UserID, update domain.PreferenceUpdate) error
}

type preferencesUseCase struct {
	preferences repository.PreferencesRepository
}

func NewPreferencesUseCase(preferences repository.PreferencesRepository) PreferencesUseCase {
	return &preferencesUseCase{
		preferences: preferences,
	}
}

func (u *preferencesUseCase) GetPreferences(ctx context.Context, userID domain.UserID) (domain.Preferences, error) {
	if userID <= 0 {
		return domain.Preferences{}, domain.ErrUnauthorized
	}

	preferences, err := u.preferences.Preferences(ctx, userID)
	if err != nil {
		return domain.Preferences{}, fmt.Errorf("get preferences: %w", err)
	}

	return preferences, nil
}

func (u *preferencesUseCase) SavePreference(ctx context.Context, userID domain.UserID, update domain.PreferenceUpdate) error {
	if userID <= 0 {
		return domain.ErrUnauthorized
	}

	if err := update.Validate(); err != nil {
		return err
	}

	if err := u.preferences.SavePreference(ctx, userID, update); err != nil {
		return fmt.Errorf("save preference: %w", err)
	}

	return nil
}
