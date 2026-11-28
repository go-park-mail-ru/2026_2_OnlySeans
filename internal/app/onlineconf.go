package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
	delivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/onlineconf"
	authservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/auth"
	service "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/redis"
)

const onlineConfServerErrorCapacity = 1

func RunOnlineConf(ctx context.Context, configPath, databasePath, redisPath, loggerPath string) (runErr error) {
	cfg, err := config.LoadOnlineConf(configPath)
	if err != nil {
		return fmt.Errorf("load onlineconf config: %w", err)
	}

	dbCfg, err := config.LoadDatabase(databasePath)
	if err != nil {
		return fmt.Errorf("load onlineconf database config: %w", err)
	}

	if dbCfg.URL == "" {
		return fmt.Errorf("%s is required", config.EnvDatabaseURL)
	}

	redisCfg, err := config.LoadRedis(redisPath)
	if err != nil {
		return fmt.Errorf("load onlineconf Redis config: %w", err)
	}

	loggerCfg, err := config.LoadLogger(loggerPath)
	if err != nil {
		return fmt.Errorf("load onlineconf logger config: %w", err)
	}

	var logOutput io.Writer = os.Stdout

	if loggerCfg.File != "" {
		if err := os.MkdirAll(filepath.Dir(loggerCfg.File), logDirectoryMode); err != nil {
			return fmt.Errorf("create onlineconf log directory: %w", err)
		}

		file, err := os.OpenFile(loggerCfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, logFileMode)
		if err != nil {
			return fmt.Errorf("open onlineconf log: %w", err)
		}

		defer closeOnlineConfResource(file, "log file", &runErr)

		logOutput = file
	}

	appLogger, err := logger.NewLogger(logOutput, loggerCfg.Level, loggerCfg.Format)
	if err != nil {
		return fmt.Errorf("create onlineconf logger: %w", err)
	}

	db, err := storage.Open(ctx, *dbCfg, appLogger)
	if err != nil {
		return fmt.Errorf("open onlineconf database: %w", err)
	}

	defer closeOnlineConfResource(db, "database", &runErr)

	connectCtx, cancelConnect := context.WithTimeout(ctx, dbCfg.ConnectTimeout)
	client, err := redis.NewClientWithPing(connectCtx, redisCfg.URL)

	cancelConnect()

	if err != nil {
		return fmt.Errorf("connect onlineconf Redis: %w", err)
	}

	defer client.Close()

	repository := service.NewPostgresRepository(db, appLogger)

	snapshots, err := service.NewSnapshotStore(repository, service.SnapshotOptions{RefreshInterval: cfg.Snapshot.RefreshInterval, LoadTimeout: cfg.Snapshot.LoadTimeout}, appLogger)
	if err != nil {
		return fmt.Errorf("create onlineconf snapshot store: %w", err)
	}

	workerCtx, cancelWorker := context.WithCancel(ctx)

	defer func() { cancelWorker(); <-snapshots.Done() }()

	if err := snapshots.Start(workerCtx); err != nil {
		return fmt.Errorf("start onlineconf snapshot worker: %w", err)
	}

	users := authservice.NewPostgresUserRepo(db, appLogger)
	sessions := authservice.NewRedisSessionStore(client, cfg.Server.SessionTTL, appLogger)

	access, err := service.NewAccess(sessions, users, appLogger)
	if err != nil {
		return fmt.Errorf("create onlineconf access checker: %w", err)
	}

	middleware, err := delivery.NewMiddleware(access, cfg.ServiceToken, cfg.Server.AllowedOrigins, appLogger)
	if err != nil {
		return fmt.Errorf("create onlineconf middleware: %w", err)
	}

	scopes, err := usecase.NewScopesUseCase(repository, snapshots, appLogger)
	if err != nil {
		return fmt.Errorf("create onlineconf scopes use case: %w", err)
	}

	configs, err := usecase.NewConfigsUseCase(repository, snapshots, appLogger)
	if err != nil {
		return fmt.Errorf("create onlineconf configs use case: %w", err)
	}

	flags, err := usecase.NewFlagsUseCase(repository, snapshots, appLogger)
	if err != nil {
		return fmt.Errorf("create onlineconf flags use case: %w", err)
	}

	evaluate, err := usecase.NewEvaluateFlagUseCase(users, snapshots, service.Evaluate, appLogger)
	if err != nil {
		return fmt.Errorf("create onlineconf evaluator use case: %w", err)
	}

	audit, err := usecase.NewListAuditUseCase(repository, appLogger)
	if err != nil {
		return fmt.Errorf("create onlineconf audit use case: %w", err)
	}

	handler, err := delivery.NewRouter(delivery.HandlerUseCases{Scopes: scopes, Configs: configs, Flags: flags, Evaluate: evaluate, Audit: audit}, middleware, snapshots, appLogger)
	if err != nil {
		return fmt.Errorf("create onlineconf router: %w", err)
	}

	server := &http.Server{Addr: cfg.Server.Addr(), Handler: handler, ReadHeaderTimeout: cfg.Server.ReadTimeout, ReadTimeout: cfg.Server.ReadTimeout, WriteTimeout: cfg.Server.WriteTimeout, IdleTimeout: cfg.Server.IdleTimeout, BaseContext: func(net.Listener) context.Context { return workerCtx }}

	if err := serveOnlineConf(ctx, server, cfg.Server, appLogger); err != nil {
		return err
	}

	return nil
}

func serveOnlineConf(ctx context.Context, server *http.Server, cfg config.Server, appLogger logger.Logger) error {
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("listen onlineconf HTTP: %w", err)
	}

	errorsCh := make(chan error, onlineConfServerErrorCapacity)

	go func() { errorsCh <- server.Serve(listener) }()

	appLogger.Info("onlineconf HTTP server started", map[string]any{"address": listener.Addr().String()})

	select {
	case err := <-errorsCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve onlineconf HTTP: %w", err)
		}

		return nil
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		if closeErr := server.Close(); closeErr != nil {
			return errors.Join(fmt.Errorf("shutdown onlineconf HTTP: %w", err), closeErr)
		}

		return fmt.Errorf("shutdown onlineconf HTTP: %w", err)
	}

	if err := <-errorsCh; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("stop onlineconf HTTP: %w", err)
	}

	return nil
}

func closeOnlineConfResource(resource io.Closer, name string, result *error) {
	if err := resource.Close(); err != nil {
		*result = errors.Join(*result, fmt.Errorf("close onlineconf %s: %w", name, err))
	}
}
