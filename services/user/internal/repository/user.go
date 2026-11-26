package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/domain"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user domain.User) (int64, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	GetUserByID(ctx context.Context, id int64) (domain.User, error)
	UpdateProfile(ctx context.Context, userID int64, name, email *string) (domain.User, error)
}
