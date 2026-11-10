package app

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
	filesservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

func initializeObjectStorages(client *s3.Client, cfg config.S3, log logger.Logger) (map[string]*filesservice.S3Storage, error) {
	storages := make(map[string]*filesservice.S3Storage)
	for _, bucket := range []string{cfg.Bucket, cfg.PostersBucket} {
		if _, exists := storages[bucket]; exists {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.BucketCheckTimeout)
		_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
		cancel()
		if err != nil {
			log.Error("check s3 bucket failed", map[string]any{"bucket": bucket, "error": err})
			return nil, fmt.Errorf("check s3 bucket %q: %w", bucket, err)
		}
		objectStorage, err := filesservice.NewS3Storage(client, bucket, cfg.PublicEndpoint, cfg.AvatarURLLifetime, log)
		if err != nil {
			return nil, fmt.Errorf("create s3 storage %q: %w", bucket, err)
		}
		storages[bucket] = objectStorage
		log.Info("s3 storage initialized", map[string]any{"bucket": bucket})
	}
	return storages, nil
}
