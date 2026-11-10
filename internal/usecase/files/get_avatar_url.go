package files

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

var ErrInvalidAvatarFilename = errors.New("invalid avatar filename")

var avatarFilenamePattern = regexp.MustCompile(`^[0-9a-f]{32}\.(jpg|png)$`)

type AvatarURLStorage interface {
	GetURL(ctx context.Context, key string) (string, error)
}

type GetAvatarURLOutput struct {
	URL string
}

type GetAvatarURLUseCase struct {
	storage AvatarURLStorage
	logger  logger.Logger
}

func NewGetAvatarURLUseCase(storage AvatarURLStorage, appLogger logger.Logger) (*GetAvatarURLUseCase, error) {
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}
	if storage == nil {
		return nil, errors.New("avatar URL storage is nil")
	}
	return &GetAvatarURLUseCase{storage: storage, logger: appLogger}, nil
}

func (uc *GetAvatarURLUseCase) Run(ctx context.Context, filename string) (*GetAvatarURLOutput, error) {
	if !avatarFilenamePattern.MatchString(filename) {
		uc.logger.WarnContext(ctx, "avatar URL request rejected: invalid filename", map[string]any{"filename_length": len(filename)})
		return nil, ErrInvalidAvatarFilename
	}
	uc.logger.DebugContext(ctx, "get avatar URL started", map[string]any{"filename": filename})
	url, err := uc.storage.GetURL(ctx, filename)
	if err != nil {
		if errors.Is(err, domain.ErrFileNotFound) {
			uc.logger.DebugContext(ctx, "avatar object not found", map[string]any{"filename": filename})
		} else {
			uc.logger.ErrorContext(ctx, "get avatar URL failed", map[string]any{"filename": filename, "error": err})
		}
		return nil, fmt.Errorf("get avatar URL: %w", err)
	}
	uc.logger.DebugContext(ctx, "avatar URL generated", map[string]any{"filename": filename})
	return &GetAvatarURLOutput{URL: url}, nil
}
