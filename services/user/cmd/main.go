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

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	usergrpc "github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/delivery/grpc"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/infrastructure/config"
	userpostgres "github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/repository/postgres"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/imageutil"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/postgres"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/s3"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
	"google.golang.org/grpc"
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

	appLogger, err := logger.NewLogger(os.Stdout, logger.LogLevelType(cfg.Logger.Level), "json")
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}

	db, err := postgres.Connect(ctx, cfg.Postgres.URL)
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	defer db.Close()

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
		s3Client, cfg.S3.Bucket, cfg.S3.PublicEndpoint, cfg.S3.AvatarURLLifetime, appLogger,
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

	userRepo := userpostgres.NewPostgresUserRepo(db, appLogger)
	avatarRepo := userpostgres.NewPostgresAvatarRepo(db)
	profileRepo := userpostgres.NewProfileRepo(db)

	users := usecase.NewUserUseCase(userRepo, avatarRepo, storage, appLogger)
	profiles := usecase.NewProfileUseCase(profileRepo, storage)

	server := grpc.NewServer(grpc.MaxRecvMsgSize(imageutil.MaxAvatarSize + (1 << 20)))
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
