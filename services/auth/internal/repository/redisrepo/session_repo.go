package redisrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/rueidis"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/sessionpolicy"
)

const (
	sessionKeyPrefix     = "session:"
	refreshNumberBase    = 10
	refreshSessionScript = `
local session_key_index = 1
local expected_session_arg_index = 1
local updated_session_arg_index = 2
local previous_expiry_arg_index = 3
local next_expiry_arg_index = 4
local seconds_part_index = 1
local microseconds_part_index = 2
local current = redis.call('GET', KEYS[session_key_index])
if not current or redis.call('PTTL', KEYS[session_key_index]) <= 0 then
    return false
end
if current ~= ARGV[expected_session_arg_index] then
    return current
end
local milliseconds_per_second = 1000
local current_time = redis.call('TIME')
local now = current_time[seconds_part_index] * milliseconds_per_second + math.floor(current_time[microseconds_part_index] / milliseconds_per_second)
if tonumber(ARGV[previous_expiry_arg_index]) <= now then
    return false
end
local remaining = tonumber(ARGV[next_expiry_arg_index]) - now
if remaining <= 0 then
    return false
end
redis.call('SET', KEYS[session_key_index], ARGV[updated_session_arg_index], 'PX', remaining, 'XX')
return ARGV[updated_session_arg_index]
`
)

const (
	sessionCreateOperation             = "session.create"
	sessionGetOperation                = "session.get"
	sessionDeleteOperation             = "session.delete"
	sessionRefreshOperation            = "session.refresh"
	encodeSessionErrorFormat           = "encode session: %w"
	createSessionErrorFormat           = "create session: %w"
	getSessionErrorFormat              = "get session: %w"
	decodeSessionErrorFormat           = "decode session: %w"
	deleteSessionErrorFormat           = "delete session: %w"
	getSessionForRefreshErrorFormat    = "get session for refresh: %w"
	decodeSessionForRefreshErrorFormat = "decode session for refresh: %w"
	encodeRefreshedSessionErrorFormat  = "encode refreshed session: %w"
	refreshSessionErrorFormat          = "refresh session: %w"
	decodeRefreshedSessionErrorFormat  = "decode refreshed session: %w"
	sessionBackendField                = "backend"
	sessionBackendName                 = "redis"
	sessionOperationField              = "operation"
	sessionDurationField               = "duration_ms"
	sessionOutcomeField                = "outcome"
	sessionErrorField                  = "error"
	sessionOutcomeNotFound             = "not_found"
	sessionOutcomeError                = "error"
	sessionOutcomeOK                   = "ok"
)

var refreshSessionLua = rueidis.NewLuaScript(refreshSessionScript)

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

	raw, err := s.client.Do(ctx, s.client.B().Get().Key(key).Build()).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, domain.ErrSessionNotFound
		}

		return nil, fmt.Errorf(getSessionForRefreshErrorFormat, err)
	}

	var current sessionDTO
	if err = json.Unmarshal([]byte(raw), &current); err != nil {
		return nil, fmt.Errorf(decodeSessionForRefreshErrorFormat, err)
	}

	if !current.ExpiresAt.After(time.Now()) {
		return nil, domain.ErrSessionNotFound
	}

	if !sessionpolicy.NeedsRefresh(current.LastRefreshedAt, started) {
		return &domain.Session{
			ID:              current.ID,
			UserID:          current.UserID,
			Role:            current.Role,
			ExpiresAt:       current.ExpiresAt,
			LastRefreshedAt: current.LastRefreshedAt,
		}, nil
	}

	oldExpiresAt := current.ExpiresAt
	current.ExpiresAt = expiresAt.UTC().Truncate(time.Millisecond)
	current.LastRefreshedAt = started.UTC().Truncate(time.Millisecond)

	data, err := json.Marshal(current)
	if err != nil {
		return nil, fmt.Errorf(encodeRefreshedSessionErrorFormat, err)
	}

	result, err := refreshSessionLua.Exec(ctx, s.client, []string{key}, []string{
		raw,
		string(data),
		strconv.FormatInt(oldExpiresAt.UnixMilli(), refreshNumberBase),
		strconv.FormatInt(current.ExpiresAt.UnixMilli(), refreshNumberBase),
	}).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, domain.ErrSessionNotFound
		}

		return nil, fmt.Errorf(refreshSessionErrorFormat, err)
	}

	if err = json.Unmarshal([]byte(result), &current); err != nil {
		return nil, fmt.Errorf(decodeRefreshedSessionErrorFormat, err)
	}

	if !current.ExpiresAt.After(time.Now()) {
		return nil, domain.ErrSessionNotFound
	}

	return &domain.Session{
		ID:              current.ID,
		UserID:          current.UserID,
		Role:            current.Role,
		ExpiresAt:       current.ExpiresAt,
		LastRefreshedAt: current.LastRefreshedAt,
	}, nil
}
