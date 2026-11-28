package onlineconf_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	service "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconf"
)

const (
	boundaryConfigCount       = oc.MaxSnapshotBytes / oc.MaxValueBytes
	snapshotLineEnding        = "\n"
	integrationBoundaryMargin = 32
	concurrentCapacityReserve = 150
	timestampPrecisionMargin  = 16
)

func sizedSnapshotPayload(t *testing.T, wireBytes int) oc.SnapshotPayload {
	t.Helper()

	payload := oc.SnapshotPayload{Revision: 2, GeneratedAt: time.Date(2026, time.November, 28, 9, 0, 0, 123456000, time.UTC), Configs: make([]oc.SnapshotConfig, 0, boundaryConfigCount), Flags: []oc.SnapshotFlag{}}

	for index := range boundaryConfigCount {
		value := json.RawMessage(`""`)
		if index < boundaryConfigCount-1 {
			value = json.RawMessage(`"` + strings.Repeat("a", oc.MaxValueBytes-len(`""`)) + `"`)
		}

		payload.Configs = append(payload.Configs, oc.SnapshotConfig{Scope: "frontend", Key: fmt.Sprintf("value_%d", index), Value: value})
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	padding := wireBytes - len(encoded) - len(snapshotLineEnding)
	if padding < 0 || padding+len(`""`) > oc.MaxValueBytes {
		t.Fatal("invalid fixture padding", padding)
	}

	payload.Configs[len(payload.Configs)-1].Value = json.RawMessage(`"` + strings.Repeat("a", padding) + `"`)

	return payload
}

func TestSnapshotSerializedSizeBoundary(t *testing.T) {
	for _, size := range []int{oc.MaxSnapshotBytes - 1, oc.MaxSnapshotBytes, oc.MaxSnapshotBytes + 1} {
		payload := sizedSnapshotPayload(t, size)

		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}

		if len(encoded)+len(snapshotLineEnding) != size {
			t.Fatal("incorrect fixture size")
		}

		err = payload.Validate()
		if size <= oc.MaxSnapshotBytes && err != nil {
			t.Fatal(size, err)
		}

		if size > oc.MaxSnapshotBytes && !errors.Is(err, oc.ErrSnapshotTooLarge) {
			t.Fatal(size, err)
		}
	}
}

func TestSnapshotSizeAccountsForHTMLEscaping(t *testing.T) {
	payload := sizedSnapshotPayload(t, oc.MaxSnapshotBytes)
	last := &payload.Configs[len(payload.Configs)-1]
	last.Value = bytes.Replace(last.Value, []byte("a"), []byte("<"), 1)

	if len(last.Value) > oc.MaxValueBytes {
		t.Fatal("individual value limit exceeded")
	}

	if err := payload.ValidateSize(); !errors.Is(err, oc.ErrSnapshotTooLarge) {
		t.Fatal("escaped JSON exceeded limit without rejection", err)
	}
}

func TestSnapshotHTTPSizeBoundary(t *testing.T) {
	source := &snapshotStub{}
	router := newHTTPRouter(t, &handlerRepository{}, source)

	for _, size := range []int{oc.MaxSnapshotBytes, oc.MaxSnapshotBytes + 1} {
		source.value = snapshotFromPayload(sizedSnapshotPayload(t, size))

		response := httpRequest(router, http.MethodGet, oc.SnapshotAPIPath, "", "", accessTestToken)
		if size <= oc.MaxSnapshotBytes {
			if response.Code != http.StatusOK || response.Body.Len() != size {
				t.Fatalf("wire boundary: status=%d size=%d", response.Code, response.Body.Len())
			}
		} else {
			if response.Code != http.StatusRequestEntityTooLarge {
				t.Fatal(response.Code, response.Body.String())
			}

			var body struct {
				Code string `json:"code"`
			}

			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}

			if body.Code != "snapshot_too_large" {
				t.Fatal(body.Code)
			}
		}
	}
}

func TestSnapshotStoreKeepsLastSnapshotWhenNewOneTooLarge(t *testing.T) {
	loader := &controlledSnapshotLoader{value: snapshotFixture(1)}
	store := startSnapshotStore(t, loader, service.SnapshotOptions{})
	oversized := snapshotFromPayload(sizedSnapshotPayload(t, oc.MaxSnapshotBytes+1))
	loader.set(oversized, nil, 0)
	store.RequestRefresh()
	waitForSnapshot(t, func() bool { return loader.completed.Load() >= 2 })
	store.RequestRefresh()
	waitForSnapshot(t, func() bool { return loader.completed.Load() >= 3 })

	current, err := store.Current()
	if err != nil || current.Revision != 1 {
		t.Fatal("oversized snapshot replaced valid state", err)
	}

	loader.set(snapshotFixture(3), nil, 0)
	store.RequestRefresh()
	waitForSnapshot(t, func() bool { current, err := store.Current(); return err == nil && current.Revision == 3 })
}

func TestPostgresSnapshotSizeRollbackAndConcurrentCapacity(t *testing.T) {
	repo, db := newRepository(t)
	ctx := context.Background()

	scope, err := repo.CreateScope(ctx, testActor, oc.Scope{Name: "frontend", Title: "Frontend"})
	if err != nil {
		t.Fatal(err)
	}

	payload := sizedSnapshotPayload(t, oc.MaxSnapshotBytes-integrationBoundaryMargin)
	var lastID int64

	for _, config := range payload.Configs {
		if err := db.QueryRowContext(ctx, `INSERT INTO oc_configs(scope_id,key,title,value) VALUES($1,$2,$3,$4) RETURNING id`, scope.ID, config.Key, config.Key, []byte(config.Value)).Scan(&lastID); err != nil {
			t.Fatal(err)
		}
	}

	last, err := repo.GetConfig(ctx, lastID)
	if err != nil {
		t.Fatal(err)
	}

	accepted, err := repo.UpdateConfig(ctx, testActor, *last)
	if err != nil {
		t.Fatal("near-limit snapshot rejected", err)
	}

	before, err := repo.LoadSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}

	beforeWire, err := json.Marshal(oc.NewSnapshotPayload(before))
	if err != nil {
		t.Fatal(err)
	}

	if len(beforeWire)+len(snapshotLineEnding) > oc.MaxSnapshotBytes {
		t.Fatal("invalid accepted size")
	}

	beforeAudit, _, err := repo.ListAudit(ctx, oc.AuditFilter{ResourceKind: oc.ResourceConfig, ResourceID: lastID})
	if err != nil {
		t.Fatal(err)
	}

	var savedBefore []byte
	if err := db.QueryRowContext(ctx, `SELECT snapshot FROM oc_state WHERE id=1`).Scan(&savedBefore); err != nil {
		t.Fatal(err)
	}

	rejected := *accepted
	var text string

	if err := json.Unmarshal(rejected.Value, &text); err != nil {
		t.Fatal(err)
	}

	oversized, err := json.Marshal(text + strings.Repeat("a", integrationBoundaryMargin+timestampPrecisionMargin))
	if err != nil {
		t.Fatal(err)
	}

	rejected.Value = oversized
	if _, err := repo.UpdateConfig(ctx, testActor, rejected); !errors.Is(err, oc.ErrSnapshotTooLarge) {
		t.Fatal("oversized update not rejected", err)
	}

	after, err := repo.GetConfig(ctx, lastID)
	if err != nil {
		t.Fatal(err)
	}

	if after.Version != accepted.Version || !bytes.Equal(after.Value, accepted.Value) {
		t.Fatal("oversized update was not rolled back")
	}

	afterAudit, _, err := repo.ListAudit(ctx, oc.AuditFilter{ResourceKind: oc.ResourceConfig, ResourceID: lastID})
	if err != nil {
		t.Fatal(err)
	}

	if len(afterAudit) != len(beforeAudit) {
		t.Fatal("audit survived rejected update")
	}

	var savedAfter []byte
	if err := db.QueryRowContext(ctx, `SELECT snapshot FROM oc_state WHERE id=1`).Scan(&savedAfter); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(savedBefore, savedAfter) {
		t.Fatal("persisted snapshot changed after rollback")
	}

	current, err := repo.LoadSnapshot(ctx)
	if err != nil || current.Revision != before.Revision {
		t.Fatal("revision changed after rollback", err)
	}

	router := newHTTPRouter(t, repo, &snapshotStub{value: before})

	response := httpRequest(router, http.MethodPut, fmt.Sprintf("%s/configs/%d", adminAPI, lastID), string(mustJSON(t, map[string]any{"scope": rejected.Scope, "key": rejected.Key, "title": rejected.Title, "description": rejected.Description, "value": rejected.Value, "version": rejected.Version})), "admin", "")
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatal(response.Code, response.Body.String())
	}

	after.Value = mustJSON(t, text[:len(text)-concurrentCapacityReserve])
	if _, err := repo.UpdateConfig(ctx, testActor, *after); err != nil {
		t.Fatal("capacity not released", err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)

	for _, prefix := range []string{"a", "b"} {
		key := prefix + strings.Repeat("x", 63)

		wg.Go(func() {
			_, err := repo.CreateConfig(ctx, testActor, oc.Config{Scope: "frontend", Key: key, Title: "Concurrent", Value: json.RawMessage(`null`)})
			results <- err
		})
	}

	wg.Wait()
	close(results)

	successes, rejections := 0, 0

	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, oc.ErrSnapshotTooLarge):
			rejections++
		default:
			t.Fatal(err)
		}
	}

	if successes != 1 || rejections != 1 {
		t.Fatalf("capacity race: successes=%d rejections=%d", successes, rejections)
	}

	final, err := repo.LoadSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := oc.NewSnapshotPayload(final).ValidateSize(); err != nil {
		t.Fatal(err)
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}

	return encoded
}
