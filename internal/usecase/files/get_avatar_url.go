package files

import (
	"context"
	"errors"
	"fmt"
	"regexp"
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
}

func NewGetAvatarURLUseCase(storage AvatarURLStorage) (*GetAvatarURLUseCase, error) {
	if storage == nil {
		return nil, errors.New("avatar URL storage is nil")
	}
	return &GetAvatarURLUseCase{storage: storage}, nil
}

func (uc *GetAvatarURLUseCase) Run(ctx context.Context, filename string) (*GetAvatarURLOutput, error) {
	if !avatarFilenamePattern.MatchString(filename) {
		return nil, ErrInvalidAvatarFilename
	}
	url, err := uc.storage.GetURL(ctx, filename)
	if err != nil {
		return nil, fmt.Errorf("get avatar URL: %w", err)
	}
	return &GetAvatarURLOutput{URL: url}, nil
}
