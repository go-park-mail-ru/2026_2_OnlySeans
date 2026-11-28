package onlineconf_test

import (
	"database/sql"
	"encoding/json"
	"os"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	service "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconf"
	"github.com/stretchr/testify/require"
)

func TestScopedResourcePermissions(t *testing.T) {
	repo, db := governanceRepository(t)
	ctx := t.Context()
	const uid = 2
	scoped := oc.WithScopedUser(ctx, uid)
	owner := oc.AdminIdentity{ID: 3, Role: oc.SuperAdmin}
	caller := oc.AdminIdentity{ID: uid, Role: "scoped"}
	area := createScope(t, repo)
	config := createConfig(t, repo)
	_, err := repo.CreateScope(ctx, owner.ID, oc.Scope{Name: "other", Title: area.Title})
	require.NoError(t, err)
	other := config
	other.Scope = "other"
	sibling, err := repo.CreateConfig(ctx, owner.ID, other)
	require.NoError(t, err)
	flag, err := repo.CreateFlag(ctx, owner.ID, oc.FeatureFlag{Scope: area.Name, Key: config.Key, Title: config.Title})
	require.NoError(t, err)

	grant := func(kind, key, action string) {
		t.Helper()
		require.NoError(t, repo.GovernanceChange(ctx, owner, "grant", service.GovernanceInput{
			User: uid, Scope: area.Name, Kind: kind, Key: key, Action: action,
		}))
	}
	grant("config", config.Key, "read")
	for _, opts := range []oc.ListOptions{{Limit: 1}, {Query: area.Title, Limit: 2}, {Scope: area.Name, Limit: 5, Offset: 1}} {
		items, count, err := repo.ListScopes(scoped, opts)
		require.NoError(t, err)
		require.Empty(t, items)
		require.Zero(t, count)
	}
	_, err = repo.GetScope(scoped, area.ID)
	require.ErrorIs(t, err, oc.ErrNotFound)
	_, err = repo.GetConfig(scoped, sibling.ID)
	require.ErrorIs(t, err, oc.ErrNotFound)
	_, err = repo.GetFlag(scoped, flag.ID)
	require.ErrorIs(t, err, oc.ErrNotFound)
	items, count, err := repo.ListConfigs(scoped, oc.ListOptions{Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Equal(t, config.ID, items[0].ID)
	data, err := repo.GovernanceView(scoped, caller, "rollouts")
	require.NoError(t, err)
	require.JSONEq(t, `{"policies":[],"flags":[]}`, string(data))

	unchanged := func() [2]int64 {
		t.Helper()
		var v [2]int64
		require.NoError(t, db.QueryRowContext(ctx, "SELECT revision,(SELECT count(*) FROM oc_audit) FROM oc_state WHERE id=1").Scan(&v[0], &v[1]))
		return v
	}
	initial := unchanged()
	_, err = repo.CreateConfig(scoped, uid, config)
	require.ErrorIs(t, err, domain.ErrForbidden)
	_, err = repo.CreateFlag(scoped, uid, *flag)
	require.ErrorIs(t, err, domain.ErrForbidden)
	_, err = repo.CreateScope(scoped, uid, area)
	require.ErrorIs(t, err, domain.ErrForbidden)
	stale := config
	stale.Version++
	_, err = repo.UpdateConfig(scoped, uid, stale)
	require.ErrorIs(t, err, domain.ErrForbidden)
	require.ErrorIs(t, repo.DeleteConfig(scoped, uid, config.ID, stale.Version), domain.ErrForbidden)
	require.Equal(t, initial, unchanged())

	grant("scope", "", "read")
	areas, count, err := repo.ListScopes(scoped, oc.ListOptions{Query: area.Title, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Equal(t, area.ID, areas[0].ID)
	_, err = repo.GetScope(scoped, area.ID)
	require.NoError(t, err)
	_, err = repo.GetFlag(scoped, flag.ID)
	require.ErrorIs(t, err, oc.ErrNotFound)

	grant("config", "new_key", "create")
	newConfig := oc.Config{Scope: area.Name, Key: "new_key", Title: "New", Value: json.RawMessage("1")}
	created, err := repo.CreateConfig(scoped, uid, newConfig)
	require.NoError(t, err)
	_, err = repo.GetConfig(scoped, created.ID)
	require.ErrorIs(t, err, oc.ErrNotFound)
	grant("config", "new_key", "update")
	created.Title = "Edited"
	updated, err := repo.UpdateConfig(scoped, uid, *created)
	require.NoError(t, err)
	require.ErrorIs(t, repo.DeleteConfig(scoped, uid, created.ID, updated.Version), domain.ErrForbidden)
	grant("config", "new_key", "delete")
	require.NoError(t, repo.DeleteConfig(scoped, uid, created.ID, updated.Version))

	grant("flag", flag.Key, "approve")
	data, err = repo.GovernanceView(scoped, caller, "rollouts")
	require.NoError(t, err)
	var plans struct {
		Flags []struct {
			ID       int64
			Title    *string
			Approve  bool `json:"can_approve"`
			Schedule bool `json:"can_schedule"`
		}
	}
	require.NoError(t, json.Unmarshal(data, &plans))
	require.Len(t, plans.Flags, 1)
	require.Equal(t, flag.ID, plans.Flags[0].ID)
	require.True(t, plans.Flags[0].Approve)
	require.False(t, plans.Flags[0].Schedule)
	require.Nil(t, plans.Flags[0].Title)

	grant("*", "", "read")
	_, err = repo.GetFlag(scoped, flag.ID)
	require.NoError(t, err)
	_, err = repo.GetConfig(scoped, sibling.ID)
	require.ErrorIs(t, err, oc.ErrNotFound)
	require.ErrorIs(t, repo.DeleteFlag(scoped, uid, flag.ID, flag.Version), domain.ErrForbidden)

	_, err = repo.ChangeRole(ctx, owner.ID, uid, "no_access", "scoped")
	require.NoError(t, err)
	blocked, err := repo.GetUser(ctx, uid)
	require.NoError(t, err)
	require.False(t, oc.HasAdminAccess(blocked.Role))
	require.ErrorIs(t, repo.GovernanceChange(scoped, caller, "approve", service.GovernanceInput{ID: flag.ID}), domain.ErrForbidden)
}

func governanceRepository(t *testing.T) (*service.PostgresRepository, *sql.DB) {
	t.Helper()
	_, db := newRepository(t)
	ctx := t.Context()
	migration, err := os.ReadFile("../../db/migrations/000017_onlineconf_admins.up.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(migration))
	require.NoError(t, err)
	migration, err = os.ReadFile("../../db/migrations/000018_onlineconf_role_changes.up.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(migration))
	require.NoError(t, err)
	migration, err = os.ReadFile("../../db/migrations/000019_onlineconf_governance.up.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(migration))
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
      CREATE TABLE role(id bigint PRIMARY KEY,code text);
      INSERT INTO role VALUES(1,'reader'),(2,'admin'),(3,'editor');
      ALTER TABLE account ADD COLUMN role_id bigint DEFAULT 1 REFERENCES role(id);
      UPDATE account SET role_id=2 WHERE id=1;
      UPDATE account SET role_id=3 WHERE id=2;
      INSERT INTO account(id,username) VALUES(3,'super'),(4,'deleted');
      UPDATE account SET deleted_at=now() WHERE id=4;
      INSERT INTO oc_admin_access(account_id,access_role) VALUES(3,'superadmin')`)
	require.NoError(t, err)
	return service.NewPostgresRepository(db, nil), db
}

func TestGrantOnlineConfAdmin(t *testing.T) {
	accounts, db := governanceRepository(t)
	ctx := t.Context()
	admin, err := accounts.GetUser(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, domain.RoleAdmin, admin.Role)
	_, err = accounts.GrantAdmin(ctx, 1, 2)
	require.ErrorIs(t, err, domain.ErrForbidden)
	for range 2 {
		target, err := accounts.GrantAdmin(ctx, 3, 2)
		require.NoError(t, err)
		require.Equal(t, domain.RoleAdmin, target.Role)
	}
	_, err = accounts.GrantAdmin(ctx, 2, 1)
	require.ErrorIs(t, err, domain.ErrForbidden)
	_, err = accounts.GrantAdmin(ctx, 3, 4)
	require.ErrorIs(t, err, domain.ErrUserNotFound)
	self, err := accounts.GrantAdmin(ctx, 3, 3)
	require.NoError(t, err)
	require.Equal(t, oc.SuperAdmin, self.Role)
	var role string
	var actor int
	err = db.QueryRowContext(ctx, `SELECT r.code,m.granted_by FROM account a JOIN role r ON r.id=a.role_id
      JOIN oc_admin_access m ON m.account_id=a.id WHERE a.id=2`).Scan(&role, &actor)
	require.NoError(t, err)
	require.Equal(t, "editor", role)
	require.Equal(t, 3, actor)
	members, err := accounts.ListAdministrators(ctx)
	require.NoError(t, err)
	require.Len(t, members, 3)
	_, err = accounts.ChangeRole(ctx, 3, 3, domain.RoleAdmin, oc.SuperAdmin)
	require.ErrorIs(t, err, oc.ErrLastSuperAdmin)
	_, err = accounts.ChangeRole(ctx, 3, 1, "no_access", domain.RoleAdmin)
	require.NoError(t, err)
	revoked, err := accounts.GetUser(ctx, 1)
	require.NoError(t, err)
	require.False(t, oc.HasAdminAccess(revoked.Role))
	_, err = accounts.ChangeRole(ctx, 3, 1, oc.SuperAdmin, domain.RoleAdmin)
	require.ErrorIs(t, err, oc.ErrVersionConflict)
	_, err = accounts.ChangeRole(ctx, 3, 2, oc.SuperAdmin, domain.RoleAdmin)
	require.NoError(t, err)
	_, err = accounts.ChangeRole(ctx, 3, 3, domain.RoleAdmin, oc.SuperAdmin)
	require.NoError(t, err)
	_, err = accounts.ChangeRole(ctx, 3, 1, domain.RoleAdmin, "no_access")
	require.ErrorIs(t, err, domain.ErrForbidden)
	_, err = accounts.GrantAdmin(ctx, 2, 1)
	require.NoError(t, err)
}

func TestGovernanceGroupScopeAndRevocation(t *testing.T) {
	repo, db := governanceRepository(t)
	ctx := t.Context()
	owner := oc.AdminIdentity{ID: 3, Role: oc.SuperAdmin}
	change := func(cmd string, input service.GovernanceInput) {
		t.Helper()
		require.NoError(t, repo.GovernanceChange(ctx, owner, cmd, input))
	}
	change("group", service.GovernanceInput{Name: "Moderators"})
	var group int64
	require.NoError(t, db.QueryRowContext(ctx, "SELECT id FROM oc_groups WHERE name='Moderators'").Scan(&group))
	change("member", service.GovernanceInput{Group: group, User: 2})
	change("grant", service.GovernanceInput{Group: group, Scope: "frontend", Kind: "flag", Key: "banner", Action: "approve"})
	user, err := repo.GetUser(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, domain.Role("scoped"), user.Role)
	for _, sample := range []struct {
		scope, key, action string
		allowed            bool
	}{
		{"frontend", "banner", "approve", true}, {"frontend", "banner", "update", false}, {"reviews", "banner", "approve", false}, {"frontend", "other", "approve", false},
	} {
		var allowed bool
		require.NoError(t, db.QueryRowContext(ctx, "SELECT oc_permitted(2,$1,'flag',$2,$3)", sample.scope, sample.key, sample.action).Scan(&allowed))
		require.Equal(t, sample.allowed, allowed)
	}
	require.ErrorIs(t, repo.GovernanceChange(ctx, oc.AdminIdentity{ID: 2, Role: "scoped"}, "policy", service.GovernanceInput{}), domain.ErrForbidden)
	change("remove-member", service.GovernanceInput{Group: group, User: 2})
	user, err = repo.GetUser(ctx, 2)
	require.NoError(t, err)
	require.False(t, oc.HasAdminAccess(user.Role))
}

func TestSuperAdminCannotSkipRolloutStages(t *testing.T) {
	repo, _ := governanceRepository(t)
	ctx := t.Context()
	createScope(t, repo)
	super := oc.AdminIdentity{ID: 3, Role: oc.SuperAdmin}
	input := oc.FeatureFlag{Scope: "reviews", Key: "stages", Title: "stages", Enabled: true, RolloutPercent: 100}
	_, err := repo.CreateFlag(ctx, super.ID, input)
	require.Error(t, err)
	input.RolloutPercent = 0
	flag, err := repo.CreateFlag(ctx, super.ID, input)
	require.NoError(t, err)
	target := service.GovernanceInput{ID: flag.ID, Generation: 1, Policy: 1}
	require.ErrorIs(t, repo.GovernanceChange(ctx, super, "publish-full", target), oc.ErrNotFound)
	changed := *flag
	changed.RolloutPercent = 100
	_, err = repo.UpdateFlag(ctx, super.ID, changed)
	require.Error(t, err)
	require.Error(t, repo.GovernanceChange(ctx, super, "policy", service.GovernanceInput{Name: "single", Steps: []int{100}, Seconds: 1}))
	current, err := repo.GetFlag(ctx, flag.ID)
	require.NoError(t, err)
	require.Equal(t, 0, current.RolloutPercent)
	require.Equal(t, flag.Version, current.Version)
	require.NoError(t, repo.GovernanceChange(ctx, super, "schedule", target))
	target.Generation++
	require.NoError(t, repo.GovernanceChange(ctx, super, "approve", target))
}
