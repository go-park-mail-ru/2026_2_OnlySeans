package redisrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/google/uuid"
	"github.com/redis/rueidis"
)

const (
	sessionKeyPrefix = "session:"
)

type RedisSessionStore struct {
	appLogger logger.Logger
	client    rueidis.Client
}

func NewRedisSessionStore(
	client rueidis.Client,
	appLogger logger.Logger,
) *RedisSessionStore {
	return &RedisSessionStore{
		appLogger: appLogger,
		client:    client,
	}
}

func sessionKey(id uuid.UUID) string {
	return sessionKeyPrefix + id.String()
}

func (s *RedisSessionStore) Create(ctx context.Context, session domain.Session) (operationErr error) {
	started := time.Now()
	defer func() {
		s.logOperation(ctx, "session.create", started, operationErr)
	}()

	remaining := time.Until(session.ExpiresAt)
	if remaining <= 0 {
		return domain.ErrSessionNotFound
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}

	cmd := s.client.B().
		Set().
		Key(sessionKey(session.ID)).
		Value(string(data)).
		ExSeconds(int64(math.Ceil(remaining.Seconds()))).
		Build()

	res := s.client.Do(ctx, cmd)
	if res.Error() != nil {
		return fmt.Errorf("create session: %w", res.Error())
	}

	return nil
}

func (s *RedisSessionStore) Get(ctx context.Context, sessionID uuid.UUID) (_ *domain.Session, operationErr error) {
	started := time.Now()
	defer func() {
		s.logOperation(ctx, "session.get", started, operationErr)
	}()

	cmd := s.client.B().
		Get().
		Key(sessionKey(sessionID)).
		Build()

	rawSession, err := s.client.Do(ctx, cmd).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, domain.ErrSessionNotFound
		}

		return nil, fmt.Errorf("get session: %w", err)
	}

	var session domain.Session
	if err := json.Unmarshal([]byte(rawSession), &session); err != nil {
		return nil, fmt.Errorf("decode session: %w", err)
	}

	if !session.ExpiresAt.After(time.Now()) {
		return nil, domain.ErrSessionNotFound
	}

	return &session, nil
}

func (s *RedisSessionStore) Delete(ctx context.Context, sessionID uuid.UUID) (operationErr error) {
	started := time.Now()
	defer func() {
		s.logOperation(ctx, "session.delete", started, operationErr)
	}()

	cmd := s.client.B().
		Del().
		Key(sessionKey(sessionID)).
		Build()

	err := s.client.Do(ctx, cmd).Error()
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

func (s *RedisSessionStore) logOperation(ctx context.Context, operation string, started time.Time, err error) {
	if s.appLogger == nil {
		return
	}

	fields := map[string]any{
		"backend":     "redis",
		"operation":   operation,
		"duration_ms": time.Since(started).Milliseconds(),
	}

	if errors.Is(err, domain.ErrSessionNotFound) {
		fields["outcome"] = "not_found"
		s.appLogger.InfoContext(ctx, "session operation completed", fields)
		return
	}

	if err != nil {
		fields["outcome"] = "error"
		fields["error"] = err.Error()
		s.appLogger.ErrorContext(ctx, "session operation failed", fields)
		return
	}

	fields["outcome"] = "ok"
	s.appLogger.InfoContext(ctx, "session operation completed", fields)
}
