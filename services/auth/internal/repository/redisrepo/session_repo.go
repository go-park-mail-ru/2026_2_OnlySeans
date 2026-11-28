package redisrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/redis/rueidis"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/sessionpolicy"
)

const (
	sessionKeyPrefix     = "session:"
	refreshMaxAttempts   = 8
	transactionExecIndex = 2
)

const (
	sessionCreateOperation    = "session.create"
	sessionGetOperation       = "session.get"
	sessionDeleteOperation    = "session.delete"
	sessionRefreshOperation   = "session.refresh"
	encodeSessionErrorFormat  = "encode session: %w"
	createSessionErrorFormat  = "create session: %w"
	getSessionErrorFormat     = "get session: %w"
	decodeSessionErrorFormat  = "decode session: %w"
	deleteSessionErrorFormat  = "delete session: %w"
	refreshSessionErrorFormat = "refresh session: %w"
	sessionBackendField       = "backend"
	sessionBackendName        = "redis"
	sessionOperationField     = "operation"
	sessionDurationField      = "duration_ms"
	sessionOutcomeField       = "outcome"
	sessionErrorField         = "error"
	sessionOutcomeNotFound    = "not_found"
	sessionOutcomeError       = "error"
	sessionOutcomeOK          = "ok"
)

type sessionDTO struct {
	ID              uuid.UUID     `json:"id"`
	UserID          domain.UserID `json:"user_id"`
	Role            string        `json:"role"`
	ExpiresAt       time.Time     `json:"expires_at"`
	LastRefreshedAt time.Time     `json:"last_refreshed_at"`
}

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
		s.logOperation(ctx, sessionCreateOperation, started, operationErr)
	}()

	remaining := time.Until(session.ExpiresAt)
	if remaining <= 0 {
		return domain.ErrSessionNotFound
	}

	data, err := json.Marshal(sessionDTO{
		ID:              session.ID,
		UserID:          session.UserID,
		Role:            session.Role,
		ExpiresAt:       session.ExpiresAt,
		LastRefreshedAt: session.LastRefreshedAt,
	})
	if err != nil {
		return fmt.Errorf(encodeSessionErrorFormat, err)
	}

	cmd := s.client.B().
		Set().
		Key(sessionKey(session.ID)).
		Value(string(data)).
		ExSeconds(int64(math.Ceil(remaining.Seconds()))).
		Build()

	res := s.client.Do(ctx, cmd)
	if res.Error() != nil {
		return fmt.Errorf(createSessionErrorFormat, res.Error())
	}

	return nil
}

func (s *RedisSessionStore) Get(ctx context.Context, sessionID uuid.UUID) (_ *domain.Session, operationErr error) {
	started := time.Now()
	defer func() {
		s.logOperation(ctx, sessionGetOperation, started, operationErr)
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

		return nil, fmt.Errorf(getSessionErrorFormat, err)
	}

	var session sessionDTO
	if err := json.Unmarshal([]byte(rawSession), &session); err != nil {
		return nil, fmt.Errorf(decodeSessionErrorFormat, err)
	}

	if !session.ExpiresAt.After(time.Now()) {
		return nil, domain.ErrSessionNotFound
	}

	return &domain.Session{
		ID:              session.ID,
		UserID:          session.UserID,
		Role:            session.Role,
		ExpiresAt:       session.ExpiresAt,
		LastRefreshedAt: session.LastRefreshedAt,
	}, nil
}

func (s *RedisSessionStore) Delete(ctx context.Context, sessionID uuid.UUID) (operationErr error) {
	started := time.Now()
	defer func() {
		s.logOperation(ctx, sessionDeleteOperation, started, operationErr)
	}()

	cmd := s.client.B().
		Del().
		Key(sessionKey(sessionID)).
		Build()

	err := s.client.Do(ctx, cmd).Error()
	if err != nil {
		return fmt.Errorf(deleteSessionErrorFormat, err)
	}

	return nil
}

func (s *RedisSessionStore) logOperation(ctx context.Context, operation string, started time.Time, err error) {
	if s.appLogger == nil {
		return
	}

	fields := map[string]any{
		sessionBackendField:   sessionBackendName,
		sessionOperationField: operation,
		sessionDurationField:  time.Since(started).Milliseconds(),
	}

	if errors.Is(err, domain.ErrSessionNotFound) {
		fields[sessionOutcomeField] = sessionOutcomeNotFound
		s.appLogger.InfoContext(ctx, "session operation completed", fields)

		return
	}

	if err != nil {
		fields[sessionOutcomeField] = sessionOutcomeError
		fields[sessionErrorField] = err.Error()
		s.appLogger.ErrorContext(ctx, "session operation failed", fields)

		return
	}

	fields[sessionOutcomeField] = sessionOutcomeOK
	s.appLogger.InfoContext(ctx, "session operation completed", fields)
}

func (s *RedisSessionStore) Refresh(ctx context.Context, sessionID uuid.UUID, expiresAt time.Time) (_ *domain.Session, operationErr error) {
	started := time.Now()
	defer func() { s.logOperation(ctx, sessionRefreshOperation, started, operationErr) }()

	key := sessionKey(sessionID)
	var result *domain.Session

	err := s.client.Dedicated(func(client rueidis.DedicatedClient) error {
		defer func() { _ = client.Do(ctx, client.B().Unwatch().Build()).Error() }()

		for range refreshMaxAttempts {
			if err := client.Do(ctx, client.B().Watch().Key(key).Build()).Error(); err != nil {
				return err
			}

			raw, err := client.Do(ctx, client.B().Get().Key(key).Build()).ToString()
			if rueidis.IsRedisNil(err) {
				return domain.ErrSessionNotFound
			}

			if err != nil {
				return err
			}

			var current sessionDTO
			if err = json.Unmarshal([]byte(raw), &current); err != nil {
				return err
			}

			now := time.Now()
			if !current.ExpiresAt.After(now) {
				return domain.ErrSessionNotFound
			}

			if sessionpolicy.NeedsRefresh(current.LastRefreshedAt, now) {
				if !expiresAt.After(now) {
					return domain.ErrSessionNotFound
				}

				current.ExpiresAt = expiresAt.UTC().Truncate(time.Millisecond)
				current.LastRefreshedAt = now.UTC().Truncate(time.Millisecond)

				data, encodeErr := json.Marshal(current)
				if encodeErr != nil {
					return encodeErr
				}

				replies := client.DoMulti(ctx,
					client.B().Multi().Build(),
					client.B().Set().Key(key).Value(string(data)).Xx().PxatMillisecondsTimestamp(current.ExpiresAt.UnixMilli()).Build(),
					client.B().Exec().Build(),
				)
				for i, reply := range replies {
					if err = reply.Error(); err != nil {
						if i == transactionExecIndex && rueidis.IsRedisNil(err) {
							break
						}

						return err
					}
				}

				if rueidis.IsRedisNil(err) {
					continue
				}
			}

			result = &domain.Session{
				ID:              current.ID,
				UserID:          current.UserID,
				Role:            current.Role,
				ExpiresAt:       current.ExpiresAt,
				LastRefreshedAt: current.LastRefreshedAt,
			}

			return nil
		}

		return errors.New("session refresh transaction retry limit exceeded")
	})
	if err != nil {
		return nil, fmt.Errorf(refreshSessionErrorFormat, err)
	}

	return result, nil
}
