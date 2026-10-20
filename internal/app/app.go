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

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/films"
	filmsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/films/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/router"
)

const sessionTTL = 24 * time.Hour

func Run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	repo := auth.NewInMemoryUserRepo()
	sessions := auth.NewInMemorySessionStore(sessionTTL)

	passwordService := auth.BcryptPasswordService{}
	register, err := usecase.NewRegisterUserUseCase(repo, sessions, passwordService)
	if err != nil {
		return fmt.Errorf("create register use case: %w", err)
	}
	login, err := usecase.NewLoginUserUseCase(repo, sessions, passwordService)
	if err != nil {
		return fmt.Errorf("create login use case: %w", err)
	}
	logout, err := usecase.NewLogoutUserUseCase(sessions)
	if err != nil {
		return fmt.Errorf("create logout use case: %w", err)
	}
	authenticate, err := usecase.NewAuthenticateUserUseCase(sessions)
	if err != nil {
		return fmt.Errorf("create authenticate use case: %w", err)
	}
	getUser, err := usecase.NewGetUserUseCase(repo)
	if err != nil {
		return fmt.Errorf("create get user use case: %w", err)
	}

	handler, err := auth.NewHandler(auth.HandlerUseCases{
		Register:     register,
		Login:        login,
		Logout:       logout,
		Authenticate: authenticate,
		GetUser:      getUser,
	})
	if err != nil {
		return fmt.Errorf("create handler: %w", err)
	}

	filmsService, err := films.NewSeededService()
	if err != nil {
		return fmt.Errorf("seed films: %w", err)
	}
	listFilms, err := filmsusecase.NewListFilmsUseCase(filmsService)
	if err != nil {
		return fmt.Errorf("create list films use case: %w", err)
	}
	getFilm, err := filmsusecase.NewGetFilmUseCase(filmsService)
	if err != nil {
		return fmt.Errorf("create get film use case: %w", err)
	}
	listCollections, err := filmsusecase.NewListCollectionsUseCase(filmsService)
	if err != nil {
		return fmt.Errorf("create list collections use case: %w", err)
	}
	getCollection, err := filmsusecase.NewGetCollectionUseCase(filmsService)
	if err != nil {
		return fmt.Errorf("create get collection use case: %w", err)
	}
	filmsHandler, err := films.NewHandler(films.HandlerUseCases{
		ListFilms:       listFilms,
		GetFilm:         getFilm,
		ListCollections: listCollections,
		GetCollection:   getCollection,
	})
	if err != nil {
		return fmt.Errorf("create films handler: %w", err)
	}

	httpHandler := router.NewRouter(handler, filmsHandler, cfg.Server.AllowedOrigin)

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
