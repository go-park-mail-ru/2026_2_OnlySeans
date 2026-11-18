package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/domain"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user domain.User) (int64, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	GetUserByID(ctx context.Context, id int64) (domain.User, error)
	CheckUserByID(ctx context.Context, userID int64) (bool, error)
	UpdateProfile(ctx context.Context, userID int64, name, email *string) error
	UpdateAvatarURL(ctx context.Context, userID int64, newAvatarURL string) error
	UpdateUserRole(ctx context.Context, userID int64, newRole string) (string, bool, error)
}
