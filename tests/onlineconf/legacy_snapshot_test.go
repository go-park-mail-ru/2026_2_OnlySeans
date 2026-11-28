package onlineconf_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	service "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
)

const (
	legacyArrayItems       = 17000
	legacyArrayJSONBBytes  = 51000
	parallelSchemaFixtures = 8
	legacySnapshotRevision = 3
	seedLegacyStateQuery   = `UPDATE oc_state SET revision=$1,generated_at=now(),snapshot=jsonb_build_object(
 'revision',$1::bigint,'generated_at',now(),
 'scopes',(SELECT jsonb_agg(to_jsonb(s) ORDER BY s.id) FROM oc_scopes s),
 'configs',(SELECT jsonb_agg((to_jsonb(c)-'scope_id') || jsonb_build_object('scope',s.name) ORDER BY c.id) FROM oc_configs c JOIN oc_scopes s ON s.id=c.scope_id),
 'flags','[]'::jsonb) WHERE id=1`
)

func TestLegacyNumericAndUnicodeValuesRemainValid(t *testing.T) {
	for _, raw := range []string{
		"[" + strings.Repeat("1,", legacyArrayItems-1) + "1]",
		"[" + strings.Repeat("1, ", legacyArrayItems-1) + "1]",
		`"` + strings.Repeat("\u2028", legacyArrayItems) + `"`,
		`"` + strings.Repeat(`\u2028`, legacyArrayItems) + `"`,
	} {
		if err := oc.ValidateConfigValue("frontend", "legacy", json.RawMessage(raw)); err != nil {
			t.Fatal("legacy value rejected", err)
		}
	}
}

func TestJSONNumberNormalizationChoosesShortestExactForm(t *testing.T) {
	cases := []struct {
		input    string
		shortest string
	}{
		{"1", "1"}, {"10", "10"}, {"100", "100"}, {"1000", "1e3"},
		{"1.00", "1"}, {"1.25", "1.25"}, {"0.001", "1e-3"}, {"0.1", "0.1"},
		{"1e-100000000000000000000", "1e-100000000000000000000"},
		{"1e100000000000000000000", "1e100000000000000000000"},
		{"-0.00", "0"}, {"-123.4500", "-123.45"}, {"9007199254740993", "9007199254740993"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			overhead := len(`{"number":,"padding":""}`)
			padding := strings.Repeat("x", oc.MaxValueBytes-overhead-len(tc.shortest))

			value := json.RawMessage(`{"number":` + tc.input + `,"padding":"` + padding + `"}`)
			if err := oc.ValidateConfigValue("frontend", "number", value); err != nil {
				t.Fatal("exact boundary rejected", err)
			}

			oversized := json.RawMessage(`{"number":` + tc.input + `,"padding":"` + padding + `x"}`)
			if err := oc.ValidateConfigValue("frontend", "number", oversized); err != oc.ErrValueTooLarge {
				t.Fatal("oversized value accepted", err)
			}
		})
	}
}

func TestPostgresUpgradeLoadsLegacySnapshotWithoutChanges(t *testing.T) {
	repo, db := newRepository(t)
	ctx := context.Background()

	scope, err := repo.CreateScope(ctx, testActor, oc.Scope{Name: "frontend", Title: "Frontend"})
	if err != nil {
		t.Fatal(err)
	}

	fixtures := []struct {
		key   string
		value json.RawMessage
	}{
		{"legacy_array", json.RawMessage("[" + strings.Repeat("1,", legacyArrayItems-1) + "1]")},
		{"legacy_unicode", json.RawMessage(`"` + strings.Repeat("\u2028", legacyArrayItems) + `"`)},
	}
	for _, fixture := range fixtures {
		if !json.Valid(fixture.value) || len(fixture.value) > oc.MaxValueBytes {
			t.Fatal("fixture violates old value rules")
		}

		if _, err := db.ExecContext(ctx, `INSERT INTO oc_configs(scope_id,key,title,value) VALUES($1,$2,$3,$4)`, scope.ID, fixture.key, fixture.key, []byte(fixture.value)); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := db.ExecContext(ctx, seedLegacyStateQuery, legacySnapshotRevision); err != nil {
		t.Fatal(err)
	}

	var oldState []byte
	if err := db.QueryRowContext(ctx, `SELECT snapshot FROM oc_state WHERE id=1`).Scan(&oldState); err != nil {
		t.Fatal(err)
	}

	oldValues := make(map[string][]byte)

	for _, fixture := range fixtures {
		var value []byte
		if err := db.QueryRowContext(ctx, `SELECT value FROM oc_configs WHERE key=$1`, fixture.key).Scan(&value); err != nil {
			t.Fatal(err)
		}

		if !json.Valid(value) || len(value) > oc.MaxValueBytes {
			t.Fatal("old PostgreSQL value was not acceptable")
		}

		if fixture.key == "legacy_array" && len(value) != legacyArrayJSONBBytes {
			t.Fatal("incorrect old array size", len(value))
		}

		oldValues[fixture.key] = value
	}

	store, err := service.NewSnapshotStore(repo, service.SnapshotOptions{}, testutil.DiscardLogger())
	if err != nil {
		t.Fatal(err)
	}

	workerCtx, cancel := context.WithCancel(t.Context())
	t.Cleanup(func() { cancel(); <-store.Done() })

	if err := store.Start(workerCtx); err != nil {
		t.Fatal("new service failed to start with legacy database", err)
	}

	router := newHTTPRouterWithRefresher(t, repo, store, store)

	response := httpRequest(router, http.MethodGet, oc.SnapshotAPIPath, "", "", accessTestToken)
	if response.Code != http.StatusOK {
		t.Fatal(response.Code, response.Body.String())
	}

	var payload oc.SnapshotPayload

	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}

	if err := payload.Validate(); err != nil {
		t.Fatal("new client rejected legacy snapshot", err)
	}

	if payload.Revision != legacySnapshotRevision || len(payload.Configs) != len(fixtures) {
		t.Fatal("legacy snapshot changed")
	}

	for _, config := range payload.Configs {
		if config.Key == "legacy_array" {
			var values []int
			if err := json.Unmarshal(config.Value, &values); err != nil {
				t.Fatal(err)
			}

			if len(values) != legacyArrayItems {
				t.Fatal("array length changed")
			}

			for _, value := range values {
				if value != 1 {
					t.Fatal("array contents changed")
				}
			}
		}
	}

	var afterState []byte
	if err := db.QueryRowContext(ctx, `SELECT snapshot FROM oc_state WHERE id=1`).Scan(&afterState); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(oldState, afterState) {
		t.Fatal("startup rewrote persisted snapshot")
	}

	for _, fixture := range fixtures {
		var value []byte

		var version int64
		if err := db.QueryRowContext(ctx, `SELECT value,version FROM oc_configs WHERE key=$1`, fixture.key).Scan(&value, &version); err != nil {
			t.Fatal(err)
		}

		if version != 1 || !bytes.Equal(value, oldValues[fixture.key]) {
			t.Fatal("startup rewrote legacy config")
		}
	}
}

func TestPostgresSchemasAreUniqueAcrossParallelFixtures(t *testing.T) {
	var schemas sync.Map

	for index := range parallelSchemaFixtures {
		t.Run(fmt.Sprintf("fixture_%d", index), func(t *testing.T) {
			t.Parallel()
			_, db := newRepository(t)
			var schema string

			if err := db.QueryRowContext(t.Context(), `SELECT current_schema()`).Scan(&schema); err != nil {
				t.Fatal(err)
			}

			if _, exists := schemas.LoadOrStore(schema, struct{}{}); exists {
				t.Fatal("test schema reused", schema)
			}
		})
	}
}
