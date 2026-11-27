package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	"google.golang.org/grpc"

	personsgrpc "github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/delivery/grpc"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/infrastructure/config"
	personspostgres "github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/repository/postgres"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/metrics"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/postgres"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/persons"
)

func run(ctx context.Context) error {
	cfg := config.Load()

	if cfg.GRPC.Port == "" {
		return errors.New("grpc.port is required")
	}

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

	personRepo := personspostgres.NewPersonRepo(db, appLogger, appMetrics)
	catalogRepo := personspostgres.NewCatalogRepo(db, appLogger, appMetrics)

	persons := usecase.NewPersonUseCase(personRepo)
	catalog := usecase.NewCatalogUseCase(catalogRepo)

	server := grpc.NewServer(grpc.ChainUnaryInterceptor(grpcutil.UnaryServerRequestID, grpcutil.UnaryServerLogging("persons", appLogger)))
	pb.RegisterPersonServiceServer(server, personsgrpc.NewPersonHandler(persons, catalog))

	addr := net.JoinHostPort(cfg.GRPC.Host, cfg.GRPC.Port)
	listener, err := net.Listen("tcp", addr)

	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	defer func() { _ = listener.Close() }()

	stopServer := context.AfterFunc(ctx, server.GracefulStop)
	defer stopServer()

	appLogger.Info("Persons gRPC server started", map[string]any{
		"address": listener.Addr().String(),
	})

	if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("serve Persons gRPC: %w", err)
	}

	return nil
}
