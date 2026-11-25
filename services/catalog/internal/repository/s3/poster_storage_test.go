package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	shareds3 "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/s3"
)

type objectStorageStub struct {
	upload     func(context.Context, string, string, io.ReadSeeker) error
	getURL     func(context.Context, string) (string, error)
	deleteFile func(context.Context, string) error
}

func (s objectStorageStub) Upload(ctx context.Context, key, contentType string, body io.ReadSeeker) error {
	return s.upload(ctx, key, contentType, body)
}

func (s objectStorageStub) GetURL(ctx context.Context, key string) (string, error) {
	return s.getURL(ctx, key)
}

func (s objectStorageStub) DeleteFile(ctx context.Context, key string) error {
	return s.deleteFile(ctx, key)
}

func TestPosterStorageGetURL(t *testing.T) {
	failure := errors.New("storage failure")

	cases := []struct {
		name       string
		storageErr error
		wantErr    error
		wantURL    string
	}{
		{
			name:    "signed URL",
			wantURL: "https://example.com/poster.png",
		},
		{
			name:       "missing object",
			storageErr: shareds3.ErrFileNotFound,
			wantErr:    domain.ErrFileNotFound,
		},
		{
			name:       "wrapped missing object",
			storageErr: fmt.Errorf("lookup: %w", shareds3.ErrFileNotFound),
			wantErr:    domain.ErrFileNotFound,
		},
		{
			name:       "storage failure",
			storageErr: failure,
			wantErr:    failure,
		},
		{
			name:       "cancellation",
			storageErr: context.Canceled,
			wantErr:    context.Canceled,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			storage := objectStorageStub{
				getURL: func(gotCtx context.Context, key string) (string, error) {
					require.Equal(t, ctx, gotCtx)
					require.Equal(t, "poster.png", key)
					return tc.wantURL, tc.storageErr
				},
			}
			adapter, err := NewPosterStorage(storage)
			require.NoError(t, err)
			url, err := adapter.GetURL(ctx, "poster.png")
			require.ErrorIs(t, err, tc.wantErr)
			require.Equal(t, tc.wantURL, url)
		})
	}
}

func TestPosterStorageForwardsUploadAndDelete(t *testing.T) {
	failure := errors.New("storage failure")

	cases := []struct {
		name       string
		operation  string
		storageErr error
	}{
		{
			name:      "upload",
			operation: "upload",
		},
		{
			name:       "upload failure",
			operation:  "upload",
			storageErr: failure,
		},
		{
			name:      "delete",
			operation: "delete",
		},
		{
			name:       "delete failure",
			operation:  "delete",
			storageErr: failure,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			body := bytes.NewReader([]byte("image-data"))
			calls := 0
			storage := objectStorageStub{
				upload: func(gotCtx context.Context, key, contentType string, gotBody io.ReadSeeker) error {
					calls++
					require.Equal(t, ctx, gotCtx)
					require.Equal(t, "poster.png", key)
					require.Equal(t, "image/png", contentType)
					require.Same(t, body, gotBody)
					return tc.storageErr
				},
				deleteFile: func(gotCtx context.Context, key string) error {
					calls++
					require.Equal(t, ctx, gotCtx)
					require.Equal(t, "poster.png", key)
					return tc.storageErr
				},
			}
			adapter, err := NewPosterStorage(storage)
			require.NoError(t, err)

			if tc.operation == "upload" {
				err = adapter.Upload(ctx, "poster.png", "image/png", body)
			} else {
				err = adapter.Delete(ctx, "poster.png")
			}

			require.ErrorIs(t, err, tc.storageErr)
			require.Equal(t, 1, calls)
		})
	}
}
