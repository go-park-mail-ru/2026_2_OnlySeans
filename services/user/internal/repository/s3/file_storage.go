package s3

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/domain"
	shareds3 "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/s3"
)

type FileStorage struct {
	storage shareds3.FileStorage
}

func NewFileStorage(storage shareds3.FileStorage) *FileStorage {
	return &FileStorage{
		storage: storage,
	}
}

func (s *FileStorage) UploadFile(
	ctx context.Context,
	key, contentType string,
	data []byte,
) error {
	return s.storage.UploadFile(ctx, key, contentType, data)
}

func (s *FileStorage) GetURL(ctx context.Context, key string) (string, error) {
	url, err := s.storage.GetURL(ctx, key)
	if errors.Is(err, shareds3.ErrFileNotFound) {
		return "", fmt.Errorf("get file URL: %w: %w", domain.ErrFileNotFound, err)
	}

	return url, err
}

func (s *FileStorage) DeleteFile(ctx context.Context, key string) error {
	return s.storage.DeleteFile(ctx, key)
}
