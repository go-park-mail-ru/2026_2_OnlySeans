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

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/films"
)

func Run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	repo := auth.NewInMemoryUserRepo()
	// Session = nil — модуль сессий/токенов (куки) делает другой
	// человек в команде. Когда он реализует auth.SessionIssuer,
	// достаточно будет передать реализацию сюда вторым аргументом.
	useCase := auth.NewUseCase(repo, nil)
	handler := auth.NewHandler(useCase)

	filmsService, err := films.NewSeededService()
	if err != nil {
		return fmt.Errorf("seed films: %w", err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/api/register", handler.Register)
	router.HandleFunc("/api/login", handler.Login)
	films.NewHandler(filmsService).RegisterRoutes(router)

	server := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      withCORS(router),
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

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
