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
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/router"
)

const sessionTTL = 24 * time.Hour

func Run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var (
		users        auth.UserRepository
		filmsService films.Service
		reviewsRepo  reviews.ReviewRepository
	)

	if cfg.Database.URL != "" {
		//база
		db, err := storage.Open(context.Background(), cfg.Database)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer db.Close()
		users = auth.NewPostgresUserRepo(db)
		filmsService = films.NewPostgresService(db)
		reviewsRepo = reviews.NewPostgresReviewRepo(db)
		log.Println("storage: postgres")
	} else {
		//память
		users = auth.NewInMemoryUserRepo()
		filmsService, err = films.NewSeededService()
		if err != nil {
			return fmt.Errorf("seed films: %w", err)
		}
		reviewsRepo = reviews.NewInMemoryReviewRepo()
		log.Println("storage: in=memory")
	}

	sessions := auth.NewInMemorySessionStore(sessionTTL)

	handler, err := auth.NewHandler(users, sessions)
	if err != nil {
		return fmt.Errorf("create handler: %w", err)
	}

	reviewsUsecase, err := reviews.NewUsecase(reviewsRepo, filmsService)
	if err != nil {
		return fmt.Errorf("create reviews usecase: %w", err)
	}
	reviewsHandler, err := reviews.NewHandler(reviewsUsecase)
	if err != nil {
		return fmt.Errorf("create reviews handler: %w", err)
	}

	httpHandler := router.NewRouter(handler, films.NewHandler(filmsService), reviewsHandler, cfg.Server.AllowedOrigin)

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
