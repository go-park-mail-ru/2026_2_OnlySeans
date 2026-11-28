package onlineconf

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
)

const (
	configColumns     = `c.id, s.name, c.key, c.title, c.description, c.value, c.version, c.created_at, c.updated_at`
	configFrom        = ` FROM oc_configs c JOIN oc_scopes s ON s.id=c.scope_id`
	configFilter      = ` WHERE ($1 = '' OR s.name=$1) AND ($2 = '' OR strpos(lower(c.key COLLATE pg_catalog."pg_c_utf8"),lower($2 COLLATE pg_catalog."pg_c_utf8")) > 0 OR strpos(lower(c.title COLLATE pg_catalog."pg_c_utf8"),lower($2 COLLATE pg_catalog."pg_c_utf8")) > 0)`
	getConfigQuery    = `SELECT ` + configColumns + configFrom + ` WHERE c.id=$1`
	countConfigsQuery = `SELECT count(*)` + configFrom + configFilter
	listConfigsQuery  = `SELECT ` + configColumns + configFrom + configFilter + ` ORDER BY c.id LIMIT $3 OFFSET $4`
	createConfigQuery = `INSERT INTO oc_configs(scope_id,key,title,description,value) VALUES($1,$2,$3,$4,$5) RETURNING id`
	updateConfigQuery = `UPDATE oc_configs SET title=$2,description=$3,value=$4,version=version+1 WHERE id=$1 AND version=$5 RETURNING id`
	deleteConfigQuery = `DELETE FROM oc_configs WHERE id=$1 AND version=$2`
)

func scanConfig(row rowScanner) (oc.Config, error) {
	var value oc.Config
	if err := row.Scan(&value.ID, &value.Scope, &value.Key, &value.Title, &value.Description, &value.Value, &value.Version, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return oc.Config{}, fmt.Errorf("scan onlineconf config: %w", err)
	}

	return value, nil
}

func (r *PostgresRepository) GetConfig(ctx context.Context, id int64) (*oc.Config, error) {
	value, err := scanConfig(r.db.QueryRowContext(ctx, getConfigQuery, id))
	if err != nil {
		return nil, repositoryError(err)
	}

	return &value, nil
}

func (r *PostgresRepository) ListConfigs(ctx context.Context, options oc.ListOptions) ([]oc.Config, int, error) {
	options, err := listOptions(options)
	if err != nil {
		return nil, 0, err
	}

	return listPage(ctx, r.db, countConfigsQuery, listConfigsQuery, []any{options.Scope, options.Query}, options.Limit, options.Offset, scanConfig)
}

func (r *PostgresRepository) CreateConfig(ctx context.Context, actorID domain.UserID, input oc.Config) (*oc.Config, error) {
	if err := input.ValidateForWrite(); err != nil {
		return nil, err
	}
	var result oc.Config

	err := r.mutate(ctx, actorID, func(m *mutation) error {
		scope, err := scopeID(ctx, m.tx, input.Scope)
		if err != nil {
			return err
		}

		var id int64
		if err := m.tx.QueryRowContext(ctx, createConfigQuery, scope, input.Key, input.Title, input.Description, []byte(input.Value)).Scan(&id); err != nil {
			return err
		}

		result, err = scanConfig(m.tx.QueryRowContext(ctx, getConfigQuery, id))
		if err != nil {
			return err
		}

		return m.audit(ctx, oc.ResourceConfig, result.ID, result.Scope, result.Key, oc.ActionCreate, nil, result)
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *PostgresRepository) UpdateConfig(ctx context.Context, actorID domain.UserID, input oc.Config) (*oc.Config, error) {
	if err := validateResource(input.ID, input.Version); err != nil {
		return nil, err
	}

	if err := input.ValidateForWrite(); err != nil {
		return nil, err
	}
	var result oc.Config

	err := r.mutate(ctx, actorID, func(m *mutation) error {
		before, err := scanConfig(m.tx.QueryRowContext(ctx, getConfigQuery, input.ID))
		if err != nil {
			return err
		}

		if err := checkVersion(input.Version, before.Version); err != nil {
			return err
		}

		if input.Scope != before.Scope || input.Key != before.Key {
			return oc.ErrImmutableIdentity
		}

		var id int64
		if err := m.tx.QueryRowContext(ctx, updateConfigQuery, input.ID, input.Title, input.Description, []byte(input.Value), input.Version).Scan(&id); err != nil {
			return err
		}

		result, err = scanConfig(m.tx.QueryRowContext(ctx, getConfigQuery, id))
		if err != nil {
			return err
		}

		return m.audit(ctx, oc.ResourceConfig, result.ID, result.Scope, result.Key, oc.ActionUpdate, before, result)
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *PostgresRepository) DeleteConfig(ctx context.Context, actorID domain.UserID, id, version int64) error {
	if err := validateResource(id, version); err != nil {
		return err
	}

	return r.mutate(ctx, actorID, func(m *mutation) error {
		before, err := scanConfig(m.tx.QueryRowContext(ctx, getConfigQuery, id))
		if err != nil {
			return err
		}

		if err := checkVersion(version, before.Version); err != nil {
			return err
		}

		if err := deleteVersioned(ctx, m.tx, deleteConfigQuery, id, version); err != nil {
			return err
		}

		return m.audit(ctx, oc.ResourceConfig, id, before.Scope, before.Key, oc.ActionDelete, before, nil)
	})
}
