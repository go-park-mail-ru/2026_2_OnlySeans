package onlineconf

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
)

const effectiveRole = `COALESCE(m.access_role, CASE WHEN r.code='admin' THEN 'admin'
 WHEN EXISTS(SELECT FROM oc_effective_grants g WHERE g.account_id=a.id) THEN 'scoped' ELSE r.code END)`

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
		args = []any{actor.ID, actor.Role != "scoped"}
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
   JOIN oc_policies p ON p.id=q.policy_id WHERE q.flag_id=$1 FOR UPDATE OF q`, v.ID, caller.Role != "scoped", actor.ID, verb).Scan(&generation, &due, &approved, &allowed, &next)
		if errors.Is(err, sql.ErrNoRows) && caller.Role == "scoped" {
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

// Advance only approved, due stages. The revision, flag update, approval consumption
// and history entry commit together under the same lock as manual flag edits.
func (r *PostgresRepository) RunRollouts(ctx context.Context) {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if err := r.advanceDue(ctx); err != nil && ctx.Err() == nil {
				r.appLogger.ErrorContext(ctx, "scheduled rollout failed", map[string]any{"error": err.Error()})
			}
		}
	}
}
func (r *PostgresRepository) advanceDue(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, "SELECT flag_id,approved_by,generation FROM oc_rollouts WHERE approved_by IS NOT NULL AND due_at<=now() ORDER BY due_at,flag_id LIMIT 50")
	if err != nil {
		return err
	}
	type job struct {
		id, gen int64
		actor   domain.UserID
	}
	var jobs []job
	for rows.Next() {
		var j job
		if err := rows.Scan(&j.id, &j.actor, &j.gen); err != nil {
			rows.Close()
			return err
		}
		jobs = append(jobs, j)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	var jobErrors error
	for _, j := range jobs {
		err := r.mutate(ctx, j.actor, func(m *mutation) error {
			before, e := scanFlag(m.tx.QueryRowContext(ctx, getFlagQuery, j.id))
			if e != nil {
				return e
			}
			caller, e := readIdentity(m.tx.QueryRowContext(ctx, accountAccess+"a.id=$1", j.actor))
			if e != nil {
				return e
			}
			var pct, seconds int
			var allowed bool
			e = m.tx.QueryRowContext(ctx, `SELECT (SELECT min(x) FROM unnest(p.steps)x WHERE x>$4),p.wait_seconds,
   ($5 OR oc_permitted($2,$6,'flag',$7,'approve')) FROM oc_rollouts q JOIN oc_policies p ON p.id=q.policy_id
   WHERE q.flag_id=$1 AND q.approved_by=$2 AND q.generation=$3 AND q.due_at<=now() FOR UPDATE OF q`, j.id, j.actor, j.gen, effectivePercent(before), caller.Role == oc.SuperAdmin || caller.Role == domain.RoleAdmin, before.Scope, before.Key).Scan(&pct, &seconds, &allowed)
			if e != nil {
				return e
			}
			if !oc.HasAdminAccess(caller.Role) || !allowed {
				return domain.ErrForbidden
			}
			if _, e = m.tx.ExecContext(ctx, `UPDATE oc_rollouts SET generation=generation+1,approved_by=NULL,not_before=now()+$3::int*interval '1 second',due_at=CASE WHEN $2::int<100 THEN now()+$3::int*interval '1 second' END WHERE flag_id=$1`, j.id, pct, seconds); e != nil {
				return e
			}
			return m.publishRollout(ctx, before, pct)
		})
		if errors.Is(err, domain.ErrForbidden) || errors.Is(err, domain.ErrUserNotFound) || errors.Is(err, oc.ErrNotFound) {
			_, err = r.db.ExecContext(ctx, "UPDATE oc_rollouts SET approved_by=NULL,generation=generation+1 WHERE flag_id=$1 AND generation=$2", j.id, j.gen)
		}
		if err != nil {
			jobErrors = errors.Join(jobErrors, err)
		}
	}
	return jobErrors
}

func (m *mutation) publishRollout(ctx context.Context, before oc.FeatureFlag, percent int) error {
	if _, err := m.tx.ExecContext(ctx, "SELECT set_config('onlineconf.approved_flag',$1,true)", fmt.Sprint(before.ID)); err != nil {
		return err
	}
	if _, err := m.tx.ExecContext(ctx, "UPDATE oc_flags SET enabled=true,rollout_percent=$2,version=version+1 WHERE id=$1", before.ID, percent); err != nil {
		return err
	}
	after, err := scanFlag(m.tx.QueryRowContext(ctx, getFlagQuery, before.ID))
	if err != nil {
		return err
	}
	return m.audit(ctx, oc.ResourceFlag, before.ID, before.Scope, before.Key, oc.ActionUpdate, before, after)
}

func effectivePercent(f oc.FeatureFlag) int {
	if !f.Enabled {
		return 0
	}
	return f.RolloutPercent
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
