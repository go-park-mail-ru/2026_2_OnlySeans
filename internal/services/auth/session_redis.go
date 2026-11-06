package auth

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/redis/rueidis"
)

const (
	sessionKeyPrefix = "session:"
)

type RedisSessionStore struct {
	client    rueidis.Client
	ttl       time.Duration
	appLogger logger.Logger
}

func NewRedisSessionStore(
	client rueidis.Client,
	ttl time.Duration,
	appLogger logger.Logger,
) *RedisSessionStore {
	return &RedisSessionStore{
		client:    client,
		ttl:       ttl,
		appLogger: appLogger,
	}
}

func sessionKey(id string) string {
	return sessionKeyPrefix + id
}

func (s *RedisSessionStore) Create(ctx context.Context, userID domain.UserID) (*domain.Session, error) {
	id, err := generateSessionID()
	if err != nil {
		s.appLogger.Error("generate session id failed", map[string]any{
			"user_id": userID,
			"error":   err,
		})
		return nil, fmt.Errorf("generate session id: %w", err)
	}

	cmd := s.client.B().
		Set().
		Key(sessionKey(id)).
		Value(strconv.Itoa(int(userID))).
		ExSeconds(int64(s.ttl.Seconds())).
		Build()

	res := s.client.Do(ctx, cmd)
	if res.Error() != nil {
		s.appLogger.Error("redis create session failed", map[string]any{
			"user_id": userID,
			"error":   res.Error(),
		})
		return nil, fmt.Errorf("create session: %w", res.Error())
	}

	session := &domain.Session{
		ID:        id,
		UserID:    userID,
		ExpiresAt: time.Now().Add(s.ttl),
	}

	return session, nil
}

func (s *RedisSessionStore) Get(ctx context.Context, sessionID string) (*domain.Session, error) {
	cmd := s.client.B().
		Get().
		Key(sessionKey(sessionID)).
		Build()

	rawUserID, err := s.client.Do(ctx, cmd).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, domain.ErrSessionNotFound
		}

		s.appLogger.Error("redis get session failed", map[string]any{
			"error": err,
		})
		return nil, fmt.Errorf("get session: %w", err)
	}

	userID, err := strconv.Atoi(rawUserID)
	if err != nil {
		s.appLogger.Error("parse session user id failed", map[string]any{
			"error": err,
		})
		return nil, fmt.Errorf("parse user id: %w", err)
	}

	session := &domain.Session{
		ID:     sessionID,
		UserID: domain.UserID(userID),
	}

	return session, nil
}

func (s *RedisSessionStore) Delete(ctx context.Context, sessionID string) error {
	cmd := s.client.B().
		Del().
		Key(sessionKey(sessionID)).
		Build()

	err := s.client.Do(ctx, cmd).Error()
	if err != nil {
		s.appLogger.Error("redis delete session failed", map[string]any{
			"error": err,
		})
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}
