package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
)

type PreferencesRepository interface {
	Preferences(ctx context.Context, user domain.UserID) (domain.Preferences, error)
	SavePreference(ctx context.Context, user domain.UserID, update domain.PreferenceUpdate) error
}
