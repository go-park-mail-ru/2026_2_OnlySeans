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
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/router"
	authservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/auth"
	filmsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/films"
	authusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	filmsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/films"
)

func Run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	repo := authservice.NewInMemoryUserRepo()
	sessions := authservice.NewInMemorySessionStore(cfg.Server.SessionTTL)

	passwordService := authservice.BcryptPasswordService{}

	filmsRepo, err := filmsservice.NewSeededService()
	if err != nil {
		return fmt.Errorf("seed films: %w", err)
	}

	registerUC, err := authusecase.NewRegisterUserUseCase(repo, sessions, passwordService)
	if err != nil {
		return fmt.Errorf("can't create register use case: %w", err)
	}

	loginUC, err := authusecase.NewLoginUserUseCase(repo, sessions, passwordService)
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

	getUserUC, err := authusecase.NewGetUserUseCase(repo)
	if err != nil {
		return fmt.Errorf("can't create get user use case: %w", err)
	}

	listFilms, err := filmsusecase.NewListFilmsUseCase(filmsRepo)
	if err != nil {
		return fmt.Errorf("create list films use case: %w", err)
	}

	getFilm, err := filmsusecase.NewGetFilmUseCase(filmsRepo)
	if err != nil {
		return fmt.Errorf("create get film use case: %w", err)
	}

	listCollections, err := filmsusecase.NewListCollectionsUseCase(filmsRepo)
	if err != nil {
		return fmt.Errorf("create list collections use case: %w", err)
	}

	getCollection, err := filmsusecase.NewGetCollectionUseCase(filmsRepo)
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
		ListFilms:       listFilms,
		GetFilm:         getFilm,
		ListCollections: listCollections,
		GetCollection:   getCollection,
	})
	if err != nil {
		return fmt.Errorf("create films handler: %w", err)
	}

	httpHandler := router.NewRouter(authHandler, filmsHandler, cfg.Server.AllowedOrigin)

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
