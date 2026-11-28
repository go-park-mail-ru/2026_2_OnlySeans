package onlineconf

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
)

type GovernanceInput struct {
	ID         int64      `json:"id"`
	Group      int64      `json:"group"`
	User       int64      `json:"user"`
	Scope      string     `json:"scope"`
	Kind       string     `json:"kind"`
	Key        string     `json:"key"`
	Action     string     `json:"action"`
	Name       string     `json:"name"`
	Steps      []int      `json:"steps"`
	Seconds    int        `json:"seconds"`
	Policy     int64      `json:"policy"`
	Generation int64      `json:"generation"`
	Start      *time.Time `json:"start"`
}

func (r *PostgresRepository) GovernanceView(ctx context.Context, actor oc.AdminIdentity, section string) (json.RawMessage, error) {
	if !oc.HasAdminAccess(actor.Role) {
		return nil, domain.ErrForbidden
	}
	var query string
	var args []any
	switch section {
	case "my-grants":
		query = `SELECT coalesce(jsonb_agg(to_jsonb(t)), '[]') FROM (SELECT DISTINCT scope,kind,key,action FROM oc_effective_grants WHERE account_id=$1)t`
		args = []any{actor.ID}
	case "access":
		if actor.Role != oc.SuperAdmin {
			return nil, domain.ErrForbidden
		}
		query = `SELECT jsonb_build_object('groups', (SELECT coalesce(jsonb_agg(to_jsonb(g) ORDER BY name),'[]') FROM oc_groups g),
 'accounts',(SELECT coalesce(jsonb_agg(jsonb_build_object('id',id,'username',username) ORDER BY lower(username),id),'[]')
             FROM account WHERE deleted_at IS NULL),
 'members',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY username),'[]') FROM (SELECT m.group_id,m.account_id,a.username FROM oc_members m JOIN account a ON a.id=m.account_id)t),
 'grants',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY id),'[]') FROM (SELECT g.*,a.username FROM oc_grants g LEFT JOIN account a ON a.id=g.account_id)t))`
	case "rollouts":
		query = `WITH visible_flags AS (
 SELECT f.id,s.name AS scope,f.key,CASE WHEN $2 OR oc_permitted($1,s.name,'flag',f.key,'read') THEN f.title END AS title,f.enabled,f.rollout_percent,p.id AS policy_id,p.name AS policy,p.wait_seconds,
 q.due_at,q.generation,q.approved_by,
 (SELECT min(step) FROM unnest(p.steps) step WHERE step>CASE WHEN f.enabled THEN f.rollout_percent ELSE 0 END) AS next_percent,
 ($2 OR oc_permitted($1,s.name,'flag',f.key,'update')) AS can_schedule,
 ($2 OR oc_permitted($1,s.name,'flag',f.key,'approve')) AS can_approve
 FROM oc_rollouts q JOIN oc_flags f ON f.id=q.flag_id JOIN oc_scopes s ON s.id=f.scope_id JOIN oc_policies p ON p.id=q.policy_id
 WHERE $2 OR oc_permitted($1,s.name,'flag',f.key,'read') OR oc_permitted($1,s.name,'flag',f.key,'approve') OR oc_permitted($1,s.name,'flag',f.key,'update'))
 SELECT jsonb_build_object('flags',(SELECT coalesce(jsonb_agg(to_jsonb(v) ORDER BY scope,key),'[]') FROM visible_flags v),
 'policies',(SELECT coalesce(jsonb_agg(to_jsonb(p) ORDER BY id),'[]') FROM oc_policies p
 WHERE $2 OR EXISTS(SELECT FROM visible_flags v WHERE v.can_schedule OR v.policy_id=p.id)))`
		args = []any{actor.ID, actor.Role != oc.ScopedAdmin}
	default:
		return nil, oc.ErrNotFound
	}
	var data []byte
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&data)
	return data, err
}

func (r *PostgresRepository) GovernanceChange(ctx context.Context, actor oc.AdminIdentity, cmd string, v GovernanceInput) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	caller, err := readIdentity(tx.QueryRowContext(ctx, accountAccess+"a.id=$1", actor.ID))
	if err != nil {
		return err
	}
	if !oc.HasAdminAccess(caller.Role) {
		return domain.ErrForbidden
	}
	var result sql.Result
	switch cmd {
	case "schedule", "approve", "stop":
		verb := "update"
		if cmd == "approve" {
			verb = "approve"
		}
		var generation int64
		var next sql.NullInt64
		var due sql.NullTime
		var approved sql.NullInt64
		var allowed bool
		err = tx.QueryRowContext(ctx, `SELECT q.generation,q.due_at,q.approved_by,($2 OR oc_permitted($3,s.name,'flag',f.key,$4)),
   (SELECT min(step) FROM unnest(p.steps) step WHERE step>CASE WHEN f.enabled THEN f.rollout_percent ELSE 0 END)
   FROM oc_rollouts q JOIN oc_flags f ON f.id=q.flag_id JOIN oc_scopes s ON s.id=f.scope_id
   JOIN oc_policies p ON p.id=q.policy_id WHERE q.flag_id=$1 FOR UPDATE OF q`, v.ID, caller.Role != oc.ScopedAdmin, actor.ID, verb).Scan(&generation, &due, &approved, &allowed, &next)
		if errors.Is(err, sql.ErrNoRows) && caller.Role == oc.ScopedAdmin {
			return domain.ErrForbidden
		}
		if err != nil {
			return repositoryError(err)
		}
		if !allowed {
			return domain.ErrForbidden
		}
		if v.Generation != generation {
			return oc.ErrVersionConflict
		}
		switch cmd {
		case "schedule":
			if !next.Valid {
				return oc.ShowcaseError("Флаг уже достиг 100%.")
			}
			result, err = tx.ExecContext(ctx, `UPDATE oc_rollouts SET policy_id=$2,due_at=greatest(coalesce($3,now()),not_before),approved_by=NULL,generation=generation+1 WHERE flag_id=$1 AND (policy_id=$2 OR EXISTS(SELECT FROM oc_flags WHERE id=$1 AND (NOT enabled OR rollout_percent=0)))`, v.ID, v.Policy, v.Start)
		case "approve":
			if !due.Valid || approved.Valid || !next.Valid {
				return oc.ErrVersionConflict
			}
			result, err = tx.ExecContext(ctx, "UPDATE oc_rollouts SET approved_by=$2,generation=generation+1 WHERE flag_id=$1", v.ID, actor.ID)
		case "stop":
			result, err = tx.ExecContext(ctx, "UPDATE oc_rollouts SET approved_by=NULL,due_at=NULL,generation=generation+1 WHERE flag_id=$1", v.ID)
		}
	default:
		if caller.Role != oc.SuperAdmin {
			return domain.ErrForbidden
		}
		switch cmd {
		case "group":
			result, err = tx.ExecContext(ctx, "INSERT INTO oc_groups(name) VALUES($1)", strings.TrimSpace(v.Name))
		case "delete-group":
			result, err = tx.ExecContext(ctx, "DELETE FROM oc_groups WHERE id=$1", v.ID)
		case "member":
			result, err = tx.ExecContext(ctx, "INSERT INTO oc_members SELECT $1,id FROM account WHERE id=$2 AND deleted_at IS NULL", v.Group, v.User)
		case "remove-member":
			result, err = tx.ExecContext(ctx, "DELETE FROM oc_members WHERE group_id=$1 AND account_id=$2", v.Group, v.User)
		case "grant":
			result, err = tx.ExecContext(ctx, "INSERT INTO oc_grants(account_id,group_id,scope,kind,key,action) VALUES(NULLIF($1::bigint,0),NULLIF($2::bigint,0),$3,$4,$5,$6)", v.User, v.Group, v.Scope, v.Kind, v.Key, v.Action)
		case "revoke":
			result, err = tx.ExecContext(ctx, "DELETE FROM oc_grants WHERE id=$1", v.ID)
		case "policy":
			result, err = tx.ExecContext(ctx, "INSERT INTO oc_policies(name,steps,wait_seconds) VALUES($1,$2,$3)", strings.TrimSpace(v.Name), v.Steps, v.Seconds)
		default:
			return oc.ErrNotFound
		}
	}
	if err != nil {
		return repositoryError(err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		if cmd == "schedule" {
			return oc.ShowcaseError("Регламент можно сменить до начала раскатки или после сброса охвата до 0%.")
		}
		return oc.ErrNotFound
	}
	return tx.Commit()
}

func (r *PostgresRepository) MayAccess(ctx context.Context, uid domain.UserID, kind string, id int64, action string) (bool, error) {
	var q string
	switch kind {
	case "config":
		q = "SELECT oc_permitted($1,s.name,'config',c.key,$3) FROM oc_configs c JOIN oc_scopes s ON s.id=c.scope_id WHERE c.id=$2"
	case "flag":
		q = "SELECT oc_permitted($1,s.name,'flag',f.key,$3) FROM oc_flags f JOIN oc_scopes s ON s.id=f.scope_id WHERE f.id=$2"
	case "scope":
		q = "SELECT oc_permitted($1,name,'scope','',$3) FROM oc_scopes WHERE id=$2"
	default:
		return false, nil
	}
	var ok bool
	err := r.db.QueryRowContext(ctx, q, uid, id, action).Scan(&ok)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return ok, err
}
