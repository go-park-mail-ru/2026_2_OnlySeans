package http_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"

	assetshttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/assets"
	authhttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

type assetTransport func(*http.Request) (*http.Response, error)

func (f assetTransport) Do(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestGatewayStaticAssets(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		s3Status   int
		wantStatus int
		wantCalls  int
	}{
		{
			name:       "existing collection cover",
			path:       "/static/collections/cover.png",
			s3Status:   http.StatusOK,
			wantStatus: http.StatusOK,
			wantCalls:  1,
		},
		{
			name:       "missing object",
			path:       "/static/collections/cover.png",
			s3Status:   http.StatusNotFound,
			wantStatus: http.StatusNotFound,
			wantCalls:  1,
		},
		{
			name:       "empty key",
			path:       "/static/",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid key",
			path:       "/static/collections/invalid..png",
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			client := awss3.NewFromConfig(aws.Config{
				Region:           "us-east-1",
				Credentials:      credentials.NewStaticCredentialsProvider("test", "test", ""),
				RetryMaxAttempts: 1,
				HTTPClient: assetTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					require.Equal(t, http.MethodGet, r.Method)
					require.Equal(t, "/avatars/collections/cover.png", r.URL.Path)
					body := "image-data"
					if tt.s3Status != http.StatusOK {
						body = `<Error><Code>NoSuchKey</Code><Message>Not found</Message></Error>`
					}
					return &http.Response{
						StatusCode: tt.s3Status,
						Header: http.Header{
							"Content-Type": []string{"image/png"},
						},
						Body: io.NopCloser(strings.NewReader(body)),
					}, nil
				}),
			}, func(options *awss3.Options) {
				options.BaseEndpoint = aws.String("http://storage.test")
				options.UsePathStyle = true
			})
			appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelInfo, "json")
			require.NoError(t, err)

			auth := authStub{}
			users := userStub{}
			handler, err := router.NewRouter(router.Handlers{
				Auth:   authhttp.NewAuthHandler(auth, users, appLogger),
				Assets: assetshttp.NewHandler(client, "avatars", appLogger),
			}, middleware.NewAuthMiddleware(auth, users, appLogger), nil, appLogger)
			require.NoError(t, err)

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tt.path, nil))
			require.Equal(t, tt.wantStatus, response.Code)
			require.Equal(t, tt.wantCalls, calls)

			if tt.wantStatus == http.StatusOK {
				require.Equal(t, "image-data", response.Body.String())
				require.Equal(t, "image/png", response.Header().Get("Content-Type"))
				require.Equal(t, "public, max-age=3600", response.Header().Get("Cache-Control"))
			}
		})
	}
}
