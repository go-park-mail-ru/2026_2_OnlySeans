package s3_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	shareds3 "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/s3"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
)

type httpClientFunc func(*http.Request) (*http.Response, error)

func (f httpClientFunc) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestS3StorageUpload(t *testing.T) {
	failure := errors.New("transport unavailable")

	cases := []struct {
		name         string
		stream       bool
		data         []byte
		offset       int64
		transportErr error
	}{
		{
			name: "existing byte upload",
			data: []byte("image-data"),
		},
		{
			name: "empty byte upload",
		},
		{
			name:   "stream upload",
			stream: true,
			data:   []byte("image-data"),
		},
		{
			name:   "stream current position",
			stream: true,
			data:   []byte("xximage-data"),
			offset: 2,
		},
		{
			name:         "stream transport failure",
			stream:       true,
			data:         []byte("image-data"),
			transportErr: failure,
		},
		{
			name:         "byte transport failure",
			data:         []byte("image-data"),
			transportErr: failure,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := awss3.NewFromConfig(aws.Config{
				Region:                     "us-east-1",
				Credentials:                credentials.NewStaticCredentialsProvider("test-key", "test-secret", ""),
				RetryMaxAttempts:           1,
				RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
				HTTPClient: httpClientFunc(func(request *http.Request) (*http.Response, error) {
					calls++
					require.Equal(t, http.MethodPut, request.Method)
					require.Equal(t, "/posters/poster.png", request.URL.Path)
					require.Equal(t, "image/png", request.Header.Get("Content-Type"))
					var body io.Reader = bytes.NewReader(nil)
					if request.Body != nil {
						body = request.Body
					}

					data, err := io.ReadAll(body)
					require.NoError(t, err)
					require.Equal(t, string(tc.data[tc.offset:]), string(data))
					if tc.transportErr != nil {
						return nil, tc.transportErr
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     make(http.Header),
						Body:       io.NopCloser(bytes.NewReader(nil)),
					}, nil
				}),
			}, func(options *awss3.Options) {
				options.BaseEndpoint = aws.String("https://storage.example.com")
				options.UsePathStyle = true
			})
			storage, err := shareds3.NewS3Storage(client, "posters", "", time.Minute, nil)
			require.NoError(t, err)

			if tc.stream {
				body := bytes.NewReader(tc.data)
				_, err := body.Seek(tc.offset, io.SeekStart)
				require.NoError(t, err)
				err = storage.Upload(t.Context(), "poster.png", "image/png", body)
				require.ErrorIs(t, err, tc.transportErr)
			} else {
				err = storage.UploadFile(t.Context(), "poster.png", "image/png", tc.data)
				require.ErrorIs(t, err, tc.transportErr)
			}

			require.Equal(t, 1, calls)
		})
	}
}
