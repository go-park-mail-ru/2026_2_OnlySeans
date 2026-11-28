package onlineconf

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

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
	"github.com/stretchr/testify/assert"
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
	require.Equal(t, domain.ReviewPolicy{SnapshotReady: true, WriteEnabled: true, MaxLength: 250, Revision: 1}, cache.ReviewPolicy(7))

	require.Error(t, cache.Refresh(t.Context()))
	require.Equal(t, domain.ReviewPolicy{SnapshotReady: true, WriteEnabled: true, MaxLength: 250, Revision: 1}, cache.ReviewPolicy(7))
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
		_, _ = w.Write([]byte(`{"revision":9,"generated_at":"2026-11-28T09:00:00Z","configs":[{"scope":"reviews","key":"max_length","value":500}],"flags":[{"scope":"reviews","key":"write_enabled","enabled":true,"id":1,"rollout_percent":100,"salt":"stable","force_on_user_ids":[],"force_off_user_ids":[]}]}`))
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.Client(), server.URL, "/snapshot", "secret")
	require.NoError(t, err)
	snapshot, err := client.Fetch(t.Context())
	require.NoError(t, err)
	require.Equal(t, int64(9), snapshot.Revision())
	require.Equal(t, domain.ReviewPolicy{SnapshotReady: true, WriteEnabled: true, MaxLength: 500, Revision: 9}, snapshot.reviewPolicy(7))
	require.False(t, snapshot.reviewPolicy(0).WriteEnabled, "guest writes are always disabled")
}

func TestSnapshotRejectsInvalidKnownValueAndLowerRevision(t *testing.T) {
	_, err := buildSnapshot(snapshotDTO{
		Revision: 1,
		Configs:  []configDTO{{Scope: reviewsScope, Key: maxLengthSnapshotKey, Value: json.RawMessage(`49`)}},
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
		SnapshotReady: true,
		WriteEnabled:  false,
		MaxLength:     domain.MaxReviewContentLen,
		Revision:      2,
	}, cache.ReviewPolicy(7))
}

func testSnapshotFromJSON(t *testing.T, revision int64, enabled bool, maxLength int) *Snapshot {
	t.Helper()
	raw := snapshotDTO{Revision: revision}
	raw.Configs = []configDTO{{Scope: reviewsScope, Key: maxLengthSnapshotKey, Value: json.RawMessage(strconv.Itoa(maxLength))}}
	rollout := 100
	raw.Flags = []flagDTO{{Scope: reviewsScope, Key: writeEnabledSnapshotKey, Enabled: enabled, Rollout: &rollout, Salt: "stable"}}
	snapshot, err := buildSnapshot(raw)
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
	client, err := NewHTTPClient(server.Client(), server.URL, "/internal/onlineconf/snapshot", "test-token")
	require.NoError(t, err)
	cache, err := NewCache(client, time.Second, testutil.DiscardLogger())
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
