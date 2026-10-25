package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	filmsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/films"
	reviewsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	authservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/auth"
	filmsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/films"
	reviewsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	authusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	filmsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/films"
	reviewsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/reviews"
)

func Run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var (
		useRepo     domain.UserRepository
		filmsRepo   filmsservice.Service
		reviewsRepo domain.ReviewRepository
	)

	if cfg.Database.URL != "" {
		db, err := storage.Open(context.Background(), cfg.Database)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer db.Close()

		useRepo = authservice.NewPostgresUserRepo(db)
		filmsRepo = filmsservice.NewPostgresService(db)
		reviewsRepo = reviewsservice.NewPostgresReviewRepo(db)
		log.Println("storage: postgres")
	} else {
		useRepo = authservice.NewInMemoryUserRepo()
		filmsRepo, err = filmsservice.NewSeededService()
		if err != nil {
			return fmt.Errorf("seed films: %w", err)
		}
		reviewsRepo = reviewsservice.NewInMemoryReviewRepo()
		log.Println("storage: in-memory")
	}

	sessions := authservice.NewInMemorySessionStore(cfg.Server.SessionTTL)

	passwordService := authservice.BcryptPasswordService{}

	registerUC, err := authusecase.NewRegisterUserUseCase(useRepo, sessions, passwordService)
	if err != nil {
		return fmt.Errorf("can't create register use case: %w", err)
	}

	loginUC, err := authusecase.NewLoginUserUseCase(useRepo, sessions, passwordService)
	if err != nil {
		return fmt.Errorf("can't create login use case: %w", err)
	}

	logoutUC, err := authusecase.NewLogoutUserUseCase(sessions)
	if err != nil {
		return fmt.Errorf("can't create logout use case: %w", err)
	}

	authenticateUC, err := authusecase.NewAuthenticateUserUseCase(sessions)
	if err != nil {
		return fmt.Errorf("can't create authenticate use case: %w", err)
	}

	getUserUC, err := authusecase.NewGetUserUseCase(useRepo)
	if err != nil {
		return fmt.Errorf("can't create get user use case: %w", err)
	}

	listFilmsUC, err := filmsusecase.NewListFilmsUseCase(filmsRepo)
	if err != nil {
		return fmt.Errorf("create list films use case: %w", err)
	}

	getFilmUC, err := filmsusecase.NewGetFilmUseCase(filmsRepo)
	if err != nil {
		return fmt.Errorf("create get film use case: %w", err)
	}

	listCollectionsUC, err := filmsusecase.NewListCollectionsUseCase(filmsRepo)
	if err != nil {
		return fmt.Errorf("create list collections use case: %w", err)
	}

	getCollectionUC, err := filmsusecase.NewGetCollectionUseCase(filmsRepo)
	if err != nil {
		return fmt.Errorf("create get collection use case: %w", err)
	}

	authHandler, err := authdelivery.NewHandler(authdelivery.HandlerUseCases{
		Register:     registerUC,
		Login:        loginUC,
		Logout:       logoutUC,
		Authenticate: authenticateUC,
		GetUser:      getUserUC,
	})
	if err != nil {
		return fmt.Errorf("create handler: %w", err)
	}

	filmsHandler, err := filmsdelivery.NewHandler(filmsdelivery.HandlerUseCases{
		ListFilms:       listFilmsUC,
		GetFilm:         getFilmUC,
		ListCollections: listCollectionsUC,
		GetCollection:   getCollectionUC,
	})
	if err != nil {
		return fmt.Errorf("create films handler: %w", err)
	}

	listReviewsUC, err := reviewsusecase.NewListReviewsUseCase(reviewsRepo, filmsRepo)
	if err != nil {
		return fmt.Errorf("create list reviews use case: %w", err)
	}

	createReviewUC, err := reviewsusecase.NewCreateReviewUseCase(reviewsRepo, filmsRepo)
	if err != nil {
		return fmt.Errorf("create create review use case: %w", err)
	}

	updateReviewUC, err := reviewsusecase.NewUpdateReviewUseCase(reviewsRepo)
	if err != nil {
		return fmt.Errorf("create update review use case: %w", err)
	}

	deleteReviewUC, err := reviewsusecase.NewDeleteReviewUseCase(reviewsRepo)
	if err != nil {
		return fmt.Errorf("create delete review use case: %w", err)
	}

	reviewsHandler, err := reviewsdelivery.NewHandler(reviewsdelivery.HandlerUseCases{
		List:   listReviewsUC,
		Create: createReviewUC,
		Update: updateReviewUC,
		Delete: deleteReviewUC,
	})
	if err != nil {
		return fmt.Errorf("create reviews handler: %w", err)
	}

	httpHandler := router.NewRouter(authHandler, filmsHandler, reviewsHandler, cfg.Server.AllowedOrigin)

	server := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      httpHandler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	return serve(server, cfg.Server.ShutdownTimeout)
}

func serve(server *http.Server, shutdownTimeout time.Duration) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("server started on %s", server.Addr)
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

	log.Println("shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	return nil
}
