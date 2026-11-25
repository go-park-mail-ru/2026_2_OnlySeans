package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	grpcgo "google.golang.org/grpc"

	cataloggrpc "github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/delivery/grpc"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/infrastructure/config"
	catalogpostgres "github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository/postgres"
	catalogs3 "github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository/s3"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/postgres"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/s3"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate catalog config: %w", err)
	}

	appLogger, err := logger.NewLogger(os.Stdout, logger.LogLevelType(cfg.Logger.Level), "json")
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}

	db, err := postgres.Connect(ctx, cfg.Postgres.URL)
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			appLogger.Error("close database failed", map[string]any{
				"error": err,
			})
		}
	}()

	s3Client, err := s3.NewS3Client(s3.ClientConfig{
		Endpoint:  cfg.S3.Endpoint,
		Region:    cfg.S3.Region,
		AccessKey: cfg.S3.AccessKey,
		SecretKey: cfg.S3.SecretKey,
	})
	if err != nil {
		return fmt.Errorf("create S3 client: %w", err)
	}

	storage, err := s3.NewS3Storage(s3Client, cfg.S3.Bucket, cfg.S3.PublicEndpoint, cfg.S3.PosterURLLifetime, appLogger)
	if err != nil {
		return fmt.Errorf("create S3 storage: %w", err)
	}

	bucketCtx, cancel := context.WithTimeout(ctx, cfg.S3.BucketCheckTimeout)
	_, err = s3Client.HeadBucket(bucketCtx, &awss3.HeadBucketInput{
		Bucket: aws.String(cfg.S3.Bucket),
	})

	cancel()

	if err != nil {
		return fmt.Errorf("check S3 bucket %q: %w", cfg.S3.Bucket, err)
	}

	posterStorage, err := catalogs3.NewPosterStorage(storage)
	if err != nil {
		return fmt.Errorf("create poster storage: %w", err)
	}

	filmsRepo := catalogpostgres.NewFilmRepo(db, appLogger)
	ratingsRepo := catalogpostgres.NewRatingRepo(db, appLogger)
	libraryRepo := catalogpostgres.NewLibraryRepo(db, appLogger)
	preferencesRepo := catalogpostgres.NewPreferencesRepo(db, appLogger)

	postersRepo, err := catalogpostgres.NewPosterRepo(db, appLogger)
	if err != nil {
		return fmt.Errorf("create poster repository: %w", err)
	}

	filmsWithPosters, err := usecase.NewPosterURLReader(filmsRepo, filmsRepo, filmsRepo, posterStorage)
	if err != nil {
		return fmt.Errorf("create film reader with poster URLs: %w", err)
	}

	posters, err := usecase.NewPosterUseCase(postersRepo, filmsRepo, posterStorage, usecase.PosterOptions{
		MaxSize:        cfg.S3.MaxPosterSize,
		MaxDimension:   cfg.S3.MaxPosterDimension,
		CleanupTimeout: cfg.S3.PosterCleanupTimeout,
	}, appLogger)
	if err != nil {
		return fmt.Errorf("create poster use case: %w", err)
	}

	handler := cataloggrpc.NewCatalogHandler(
		usecase.NewFilmUseCase(filmsWithPosters, filmsWithPosters, filmsRepo),
		usecase.NewCollectionUseCase(filmsWithPosters),
		usecase.NewRatingUseCase(ratingsRepo, ratingsRepo, filmsRepo),
		usecase.NewLibraryUseCase(libraryRepo),
		usecase.NewPreferencesUseCase(preferencesRepo),
		posters,
	)

	server, err := cataloggrpc.NewServer(handler, cfg.S3.MaxPosterSize)
	if err != nil {
		return fmt.Errorf("create catalog gRPC server: %w", err)
	}
	defer server.Stop()

	listener, err := net.Listen("tcp", cfg.Address())
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.Address(), err)
	}

	defer func() { _ = listener.Close() }()
	appLogger.Info("Catalog gRPC server started", map[string]any{
		"address": listener.Addr().String(),
	})

	return serve(ctx, server, listener, cfg.ShutdownTimeout)
}

func serve(ctx context.Context, server *grpcgo.Server, listener net.Listener, shutdownTimeout time.Duration) error {
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.Serve(listener) }()
	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, grpcgo.ErrServerStopped) {
			return fmt.Errorf("serve catalog gRPC: %w", err)
		}

		return nil
	case <-ctx.Done():
	}

	stopped := make(chan struct{})

	go func() {
		server.GracefulStop()
		close(stopped)
	}()

	timer := time.NewTimer(shutdownTimeout)
	defer timer.Stop()
	select {
	case <-stopped:
	case <-timer.C:
		server.Stop()
	}

	if err := <-serverErr; err != nil && !errors.Is(err, grpcgo.ErrServerStopped) {
		return fmt.Errorf("serve catalog gRPC: %w", err)
	}

	return nil
}
