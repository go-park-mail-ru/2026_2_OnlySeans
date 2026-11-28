package onlineconf

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
)

const (
	loadStateQuery       = `SELECT revision,generated_at FROM oc_state WHERE id=1`
	saveStateQuery       = `UPDATE oc_state SET revision=$1,snapshot=$2,generated_at=$3 WHERE id=1`
	snapshotScopesQuery  = `SELECT ` + scopeColumns + ` FROM oc_scopes ORDER BY id`
	snapshotConfigsQuery = `SELECT ` + configColumns + configFrom + ` ORDER BY c.id`
	snapshotFlagsQuery   = `SELECT ` + flagColumns + flagFrom + ` ORDER BY f.id`
)

func loadSnapshot(ctx context.Context, tx *sql.Tx) (*oc.Snapshot, error) {
	var result oc.Snapshot
	if err := tx.QueryRowContext(ctx, loadStateQuery).Scan(&result.Revision, &result.GeneratedAt); err != nil {
		return nil, err
	}

	result.GeneratedAt = result.GeneratedAt.UTC()

	scopes, err := tx.QueryContext(ctx, snapshotScopesQuery)
	if err != nil {
		return nil, err
	}

	result.Scopes, err = collectRows(scopes, scanScope)
	if err != nil {
		return nil, err
	}

	configs, err := tx.QueryContext(ctx, snapshotConfigsQuery)
	if err != nil {
		return nil, err
	}

	result.Configs, err = collectRows(configs, scanConfig)
	if err != nil {
		return nil, err
	}

	flags, err := tx.QueryContext(ctx, snapshotFlagsQuery)
	if err != nil {
		return nil, err
	}

	result.Flags, err = collectRows(flags, scanFlag)
	if err != nil {
		return nil, err
	}

	for _, scope := range result.Scopes {
		if err := scope.Validate(); err != nil {
			return nil, fmt.Errorf("%w: %w", oc.ErrInvalidSnapshot, err)
		}
	}

	for _, config := range result.Configs {
		if err := config.Validate(); err != nil {
			return nil, fmt.Errorf("%w: %w", oc.ErrInvalidSnapshot, err)
		}
	}

	for _, flag := range result.Flags {
		if err := flag.Validate(); err != nil {
			return nil, fmt.Errorf("%w: %w", oc.ErrInvalidSnapshot, err)
		}
	}

	return &result, nil
}

func (r *PostgresRepository) LoadSnapshot(ctx context.Context) (*oc.Snapshot, error) {
	return readTransaction(ctx, r.db, func(tx *sql.Tx) (*oc.Snapshot, error) {
		snapshot, err := loadSnapshot(ctx, tx)
		if err != nil {
			return nil, err
		}

		if err := oc.NewSnapshotPayload(snapshot).ValidateSize(); err != nil {
			return nil, err
		}

		return snapshot, nil
	})
}

func (m *mutation) saveSnapshot(ctx context.Context) error {
	snapshot, err := loadSnapshot(ctx, m.tx)
	if err != nil {
		return err
	}

	snapshot.Revision = m.revision
	snapshot.GeneratedAt = time.Now().UTC().Truncate(time.Microsecond)

	if err := oc.NewSnapshotPayload(snapshot).ValidateSize(); err != nil {
		return fmt.Errorf("validate snapshot before commit: %w", err)
	}

	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("encode snapshot: %w", err)
	}

	if _, err := m.tx.ExecContext(ctx, saveStateQuery, snapshot.Revision, encoded, snapshot.GeneratedAt); err != nil {
		return fmt.Errorf("save onlineconf snapshot: %w", err)
	}

	return nil
}
