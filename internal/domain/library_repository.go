package domain

import (
	"context"
	"encoding/json"
)

type LibraryRepository interface {
	Get(ctx context.Context, user UserID) (Library, error)
	SetSaved(ctx context.Context, user UserID, list List, film int64, saved bool) error
	Preferences(ctx context.Context, user UserID) (json.RawMessage, error)
	SavePreference(ctx context.Context, user UserID, key string, value json.RawMessage) error
}
