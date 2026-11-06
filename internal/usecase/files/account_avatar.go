package files

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type AccountAvatarRepository interface {
	GetAvatar(ctx context.Context, userID domain.UserID) (*domain.AvatarFile, error)
	DeleteAvatar(ctx context.Context, userID domain.UserID) error
}

type AccountAvatarUseCase struct {
	avatars AccountAvatarRepository
	storage AvatarURLStorage
}

func NewAccountAvatarUseCase(avatars AccountAvatarRepository, storage AvatarURLStorage) (*AccountAvatarUseCase, error) {
	if avatars == nil {
		return nil, errors.New("account avatar repository is nil")
	}
	if storage == nil {
		return nil, errors.New("avatar URL storage is nil")
	}
	return &AccountAvatarUseCase{avatars: avatars, storage: storage}, nil
}

func (uc *AccountAvatarUseCase) GetURL(ctx context.Context, userID domain.UserID) (string, error) {
	if userID <= 0 {
		return "", domain.ErrUnauthorized
	}
	avatar, err := uc.avatars.GetAvatar(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get account avatar: %w", err)
	}
	if avatar == nil {
		return "", domain.ErrFileNotFound
	}
	url, err := uc.storage.GetURL(ctx, avatar.Filename)
	if err != nil {
		return "", fmt.Errorf("get account avatar URL: %w", err)
	}
	return url, nil
}

func (uc *AccountAvatarUseCase) Delete(ctx context.Context, userID domain.UserID) error {
	if userID <= 0 {
		return domain.ErrUnauthorized
	}
	if err := uc.avatars.DeleteAvatar(ctx, userID); err != nil {
		return fmt.Errorf("delete account avatar: %w", err)
	}
	return nil
}
