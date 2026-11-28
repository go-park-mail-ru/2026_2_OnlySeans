package onlineconf

import (
	"context"
	"database/sql"
	"errors"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
)

// An explicit OnlyAdmin assignment takes precedence over product roles and grants.
const effectiveRole = `COALESCE(
    m.access_role,
    CASE
        WHEN r.code = 'admin' THEN 'admin'
        WHEN EXISTS (SELECT FROM oc_effective_grants g WHERE g.account_id = a.id) THEN 'scoped'
        ELSE r.code
    END
)`

const accountAccess = `SELECT a.id, a.username, ` + effectiveRole + `
    FROM account a JOIN role r ON r.id=a.role_id
    LEFT JOIN oc_admin_access m ON m.account_id=a.id
    WHERE a.deleted_at IS NULL AND `

func readIdentity(row *sql.Row) (oc.AdminIdentity, error) {
	var u oc.AdminIdentity
	err := row.Scan(&u.ID, &u.Username, &u.Role)
	if errors.Is(err, sql.ErrNoRows) {
		err = domain.ErrUserNotFound
	}
	return u, err
}

// GetUser supplies only the fields needed by OnlineConf's session access check.
func (r *PostgresRepository) GetUser(ctx context.Context, id domain.UserID) (*domain.User, error) {
	u, err := readIdentity(r.db.QueryRowContext(ctx, accountAccess+"a.id=$1", id))
	if err != nil {
		return nil, err
	}
	return &domain.User{ID: u.ID, Username: u.Username, Role: u.Role}, nil
}

func (r *PostgresRepository) FindAccount(ctx context.Context, username string) (oc.AdminIdentity, error) {
	return readIdentity(r.db.QueryRowContext(ctx, accountAccess+"a.username=$1", username))
}

func (r *PostgresRepository) ListAdministrators(ctx context.Context) ([]oc.AdminIdentity, error) {
	rows, err := r.db.QueryContext(ctx, accountAccess+effectiveRole+" IN ('admin','superadmin','scoped') ORDER BY lower(a.username), a.id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []oc.AdminIdentity{}
	for rows.Next() {
		var user oc.AdminIdentity
		if err := rows.Scan(&user.ID, &user.Username, &user.Role); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *PostgresRepository) AuditAuthors(ctx context.Context) ([]oc.AuditAuthor, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT actor_id,actor_name FROM
      (SELECT DISTINCT ON (actor_id) actor_id,actor_name FROM oc_audit ORDER BY actor_id,created_at DESC,id DESC) authors
      ORDER BY lower(actor_name),actor_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	authors := []oc.AuditAuthor{}
	for rows.Next() {
		var a oc.AuditAuthor
		if err := rows.Scan(&a.ID, &a.Username); err != nil {
			return nil, err
		}
		authors = append(authors, a)
	}
	return authors, rows.Err()
}

// The original empty PUT grants access without changing an existing administrator's level.
func (r *PostgresRepository) GrantAdmin(ctx context.Context, actor, target domain.UserID) (oc.AdminIdentity, error) {
	return r.ChangeRole(ctx, actor, target, domain.RoleAdmin, "")
}

func (r *PostgresRepository) ChangeRole(ctx context.Context, actor, target domain.UserID, role, previous domain.Role) (oc.AdminIdentity, error) {
	if !oc.CanAssignRole(role) {
		return oc.AdminIdentity{}, domain.ErrForbidden
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return oc.AdminIdentity{}, err
	}
	defer tx.Rollback()
	// Role changes are rare. Serialize them so concurrent demotions cannot remove both final superadmins.
	if _, err = tx.ExecContext(ctx, "LOCK TABLE oc_admin_access IN SHARE ROW EXCLUSIVE MODE"); err != nil {
		return oc.AdminIdentity{}, err
	}
	caller, err := readIdentity(tx.QueryRowContext(ctx, accountAccess+"a.id=$1 FOR SHARE OF a", actor))
	if err != nil {
		return oc.AdminIdentity{}, err
	}
	if caller.Role != oc.SuperAdmin {
		return oc.AdminIdentity{}, domain.ErrForbidden
	}
	user, err := readIdentity(tx.QueryRowContext(ctx, accountAccess+"a.id=$1 FOR SHARE OF a", target))
	if err != nil {
		return oc.AdminIdentity{}, err
	}
	if previous != "" && previous != user.Role {
		return oc.AdminIdentity{}, oc.ErrVersionConflict
	}
	if previous == "" && oc.HasAdminAccess(user.Role) || role == user.Role {
		return user, tx.Commit()
	}
	if user.Role == oc.SuperAdmin {
		var hasOther bool
		err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM oc_admin_access m JOIN account a ON a.id=m.account_id
            WHERE m.access_role='superadmin' AND a.deleted_at IS NULL AND a.id<>$1)`, target).Scan(&hasOther)
		if err != nil {
			return oc.AdminIdentity{}, err
		}
		if !hasOther {
			return oc.AdminIdentity{}, oc.ErrLastSuperAdmin
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO oc_admin_access(account_id,access_role,granted_by) VALUES($1,$2,$3)
       ON CONFLICT(account_id) DO UPDATE SET access_role=excluded.access_role,granted_by=excluded.granted_by,granted_at=now()`, target, role, actor)
	if err != nil {
		return oc.AdminIdentity{}, err
	}
	user.Role = role
	return user, tx.Commit()
}
