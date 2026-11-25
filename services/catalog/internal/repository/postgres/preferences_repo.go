package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/storage"
)

const (
	preferencesQuery    = `SELECT preferences FROM account WHERE id = $1 AND deleted_at IS NULL`
	savePreferenceQuery = `
		UPDATE account SET preferences = jsonb_set(preferences, ARRAY[$2]::text[], $3::jsonb, true)
		WHERE id = $1 AND deleted_at IS NULL`
)

type PreferencesRepo struct {
	db         *sql.DB
	operations storage.OperationRecorder
}

type preferencesRow struct {
	SidebarCollapsed *bool               `json:"frame:sidebar-collapsed"`
	CatalogView      *domain.CatalogView `json:"frame:catalog-view"`
}

func NewPreferencesRepo(db *sql.DB, appLogger logger.Logger, observers ...storage.ExternalObserver) *PreferencesRepo {
	return &PreferencesRepo{
		db:         db,
		operations: storage.NewOperationRecorder(appLogger, observers...),
	}
}

func (r *PreferencesRepo) Preferences(ctx context.Context, user domain.UserID) (_ domain.Preferences, operationErr error) {
	operationLog := r.operations.Start(ctx, "postgres", "preferences.preferences", map[string]any{
		"user_id": user,
	})
	defer func() { operationLog.Finish(operationErr) }()

	var data []byte
	operationLog.Stage = "query"

	if err := r.db.QueryRowContext(ctx, preferencesQuery, user).Scan(&data); err != nil {
		return domain.Preferences{}, fmt.Errorf("read preferences: %w", err)
	}

	var row preferencesRow
	if err := json.Unmarshal(data, &row); err != nil {
		return domain.Preferences{}, fmt.Errorf("decode preferences: %w", err)
	}

	return domain.Preferences{
		SidebarCollapsed: row.SidebarCollapsed,
		CatalogView:      row.CatalogView,
	}, nil
}

func (r *PreferencesRepo) SavePreference(ctx context.Context, user domain.UserID, update domain.PreferenceUpdate) (operationErr error) {
	operationLog := r.operations.Start(ctx, "postgres", "preferences.save_preference", map[string]any{
		"user_id": user,
	})
	defer func() { operationLog.Finish(operationErr) }()

	if err := update.Validate(); err != nil {
		operationLog.Rejected = true
		return err
	}

	var key string
	var value any

	if update.SidebarCollapsed != nil {
		key = "frame:sidebar-collapsed"
		value = *update.SidebarCollapsed
	} else {
		key = "frame:catalog-view"
		value = *update.CatalogView
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode preference: %w", err)
	}

	operationLog.Stage = "query"

	if _, err := r.db.ExecContext(ctx, savePreferenceQuery, user, key, string(data)); err != nil {
		return fmt.Errorf("save preference: %w", err)
	}

	return nil
}
