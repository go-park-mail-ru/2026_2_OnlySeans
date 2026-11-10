package auth

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/redis/rueidis"
)

const (
	sessionKeyPrefix = "session:"
)

type RedisSessionStore struct {
	log    logger.Logger
	client rueidis.Client
	ttl    time.Duration
}

func NewRedisSessionStore(
	client rueidis.Client,
	ttl time.Duration,
	appLogger logger.Logger,
) *RedisSessionStore {
	return &RedisSessionStore{log: appLogger,
		client: client,
		ttl:    ttl,
	}
}

func sessionKey(id string) string {
	return sessionKeyPrefix + id
}

func (s *RedisSessionStore) Create(ctx context.Context, userID domain.UserID) (_ *domain.Session, operationErr error) {
	operationLog := storage.StartOperation(ctx, s.log, "redis", "session.create", map[string]any{"user_id": userID})
	defer func() { operationLog.Finish(operationErr) }()

	id, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("generate session id: %w", err)
	}

	cmd := s.client.B().
		Set().
		Key(sessionKey(id)).
		Value(strconv.Itoa(int(userID))).
		ExSeconds(int64(s.ttl.Seconds())).
		Build()

	operationLog.Stage = "command"
	res := s.client.Do(ctx, cmd)
	if res.Error() != nil {
		return nil, fmt.Errorf("create session: %w", res.Error())
	}

	session := &domain.Session{
		ID:        id,
		UserID:    userID,
		ExpiresAt: time.Now().Add(s.ttl),
	}

	return session, nil
}

func (s *RedisSessionStore) Get(ctx context.Context, sessionID string) (_ *domain.Session, operationErr error) {
	operationLog := storage.StartOperation(ctx, s.log, "redis", "session.get", map[string]any{})
	defer func() { operationLog.Finish(operationErr) }()

	cmd := s.client.B().
		Get().
		Key(sessionKey(sessionID)).
		Build()

	operationLog.Stage = "command"
	rawUserID, err := s.client.Do(ctx, cmd).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, domain.ErrSessionNotFound
		}

		return nil, fmt.Errorf("get session: %w", err)
	}

	userID, err := strconv.Atoi(rawUserID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}

	session := &domain.Session{
		ID:     sessionID,
		UserID: domain.UserID(userID),
	}

	return session, nil
}

func (s *RedisSessionStore) Delete(ctx context.Context, sessionID string) (operationErr error) {
	operationLog := storage.StartOperation(ctx, s.log, "redis", "session.delete", map[string]any{})
	defer func() { operationLog.Finish(operationErr) }()

	cmd := s.client.B().
		Del().
		Key(sessionKey(sessionID)).
		Build()

	operationLog.Stage = "command"
	err := s.client.Do(ctx, cmd).Error()
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}
