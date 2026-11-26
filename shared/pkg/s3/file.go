package s3

import (
	"context"
	"errors"
)

var ErrFileNotFound = errors.New("file not found")

type FileStorage interface {
	UploadFile(ctx context.Context, key, contentType string, data []byte) error
	GetURL(ctx context.Context, key string) (string, error)
	DeleteFile(ctx context.Context, key string) error
}
