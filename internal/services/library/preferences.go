package library

import (
	"context"
	"encoding/json"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

const (
	preferencesQuery    = `SELECT preferences FROM account WHERE id = $1 AND deleted_at IS NULL`
	savePreferenceQuery = `
		UPDATE account SET preferences = jsonb_set(preferences, ARRAY[$2]::text[], $3::jsonb, true)
		WHERE id = $1 AND deleted_at IS NULL`
)

var preferenceValues = map[string][]string{
	"frame:sidebar-collapsed": {"true", "false"},
	"frame:catalog-view":      {`"grid"`, `"list"`},
}

func (r *PostgresRepo) Preferences(ctx context.Context, user domain.UserID) (json.RawMessage, error) {
	var data json.RawMessage
	err := r.db.QueryRowContext(ctx, preferencesQuery, user).Scan(&data)
	return data, err
}

func (r *PostgresRepo) SavePreference(ctx context.Context, user domain.UserID, key string, value json.RawMessage) error {
	if !allowedPreference(key, string(value)) {
		return ErrInvalid
	}
	_, err := r.db.ExecContext(ctx, savePreferenceQuery, user, key, string(value))
	return err
}

func allowedPreference(key, value string) bool {
	for _, allowed := range preferenceValues[key] {
		if value == allowed {
			return true
		}
	}
	return false
}
