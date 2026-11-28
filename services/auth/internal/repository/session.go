package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/domain"
)

type SessionRepository interface {
	Create(ctx context.Context, session domain.Session) error
	Get(ctx context.Context, sessionID uuid.UUID) (*domain.Session, error)
	Delete(ctx context.Context, sessionID uuid.UUID) error
	Refresh(ctx context.Context, sessionID uuid.UUID, expiresAt time.Time) (*domain.Session, error)
}
