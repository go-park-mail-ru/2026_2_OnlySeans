package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
	reviewsgrpc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/reviews/grpc"
	filmsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/films"
	reviewsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	reviewsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	reviewspb "github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/reviews"
	"google.golang.org/grpc"
)

const (
	defaultListenAddress      = "127.0.0.1:50051"
	defaultDatabaseConfigPath = "configs/database.yaml"
	defaultLoggerConfigPath   = "configs/logger.yaml"
)

func main() {
	listenAddress := flag.String("listen", defaultListenAddress, "gRPC listen address")
	databaseConfigPath := flag.String("database-config", defaultDatabaseConfigPath, "path to database config file")
	loggerConfigPath := flag.String("logger-config", defaultLoggerConfigPath, "path to logger config file")
	flag.Parse()

	if err := run(*listenAddress, *databaseConfigPath, *loggerConfigPath); err != nil {
		log.Fatal(err)
	}
}

func run(listenAddress, databaseConfigPath, loggerConfigPath string) error {
	dbCfg, err := config.LoadDatabase(databaseConfigPath)
	if err != nil {
		return fmt.Errorf("load database config: %w", err)
	}
	if dbCfg.URL == "" {
		return fmt.Errorf("%s is required", config.EnvDatabaseURL)
	}

	loggerCfg, err := config.LoadLogger(loggerConfigPath)
	if err != nil {
		return fmt.Errorf("load logger config: %w", err)
	}

	appLogger, err := logger.NewLogger(os.Stdout, loggerCfg.Level, loggerCfg.Format)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}

	db, err := storage.Open(context.Background(), *dbCfg, appLogger)
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	defer db.Close()

	// юзкейс проверяет, что фильм существует, и читает отзывы
	reviewsRepo := reviewsservice.NewPostgresReviewRepo(db, appLogger)
	filmsRepo := filmsservice.NewPostgresService(db, appLogger)
	operations, err := reviewsusecase.NewOperations(reviewsRepo, filmsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create reviews use cases: %w", err)
	}

	handler, err := reviewsgrpc.NewHandler(operations)
	if err != nil {
		return fmt.Errorf("create reviews handler: %w", err)
	}

	// принимает TCP-соединения на указанном адресе
	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", listenAddress, err)
	}
	defer listener.Close()

	token := os.Getenv(reviewsgrpc.EnvToken)
	if token == "" && !listener.Addr().(*net.TCPAddr).IP.IsLoopback() {
		return fmt.Errorf("%s is required when listening outside loopback", reviewsgrpc.EnvToken)
	}

	appLogger.Info("reviews gRPC server started", map[string]any{"address": listener.Addr().String()})
	if err := serve(listener, handler, token); err != nil {
		return fmt.Errorf("serve reviews gRPC: %w", err)
	}
	return nil
}

func serve(listener net.Listener, handler *reviewsgrpc.Handler, token string) error {
	server := grpc.NewServer(grpc.UnaryInterceptor(reviewsgrpc.AuthenticateBackend(token)))
	// регистрируем обработчик четырёх RPC, чтобы сервер мог направлять ему запросы
	reviewspb.RegisterReviewsServiceServer(server, handler)
	// принимает соединения и держит процесс запущенным, пока работает сервер
	return server.Serve(listener)
}
