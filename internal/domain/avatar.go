package domain

import (
	"context"
)

type AvatarFile struct {
	Filename    string
	ContentType string
	Size        int64
}

type AvatarRepository interface {
	GetAvatar(ctx context.Context, userID UserID) (*AvatarFile, error)
	DeleteAvatar(ctx context.Context, userID UserID) error
	SaveAvatar(ctx context.Context, userID UserID, file AvatarFile) error
}
