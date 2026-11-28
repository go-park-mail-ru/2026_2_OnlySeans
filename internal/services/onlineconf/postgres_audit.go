package onlineconf

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"

	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
)

const (
	createAuditQuery = `INSERT INTO oc_audit(actor_id,actor_name,resource_type,resource_id,scope,key,action,before,after,revision) VALUES($1,$2,$3,$4,$5,NULLIF($6,''),$7,$8,$9,$10)`
	auditColumns     = `id,actor_id,actor_name,resource_type,resource_id,scope,COALESCE(key,''),action,before,after,revision,created_at`
	auditFilter      = ` WHERE ($1 = '' OR resource_type=$1) AND ($2::bigint=0 OR resource_id=$2) AND ($3::bigint=0 OR actor_id=$3)`
	countAuditQuery  = `SELECT count(*) FROM oc_audit` + auditFilter
	listAuditQuery   = `SELECT ` + auditColumns + ` FROM oc_audit` + auditFilter + ` ORDER BY created_at DESC,id DESC LIMIT $4 OFFSET $5`
	auditSaltField   = "salt"
)

func auditJSON(value any) ([]byte, error) {
	if flag, ok := value.(oc.FeatureFlag); ok {
		encoded, err := json.Marshal(flag)
		if err != nil {
			return nil, err
		}

		var fields map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &fields); err != nil {
			return nil, err
		}

		delete(fields, auditSaltField)

		return json.Marshal(fields)
	}

	return json.Marshal(value)
}

func (m *mutation) authorize(ctx context.Context, kind oc.ResourceKind, scope, key string, action oc.AuditAction) error {
	if uid := oc.ScopedUser(ctx); uid != 0 {
		var allowed bool
		if err := m.tx.QueryRowContext(ctx, "SELECT oc_permitted($1,$2,$3,$4,$5)", uid, scope, kind, key, action).Scan(&allowed); err != nil {
			return err
		}
		if !allowed {
			return domain.ErrForbidden
		}
	}
	return nil
}

func (m *mutation) audit(ctx context.Context, kind oc.ResourceKind, id int64, scope, key string, action oc.AuditAction, before, after any) error {
	if err := m.authorize(ctx, kind, scope, key, action); err != nil {
		return err
	}
	previous, err := auditJSON(before)
	if err != nil {
		return fmt.Errorf("encode audit before: %w", err)
	}

	next, err := auditJSON(after)
	if err != nil {
		return fmt.Errorf("encode audit after: %w", err)
	}

	if _, err := m.tx.ExecContext(ctx, createAuditQuery, m.actorID, m.actorName, kind, id, scope, key, action, previous, next, m.revision); err != nil {
		return fmt.Errorf("save onlineconf audit: %w", err)
	}

	return nil
}

func scanAudit(row rowScanner) (oc.AuditEntry, error) {
	var entry oc.AuditEntry
	if err := row.Scan(&entry.ID, &entry.ActorID, &entry.ActorName, &entry.ResourceKind, &entry.ResourceID, &entry.Scope, &entry.Key, &entry.Action, &entry.Before, &entry.After, &entry.Revision, &entry.CreatedAt); err != nil {
		return oc.AuditEntry{}, fmt.Errorf("scan onlineconf audit: %w", err)
	}

	entry.CreatedAt = entry.CreatedAt.UTC()

	return entry, nil
}

func (r *PostgresRepository) ListAudit(ctx context.Context, filter oc.AuditFilter) ([]oc.AuditEntry, int, error) {
	switch filter.ResourceKind {
	case "", oc.ResourceScope, oc.ResourceConfig, oc.ResourceFlag:
	default:
		return nil, 0, oc.ErrInvalidAuditFilter
	}

	if filter.ResourceID < 0 || filter.ResourceID > oc.MaxSafeID || filter.ActorID < 0 || filter.ActorID > oc.MaxSafeID {
		return nil, 0, oc.ErrInvalidAuditFilter
	}

	options, err := listOptions(oc.ListOptions{Limit: filter.Limit, Offset: filter.Offset})
	if err != nil {
		return nil, 0, err
	}

	return listPage(ctx, r.db, countAuditQuery, listAuditQuery, []any{filter.ResourceKind, filter.ResourceID, filter.ActorID}, options.Limit, options.Offset, scanAudit)
}
