package redisrepo_test

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/repository/redisrepo"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/usecase"
)

const refreshTTL = 72 * time.Hour

func testRedis(t *testing.T) rueidis.Client {
	t.Helper()

	url := os.Getenv("AUTH_REDIS_TEST_URL")
	if url == "" {
		t.Skip("set AUTH_REDIS_TEST_URL for Redis integration")
	}

	options, err := rueidis.ParseURL(url)
	require.NoError(t, err)

	options.DisableCache = true
	client, err := rueidis.NewClient(options)
	require.NoError(t, err)
	t.Cleanup(client.Close)

	return client
}
func createSession(t *testing.T, client rueidis.Client) (domain.Session, string) {
	t.Helper()

	session := domain.Session{ID: uuid.New(), UserID: 7, Role: "admin", ExpiresAt: time.Now().Add(time.Hour)}
	store := redisrepo.NewRedisSessionStore(client, nil)
	require.NoError(t, store.Create(t.Context(), session))
	key := "session:" + session.ID.String()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		require.NoError(t, client.Do(ctx, client.B().Del().Key(key).Build()).Error())
	})

	return session, key
}
func TestRedisRefreshUpdatesExpiryAndTTL(t *testing.T) {
	client := testRedis(t)
	original, key := createSession(t, client)
	store := redisrepo.NewRedisSessionStore(client, nil)
	sessions := usecase.NewSessionUseCase(store, refreshTTL)
	before := time.Now()
	refreshed, err := sessions.Refresh(t.Context(), original.ID)
	require.NoError(t, err)
	require.Equal(t, original.ID, refreshed.ID)
	require.Equal(t, original.UserID, refreshed.UserID)
	require.Equal(t, original.Role, refreshed.Role)
	require.WithinDuration(t, before.Add(refreshTTL), refreshed.ExpiresAt, time.Second)
	stored, err := store.Get(t.Context(), original.ID)
	require.NoError(t, err)
	require.Equal(t, refreshed, *stored)
	ttl, err := client.Do(t.Context(), client.B().Pttl().Key(key).Build()).AsInt64()
	require.NoError(t, err)
	require.InDelta(t, time.Until(stored.ExpiresAt).Milliseconds(), ttl, float64(time.Second.Milliseconds()))
	require.NoError(t, sessions.Delete(t.Context(), original.ID))
	_, err = sessions.Refresh(t.Context(), original.ID)
	require.ErrorIs(t, err, domain.ErrSessionNotFound)
	exists, err := client.Do(t.Context(), client.B().Exists().Key(key).Build()).AsInt64()
	require.NoError(t, err)
	require.Zero(t, exists)
}
func TestRedisRefreshRejectsExpiredPayload(t *testing.T) {
	client := testRedis(t)
	original, key := createSession(t, client)
	raw, err := client.Do(t.Context(), client.B().Get().Key(key).Build()).ToString()
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &payload))
	payload["expires_at"] = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano)
	expired, err := json.Marshal(payload)
	require.NoError(t, err)
	require.NoError(t, client.Do(t.Context(), client.B().Set().Key(key).Value(string(expired)).Ex(time.Minute).Build()).Error())
	store := redisrepo.NewRedisSessionStore(client, nil)
	_, err = store.Refresh(t.Context(), original.ID, time.Now().Add(refreshTTL))
	require.ErrorIs(t, err, domain.ErrSessionNotFound)
	current, err := client.Do(t.Context(), client.B().Get().Key(key).Build()).ToString()
	require.NoError(t, err)
	require.Equal(t, string(expired), current)
}

type logoutAfterReadClient struct {
	rueidis.Client
	key         string
	once        sync.Once
	deleteError error
}

func (c *logoutAfterReadClient) Do(ctx context.Context, command rueidis.Completed) rueidis.RedisResult {
	parts := command.Commands()
	matches := len(parts) == 2 && parts[0] == "GET" && parts[1] == c.key
	result := c.Client.Do(ctx, command)

	if matches {
		c.once.Do(func() { c.deleteError = c.Client.Do(ctx, c.Client.B().Del().Key(c.key).Build()).Error() })
	}

	return result
}
func TestRedisRefreshDoesNotRestoreConcurrentLogout(t *testing.T) {
	client := testRedis(t)
	original, key := createSession(t, client)
	proxy := &logoutAfterReadClient{Client: client, key: key}
	store := redisrepo.NewRedisSessionStore(proxy, nil)
	_, err := store.Refresh(t.Context(), original.ID, time.Now().Add(refreshTTL))
	require.NoError(t, proxy.deleteError)
	require.ErrorIs(t, err, domain.ErrSessionNotFound)
	exists, err := client.Do(t.Context(), client.B().Exists().Key(key).Build()).AsInt64()
	require.NoError(t, err)
	require.Zero(t, exists)
}
func TestRedisConcurrentRefresh(t *testing.T) {
	const requests = 8
	client := testRedis(t)
	original, _ := createSession(t, client)
	store := redisrepo.NewRedisSessionStore(client, nil)
	sessions := usecase.NewSessionUseCase(store, refreshTTL)
	errors := make(chan error, requests)

	var wg sync.WaitGroup
	for range requests {
		wg.Go(func() { _, err := sessions.Refresh(t.Context(), original.ID); errors <- err })
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		require.NoError(t, err)
	}

	result, err := store.Get(t.Context(), original.ID)
	require.NoError(t, err)
	require.Equal(t, original.UserID, result.UserID)
	require.WithinDuration(t, time.Now().Add(refreshTTL), result.ExpiresAt, time.Second)
}
