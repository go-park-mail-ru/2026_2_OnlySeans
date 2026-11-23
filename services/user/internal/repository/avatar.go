package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/domain"
)

type AvatarRepository interface {
	GetAvatar(ctx context.Context, userID int64) (*domain.AvatarFile, error)
	DeleteAvatar(ctx context.Context, userID int64) error
	SaveAvatar(ctx context.Context, userID int64, file domain.AvatarFile) error
}
