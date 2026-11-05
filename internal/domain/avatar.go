package domain

import (
	"context"
	"errors"
)

type AvatarFile struct {
	Filename    string
	ContentType string
	Size        int64
}

type AvatarRepository interface {
	SaveAvatar(ctx context.Context, userID UserID, file AvatarFile) error
}

// A failed commit can have an unknown outcome. Keep its S3 object until reconciled.
var ErrAvatarCommitUnknown = errors.New("avatar transaction commit outcome is unknown")
