package library

import (
	"context"
	"encoding/json"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
)

const (
	preferencesQuery    = `SELECT preferences FROM account WHERE id = $1 AND deleted_at IS NULL`
	savePreferenceQuery = `
		UPDATE account SET preferences = jsonb_set(preferences, ARRAY[$2]::text[], $3::jsonb, true)
		WHERE id = $1 AND deleted_at IS NULL`
)

func (r *PostgresRepo) Preferences(ctx context.Context, user domain.UserID) (_ json.RawMessage, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.log, "postgres", "preferences.preferences", map[string]any{"user_id": user})
	defer func() { operationLog.Finish(operationErr) }()

	var data json.RawMessage
	operationLog.Stage = "query"
	err := r.db.QueryRowContext(ctx, preferencesQuery, user).Scan(&data)
	return data, err
}

func (r *PostgresRepo) SavePreference(ctx context.Context, user domain.UserID, key string, value json.RawMessage) (operationErr error) {
	operationLog := storage.StartOperation(ctx, r.log, "postgres", "preferences.save_preference", map[string]any{"user_id": user})
	defer func() { operationLog.Finish(operationErr) }()

	operationLog.Stage = "query"
	_, err := r.db.ExecContext(ctx, savePreferenceQuery, user, key, string(value))
	return err
}
