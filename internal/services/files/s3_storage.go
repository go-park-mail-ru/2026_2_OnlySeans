package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

const AvatarURLLifetime = 15 * time.Minute

type S3Storage struct {
	client    *s3.Client
	bucket    string
	presigner *s3.PresignClient
}

func NewS3Storage(client *s3.Client, bucket, publicEndpoint string) (*S3Storage, error) {
	if client == nil {
		return nil, errors.New("s3 client is nil")
	}
	if strings.TrimSpace(bucket) == "" {
		return nil, errors.New("s3 bucket is required")
	}

	return &S3Storage{
		client: client,
		bucket: bucket,
		presigner: s3.NewPresignClient(client, func(options *s3.PresignOptions) {
			if publicEndpoint != "" {
				options.ClientOptions = append(options.ClientOptions, func(clientOptions *s3.Options) {
					clientOptions.BaseEndpoint = aws.String(publicEndpoint)
				})
			}
		}),
	}, nil
}

func (s *S3Storage) Upload(ctx context.Context, key, contentType string, body io.ReadSeeker) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
		Body:        body,
	})

	if err != nil {
		return fmt.Errorf("upload object: %w", err)
	}

	return nil
}

func (s *S3Storage) GetURL(ctx context.Context, key string) (string, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return "", domain.ErrFileNotFound
		}
		return "", fmt.Errorf("check avatar object: %w", err)
	}
	request, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, func(options *s3.PresignOptions) {
		options.Expires = AvatarURLLifetime
	})
	if err != nil {
		return "", fmt.Errorf("sign avatar URL: %w", err)
	}
	return request.URL, nil
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete avatar object: %w", err)
	}
	return nil
}
