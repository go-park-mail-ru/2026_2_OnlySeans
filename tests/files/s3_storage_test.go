package files_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/files"
	"github.com/stretchr/testify/require"
)

func TestS3StorageUpload(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr bool
	}{
		{
			name:   "object uploaded",
			status: http.StatusOK,
		},
		{
			name:    "access denied",
			status:  http.StatusForbidden,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contents := []byte("test image contents")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut || r.URL.Path != "/avatars/test.jpg" {
					t.Errorf("unexpected upload request: %s %s", r.Method, r.URL.Path)
				}
				if got := r.Header.Get("Content-Type"); got != "image/jpeg" {
					t.Errorf("unexpected content type: %s", got)
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read upload: %v", err)
				}
				if !bytes.Equal(contents, body) {
					t.Errorf("uploaded contents differ: %q", body)
				}
				w.WriteHeader(tt.status)
			}))
			defer server.Close()

			client := s3.NewFromConfig(aws.Config{
				Region:      "us-east-1",
				Credentials: credentials.NewStaticCredentialsProvider("test-key", "test-secret", ""),
			}, func(options *s3.Options) {
				options.BaseEndpoint = aws.String(server.URL)
				options.UsePathStyle = true
				options.RetryMaxAttempts = 1
			})
			storage, err := files.NewS3Storage(client, "avatars", "", 15*time.Minute)
			require.NoError(t, err)
			err = storage.Upload(t.Context(), "test.jpg", "image/jpeg", bytes.NewReader(contents))
			if tt.wantErr {
				require.ErrorContains(t, err, "upload object")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestS3StorageGetURL(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr bool
		missing bool
	}{
		{
			name:   "existing object",
			status: http.StatusOK,
		},
		{
			name:    "missing object",
			status:  http.StatusNotFound,
			wantErr: true,
			missing: true,
		},
		{
			name:    "access denied is not missing",
			status:  http.StatusForbidden,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const key = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.png"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodHead || r.URL.Path != "/avatars/"+key {
					t.Errorf("unexpected object check: %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(tt.status)
			}))
			defer server.Close()
			client := s3.NewFromConfig(aws.Config{
				Region:      "us-east-1",
				Credentials: credentials.NewStaticCredentialsProvider("test-key", "test-secret", ""),
			}, func(options *s3.Options) {
				options.BaseEndpoint = aws.String(server.URL)
				options.UsePathStyle = true
				options.RetryMaxAttempts = 1
			})
			// Internal checks use the test server; links use the browser-facing endpoint.
			storage, err := files.NewS3Storage(client, "avatars", "https://media.example.com", 2*time.Minute)
			require.NoError(t, err)
			link, err := storage.GetURL(t.Context(), key)
			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, link)
				if tt.missing {
					require.ErrorIs(t, err, domain.ErrFileNotFound)
				} else {
					require.NotErrorIs(t, err, domain.ErrFileNotFound)
				}
				return
			}
			require.NoError(t, err)
			parsed, err := url.Parse(link)
			require.NoError(t, err)
			require.Equal(t, "https", parsed.Scheme)
			require.Equal(t, "media.example.com", parsed.Host)
			require.Equal(t, "/avatars/"+key, parsed.Path)
			require.Equal(t, "120", parsed.Query().Get("X-Amz-Expires"))
			require.NotEmpty(t, parsed.Query().Get("X-Amz-Signature"))
		})
	}
}
