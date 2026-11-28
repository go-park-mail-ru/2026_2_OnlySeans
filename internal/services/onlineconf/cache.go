package onlineconf

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

var ErrStaleSnapshot = errors.New("onlineconf snapshot revision decreased")

type SettingsProvider interface {
	GetConfig(scope, key string) (json.RawMessage, bool)
	EvaluateFlag(scope, key string, userID domain.UserID) oc.Evaluation
}

type CacheStatus struct {
	Revision    int64
	LastSuccess time.Time
	LastError   string
}

type Cache struct {
	source    SnapshotSource
	interval  time.Duration
	logger    logger.Logger
	current   atomic.Pointer[Snapshot]
	refreshMu sync.Mutex
	metaMu    sync.RWMutex
	meta      CacheStatus
}

func NewCache(source SnapshotSource, interval time.Duration, appLogger logger.Logger) (*Cache, error) {
	if source == nil {
		return nil, errors.New("onlineconf snapshot source is nil")
	}

	if interval <= 0 {
		return nil, errors.New("onlineconf refresh interval must be positive")
	}

	if appLogger == nil {
		return nil, errors.New("onlineconf logger is nil")
	}

	return &Cache{source: source, interval: interval, logger: appLogger}, nil
}

func (c *Cache) ReviewPolicy(userID domain.UserID) domain.ReviewPolicy {
	return c.current.Load().ReviewPolicy(userID)
}

func (c *Cache) PublicRuntime(userID domain.UserID) domain.PublicRuntime {
	return c.current.Load().PublicRuntime(userID)
}

func (c *Cache) SettingsSnapshot() *Snapshot {
	return c.current.Load()
}

func (c *Cache) GetConfig(scope, key string) (json.RawMessage, bool) {
	return c.SettingsSnapshot().GetConfig(scope, key)
}

func (c *Cache) EvaluateFlag(scope, key string, userID domain.UserID) oc.Evaluation {
	return c.SettingsSnapshot().EvaluateFlag(scope, key, userID)
}

func (c *Cache) Refresh(ctx context.Context) error {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()

	next, err := c.source.Fetch(ctx)
	if err != nil {
		c.recordError(err)
		return err
	}

	if next == nil {
		err := errors.New("onlineconf snapshot source returned nil")
		c.recordError(err)

		return err
	}

	current := c.current.Load()
	if current != nil && next.revision < current.revision {
		err := fmt.Errorf("%w: current=%d received=%d", ErrStaleSnapshot, current.revision, next.revision)
		c.recordError(err)

		return err
	}

	if current == nil || next.revision > current.revision {
		c.current.Store(next)
	}

	c.metaMu.Lock()
	c.meta = CacheStatus{Revision: next.revision, LastSuccess: time.Now()}
	c.metaMu.Unlock()

	return nil
}

func (c *Cache) Run(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.Refresh(ctx); err != nil && !errors.Is(err, context.Canceled) {
				c.logger.WarnContext(ctx, "onlineconf snapshot refresh failed", map[string]any{"error": err})
			}
		}
	}
}

func (c *Cache) Status() CacheStatus {
	c.metaMu.RLock()
	defer c.metaMu.RUnlock()

	return c.meta
}

func (c *Cache) recordError(err error) {
	c.metaMu.Lock()

	c.meta.LastError = err.Error()
	if current := c.current.Load(); current != nil {
		c.meta.Revision = current.revision
	}
	c.metaMu.Unlock()
}
