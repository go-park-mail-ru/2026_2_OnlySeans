package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/domain"
	"github.com/google/uuid"
)

type SessionRepository interface {
	Create(ctx context.Context, session domain.Session) error
	Get(ctx context.Context, sessionID uuid.UUID) (*domain.Session, error)
	Delete(ctx context.Context, sessionID uuid.UUID) error
}
