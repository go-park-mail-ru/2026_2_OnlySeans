package usecase

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type SessionStore interface {
	Create(ctx context.Context, userID domain.UserID) (*domain.Session, error)
	Get(ctx context.Context, sessionID string) (*domain.Session, error)
	Delete(ctx context.Context, sessionID string) error
}
