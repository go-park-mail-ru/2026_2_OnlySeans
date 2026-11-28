package onlineconf

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
)

const (
	scopeColumns     = `id, name, title, description, version, created_at, updated_at`
	getScopeQuery    = `SELECT ` + scopeColumns + ` FROM oc_scopes WHERE id = $1`
	createScopeQuery = `INSERT INTO oc_scopes(name,title,description) VALUES($1,$2,$3) RETURNING ` + scopeColumns
	updateScopeQuery = `UPDATE oc_scopes SET title=$2, description=$3, version=version+1 WHERE id=$1 AND version=$4 RETURNING ` + scopeColumns
	deleteScopeQuery = `DELETE FROM oc_scopes WHERE id=$1 AND version=$2`
	scopeFilter      = ` WHERE ($1 = '' OR name = $1) AND ($2 = '' OR strpos(lower(name COLLATE pg_catalog."pg_c_utf8"),lower($2 COLLATE pg_catalog."pg_c_utf8")) > 0 OR strpos(lower(title COLLATE pg_catalog."pg_c_utf8"),lower($2 COLLATE pg_catalog."pg_c_utf8")) > 0)`
	countScopesQuery = `SELECT count(*) FROM oc_scopes` + scopeFilter
	listScopesBase   = `SELECT ` + scopeColumns + ` FROM oc_scopes` + scopeFilter
	listScopesQuery  = listScopesBase + ` ORDER BY id LIMIT $3 OFFSET $4`
)

func scanScope(row rowScanner) (oc.Scope, error) {
	var value oc.Scope
	if err := row.Scan(&value.ID, &value.Name, &value.Title, &value.Description, &value.Version, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return oc.Scope{}, fmt.Errorf("scan onlineconf scope: %w", err)
	}

	return value, nil
}

func (r *PostgresRepository) GetScope(ctx context.Context, id int64) (*oc.Scope, error) {
	query := getScopeQuery
	args := []any{id}
	if uid := oc.ScopedUser(ctx); uid != 0 {
		query += ` AND oc_permitted($2,name,'scope','','read')`
		args = append(args, uid)
	}
	value, err := scanScope(r.db.QueryRowContext(ctx, query, args...))
	if err != nil {
		return nil, repositoryError(err)
	}

	return &value, nil
}

func (r *PostgresRepository) ListScopes(ctx context.Context, options oc.ListOptions) ([]oc.Scope, int, error) {
	options, err := listOptions(options)
	if err != nil {
		return nil, 0, err
	}

	if uid := oc.ScopedUser(ctx); uid != 0 {
		filter := ` AND oc_permitted($3,oc_scopes.name,'scope','','read')`
		return listPage(ctx, r.db, countScopesQuery+filter, listScopesBase+filter+" ORDER BY id LIMIT $4 OFFSET $5", []any{options.Scope, options.Query, uid}, options.Limit, options.Offset, scanScope)
	}
	return listPage(ctx, r.db, countScopesQuery, listScopesQuery, []any{options.Scope, options.Query}, options.Limit, options.Offset, scanScope)
}

func (r *PostgresRepository) CreateScope(ctx context.Context, actorID domain.UserID, input oc.Scope) (*oc.Scope, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	var result oc.Scope

	err := r.mutate(ctx, actorID, func(m *mutation) error {
		if err := m.authorize(ctx, oc.ResourceScope, input.Name, "", oc.ActionCreate); err != nil {
			return err
		}

		var err error

		result, err = scanScope(m.tx.QueryRowContext(ctx, createScopeQuery, input.Name, input.Title, input.Description))
		if err != nil {
			return err
		}

		return m.audit(ctx, oc.ResourceScope, result.ID, result.Name, "", oc.ActionCreate, nil, result)
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *PostgresRepository) UpdateScope(ctx context.Context, actorID domain.UserID, input oc.Scope) (*oc.Scope, error) {
	if err := validateResource(input.ID, input.Version); err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}
	var result oc.Scope

	err := r.mutate(ctx, actorID, func(m *mutation) error {
		before, err := scanScope(m.tx.QueryRowContext(ctx, getScopeQuery, input.ID))
		if err != nil {
			return err
		}

		if err := m.authorize(ctx, oc.ResourceScope, before.Name, "", oc.ActionUpdate); err != nil {
			return err
		}

		if err := checkVersion(input.Version, before.Version); err != nil {
			return err
		}

		if input.Name != before.Name {
			return oc.ErrImmutableIdentity
		}

		result, err = scanScope(m.tx.QueryRowContext(ctx, updateScopeQuery, input.ID, input.Title, input.Description, input.Version))
		if err != nil {
			return err
		}

		return m.audit(ctx, oc.ResourceScope, result.ID, result.Name, "", oc.ActionUpdate, before, result)
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *PostgresRepository) DeleteScope(ctx context.Context, actorID domain.UserID, id, version int64) error {
	if err := validateResource(id, version); err != nil {
		return err
	}

	return r.mutate(ctx, actorID, func(m *mutation) error {
		before, err := scanScope(m.tx.QueryRowContext(ctx, getScopeQuery, id))
		if err != nil {
			return err
		}

		if err := m.authorize(ctx, oc.ResourceScope, before.Name, "", oc.ActionDelete); err != nil {
			return err
		}

		if err := checkVersion(version, before.Version); err != nil {
			return err
		}

		if err := deleteVersioned(ctx, m.tx, deleteScopeQuery, id, version); err != nil {
			return err
		}

		return m.audit(ctx, oc.ResourceScope, id, before.Name, "", oc.ActionDelete, before, nil)
	})
}

func deleteVersioned(ctx context.Context, tx *sql.Tx, query string, id, version int64) error {
	result, err := tx.ExecContext(ctx, query, id, version)
	if err != nil {
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if count == 0 {
		return oc.ErrVersionConflict
	}

	return nil
}
