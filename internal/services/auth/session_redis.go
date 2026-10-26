package auth

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/redis/rueidis"
)

const (
	sessionKeyPrefix = "session:"
)

type RedisSessionStore struct {
	client rueidis.Client
	ttl    time.Duration
}

func NewRedisSessionStore(client rueidis.Client, ttl time.Duration) *RedisSessionStore {
	return &RedisSessionStore{
		client: client,
		ttl:    ttl,
	}
}

func sessionKey(id string) string {
	return sessionKeyPrefix + id
}

func (s *RedisSessionStore) Create(ctx context.Context, userID domain.UserID) (*domain.Session, error) {
	id, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("generate session id: %w", err)
	}

	cmd := s.client.B().Set().Key(sessionKey(id)).Value(strconv.Itoa(int(userID))).ExSeconds(int64(s.ttl.Seconds())).Build()

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

func (s *RedisSessionStore) Get(ctx context.Context, sessionID string) (*domain.Session, error) {
	cmd := s.client.B().Get().Key(sessionKey(sessionID)).Build()

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

func (s *RedisSessionStore) Delete(ctx context.Context, sessionID string) error {
	cmd := s.client.B().Del().Key(sessionKey(sessionID)).Build()
	err := s.client.Do(ctx, cmd).Error()
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
