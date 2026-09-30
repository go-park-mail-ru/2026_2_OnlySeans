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

	useCase, err := auth.NewUseCase(repo, sessions)
	if err != nil {
		return fmt.Errorf("create usecase: %w", err)
	}
	handler, err := auth.NewHandler(useCase)
	if err != nil {
		return fmt.Errorf("create handler: %w", err)
	}

	filmsService, err := films.NewSeededService()
	if err != nil {
		return fmt.Errorf("seed films: %w", err)
	}

	httpHandler := router.NewRouter(handler, films.NewHandler(filmsService), cfg.Server.AllowedOrigin)

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
