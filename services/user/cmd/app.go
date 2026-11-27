package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"google.golang.org/grpc"

	usergrpc "github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/delivery/grpc"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/infrastructure/config"
	userpostgres "github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/repository/postgres"
	users3 "github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/repository/s3"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/imageutil"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/metrics"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/password"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/postgres"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/s3"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
)

func run(ctx context.Context) error {
	cfg := config.Load()

	appLogger, err := logger.NewLogger(os.Stdout, logger.LogLevelType(cfg.Logger.Level), "json")
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}

	appMetrics := metrics.New()
	closeMetrics, err := appMetrics.Listen(os.Getenv("METRICS_ADDR"))
	if err != nil {
		return err
	}

	defer closeMetrics()

	db, err := postgres.Connect(ctx, cfg.Postgres.URL)
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	defer func() { _ = db.Close() }()

	s3Client, err := s3.NewS3Client(s3.ClientConfig{
		Endpoint:  cfg.S3.Endpoint,
		Region:    cfg.S3.Region,
		AccessKey: cfg.S3.AccessKey,
		SecretKey: cfg.S3.SecretKey,
	})
	if err != nil {
		return fmt.Errorf("create S3 client: %w", err)
	}

	storage, err := s3.NewS3Storage(
		s3Client, cfg.S3.Bucket, cfg.S3.PublicEndpoint, cfg.S3.AvatarURLLifetime, appLogger, appMetrics,
	)
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

	userRepo := userpostgres.NewPostgresUserRepo(db, appLogger, appMetrics)
	avatarRepo := userpostgres.NewPostgresAvatarRepo(db, appLogger, appMetrics)
	profileRepo := userpostgres.NewProfileRepo(db, appLogger, appMetrics)

	files := users3.NewFileStorage(storage)
	hasher := &password.BcryptPasswordService{}

	users := usecase.NewUserUseCase(userRepo, avatarRepo, files, hasher, appLogger)
	profiles := usecase.NewProfileUseCase(profileRepo, files)

	server := grpc.NewServer(grpc.ChainUnaryInterceptor(grpcutil.UnaryServerRequestID, grpcutil.UnaryServerLogging("user", appLogger)), grpc.MaxRecvMsgSize(imageutil.MaxAvatarSize+(1<<20)))
	pb.RegisterUserServiceServer(server, usergrpc.NewUserHandler(users, profiles))

	addr := net.JoinHostPort(cfg.GRPC.Host, cfg.GRPC.Port)
	listener, err := net.Listen("tcp", addr)

	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	stopServer := context.AfterFunc(ctx, server.GracefulStop)
	defer stopServer()

	appLogger.Info("User gRPC server started", map[string]any{"address": listener.Addr().String()})

	if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("serve User gRPC: %w", err)
	}

	return nil
}
