package onlineconf

import (
	"context"
	"encoding/json"
	"fmt"

	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
)

const (
	createAuditQuery = `INSERT INTO oc_audit(actor_id,actor_name,resource_type,resource_id,scope,key,action,before,after,revision) VALUES($1,$2,$3,$4,$5,NULLIF($6,''),$7,$8,$9,$10)`
	auditColumns     = `id,actor_id,actor_name,resource_type,resource_id,scope,COALESCE(key,''),action,before,after,revision,created_at`
	auditFilter      = ` WHERE ($1 = '' OR resource_type=$1) AND ($2::bigint=0 OR resource_id=$2)`
	countAuditQuery  = `SELECT count(*) FROM oc_audit` + auditFilter
	listAuditQuery   = `SELECT ` + auditColumns + ` FROM oc_audit` + auditFilter + ` ORDER BY created_at DESC,id DESC LIMIT $3 OFFSET $4`
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

func (m *mutation) audit(ctx context.Context, kind oc.ResourceKind, id int64, scope, key string, action oc.AuditAction, before, after any) error {
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

	if filter.ResourceID < 0 || filter.ResourceID > oc.MaxSafeID {
		return nil, 0, oc.ErrInvalidAuditFilter
	}

	options, err := listOptions(oc.ListOptions{Limit: filter.Limit, Offset: filter.Offset})
	if err != nil {
		return nil, 0, err
	}

	return listPage(ctx, r.db, countAuditQuery, listAuditQuery, []any{filter.ResourceKind, filter.ResourceID}, options.Limit, options.Offset, scanAudit)
}
