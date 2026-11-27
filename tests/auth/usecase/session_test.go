package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/usecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/repository"
)

type sessionRepositoryStub struct {
	repository.SessionRepository
	create func(context.Context, domain.Session) error
	get    func(context.Context, uuid.UUID) (*domain.Session, error)
	delete func(context.Context, uuid.UUID) error
}

func (s sessionRepositoryStub) Create(ctx context.Context, session domain.Session) error {
	return s.create(ctx, session)
}
func (s sessionRepositoryStub) Get(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	return s.get(ctx, id)
}
func (s sessionRepositoryStub) Delete(ctx context.Context, id uuid.UUID) error {
	return s.delete(ctx, id)
}

func TestSessionCreate(t *testing.T) {
	failure := errors.New("redis unavailable")
	for _, tt := range []struct {
		name    string
		repoErr error
	}{
		{
			name: "new session",
		},
		{
			name:    "storage failure",
			repoErr: failure,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var saved domain.Session
			repo := sessionRepositoryStub{
				create: func(_ context.Context, session domain.Session) error {
					saved = session
					return tt.repoErr
				},
			}
			uc := usecase.NewSessionUseCase(repo, time.Hour)
			before := time.Now()
			session, err := uc.Create(t.Context(), 7, "reader")
			require.ErrorIs(t, err, tt.repoErr)
			require.NotEqual(t, uuid.Nil, saved.ID)
			require.Equal(t, domain.UserID(7), saved.UserID)
			require.Equal(t, "reader", saved.Role)
			require.WithinDuration(t, before.Add(time.Hour), saved.ExpiresAt, time.Second)

			if tt.repoErr == nil {
				require.Equal(t, saved, session)
			} else {
				require.Empty(t, session)
			}
		})
	}
}

func TestSessionCheck(t *testing.T) {
	failure := errors.New("redis unavailable")
	valid := domain.Session{
		ID:        uuid.New(),
		UserID:    7,
		Role:      "reader",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	expired := valid
	expired.ExpiresAt = time.Now().Add(-time.Second)

	tests := []struct {
		name    string
		session *domain.Session
		repoErr error
		wantErr error
	}{
		{
			name:    "active session",
			session: &valid,
		},
		{
			name:    "missing session",
			wantErr: domain.ErrSessionNotFound,
		},
		{
			name:    "expired session",
			session: &expired,
			wantErr: domain.ErrSessionExpired,
		},
		{
			name:    "storage failure",
			repoErr: failure,
			wantErr: failure,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := sessionRepositoryStub{
				get: func(_ context.Context, id uuid.UUID) (*domain.Session, error) {
					require.Equal(t, valid.ID, id)
					return tt.session, tt.repoErr
				},
			}
			uc := usecase.NewSessionUseCase(repo, time.Hour)
			session, err := uc.Check(t.Context(), valid.ID)
			require.ErrorIs(t, err, tt.wantErr)

			if tt.wantErr == nil {
				require.Equal(t, valid, session)
			} else {
				require.Empty(t, session)
			}
		})
	}
}

func TestSessionDelete(t *testing.T) {
	failure := errors.New("redis unavailable")
	for _, tt := range []struct {
		name    string
		repoErr error
	}{
		{
			name: "deleted",
		},
		{
			name:    "storage failure",
			repoErr: failure,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			id := uuid.New()
			calls := 0
			repo := sessionRepositoryStub{
				delete: func(_ context.Context, sessionID uuid.UUID) error {
					calls++
					require.Equal(t, id, sessionID)
					return tt.repoErr
				},
			}
			uc := usecase.NewSessionUseCase(repo, time.Hour)
			require.ErrorIs(t, uc.Delete(t.Context(), id), tt.repoErr)
			require.Equal(t, 1, calls)
		})
	}
}
