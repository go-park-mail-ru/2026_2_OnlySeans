package usecase

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, email, username, passwordHash string) (domain.UserID, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUser(ctx context.Context, id domain.UserID) (*domain.User, error)
}
