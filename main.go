package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"kinopoisk-auth/auth"
	"kinopoisk-auth/config"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "путь к файлу конфигурации")
	flag.Parse()

	if err := run(*configPath); err != nil {
		log.Fatal(err)
	}
}

func run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("конфигурация: %w", err)
	}

	// main.go только "собирает" модули вместе — сам не содержит
	// бизнес-логики.
	repo := auth.NewInMemoryUserRepo()
	// Session = nil — модуль сессий/токенов (куки) делает другой
	// человек в команде. Когда он реализует auth.SessionIssuer,
	// достаточно будет передать реализацию сюда вторым аргументом.
	useCase := auth.NewUseCase(repo, nil)
	handler := auth.NewHandler(useCase)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/register", handler.Register)
	mux.HandleFunc("/api/login", handler.Login)

	withCORSHandler := withCORS(mux)

	server := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      withCORSHandler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("сервер запущен на %s", server.Addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("сервер: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	log.Println("остановка сервера")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("остановка сервера: %w", err)
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
