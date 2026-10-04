package auth_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth"
)

func TestInMemorySessionStore_CreateGetDelete(t *testing.T) {
	ctx := t.Context()
	store := auth.NewInMemorySessionStore(time.Hour)

	s, err := store.Create(ctx, 42)
	require.NoError(t, err, "failed to create session")

	got, err := store.Get(ctx, s.ID)
	require.NoError(t, err, "failed to get session")
	assert.Equal(t, auth.UserID(42), got.UserID)

	require.NoError(t, store.Delete(ctx, s.ID), "failed to delete session")

	_, err = store.Get(ctx, s.ID)
	assert.ErrorIs(t, err, auth.ErrSessionNotFound)
}

func TestInMemorySessionStore_Expired(t *testing.T) {
	ctx := t.Context()
	store := auth.NewInMemorySessionStore(-time.Hour)

	s, err := store.Create(ctx, 1)
	require.NoError(t, err, "failed to create session")

	_, err = store.Get(ctx, s.ID)
	assert.ErrorIs(t, err, auth.ErrSessionNotFound)
}

func TestInMemorySessionStore_Get_Unknown(t *testing.T) {
	ctx := t.Context()
	store := auth.NewInMemorySessionStore(time.Hour)

	_, err := store.Get(ctx, "unknown")
	assert.ErrorIs(t, err, auth.ErrSessionNotFound)
}
