package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemorySessionStore_CreateGetDelete(t *testing.T) {
	store := NewInMemorySessionStore(time.Hour)

	s, err := store.Create(42)
	require.NoError(t, err, "failed to create session")

	got, err := store.Get(s.ID)
	require.NoError(t, err, "failed to get session")
	assert.Equal(t, UserID(42), got.UserID)

	require.NoError(t, store.Delete(s.ID), "failed to delete session")

	_, err = store.Get(s.ID)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestInMemorySessionStore_Expired(t *testing.T) {
	store := NewInMemorySessionStore(time.Hour)

	s, err := store.Create(1)
	require.NoError(t, err, "failed to create session")

	store.now = func() time.Time { return time.Now().Add(2 * time.Hour) }

	_, err = store.Get(s.ID)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestInMemorySessionStore_Get_Unknown(t *testing.T) {
	store := NewInMemorySessionStore(time.Hour)

	_, err := store.Get("unknown")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}
