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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	filesdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/files"
	filmsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/films"
	persondelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/persons"
	ratingsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/ratings"
	reviewsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	authservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/auth"
	filesservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/files"
	filmsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/films"
	personservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/persons"
	ratingsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/ratings"
	reviewsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	authusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	filesusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/files"
	filmsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/films"
	personusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/persons"
	ratingsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/ratings"
	reviewsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/redis"
)

func Run(configPath, databaseConfigPath, redisConfigPath, s3ConfigPath, loggerConfigPath string) error {
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

	s3Cfg, err := config.LoadS3(s3ConfigPath)
	if err != nil {
		return fmt.Errorf("load s3 config: %w", err)
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

	s3Client, err := storage.NewS3Client(*s3Cfg)
	if err != nil {
		appLogger.Error("create s3 client failed", map[string]any{"error": err})
		return fmt.Errorf("create s3 client: %w", err)
	}

	s3Ctx, cancelS3 := context.WithTimeout(context.Background(), s3Cfg.BucketCheckTimeout)

	_, err = s3Client.HeadBucket(s3Ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s3Cfg.Bucket),
	})
	cancelS3()

	if err != nil {
		appLogger.Error("check s3 bucket failed", map[string]any{"error": err})
		return fmt.Errorf("check s3 bucket: %w", err)
	}

	appLogger.Info("s3 bucket available", map[string]any{"bucket": s3Cfg.Bucket})

	avatarStorage, err := filesservice.NewS3Storage(s3Client, s3Cfg.Bucket, s3Cfg.PublicEndpoint, s3Cfg.AvatarURLLifetime)
	if err != nil {
		appLogger.Error("create avatar storage failed", map[string]any{"error": err})
		return fmt.Errorf("create avatar storage: %w", err)
	}

	var (
		userRepo    domain.UserRepository
		filmsRepo   domain.FilmsRepository
		ratingsRepo domain.FilmRatingRepository
		reviewsRepo domain.ReviewRepository
		personsRepo domain.PersonRepository
	)

	db, err := storage.Open(context.Background(), *dbCfg, appLogger)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	appLogger.Info("database connected", nil)
	defer db.Close()

	avatarRepo, err := filesservice.NewPostgresAvatarRepo(db)
	if err != nil {
		appLogger.Error("create avatar repository failed", map[string]any{"error": err})
		return fmt.Errorf("create avatar repository: %w", err)
	}
	uploadAvatarUC, err := filesusecase.NewUploadAvatarUseCase(avatarStorage, avatarRepo, filesusecase.UploadAvatarOptions{
		MaxSize:        s3Cfg.MaxAvatarSize,
		MaxDimension:   s3Cfg.MaxAvatarDimension,
		CleanupTimeout: s3Cfg.AvatarCleanupTimeout,
	}, appLogger)
	if err != nil {
		appLogger.Error("create upload avatar use case failed", map[string]any{"error": err})
		return fmt.Errorf("create upload avatar use case: %w", err)
	}
	getAvatarURLUC, err := filesusecase.NewGetAvatarURLUseCase(avatarStorage, appLogger)
	if err != nil {
		appLogger.Error("create get avatar URL use case failed", map[string]any{"error": err})
		return fmt.Errorf("create get avatar URL use case: %w", err)
	}
	accountAvatarUC, err := filesusecase.NewAccountAvatarUseCase(avatarRepo, avatarStorage, appLogger)
	if err != nil {
		appLogger.Error("create account avatar use case failed", map[string]any{"error": err})
		return fmt.Errorf("create account avatar use case: %w", err)
	}
	filesHandler, err := filesdelivery.NewHandler(uploadAvatarUC, getAvatarURLUC, accountAvatarUC, appLogger)
	if err != nil {
		appLogger.Error("create files handler failed", map[string]any{"error": err})
		return fmt.Errorf("create files handler: %w", err)
	}

	userRepo = authservice.NewPostgresUserRepo(db)
	filmsRepo = filmsservice.NewPostgresService(db)
	reviewsRepo = reviewsservice.NewPostgresReviewRepo(db)
	personsRepo = personservice.NewPersonRepo(db)
	postgresRatingsRepo := ratingsservice.NewPostgresRepo(db, appLogger)
	ratingsRepo = postgresRatingsRepo

	newClient, err := redis.NewClientWithPing(context.Background(), redisCfg.URL)
	if err != nil {
		return fmt.Errorf("connect redis: %w", err)
	}

	appLogger.Info("redis connected", nil)
	defer newClient.Close()

	sessions := authservice.NewRedisSessionStore(newClient, cfg.Server.SessionTTL)

	passwordService := authservice.BcryptPasswordService{}

	registerUC, err := authusecase.NewRegisterUserUseCase(userRepo, sessions, passwordService, appLogger)
	if err != nil {
		return fmt.Errorf("can't create register use case: %w", err)
	}

	loginUC, err := authusecase.NewLoginUserUseCase(userRepo, sessions, passwordService, appLogger)
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

	getUserUC, err := authusecase.NewGetUserUseCase(userRepo, appLogger)
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

	createFilmUC, err := filmsusecase.NewCreateFilmUseCase(filmsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create film use case: %w", err)
	}

	updateFilmUC, err := filmsusecase.NewUpdateFilmUseCase(filmsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create update film use case: %w", err)
	}

	deleteFilmUC, err := filmsusecase.NewDeleteFilmUseCase(filmsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create delete film use case: %w", err)
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
		CreateFilm:      createFilmUC,
		UpdateFilm:      updateFilmUC,
		DeleteFilm:      deleteFilmUC,
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
		return fmt.Errorf("create review use case: %w", err)
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
		CreatePerson: createPersonUC,
		GetPerson:    getPersonUC,
		ListPersons:  listPersonsUC,
		UpdatePerson: updatePersonUC,
		DeletePerson: deletePersonUC,
	}, appLogger)
	if err != nil {
		return fmt.Errorf("create persons handler: %w", err)
	}

	setFilmRatingUC, err := ratingsusecase.NewSetFilmRatingUseCase(ratingsRepo, filmsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create set film rating use case: %w", err)
	}

	getFilmRatingUC, err := ratingsusecase.NewGetFilmRatingUseCase(ratingsRepo, filmsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create get film rating use case: %w", err)
	}

	getCollectionRatingsUC, err := ratingsusecase.NewGetCollectionRatingsUseCase(postgresRatingsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create get collection ratings use case: %w", err)
	}

	deleteFilmRatingUC, err := ratingsusecase.NewDeleteFilmRatingUseCase(ratingsRepo, filmsRepo, appLogger)
	if err != nil {
		return fmt.Errorf("create delete film rating use case: %w", err)
	}

	ratingsHandler, err := ratingsdelivery.NewHandler(ratingsdelivery.HandlerUseCases{
		SetRating:               setFilmRatingUC,
		GetRating:               getFilmRatingUC,
		GetRatingsForCollection: getCollectionRatingsUC,
		DeleteRating:            deleteFilmRatingUC,
	}, appLogger)
	if err != nil {
		return fmt.Errorf("create ratings handler: %w", err)
	}

	httpHandler, err := router.NewRouter(
		authHandler,
		filmsHandler,
		reviewsHandler,
		personsHandler,
		filesHandler,
		ratingsHandler,
		cfg.Server.AllowedOrigins,
		appLogger,
	)
	if err != nil {
		return fmt.Errorf("create router: %w", err)
	}

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
