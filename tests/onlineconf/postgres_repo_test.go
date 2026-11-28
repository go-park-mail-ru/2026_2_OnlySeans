package onlineconf_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	evaluator "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconfevaluator"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	service "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconf"
)

const (
	schemaNonceBytes               = 16
	testDatabaseEnv                = "ONLINECONF_TEST_DATABASE_URL"
	testActor        domain.UserID = 1
	testOtherUser    domain.UserID = 2
	fixtureQuery                   = `CREATE TABLE account(id bigint PRIMARY KEY,username text NOT NULL,deleted_at timestamptz);
 INSERT INTO account(id,username) VALUES(1,'admin'),(2,'reader');
 CREATE FUNCTION set_updated_at() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN NEW.updated_at=now(); RETURN NEW; END; $$;`
)

func newRepository(t *testing.T) (oc.Repository, *sql.DB) {
	t.Helper()

	url := os.Getenv(testDatabaseEnv)
	if url == "" {
		t.Skip("set ONLINECONF_TEST_DATABASE_URL to run PostgreSQL integration tests")
	}

	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}

	control := stdlib.OpenDB(*cfg)

	t.Cleanup(func() {
		if err := control.Close(); err != nil {
			t.Error(err)
		}
	})

	nonce := make([]byte, schemaNonceBytes)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatal(err)
	}

	schema := fmt.Sprintf("oc_test_%x", nonce)
	ctx := context.Background()

	if _, err := control.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}

	cfg.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*cfg)

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}

		if _, err := control.ExecContext(ctx, `DROP SCHEMA `+schema+` CASCADE`); err != nil {
			t.Error(err)
		}
	})

	if _, err := db.ExecContext(ctx, fixtureQuery); err != nil {
		t.Fatal(err)
	}

	migration, err := os.ReadFile(filepath.Join("..", "..", "db", "migrations", "000016_onlineconf.up.sql"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.ExecContext(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}

	return service.NewPostgresRepository(db, nil), db
}

func createScope(t *testing.T, repo oc.Repository) oc.Scope {
	t.Helper()

	scope, err := repo.CreateScope(context.Background(), testActor, oc.Scope{Name: "reviews", Title: "Отзывы"})
	if err != nil {
		t.Fatal(err)
	}

	return *scope
}

func createConfig(t *testing.T, repo oc.Repository) oc.Config {
	t.Helper()

	config, err := repo.CreateConfig(context.Background(), testActor, oc.Config{Scope: oc.ReviewsScope, Key: oc.ReviewMaxLengthKey, Title: "Лимит", Value: json.RawMessage(`500`)})
	if err != nil {
		t.Fatal(err)
	}

	return *config
}

func TestPostgresConfigsVersionsAndAudit(t *testing.T) {
	repo, db := newRepository(t)
	createScope(t, repo)
	config := createConfig(t, repo)
	ctx := context.Background()

	if _, err := repo.CreateConfig(ctx, testActor, config); !errors.Is(err, oc.ErrDuplicateKey) {
		t.Fatalf("duplicate key: %v", err)
	}

	stale := config
	config.Value = json.RawMessage(`200`)
	updated, err := repo.UpdateConfig(ctx, testActor, config)

	if err != nil || updated.Version != config.Version+1 {
		t.Fatalf("update: %v %v", updated, err)
	}

	if _, err := repo.UpdateConfig(ctx, testActor, stale); !errors.Is(err, oc.ErrVersionConflict) {
		t.Fatalf("stale update: %v", err)
	}

	if err := repo.DeleteConfig(ctx, testActor, config.ID, stale.Version); !errors.Is(err, oc.ErrVersionConflict) {
		t.Fatalf("stale delete: %v", err)
	}

	items, total, err := repo.ListConfigs(ctx, oc.ListOptions{Scope: oc.ReviewsScope, Query: "ЛИМИТ", Limit: 1})
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("search: %v %d %v", items, total, err)
	}

	_, total, err = repo.ListConfigs(ctx, oc.ListOptions{Query: "%"})
	if err != nil || total != 0 {
		t.Fatalf("search must use literal substring: %d %v", total, err)
	}

	snapshot, err := repo.LoadSnapshot(ctx)
	if err != nil || snapshot.Revision != 3 || string(snapshot.Configs[0].Value) != "200" {
		t.Fatalf("snapshot: %v %v", snapshot, err)
	}

	var persisted []byte
	if err := db.QueryRowContext(ctx, `SELECT snapshot FROM oc_state`).Scan(&persisted); err != nil {
		t.Fatal(err)
	}

	var saved oc.Snapshot
	if err := json.Unmarshal(persisted, &saved); err != nil {
		t.Fatal(err)
	}

	if saved.Revision != snapshot.Revision || !reflect.DeepEqual(saved.Configs, snapshot.Configs) {
		t.Fatal("persisted snapshot mismatch")
	}

	if err := repo.DeleteConfig(ctx, testActor, updated.ID, updated.Version); err != nil {
		t.Fatal(err)
	}

	history, total, err := repo.ListAudit(ctx, oc.AuditFilter{ResourceKind: oc.ResourceConfig, ResourceID: config.ID})
	if err != nil || total != 3 || history[0].Action != oc.ActionDelete || string(history[0].After) != "null" || history[0].ActorName != "admin" {
		t.Fatalf("audit: %v %d %v", history, total, err)
	}

	if _, err := repo.GetConfig(ctx, config.ID); !errors.Is(err, oc.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestPostgresFlagsOverridesAndSalt(t *testing.T) {
	repo, _ := newRepository(t)
	createScope(t, repo)

	ctx := context.Background()
	input := oc.FeatureFlag{Scope: oc.ReviewsScope, Key: "write_enabled", Title: "Отзывы", Enabled: true, RolloutPercent: 25, ForceOnUserIDs: []domain.UserID{testActor}, ForceOffUserIDs: []domain.UserID{testOtherUser}}

	flag, err := repo.CreateFlag(ctx, testActor, input)
	if err != nil {
		t.Fatal(err)
	}

	if len(flag.Salt) != 32 || !evaluator.Evaluate(flag, testActor).Enabled || evaluator.Evaluate(flag, testOtherUser).Enabled {
		t.Fatal("salt or overrides")
	}

	salt := flag.Salt
	flag.RolloutPercent = 100
	flag.Salt = ""
	flag.ForceOnUserIDs = nil
	flag.ForceOffUserIDs = nil
	flag, err = repo.UpdateFlag(ctx, testActor, *flag)

	if err != nil || flag.Salt != salt || len(flag.ForceOnUserIDs) != 0 || len(flag.ForceOffUserIDs) != 0 {
		t.Fatalf("update: %v %v", flag, err)
	}

	input.ForceOffUserIDs = []domain.UserID{testActor}
	if _, err := repo.CreateFlag(ctx, testActor, input); !errors.Is(err, oc.ErrConflictingOverrides) {
		t.Fatalf("intersection: %v", err)
	}

	history, _, err := repo.ListAudit(ctx, oc.AuditFilter{ResourceKind: oc.ResourceFlag})
	if err != nil {
		t.Fatal(err)
	}

	var before map[string]json.RawMessage
	if err := json.Unmarshal(history[0].Before, &before); err != nil {
		t.Fatal(err)
	}

	if _, exists := before["salt"]; exists {
		t.Fatal("salt leaked to audit")
	}

	if err := repo.DeleteFlag(ctx, testActor, flag.ID, flag.Version); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresMutationRollsBackWithAuditFailure(t *testing.T) {
	repo, db := newRepository(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `ALTER TABLE oc_audit ADD CONSTRAINT fail_audit CHECK(actor_name <> 'admin')`); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.CreateScope(ctx, testActor, oc.Scope{Name: "reviews", Title: "Отзывы"}); err == nil {
		t.Fatal("audit failure ignored")
	}

	scopes, total, err := repo.ListScopes(ctx, oc.ListOptions{})
	if err != nil || total != 0 || len(scopes) != 0 {
		t.Fatalf("mutation survived: %v %v", scopes, err)
	}

	snapshot, err := repo.LoadSnapshot(ctx)
	if err != nil || snapshot.Revision != 0 || len(snapshot.Scopes) != 0 {
		t.Fatalf("snapshot changed: %v %v", snapshot, err)
	}
}

func TestPostgresConcurrentVersionConflict(t *testing.T) {
	repo, _ := newRepository(t)
	createScope(t, repo)
	config := createConfig(t, repo)
	var wg sync.WaitGroup
	results := make(chan error, 2)

	for range 2 {
		wg.Go(func() { _, err := repo.UpdateConfig(context.Background(), testActor, config); results <- err })
	}

	wg.Wait()
	close(results)
	var successes, conflicts int

	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, oc.ErrVersionConflict):
			conflicts++
		default:
			t.Fatal(err)
		}
	}

	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
}

func TestPostgresJSONAndRejectedWrites(t *testing.T) {
	repo, _ := newRepository(t)
	createScope(t, repo)

	ctx := context.Background()
	for i, value := range []string{`{}`, `[]`, `"text"`, `12.5`, `true`, `null`} {
		_, err := repo.CreateConfig(ctx, testActor, oc.Config{Scope: oc.ReviewsScope, Key: fmt.Sprintf("config_%d", i), Title: "JSON", Value: json.RawMessage(value)})
		if err != nil {
			t.Fatal(value, err)
		}
	}

	input := oc.FeatureFlag{Scope: oc.ReviewsScope, Key: "missing_user", Title: "Flag", ForceOnUserIDs: []domain.UserID{999}}
	if _, err := repo.CreateFlag(ctx, testActor, input); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("missing account: %v", err)
	}

	flags, total, err := repo.ListFlags(ctx, oc.ListOptions{})
	if err != nil || total != 0 || len(flags) != 0 {
		t.Fatal("flag not rolled back", err)
	}

	scope, err := repo.GetScope(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if err := repo.DeleteScope(ctx, testActor, scope.ID, scope.Version); !errors.Is(err, oc.ErrResourceInUse) {
		t.Fatalf("scope in use: %v", err)
	}

	if _, _, err := repo.ListConfigs(ctx, oc.ListOptions{Limit: -1}); !errors.Is(err, oc.ErrInvalidPagination) {
		t.Fatal(err)
	}
}

func TestPostgresSnapshotConsistentDuringWrites(t *testing.T) {
	repo, _ := newRepository(t)
	createScope(t, repo)
	config := createConfig(t, repo)
	ctx := context.Background()
	failures := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Go(func() {
		for range 20 {
			updated, err := repo.UpdateConfig(ctx, testActor, config)
			if err != nil {
				failures <- err
				return
			}

			config = *updated
		}
		failures <- nil
	})
	wg.Go(func() {
		for range 20 {
			snapshot, err := repo.LoadSnapshot(ctx)
			if err != nil {
				failures <- err
				return
			}

			if len(snapshot.Configs) != 1 || snapshot.Revision != snapshot.Configs[0].Version+1 {
				failures <- fmt.Errorf("mixed snapshot: %+v", snapshot)
				return
			}
		}
		failures <- nil
	})
	wg.Wait()
	close(failures)

	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
}
