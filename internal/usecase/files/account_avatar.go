package files

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type AccountAvatarRepository interface {
	GetAvatar(ctx context.Context, userID domain.UserID) (*domain.AvatarFile, error)
	DeleteAvatar(ctx context.Context, userID domain.UserID) error
}

type AccountAvatarUseCase struct {
	avatars AccountAvatarRepository
	storage AvatarURLStorage
	logger  logger.Logger
}

func NewAccountAvatarUseCase(avatars AccountAvatarRepository, storage AvatarURLStorage, appLogger logger.Logger) (*AccountAvatarUseCase, error) {
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}
	if avatars == nil {
		return nil, errors.New("account avatar repository is nil")
	}
	if storage == nil {
		return nil, errors.New("avatar URL storage is nil")
	}
	return &AccountAvatarUseCase{avatars: avatars, storage: storage, logger: appLogger}, nil
}

func (uc *AccountAvatarUseCase) GetURL(ctx context.Context, userID domain.UserID) (string, error) {
	uc.logger.DebugContext(ctx, "get account avatar started", map[string]any{"user_id": userID})
	if userID <= 0 {
		uc.logger.WarnContext(ctx, "get account avatar rejected: invalid user", nil)
		return "", domain.ErrUnauthorized
	}
	avatar, err := uc.avatars.GetAvatar(ctx, userID)
	if err != nil {
		uc.logFailure(ctx, "get account avatar failed", userID, err)
		return "", fmt.Errorf("get account avatar: %w", err)
	}
	if avatar == nil {
		uc.logger.DebugContext(ctx, "account avatar not found", map[string]any{"user_id": userID})
		return "", domain.ErrFileNotFound
	}
	url, err := uc.storage.GetURL(ctx, avatar.Filename)
	if err != nil {
		uc.logFailure(ctx, "get account avatar URL failed", userID, err)
		return "", fmt.Errorf("get account avatar URL: %w", err)
	}
	uc.logger.DebugContext(ctx, "account avatar URL generated", map[string]any{"user_id": userID, "filename": avatar.Filename})
	return url, nil
}

func (uc *AccountAvatarUseCase) Delete(ctx context.Context, userID domain.UserID) error {
	uc.logger.DebugContext(ctx, "detach account avatar started", map[string]any{"user_id": userID})
	if userID <= 0 {
		uc.logger.WarnContext(ctx, "detach account avatar rejected: invalid user", nil)
		return domain.ErrUnauthorized
	}
	if err := uc.avatars.DeleteAvatar(ctx, userID); err != nil {
		uc.logFailure(ctx, "detach account avatar failed", userID, err)
		return fmt.Errorf("delete account avatar: %w", err)
	}
	uc.logger.InfoContext(ctx, "account avatar detached", map[string]any{"user_id": userID})
	return nil
}

func (uc *AccountAvatarUseCase) logFailure(ctx context.Context, message string, userID domain.UserID, err error) {
	fields := map[string]any{"user_id": userID, "error": err}
	switch {
	case errors.Is(err, domain.ErrFileNotFound):
		uc.logger.DebugContext(ctx, message, fields)
	case errors.Is(err, domain.ErrUserNotFound), errors.Is(err, domain.ErrUnauthorized):
		uc.logger.WarnContext(ctx, message, fields)
	default:
		uc.logger.ErrorContext(ctx, message, fields)
	}
}
