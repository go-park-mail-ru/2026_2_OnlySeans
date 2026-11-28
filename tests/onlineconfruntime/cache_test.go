package onlineconfruntime_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	service "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
)

func TestPublicRuntimeHomeBlocks(t *testing.T) {
	const blocks = `[{"type":"banner","title":"Premiere"},{"type":"collection","id":2}]`
	raw := oc.SnapshotPayload{
		Revision: 9, GeneratedAt: time.Now().UTC(),
		Configs: []oc.SnapshotConfig{{Scope: "frontend", Key: "home_blocks", Value: json.RawMessage(blocks)}},
		Flags:   []oc.SnapshotFlag{{ID: 1, Scope: "frontend", Key: "home_blocks_enabled", Salt: "stable", Enabled: true, RolloutPercent: 0, ForceOnUserIDs: []domain.UserID{7}}},
	}
	s := snapshotFromPayload(t, raw)
	got := s.PublicRuntime(7)
	require.True(t, got.HomeBlocksEnabled)
	require.Equal(t, int64(9), got.Revision)
	require.JSONEq(t, blocks, string(got.HomeBlocks))

	for _, id := range []domain.UserID{0, 8} {
		got = s.PublicRuntime(id)
		require.False(t, got.HomeBlocksEnabled)
		require.JSONEq(t, "[]", string(got.HomeBlocks))
	}

	raw.Flags[0].RolloutPercent = 100
	raw.Flags[0].ForceOnUserIDs = nil
	raw.Flags[0].ForceOffUserIDs = []domain.UserID{7}
	s = snapshotFromPayload(t, raw)
	require.False(t, s.PublicRuntime(7).HomeBlocksEnabled)
	require.True(t, s.PublicRuntime(8).HomeBlocksEnabled)

	raw.Configs = []oc.SnapshotConfig{}
	s = snapshotFromPayload(t, raw)
	require.JSONEq(t, "[]", string(s.PublicRuntime(8).HomeBlocks))

	raw.Configs = []oc.SnapshotConfig{{Scope: "frontend", Key: "home_blocks", Value: json.RawMessage(`"wrong type"`)}}
	s = snapshotFromPayload(t, raw)
	require.JSONEq(t, "[]", string(s.PublicRuntime(8).HomeBlocks))
	require.False(t, (service.FallbackProvider{}).PublicRuntime(7).SnapshotReady)
	require.JSONEq(t, "[]", string((service.FallbackProvider{}).PublicRuntime(7).HomeBlocks))
}

type sourceResult struct {
	snapshot *service.Snapshot
	err      error
}

type sequenceSource struct {
	mu      sync.Mutex
	results []sourceResult
	calls   int
}

func (s *sequenceSource) Fetch(context.Context) (*service.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls++

	index := s.calls - 1
	if index >= len(s.results) {
		index = len(s.results) - 1
	}

	return s.results[index].snapshot, s.results[index].err
}

func (s *sequenceSource) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.calls
}

func TestCacheColdStartAndLastKnownGood(t *testing.T) {
	wanted := testSnapshotFromJSON(t, 1, true, 250)
	source := &sequenceSource{results: []sourceResult{{snapshot: wanted}, {err: errors.New("source unavailable")}}}
	cache, err := service.NewCache(source, time.Second, testutil.DiscardLogger())
	require.NoError(t, err)

	cold := cache.ReviewPolicy(7)
	require.False(t, cold.WriteEnabled)
	require.Equal(t, domain.MaxReviewContentLen, cold.MaxLength)
	require.Zero(t, cold.Revision)

	require.NoError(t, cache.Refresh(t.Context()))
	require.Equal(t, domain.ReviewPolicy{SnapshotReady: true, WriteEnabled: true, MaxLength: 250, Revision: 1}, cache.ReviewPolicy(7))

	require.Error(t, cache.Refresh(t.Context()))
	require.Equal(t, domain.ReviewPolicy{SnapshotReady: true, WriteEnabled: true, MaxLength: 250, Revision: 1}, cache.ReviewPolicy(7))
	require.Equal(t, int64(1), cache.Status().Revision)
}

func TestReviewPolicyDoesNotFetchOnProductPath(t *testing.T) {
	source := &sequenceSource{results: []sourceResult{{snapshot: testSnapshotFromJSON(t, 4, true, 400)}}}
	cache, err := service.NewCache(source, time.Second, testutil.DiscardLogger())
	require.NoError(t, err)
	require.NoError(t, cache.Refresh(t.Context()))

	for range 100 {
		_ = cache.ReviewPolicy(42)
	}

	require.Equal(t, 1, source.callCount(), "policy reads must use memory only")
}

func TestCacheWorkerUpdatesAndStops(t *testing.T) {
	source := &sequenceSource{results: []sourceResult{
		{snapshot: testSnapshotFromJSON(t, 1, true, 100)},
		{snapshot: testSnapshotFromJSON(t, 2, false, 200)},
	}}
	cache, err := service.NewCache(source, 5*time.Millisecond, testutil.DiscardLogger())
	require.NoError(t, err)
	require.NoError(t, cache.Refresh(t.Context()))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		cache.Run(ctx)
		close(done)
	}()
	require.Eventually(t, func() bool { return cache.ReviewPolicy(7).Revision == 2 }, time.Second, time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cache worker did not stop after cancellation")
	}
}

func TestHTTPClientValidatesAndAuthenticatesSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"revision":9,"generated_at":"2026-11-28T09:00:00Z","configs":[{"scope":"reviews","key":"max_length","value":500}],"flags":[{"scope":"reviews","key":"write_enabled","enabled":true,"id":1,"rollout_percent":100,"salt":"stable","force_on_user_ids":[],"force_off_user_ids":[]}]}`))
	}))
	defer server.Close()

	client, err := service.NewHTTPClient(server.Client(), server.URL, "/snapshot", "secret")
	require.NoError(t, err)
	snapshot, err := client.Fetch(t.Context())
	require.NoError(t, err)
	require.Equal(t, int64(9), snapshot.Revision())
	require.Equal(t, domain.ReviewPolicy{SnapshotReady: true, WriteEnabled: true, MaxLength: 500, Revision: 9}, snapshot.ReviewPolicy(7))
	require.False(t, snapshot.ReviewPolicy(0).WriteEnabled, "guest writes are always disabled")
}

func TestSnapshotRejectsInvalidKnownValueAndLowerRevision(t *testing.T) {
	_, err := fetchPayload(t, oc.SnapshotPayload{
		Revision: 1, GeneratedAt: time.Now().UTC(),
		Configs: []oc.SnapshotConfig{{Scope: "reviews", Key: "max_length", Value: json.RawMessage(`49`)}},
		Flags:   []oc.SnapshotFlag{},
	})
	require.Error(t, err)

	source := &sequenceSource{results: []sourceResult{
		{snapshot: testSnapshotFromJSON(t, 5, true, 500)},
		{snapshot: testSnapshotFromJSON(t, 4, false, 100)},
	}}
	cache, err := service.NewCache(source, time.Second, testutil.DiscardLogger())
	require.NoError(t, err)
	require.NoError(t, cache.Refresh(t.Context()))
	require.ErrorIs(t, cache.Refresh(t.Context()), service.ErrStaleSnapshot)
	require.Equal(t, int64(5), cache.ReviewPolicy(7).Revision)
}

func TestNewSnapshotDeletionUsesPerKeyFallback(t *testing.T) {
	empty, err := fetchPayload(t, oc.SnapshotPayload{Revision: 2, GeneratedAt: time.Now().UTC(), Configs: []oc.SnapshotConfig{}, Flags: []oc.SnapshotFlag{}})
	require.NoError(t, err)
	source := &sequenceSource{results: []sourceResult{
		{snapshot: testSnapshotFromJSON(t, 1, true, 500)},
		{snapshot: empty},
	}}
	cache, err := service.NewCache(source, time.Second, testutil.DiscardLogger())
	require.NoError(t, err)
	require.NoError(t, cache.Refresh(t.Context()))
	require.NoError(t, cache.Refresh(t.Context()))

	require.Equal(t, domain.ReviewPolicy{
		SnapshotReady: true,
		WriteEnabled:  false,
		MaxLength:     domain.MaxReviewContentLen,
		Revision:      2,
	}, cache.ReviewPolicy(7))
}

func testSnapshotFromJSON(t *testing.T, revision int64, enabled bool, maxLength int) *service.Snapshot {
	t.Helper()

	raw := oc.SnapshotPayload{Revision: revision, GeneratedAt: time.Now().UTC()}
	raw.Configs = []oc.SnapshotConfig{{Scope: "reviews", Key: "max_length", Value: json.RawMessage(strconv.Itoa(maxLength))}}
	raw.Flags = []oc.SnapshotFlag{{Scope: "reviews", Key: "write_enabled", ID: 1, Enabled: enabled, RolloutPercent: 100, Salt: "stable"}}
	snapshot, err := fetchPayload(t, raw)
	require.NoError(t, err)

	return snapshot
}

func TestServiceSnapshotContract(t *testing.T) {
	const empty = `{"revision":0,"generated_at":"2026-11-28T09:00:00Z","configs":[],"flags":[]}`
	const active = `{"revision":1,"generated_at":"2026-11-28T09:00:01Z","configs":[{"scope":"reviews","key":"max_length","value":50}],"flags":[{"id":1,"scope":"reviews","key":"write_enabled","enabled":true,"rollout_percent":25,"salt":"stable","force_on_user_ids":[101],"force_off_user_ids":[102]}]}`
	const disabled = `{"revision":2,"generated_at":"2026-11-28T09:00:02Z","configs":[],"flags":[{"id":1,"scope":"reviews","key":"write_enabled","enabled":false,"rollout_percent":100,"salt":"stable","force_on_user_ids":[101],"force_off_user_ids":[]}]}`
	var response atomic.Value
	response.Store(empty)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/onlineconf/snapshot", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response.Load().(string)))
	}))
	defer server.Close()
	client, err := service.NewHTTPClient(server.Client(), server.URL, "/internal/onlineconf/snapshot", "test-token")
	require.NoError(t, err)
	cache, err := service.NewCache(client, time.Second, testutil.DiscardLogger())
	require.NoError(t, err)

	require.NoError(t, cache.Refresh(t.Context()))
	require.Equal(t, domain.ReviewPolicy{SnapshotReady: true, MaxLength: domain.MaxReviewContentLen}, cache.ReviewPolicy(7))

	response.Store(active)
	require.NoError(t, cache.Refresh(t.Context()))
	require.Equal(t, domain.ReviewPolicy{SnapshotReady: true, Revision: 1, MaxLength: 50, WriteEnabled: true}, cache.ReviewPolicy(7)) // Bucket 2306.
	require.False(t, cache.ReviewPolicy(42).WriteEnabled)                                                                             // Bucket 6252.
	require.True(t, cache.ReviewPolicy(101).WriteEnabled)
	require.False(t, cache.ReviewPolicy(0).WriteEnabled)

	response.Store(strings.Replace(active, `"rollout_percent":25`, `"rollout_percent":100`, 1))
	response.Store(strings.Replace(response.Load().(string), `"revision":1`, `"revision":2`, 1))
	require.NoError(t, cache.Refresh(t.Context()))
	require.False(t, cache.ReviewPolicy(102).WriteEnabled)

	response.Store(strings.Replace(disabled, `"revision":2`, `"revision":3`, 1))
	require.NoError(t, cache.Refresh(t.Context()))
	require.Equal(t, domain.ReviewPolicy{SnapshotReady: true, Revision: 3, MaxLength: domain.MaxReviewContentLen}, cache.ReviewPolicy(101))

	response.Store(strings.Replace(empty, `"revision":0,`, "", 1))
	require.Error(t, cache.Refresh(t.Context()))
	require.Equal(t, int64(3), cache.ReviewPolicy(101).Revision)
}

func fetchPayload(t *testing.T, payload oc.SnapshotPayload) (*service.Snapshot, error) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(payload); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client, err := service.NewHTTPClient(server.Client(), server.URL, oc.SnapshotAPIPath, "test-token")
	require.NoError(t, err)

	return client.Fetch(t.Context())
}

func snapshotFromPayload(t *testing.T, payload oc.SnapshotPayload) *service.Snapshot {
	t.Helper()
	snapshot, err := fetchPayload(t, payload)
	require.NoError(t, err)

	return snapshot
}
