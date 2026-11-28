package onlineconf

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const (
	DefaultSnapshotRefreshInterval = 10 * time.Second
	DefaultSnapshotLoadTimeout     = 2 * time.Second
	refreshSignalCapacity          = 1
)

type SnapshotLoader interface {
	LoadSnapshot(context.Context) (*oc.Snapshot, error)
}

type SnapshotOptions struct {
	RefreshInterval time.Duration
	LoadTimeout     time.Duration
}

type SnapshotStore struct {
	loader      SnapshotLoader
	appLogger   logger.Logger
	options     SnapshotOptions
	current     atomic.Pointer[oc.Snapshot]
	refreshMu   sync.Mutex
	lifecycleMu sync.Mutex
	requests    chan struct{}
	done        chan struct{}
	running     bool
}

func NewSnapshotStore(loader SnapshotLoader, options SnapshotOptions, appLogger logger.Logger) (*SnapshotStore, error) {
	if loader == nil {
		return nil, oc.ErrNilSnapshotLoader
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	if options.RefreshInterval == 0 {
		options.RefreshInterval = DefaultSnapshotRefreshInterval
	}

	if options.LoadTimeout == 0 {
		options.LoadTimeout = DefaultSnapshotLoadTimeout
	}

	if options.RefreshInterval < 0 || options.RefreshInterval > DefaultSnapshotRefreshInterval || options.LoadTimeout < 0 || options.LoadTimeout > DefaultSnapshotLoadTimeout {
		return nil, oc.ErrInvalidSnapshotTiming
	}

	done := make(chan struct{})
	close(done)

	return &SnapshotStore{loader: loader, appLogger: appLogger, options: options, requests: make(chan struct{}, refreshSignalCapacity), done: done}, nil
}

func (s *SnapshotStore) Start(ctx context.Context) error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	if s.running {
		return oc.ErrSnapshotWorkerRunning
	}

	ticker := time.NewTicker(s.options.RefreshInterval)
	if err := s.refresh(ctx); err != nil {
		ticker.Stop()
		return fmt.Errorf("load initial onlineconf snapshot: %w", err)
	}

	s.running = true
	done := make(chan struct{})
	s.done = done

	go s.run(ctx, ticker, done)

	return nil
}

func (s *SnapshotStore) Done() <-chan struct{} {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	return s.done
}

func (s *SnapshotStore) RequestRefresh() {
	select {
	case s.requests <- struct{}{}:
	default:
	}
}

func (s *SnapshotStore) Current() (*oc.Snapshot, error) {
	snapshot := s.current.Load()
	if snapshot == nil {
		return nil, oc.ErrSnapshotNotReady
	}

	return cloneSnapshot(snapshot), nil
}

func (s *SnapshotStore) run(ctx context.Context, ticker *time.Ticker, done chan struct{}) {
	defer ticker.Stop()
	defer func() {
		s.lifecycleMu.Lock()
		s.running = false

		close(done)
		s.lifecycleMu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-s.requests:
		}

		if err := s.refresh(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}

			s.appLogger.ErrorContext(ctx, "onlineconf snapshot refresh failed", nil)
		}
	}
}

func (s *SnapshotStore) refresh(ctx context.Context) error {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("refresh onlineconf snapshot: %w", err)
	}

	loadCtx, cancel := context.WithTimeout(ctx, s.options.LoadTimeout)
	defer cancel()

	loaded, err := s.loader.LoadSnapshot(loadCtx)
	if err != nil {
		return fmt.Errorf("load onlineconf snapshot: %w", err)
	}

	if err := loadCtx.Err(); err != nil {
		return fmt.Errorf("onlineconf snapshot load deadline: %w", err)
	}

	if loaded == nil {
		return oc.ErrInvalidSnapshot
	}

	next := cloneSnapshot(loaded)
	if err := validateSnapshot(next); err != nil {
		return err
	}

	if err := oc.NewSnapshotPayload(next).ValidateSize(); err != nil {
		return fmt.Errorf("validate snapshot before publication: %w", err)
	}

	current := s.current.Load()

	if current != nil && next.Revision <= current.Revision {
		return nil
	}

	if err := loadCtx.Err(); err != nil {
		return fmt.Errorf("publish onlineconf snapshot: %w", err)
	}

	s.current.Store(next)

	return nil
}

func cloneSnapshot(source *oc.Snapshot) *oc.Snapshot {
	result := *source
	result.Scopes = slices.Clone(source.Scopes)
	result.Configs = slices.Clone(source.Configs)
	result.Flags = slices.Clone(source.Flags)

	for i := range result.Configs {
		result.Configs[i].Value = slices.Clone(source.Configs[i].Value)
	}

	for i := range result.Flags {
		result.Flags[i].ForceOnUserIDs = slices.Clone(source.Flags[i].ForceOnUserIDs)
		result.Flags[i].ForceOffUserIDs = slices.Clone(source.Flags[i].ForceOffUserIDs)
	}

	return &result
}

func validateSnapshot(snapshot *oc.Snapshot) error {
	if snapshot.Revision < 0 || snapshot.GeneratedAt.IsZero() {
		return oc.ErrInvalidSnapshot
	}

	scopes := make(map[string]struct{}, len(snapshot.Scopes))

	for _, scope := range snapshot.Scopes {
		if err := scope.Validate(); err != nil {
			return fmt.Errorf("%w: %w", oc.ErrInvalidSnapshot, err)
		}

		if _, exists := scopes[scope.Name]; exists {
			return fmt.Errorf("%w: duplicate scope", oc.ErrInvalidSnapshot)
		}

		scopes[scope.Name] = struct{}{}
	}

	configs := make(map[resourceIdentity]struct{}, len(snapshot.Configs))

	for _, config := range snapshot.Configs {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("%w: %w", oc.ErrInvalidSnapshot, err)
		}

		key := resourceIdentity{scope: config.Scope, key: config.Key}
		if _, exists := configs[key]; exists {
			return fmt.Errorf("%w: duplicate config", oc.ErrInvalidSnapshot)
		}

		configs[key] = struct{}{}
	}

	flags := make(map[resourceIdentity]struct{}, len(snapshot.Flags))
	flagIDs := make(map[int64]struct{}, len(snapshot.Flags))

	for _, flag := range snapshot.Flags {
		if err := flag.Validate(); err != nil {
			return fmt.Errorf("%w: %w", oc.ErrInvalidSnapshot, err)
		}

		if flag.ID < oc.MinUserID || flag.ID > oc.MaxSafeID {
			return fmt.Errorf("%w: invalid flag ID", oc.ErrInvalidSnapshot)
		}

		key := resourceIdentity{scope: flag.Scope, key: flag.Key}

		if _, exists := flags[key]; exists {
			return fmt.Errorf("%w: duplicate flag", oc.ErrInvalidSnapshot)
		}

		if _, exists := flagIDs[flag.ID]; exists {
			return fmt.Errorf("%w: duplicate flag ID", oc.ErrInvalidSnapshot)
		}

		flags[key] = struct{}{}
		flagIDs[flag.ID] = struct{}{}
	}

	return nil
}

type resourceIdentity struct {
	scope string
	key   string
}
