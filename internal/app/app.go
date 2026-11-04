package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	filmsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/films"
	persondelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/persons"
	reviewsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	authservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/auth"
	filmsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/films"
	personservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/persons"
	reviewsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	authusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	filmsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/films"
	personusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/persons"
	reviewsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/redis"
)

func Run(configPath, databaseConfigPath, redisConfigPath, loggerConfigPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	dbCfg, err := config.LoadDatabase(databaseConfigPath)
	if err != nil {
		return fmt.Errorf("load database config: %w", err)
	}

	if dbCfg.URL == "" {
		return fmt.Errorf("%s is required", config.EnvDatabaseURL)
	}

	redisCfg, err := config.LoadRedis(redisConfigPath)
	if err != nil {
		return fmt.Errorf("load redis config: %w", err)
	}

	loggerCfg, err := config.LoadLogger(loggerConfigPath)
	if err != nil {
		return fmt.Errorf("load logger config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(loggerCfg.File), 0o755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	logFile, err := os.OpenFile(loggerCfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer logFile.Close()

	appLogger, err := logger.NewLogger(logFile, loggerCfg.Level, loggerCfg.Format)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}

	var (
		useRepo     domain.UserRepository
		filmsRepo   domain.FilmsRepository
		reviewsRepo domain.ReviewRepository
		personsRepo domain.PersonRepository
	)

	db, err := storage.Open(context.Background(), *dbCfg, appLogger)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	appLogger.Info("database connected", nil)
	defer db.Close()

	useRepo = authservice.NewPostgresUserRepo(db, appLogger)
	filmsRepo = filmsservice.NewPostgresService(db, appLogger)
	reviewsRepo = reviewsservice.NewPostgresReviewRepo(db, appLogger)
	personsRepo = personservice.NewPersonRepo(db, appLogger)

	newClient, err := redis.NewClientWithPing(context.Background(), redisCfg.URL)
	if err != nil {
		return fmt.Errorf("connect redis: %w", err)
	}

	appLogger.Info("redis connected", nil)
	defer newClient.Close()

	sessions := authservice.NewRedisSessionStore(newClient, cfg.Server.SessionTTL, appLogger)

	passwordService := authservice.BcryptPasswordService{}

	registerUC, err := authusecase.NewRegisterUserUseCase(useRepo, sessions, passwordService, appLogger)
	if err != nil {
		return fmt.Errorf("can't create register use case: %w", err)
	}

	loginUC, err := authusecase.NewLoginUserUseCase(useRepo, sessions, passwordService, appLogger)
	if err != nil {
		return fmt.Errorf("can't create login use case: %w", err)
	}

	logoutUC, err := authusecase.NewLogoutUserUseCase(sessions, appLogger)
	if err != nil {
		return fmt.Errorf("can't create logout use case: %w", err)
	}

	authenticateUC, err := authusecase.NewAuthenticateUserUseCase(sessions, appLogger)
	if err != nil {
		return fmt.Errorf("can't create authenticate use case: %w", err)
	}

	getUserUC, err := authusecase.NewGetUserUseCase(useRepo, appLogger)
	if err != nil {
		return fmt.Errorf("can't create get user use case: %w", err)
	}

	listFilmsUC, err := filmsusecase.NewListFilmsUseCase(filmsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create list films use case: %w", err)
	}

	getFilmUC, err := filmsusecase.NewGetFilmUseCase(filmsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create get film use case: %w", err)
	}

	listCollectionsUC, err := filmsusecase.NewListCollectionsUseCase(filmsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create list collections use case: %w", err)
	}

	getCollectionUC, err := filmsusecase.NewGetCollectionUseCase(filmsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create get collection use case: %w", err)
	}

	authHandler, err := authdelivery.NewHandler(authdelivery.HandlerUseCases{
		Register:     registerUC,
		Login:        loginUC,
		Logout:       logoutUC,
		Authenticate: authenticateUC,
		GetUser:      getUserUC,
	}, appLogger)
	if err != nil {
		return fmt.Errorf("create handler: %w", err)
	}

	filmsHandler, err := filmsdelivery.NewHandler(filmsdelivery.HandlerUseCases{
		ListFilms:       listFilmsUC,
		GetFilm:         getFilmUC,
		ListCollections: listCollectionsUC,
		GetCollection:   getCollectionUC,
	}, appLogger)
	if err != nil {
		return fmt.Errorf("create films handler: %w", err)
	}

	listReviewsUC, err := reviewsusecase.NewListReviewsUseCase(reviewsRepo, filmsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create list reviews use case: %w", err)
	}

	createReviewUC, err := reviewsusecase.NewCreateReviewUseCase(reviewsRepo, filmsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create create review use case: %w", err)
	}

	updateReviewUC, err := reviewsusecase.NewUpdateReviewUseCase(reviewsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create update review use case: %w", err)
	}

	deleteReviewUC, err := reviewsusecase.NewDeleteReviewUseCase(reviewsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create delete review use case: %w", err)
	}

	reviewsHandler, err := reviewsdelivery.NewHandler(reviewsdelivery.HandlerUseCases{
		List:   listReviewsUC,
		Create: createReviewUC,
		Update: updateReviewUC,
		Delete: deleteReviewUC,
	}, appLogger)

	if err != nil {
		return fmt.Errorf("create reviews handler: %w", err)
	}

	createPersonUC, err := personusecase.NewCreatePersonUseCase(personsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create create person use case: %w", err)
	}

	getPersonUC, err := personusecase.NewGetPersonUseCase(personsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create get person use case: %w", err)
	}

	listPersonsUC, err := personusecase.NewListPersonsUseCase(personsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create list persons use case: %w", err)
	}

	updatePersonUC, err := personusecase.NewUpdatePersonUseCase(personsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create update person use case: %w", err)
	}

	deletePersonUC, err := personusecase.NewDeletePersonUseCase(personsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create delete person use case: %w", err)
	}

	personsHandler, err := persondelivery.NewHandler(persondelivery.HandlerUseCases{
		Create: createPersonUC,
		Get:    getPersonUC,
		List:   listPersonsUC,
		Update: updatePersonUC,
		Delete: deletePersonUC,
	}, appLogger)
	if err != nil {
		return fmt.Errorf("create persons handler: %w", err)
	}

	httpHandler := router.NewRouter(
		authHandler,
		filmsHandler,
		reviewsHandler,
		personsHandler,
		cfg.Server.AllowedOrigin,
		appLogger,
	)

	server := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      httpHandler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	return serve(server, cfg.Server.ShutdownTimeout, appLogger)
}

func serve(server *http.Server, shutdownTimeout time.Duration, appLogger logger.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		appLogger.Info("server started", map[string]any{"addr": server.Addr})
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server: %w", err)
		}

		return nil
	case <-ctx.Done():
	}

	appLogger.Info("shutting down", map[string]any{"addr": server.Addr})
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("server shutdown failed", map[string]any{
			"addr":  server.Addr,
			"error": err,
		})
		return fmt.Errorf("shutdown server: %w", err)
	}

	return nil
}
