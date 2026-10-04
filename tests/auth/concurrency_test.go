package auth_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth"
)

const workers = 100

func runConcurrently(work func(i int)) {
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			work(i)
		}()
	}

	close(start)
	wg.Wait()
}

func TestInMemoryUserRepo_ConcurrentCreate_SameEmail(t *testing.T) {
	repo := auth.NewInMemoryUserRepo()
	ctx := context.Background()
	errs := make([]error, workers)

	runConcurrently(func(i int) {
		_, errs[i] = repo.Create(ctx, "race@example.com", fmt.Sprintf("user%d", i), "hash")
	})

	created := 0
	for _, err := range errs {
		if err == nil {
			created++
			continue
		}
		assert.ErrorIs(t, err, auth.ErrUserExists)
	}
	assert.Equal(t, 1, created)
}

func TestInMemoryUserRepo_ConcurrentCreate_UniqueIDs(t *testing.T) {
	repo := auth.NewInMemoryUserRepo()
	ctx := context.Background()
	ids := make([]auth.UserID, workers)
	errs := make([]error, workers)

	runConcurrently(func(i int) {
		ids[i], errs[i] = repo.Create(ctx, fmt.Sprintf("user%d@example.com", i), "user", "hash")
	})

	seen := make(map[auth.UserID]struct{}, workers)
	for i, id := range ids {
		require.NoError(t, errs[i])
		seen[id] = struct{}{}

		user, err := repo.GetUser(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("user%d@example.com", i), user.Email)
	}
	assert.Len(t, seen, workers)
}

func TestInMemorySessionStore_ConcurrentCreate(t *testing.T) {
	store := auth.NewInMemorySessionStore(time.Hour)
	sessions := make([]*auth.Session, workers)
	errs := make([]error, workers)

	runConcurrently(func(i int) {
		sessions[i], errs[i] = store.Create(auth.UserID(i))
	})

	seen := make(map[string]struct{}, workers)
	for i, session := range sessions {
		require.NoError(t, errs[i])
		seen[session.ID] = struct{}{}

		got, err := store.Get(session.ID)
		require.NoError(t, err)
		assert.Equal(t, auth.UserID(i), got.UserID)
	}
	assert.Len(t, seen, workers)
}
