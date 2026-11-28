package onlineconf_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	service "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
)

const (
	snapshotTestWait = 2 * time.Second
	snapshotTestPoll = time.Millisecond
)

type controlledSnapshotLoader struct {
	mu          sync.Mutex
	value       *oc.Snapshot
	err         error
	delay       time.Duration
	active      atomic.Int32
	overlapping atomic.Bool
	completed   atomic.Int32
}

func (l *controlledSnapshotLoader) LoadSnapshot(ctx context.Context) (*oc.Snapshot, error) {
	if l.active.Add(1) > 1 {
		l.overlapping.Store(true)
	}
	defer l.active.Add(-1)
	defer l.completed.Add(1)
	l.mu.Lock()
	value, err, delay := l.value, l.err, l.delay
	l.mu.Unlock()

	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	return value, err
}

func (l *controlledSnapshotLoader) set(value *oc.Snapshot, err error, delay time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.value, l.err, l.delay = value, err, delay
}

func snapshotFixture(revision int64) *oc.Snapshot {
	return &oc.Snapshot{Revision: revision, GeneratedAt: time.Now().UTC(), Configs: []oc.Config{{Scope: "frontend", Key: "counter", Title: "Counter", Value: json.RawMessage(`1`)}}, Flags: []oc.FeatureFlag{{ID: 1, Scope: "frontend", Key: "test", Title: "Test", Salt: "stable", Enabled: true, ForceOnUserIDs: []domain.UserID{testActor}}}}
}

func waitForSnapshot(t *testing.T, condition func() bool) {
	t.Helper()

	timer := time.NewTimer(snapshotTestWait)
	defer timer.Stop()

	ticker := time.NewTicker(snapshotTestPoll)
	defer ticker.Stop()

	for {
		if condition() {
			return
		}
		select {
		case <-timer.C:
			t.Fatal("snapshot condition timed out")
		case <-ticker.C:
		}
	}
}

func startSnapshotStore(t *testing.T, loader *controlledSnapshotLoader, options service.SnapshotOptions) *service.SnapshotStore {
	t.Helper()

	store, err := service.NewSnapshotStore(loader, options, testutil.DiscardLogger())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := store.Start(ctx); err != nil {
		cancel()
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cancel()
		select {
		case <-store.Done():
		case <-time.After(snapshotTestWait):
			t.Error("snapshot worker did not stop")
		}
	})

	return store
}

func TestSnapshotStoreStartupAndCopyIsolation(t *testing.T) {
	source := snapshotFixture(1)
	loader := &controlledSnapshotLoader{value: source}
	store := startSnapshotStore(t, loader, service.SnapshotOptions{})

	current, err := store.Current()
	if err != nil || current.Revision != 1 {
		t.Fatalf("current: %v %v", current, err)
	}

	current.Configs[0].Value[0] = '2'
	current.Flags[0].ForceOnUserIDs[0] = testOtherUser
	source.Configs[0].Value[0] = '3'
	source.Flags[0].ForceOnUserIDs[0] = testOtherUser

	unchanged, err := store.Current()
	if err != nil || string(unchanged.Configs[0].Value) != "1" || unchanged.Flags[0].ForceOnUserIDs[0] != testActor {
		t.Fatal("published snapshot was mutable", err)
	}

	if err := store.Start(context.Background()); !errors.Is(err, oc.ErrSnapshotWorkerRunning) {
		t.Fatal(err)
	}
}

func TestSnapshotStoreCommitSignalFailureAndRevision(t *testing.T) {
	loader := &controlledSnapshotLoader{value: snapshotFixture(1)}
	store := startSnapshotStore(t, loader, service.SnapshotOptions{})
	loader.set(snapshotFixture(2), nil, 0)
	store.RequestRefresh()
	waitForSnapshot(t, func() bool { current, err := store.Current(); return err == nil && current.Revision == 2 })

	for _, next := range []struct {
		snapshot *oc.Snapshot
		err      error
	}{
		{err: errors.New("database unavailable")},
		{snapshot: snapshotFixture(1)},
		{snapshot: &oc.Snapshot{Revision: 3}},
	} {
		completed := loader.completed.Load()
		loader.set(next.snapshot, next.err, 0)
		store.RequestRefresh()
		waitForSnapshot(t, func() bool { return loader.completed.Load() > completed })

		current, err := store.Current()
		if err != nil || current.Revision != 2 {
			t.Fatalf("last successful snapshot lost: %v %v", current, err)
		}
	}

	loader.set(snapshotFixture(4), nil, 0)
	store.RequestRefresh()
	waitForSnapshot(t, func() bool { current, err := store.Current(); return err == nil && current.Revision == 4 })
}

func TestSnapshotStorePeriodicRefreshAndNoOverlap(t *testing.T) {
	loader := &controlledSnapshotLoader{value: snapshotFixture(1)}
	store := startSnapshotStore(t, loader, service.SnapshotOptions{RefreshInterval: 10 * time.Millisecond, LoadTimeout: time.Second})
	loader.set(snapshotFixture(2), nil, 5*time.Millisecond)
	waitForSnapshot(t, func() bool { current, err := store.Current(); return err == nil && current.Revision == 2 })
	loader.set(snapshotFixture(3), nil, 5*time.Millisecond)

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			for range 20 {
				store.RequestRefresh()

				current, err := store.Current()
				if err == nil {
					current.Configs[0].Value[0] = '9'
				}
			}
		})
	}

	wg.Wait()
	waitForSnapshot(t, func() bool { current, err := store.Current(); return err == nil && current.Revision == 3 })

	if loader.overlapping.Load() {
		t.Fatal("snapshot loads overlapped")
	}
}

func TestSnapshotStoreTimeoutAndShutdown(t *testing.T) {
	loader := &controlledSnapshotLoader{value: snapshotFixture(1)}
	store := startSnapshotStore(t, loader, service.SnapshotOptions{LoadTimeout: 10 * time.Millisecond})
	completed := loader.completed.Load()
	loader.set(snapshotFixture(2), nil, time.Second)
	store.RequestRefresh()
	waitForSnapshot(t, func() bool { return loader.completed.Load() > completed })

	current, err := store.Current()
	if err != nil || current.Revision != 1 {
		t.Fatal("timeout replaced valid snapshot", err)
	}
}

func TestSnapshotStoreFailedStartup(t *testing.T) {
	loader := &controlledSnapshotLoader{err: context.DeadlineExceeded}

	store, err := service.NewSnapshotStore(loader, service.SnapshotOptions{}, testutil.DiscardLogger())
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Start(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}

	if _, err := store.Current(); !errors.Is(err, oc.ErrSnapshotNotReady) {
		t.Fatal(err)
	}
	select {
	case <-store.Done():
	default:
		t.Fatal("failed startup left worker running")
	}
}

func TestSnapshotStoreShutdownCancelsInFlightLoad(t *testing.T) {
	loader := &controlledSnapshotLoader{value: snapshotFixture(1)}

	store, err := service.NewSnapshotStore(loader, service.SnapshotOptions{}, testutil.DiscardLogger())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	if err := store.Start(ctx); err != nil {
		t.Fatal(err)
	}

	loader.set(snapshotFixture(2), nil, time.Hour)
	store.RequestRefresh()
	waitForSnapshot(t, func() bool { return loader.active.Load() != 0 })
	cancel()
	select {
	case <-store.Done():
	case <-time.After(snapshotTestWait):
		t.Fatal("shutdown did not cancel loader")
	}

	current, err := store.Current()
	if err != nil || current.Revision != 1 {
		t.Fatal("canceled load was published", err)
	}
}
