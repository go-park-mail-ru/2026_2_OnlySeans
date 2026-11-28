package onlineconf

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
)

const (
	saltBytes   = 16
	flagColumns = `f.id,s.name,f.key,f.title,f.description,f.enabled,f.rollout_percent,f.salt,f.version,f.created_at,f.updated_at,
 COALESCE((SELECT jsonb_agg(o.account_id ORDER BY o.account_id) FROM oc_flag_overrides o WHERE o.flag_id=f.id AND o.enabled),'[]'::jsonb),
 COALESCE((SELECT jsonb_agg(o.account_id ORDER BY o.account_id) FROM oc_flag_overrides o WHERE o.flag_id=f.id AND NOT o.enabled),'[]'::jsonb)`
	flagFrom             = ` FROM oc_flags f JOIN oc_scopes s ON s.id=f.scope_id`
	flagFilter           = ` WHERE ($1 = '' OR s.name=$1) AND ($2 = '' OR strpos(lower(f.key COLLATE pg_catalog."pg_c_utf8"),lower($2 COLLATE pg_catalog."pg_c_utf8")) > 0 OR strpos(lower(f.title COLLATE pg_catalog."pg_c_utf8"),lower($2 COLLATE pg_catalog."pg_c_utf8")) > 0)`
	getFlagQuery         = `SELECT ` + flagColumns + flagFrom + ` WHERE f.id=$1`
	countFlagsQuery      = `SELECT count(*)` + flagFrom + flagFilter
	listFlagsBase        = `SELECT ` + flagColumns + flagFrom + flagFilter
	listFlagsQuery       = listFlagsBase + ` ORDER BY f.id LIMIT $3 OFFSET $4`
	createFlagQuery      = `INSERT INTO oc_flags(scope_id,key,title,description,enabled,rollout_percent,salt) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`
	updateFlagQuery      = `UPDATE oc_flags SET title=$2,description=$3,enabled=$4,rollout_percent=$5,version=version+1 WHERE id=$1 AND version=$6 RETURNING id`
	deleteFlagQuery      = `DELETE FROM oc_flags WHERE id=$1 AND version=$2`
	deleteOverridesQuery = `DELETE FROM oc_flag_overrides WHERE flag_id=$1`
	createOverrideQuery  = `INSERT INTO oc_flag_overrides(flag_id,account_id,enabled) SELECT $1,id,$3 FROM account WHERE id=$2 AND deleted_at IS NULL`
)

func scanFlag(row rowScanner) (oc.FeatureFlag, error) {
	var flag oc.FeatureFlag
	var forceOn, forceOff []byte

	err := row.Scan(&flag.ID, &flag.Scope, &flag.Key, &flag.Title, &flag.Description, &flag.Enabled, &flag.RolloutPercent, &flag.Salt, &flag.Version, &flag.CreatedAt, &flag.UpdatedAt, &forceOn, &forceOff)
	if err != nil {
		return oc.FeatureFlag{}, fmt.Errorf("scan onlineconf flag: %w", err)
	}

	if err := json.Unmarshal(forceOn, &flag.ForceOnUserIDs); err != nil {
		return flag, fmt.Errorf("decode force-on: %w", err)
	}

	if err := json.Unmarshal(forceOff, &flag.ForceOffUserIDs); err != nil {
		return flag, fmt.Errorf("decode force-off: %w", err)
	}

	return flag, nil
}

func (r *PostgresRepository) GetFlag(ctx context.Context, id int64) (*oc.FeatureFlag, error) {
	query := getFlagQuery
	args := []any{id}
	if uid := oc.ScopedUser(ctx); uid != 0 {
		query += ` AND oc_permitted($2,s.name,'flag',f.key,'read')`
		args = append(args, uid)
	}
	flag, err := scanFlag(r.db.QueryRowContext(ctx, query, args...))
	if err != nil {
		return nil, repositoryError(err)
	}

	return &flag, nil
}

func (r *PostgresRepository) ListFlags(ctx context.Context, options oc.ListOptions) ([]oc.FeatureFlag, int, error) {
	options, err := listOptions(options)
	if err != nil {
		return nil, 0, err
	}

	if uid := oc.ScopedUser(ctx); uid != 0 {
		filter := ` AND oc_permitted($3,s.name,'flag',f.key,'read')`
		return listPage(ctx, r.db, countFlagsQuery+filter, listFlagsBase+filter+" ORDER BY f.id LIMIT $4 OFFSET $5", []any{options.Scope, options.Query, uid}, options.Limit, options.Offset, scanFlag)
	}
	return listPage(ctx, r.db, countFlagsQuery, listFlagsQuery, []any{options.Scope, options.Query}, options.Limit, options.Offset, scanFlag)
}

func (r *PostgresRepository) CreateFlag(ctx context.Context, actorID domain.UserID, input oc.FeatureFlag) (*oc.FeatureFlag, error) {
	salt := make([]byte, saltBytes)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}

	input.Salt = hex.EncodeToString(salt)
	if err := input.Validate(); err != nil {
		return nil, err
	}
	var result oc.FeatureFlag

	err := r.mutate(ctx, actorID, func(m *mutation) error {
		if err := m.authorize(ctx, oc.ResourceFlag, input.Scope, input.Key, oc.ActionCreate); err != nil {
			return err
		}

		scope, err := scopeID(ctx, m.tx, input.Scope)
		if err != nil {
			return err
		}

		var id int64
		if err := m.tx.QueryRowContext(ctx, createFlagQuery, scope, input.Key, input.Title, input.Description, input.Enabled, input.RolloutPercent, input.Salt).Scan(&id); err != nil {
			return err
		}

		if err := replaceOverrides(ctx, m.tx, id, input); err != nil {
			return err
		}

		result, err = scanFlag(m.tx.QueryRowContext(ctx, getFlagQuery, id))
		if err != nil {
			return err
		}

		return m.audit(ctx, oc.ResourceFlag, result.ID, result.Scope, result.Key, oc.ActionCreate, nil, result)
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *PostgresRepository) UpdateFlag(ctx context.Context, actorID domain.UserID, input oc.FeatureFlag) (*oc.FeatureFlag, error) {
	if err := validateResource(input.ID, input.Version); err != nil {
		return nil, err
	}
	var result oc.FeatureFlag

	err := r.mutate(ctx, actorID, func(m *mutation) error {
		before, err := scanFlag(m.tx.QueryRowContext(ctx, getFlagQuery, input.ID))
		if err != nil {
			return err
		}

		if err := m.authorize(ctx, oc.ResourceFlag, before.Scope, before.Key, oc.ActionUpdate); err != nil {
			return err
		}

		if err := checkVersion(input.Version, before.Version); err != nil {
			return err
		}

		if input.Scope != before.Scope || input.Key != before.Key || (input.Salt != "" && input.Salt != before.Salt) {
			return oc.ErrImmutableIdentity
		}

		input.Salt = before.Salt
		if err := input.Validate(); err != nil {
			return err
		}

		var id int64
		if err := m.tx.QueryRowContext(ctx, updateFlagQuery, input.ID, input.Title, input.Description, input.Enabled, input.RolloutPercent, input.Version).Scan(&id); err != nil {
			return err
		}

		if err := replaceOverrides(ctx, m.tx, id, input); err != nil {
			return err
		}

		result, err = scanFlag(m.tx.QueryRowContext(ctx, getFlagQuery, id))
		if err != nil {
			return err
		}

		return m.audit(ctx, oc.ResourceFlag, result.ID, result.Scope, result.Key, oc.ActionUpdate, before, result)
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *PostgresRepository) DeleteFlag(ctx context.Context, actorID domain.UserID, id, version int64) error {
	if err := validateResource(id, version); err != nil {
		return err
	}

	return r.mutate(ctx, actorID, func(m *mutation) error {
		before, err := scanFlag(m.tx.QueryRowContext(ctx, getFlagQuery, id))
		if err != nil {
			return err
		}

		if err := m.authorize(ctx, oc.ResourceFlag, before.Scope, before.Key, oc.ActionDelete); err != nil {
			return err
		}

		if err := checkVersion(version, before.Version); err != nil {
			return err
		}

		if err := deleteVersioned(ctx, m.tx, deleteFlagQuery, id, version); err != nil {
			return err
		}

		return m.audit(ctx, oc.ResourceFlag, id, before.Scope, before.Key, oc.ActionDelete, before, nil)
	})
}

func replaceOverrides(ctx context.Context, tx *sql.Tx, id int64, flag oc.FeatureFlag) error {
	if _, err := tx.ExecContext(ctx, deleteOverridesQuery, id); err != nil {
		return err
	}

	for _, userID := range flag.ForceOnUserIDs {
		if err := insertOverride(ctx, tx, id, userID, true); err != nil {
			return err
		}
	}

	for _, userID := range flag.ForceOffUserIDs {
		if err := insertOverride(ctx, tx, id, userID, false); err != nil {
			return err
		}
	}

	return nil
}

func insertOverride(ctx context.Context, tx *sql.Tx, id int64, userID domain.UserID, enabled bool) error {
	result, err := tx.ExecContext(ctx, createOverrideQuery, id, userID, enabled)
	if err != nil {
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if count == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}
