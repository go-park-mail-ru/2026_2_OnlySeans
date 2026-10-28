package auth_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/pkg/redis"
	authservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/auth"
)

func newTestSessionStore(t *testing.T, ttl time.Duration) (*authservice.RedisSessionStore, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)

	client, err := redis.NewClientWithPing(t.Context(), fmt.Sprintf("redis://%s", mr.Addr()))
	require.NoError(t, err)
	t.Cleanup(client.Close)

	return authservice.NewRedisSessionStore(client, ttl), mr
}

func TestRedisSessionStore_CreateGetDelete(t *testing.T) {
	ctx := t.Context()
	store, _ := newTestSessionStore(t, time.Hour)

	s, err := store.Create(ctx, 42)
	require.NoError(t, err, "failed to create session")
	require.NotEmpty(t, s.ID, "session id must not be empty")
	assert.Equal(t, domain.UserID(42), s.UserID)
	assert.WithinDuration(t, time.Now().Add(time.Hour), s.ExpiresAt, 5*time.Second)

	got, err := store.Get(ctx, s.ID)
	require.NoError(t, err, "failed to get session")
	assert.Equal(t, domain.UserID(42), got.UserID)
	assert.Equal(t, s.ID, got.ID)

	require.NoError(t, store.Delete(ctx, s.ID), "failed to delete session")

	_, err = store.Get(ctx, s.ID)
	assert.ErrorIs(t, err, domain.ErrSessionNotFound)
}

func TestRedisSessionStore_Expired(t *testing.T) {
	ctx := t.Context()
	store, mr := newTestSessionStore(t, time.Hour)

	s, err := store.Create(ctx, 1)
	require.NoError(t, err, "failed to create session")

	mr.FastForward(2 * time.Hour)

	_, err = store.Get(ctx, s.ID)
	assert.ErrorIs(t, err, domain.ErrSessionNotFound)
}

func TestRedisSessionStore_Get_Unknown(t *testing.T) {
	ctx := t.Context()
	store, _ := newTestSessionStore(t, time.Hour)

	_, err := store.Get(ctx, "unknown")
	assert.ErrorIs(t, err, domain.ErrSessionNotFound)
}
