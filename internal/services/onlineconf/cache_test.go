package onlineconf

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
	"github.com/stretchr/testify/require"
)

type sourceResult struct {
	snapshot *Snapshot
	err      error
}

type sequenceSource struct {
	mu      sync.Mutex
	results []sourceResult
	calls   int
}

func (s *sequenceSource) Fetch(context.Context) (*Snapshot, error) {
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
	cache, err := NewCache(source, time.Second, testutil.DiscardLogger())
	require.NoError(t, err)

	cold := cache.ReviewPolicy(7)
	require.False(t, cold.WriteEnabled)
	require.Equal(t, domain.MaxReviewContentLen, cold.MaxLength)
	require.Zero(t, cold.Revision)

	require.NoError(t, cache.Refresh(t.Context()))
	require.Equal(t, domain.ReviewPolicy{WriteEnabled: true, MaxLength: 250, Revision: 1}, cache.ReviewPolicy(7))

	require.Error(t, cache.Refresh(t.Context()))
	require.Equal(t, domain.ReviewPolicy{WriteEnabled: true, MaxLength: 250, Revision: 1}, cache.ReviewPolicy(7))
	require.Equal(t, int64(1), cache.Status().Revision)
}

func TestReviewPolicyDoesNotFetchOnProductPath(t *testing.T) {
	source := &sequenceSource{results: []sourceResult{{snapshot: testSnapshotFromJSON(t, 4, true, 400)}}}
	cache, err := NewCache(source, time.Second, testutil.DiscardLogger())
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
	cache, err := NewCache(source, 5*time.Millisecond, testutil.DiscardLogger())
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
		_, _ = w.Write([]byte(`{"revision":9,"configs":[{"scope":"backend","key":"reviews.max_length","value":500}],"flags":[{"scope":"backend","key":"reviews.write_enabled","enabled":true,"rollout":100,"included_user_ids":[],"excluded_user_ids":[]}]}`))
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.Client(), server.URL, "/snapshot", "secret")
	require.NoError(t, err)
	snapshot, err := client.Fetch(t.Context())
	require.NoError(t, err)
	require.Equal(t, int64(9), snapshot.Revision())
	require.Equal(t, domain.ReviewPolicy{WriteEnabled: true, MaxLength: 500, Revision: 9}, snapshot.reviewPolicy(7))
	require.False(t, snapshot.reviewPolicy(0).WriteEnabled, "guest writes are always disabled")
}

func TestSnapshotRejectsInvalidKnownValueAndLowerRevision(t *testing.T) {
	_, err := buildSnapshot(snapshotDTO{
		Revision: 1,
		Configs:  []configDTO{{Scope: ProductBackendScope, Key: domain.ReviewMaxLengthKey, Value: json.RawMessage(`49`)}},
	})
	require.Error(t, err)

	source := &sequenceSource{results: []sourceResult{
		{snapshot: testSnapshotFromJSON(t, 5, true, 500)},
		{snapshot: testSnapshotFromJSON(t, 4, false, 100)},
	}}
	cache, err := NewCache(source, time.Second, testutil.DiscardLogger())
	require.NoError(t, err)
	require.NoError(t, cache.Refresh(t.Context()))
	require.ErrorIs(t, cache.Refresh(t.Context()), ErrStaleSnapshot)
	require.Equal(t, int64(5), cache.ReviewPolicy(7).Revision)
}

func TestNewSnapshotDeletionUsesPerKeyFallback(t *testing.T) {
	empty, err := buildSnapshot(snapshotDTO{Revision: 2})
	require.NoError(t, err)
	source := &sequenceSource{results: []sourceResult{
		{snapshot: testSnapshotFromJSON(t, 1, true, 500)},
		{snapshot: empty},
	}}
	cache, err := NewCache(source, time.Second, testutil.DiscardLogger())
	require.NoError(t, err)
	require.NoError(t, cache.Refresh(t.Context()))
	require.NoError(t, cache.Refresh(t.Context()))

	require.Equal(t, domain.ReviewPolicy{
		WriteEnabled: false,
		MaxLength:    domain.MaxReviewContentLen,
		Revision:     2,
	}, cache.ReviewPolicy(7))
}

func testSnapshotFromJSON(t *testing.T, revision int64, enabled bool, maxLength int) *Snapshot {
	t.Helper()
	raw := snapshotDTO{Revision: revision}
	raw.Configs = []configDTO{{Scope: ProductBackendScope, Key: domain.ReviewMaxLengthKey, Value: json.RawMessage(strconv.Itoa(maxLength))}}
	rollout := 100
	raw.Flags = []flagDTO{{Scope: ProductBackendScope, Key: domain.ReviewWriteEnabledKey, Enabled: enabled, Rollout: &rollout}}
	snapshot, err := buildSnapshot(raw)
	require.NoError(t, err)
	return snapshot
}
