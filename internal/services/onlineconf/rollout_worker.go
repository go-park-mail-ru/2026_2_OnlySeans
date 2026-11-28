package onlineconf

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
)

type approvedRollout struct {
	flagID     int64
	generation int64
	approvedBy domain.UserID
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
	var jobs []approvedRollout
	for rows.Next() {
		var job approvedRollout
		if err := rows.Scan(&job.flagID, &job.approvedBy, &job.generation); err != nil {
			rows.Close()
			return err
		}
		jobs = append(jobs, job)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	var jobErrors error
	for _, job := range jobs {
		err := r.mutate(ctx, job.approvedBy, func(m *mutation) error {
			return m.advanceApprovedRollout(ctx, job)
		})
		if errors.Is(err, domain.ErrForbidden) || errors.Is(err, domain.ErrUserNotFound) || errors.Is(err, oc.ErrNotFound) {
			_, err = r.db.ExecContext(ctx, "UPDATE oc_rollouts SET approved_by=NULL,generation=generation+1 WHERE flag_id=$1 AND generation=$2", job.flagID, job.generation)
		}
		if err != nil {
			jobErrors = errors.Join(jobErrors, err)
		}
	}
	return jobErrors
}

func (m *mutation) advanceApprovedRollout(ctx context.Context, job approvedRollout) error {
	before, err := scanFlag(m.tx.QueryRowContext(ctx, getFlagQuery, job.flagID))
	if err != nil {
		return err
	}
	caller, err := readIdentity(m.tx.QueryRowContext(ctx, accountAccess+"a.id=$1", job.approvedBy))
	if err != nil {
		return err
	}
	var nextPercent, waitSeconds int
	var allowed bool
	hasFullAccess := caller.Role == oc.SuperAdmin || caller.Role == domain.RoleAdmin
	err = m.tx.QueryRowContext(ctx, `SELECT
        (SELECT min(x) FROM unnest(p.steps) x WHERE x > $4),
        p.wait_seconds,
        ($5 OR oc_permitted($2,$6,'flag',$7,'approve'))
    FROM oc_rollouts q JOIN oc_policies p ON p.id=q.policy_id
   WHERE q.flag_id=$1 AND q.approved_by=$2 AND q.generation=$3 AND q.due_at<=now() FOR UPDATE OF q`, job.flagID, job.approvedBy, job.generation, effectivePercent(before),
		hasFullAccess, before.Scope, before.Key).Scan(&nextPercent, &waitSeconds, &allowed)
	if err != nil {
		return err
	}
	if !oc.HasAdminAccess(caller.Role) || !allowed {
		return domain.ErrForbidden
	}
	if _, err = m.tx.ExecContext(ctx, `UPDATE oc_rollouts
    SET generation=generation+1, approved_by=NULL,
        not_before=now()+$3::int*interval '1 second',
        due_at=CASE WHEN $2::int<100 THEN now()+$3::int*interval '1 second' END
    WHERE flag_id=$1`, job.flagID, nextPercent, waitSeconds); err != nil {
		return err
	}
	return m.publishRollout(ctx, before, nextPercent)
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
