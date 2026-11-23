package s3

import (
	"context"
)

type FileStorage interface {
	UploadFile(ctx context.Context, key, contentType string, data []byte) error
	GetURL(ctx context.Context, key string) (string, error)
	DeleteFile(ctx context.Context, key string) error
}
