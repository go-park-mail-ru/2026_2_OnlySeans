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

	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/redis"
	authgrpc "github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/delivery/grpc"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/infrastructure/config"
	grpcclient "github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/infrastructure/grpc_client"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/repository/redisrepo"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/password"
	pbauth "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/auth"
	pbuser "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	if cfg.GRPC.Port == "" || cfg.UserServiceAddr == "" {
		return errors.New("grpc.port and user_service_addr are required")
	}
	if cfg.SessionTTL <= 0 {
		return errors.New("session_ttl must be positive")
	}

	appLogger, err := logger.NewLogger(os.Stdout, logger.LogLevelType(cfg.Logger.Level), "json")
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}

	redisClient, err := redis.NewClientWithPing(ctx, cfg.Redis.URL)
	if err != nil {
		return fmt.Errorf("connect to Redis: %w", err)
	}
	defer redisClient.Close()

	userConn, err := grpc.NewClient(
		cfg.UserServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("create User gRPC client: %w", err)
	}
	defer userConn.Close()

	userClient := grpcclient.New(pbuser.NewUserServiceClient(userConn))
	sessionRepo := redisrepo.NewRedisSessionStore(redisClient, appLogger)
	sessions := usecase.NewSessionUseCase(sessionRepo, cfg.SessionTTL)
	auth := usecase.NewAuthUseCase(userClient, sessions, &password.BcryptPasswordService{})

	server := grpc.NewServer()
	pbauth.RegisterAuthServiceServer(server, authgrpc.NewAuthHandler(auth))

	addr := net.JoinHostPort(cfg.GRPC.Host, cfg.GRPC.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	stopServer := context.AfterFunc(ctx, server.GracefulStop)
	defer stopServer()

	appLogger.Info("Auth gRPC server started", map[string]any{"address": listener.Addr().String()})

	if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("serve Auth gRPC: %w", err)
	}

	return nil
}
