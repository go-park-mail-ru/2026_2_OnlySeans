package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/domain"
)

type ProfileReader interface {
	GetProfile(ctx context.Context, userID int64) (*domain.Profile, error)
}
