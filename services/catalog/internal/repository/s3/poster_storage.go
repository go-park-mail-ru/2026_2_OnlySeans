package s3

import (
	"context"
	"errors"
	"io"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	shareds3 "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/s3"
)

type objectStorage interface {
	Upload(ctx context.Context, key, contentType string, body io.ReadSeeker) error
	GetURL(ctx context.Context, key string) (string, error)
	DeleteFile(ctx context.Context, key string) error
}

type PosterStorage struct {
	storage objectStorage
}

func NewPosterStorage(storage objectStorage) (*PosterStorage, error) {
	if storage == nil {
		return nil, errors.New("poster object storage is required")
	}

	return &PosterStorage{
		storage: storage,
	}, nil
}

func (s *PosterStorage) Upload(ctx context.Context, key, contentType string, body io.ReadSeeker) error {
	return s.storage.Upload(ctx, key, contentType, body)
}

func (s *PosterStorage) GetURL(ctx context.Context, key string) (string, error) {
	url, err := s.storage.GetURL(ctx, key)
	if errors.Is(err, shareds3.ErrFileNotFound) {
		return "", domain.ErrFileNotFound
	}

	return url, err
}

func (s *PosterStorage) Delete(ctx context.Context, key string) error {
	return s.storage.DeleteFile(ctx, key)
}
