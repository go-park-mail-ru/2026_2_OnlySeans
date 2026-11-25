package main

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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authhttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/auth"
	fileshttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/files"
	filmshttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/films"
	libraryhttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/library"
	personshttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/persons"
	ratingshttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/ratings"
	userhttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/user"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/authclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/catalogclient"
	personsclient "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/personscleint"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/userclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/infrastructure/config"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	pbauth "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/auth"
	pbcatalog "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
	pbpersons "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/persons"
	pbuser "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	cfg := config.Load()
	if cfg.HTTP.Addr == "" || cfg.AuthServiceAddr == "" || cfg.UserServiceAddr == "" || cfg.PersonsServiceAddr == "" || cfg.CatalogServiceAddr == "" {
		return errors.New("http.addr, auth_service_addr, user_service_addr, persons_service_addr and catalog_service_addr are required")
	}

	appLogger, err := logger.NewLogger(os.Stdout, logger.LogLevelType(cfg.Logger.Level), "json")
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}

	transport := grpc.WithTransportCredentials(insecure.NewCredentials())

	authConn, err := grpc.NewClient(cfg.AuthServiceAddr, transport)
	if err != nil {
		return fmt.Errorf("create Auth gRPC client: %w", err)
	}

	defer func() { _ = authConn.Close() }()

	userConn, err := grpc.NewClient(cfg.UserServiceAddr, transport)
	if err != nil {
		return fmt.Errorf("create User gRPC client: %w", err)
	}

	defer func() { _ = userConn.Close() }()

	personsConn, err := grpc.NewClient(cfg.PersonsServiceAddr, transport)
	if err != nil {
		return fmt.Errorf("create Persons gRPC client: %w", err)
	}

	defer func() { _ = personsConn.Close() }()

	if cfg.HTTP.MaxPosterSize <= 0 || cfg.HTTP.MaxPosterSize > int64(int(^uint(0)>>1))-(1<<20) {
		return errors.New("http.max_poster_size must be positive and leave room for message overhead")
	}

	catalogConn, err := grpc.NewClient(cfg.CatalogServiceAddr, transport,
		grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(int(cfg.HTTP.MaxPosterSize)+(1<<20))),
	)
	if err != nil {
		return fmt.Errorf("create Catalog gRPC client: %w", err)
	}

	defer func() { _ = catalogConn.Close() }()

	catalogClient := catalogclient.NewCatalogClient(pbcatalog.NewCatalogServiceClient(catalogConn))

	filmsHandler, err := filmshttp.NewHandler(catalogClient, appLogger)
	if err != nil {
		return fmt.Errorf("create films handler: %w", err)
	}

	ratingsHandler, err := ratingshttp.NewHandler(catalogClient, appLogger)
	if err != nil {
		return fmt.Errorf("create ratings handler: %w", err)
	}

	libraryHandler := libraryhttp.NewHandler(catalogClient, appLogger)
	personalRatingHandler := ratingshttp.NewMyRatingHandler(catalogClient, appLogger)

	postersHandler, err := fileshttp.NewPosterHandler(catalogClient, cfg.HTTP.MaxPosterSize, appLogger)
	if err != nil {
		return fmt.Errorf("create posters handler: %w", err)
	}

	authClient := authclient.NewAuthClient(pbauth.NewAuthServiceClient(authConn))
	userClient := userclient.NewUserClient(pbuser.NewUserServiceClient(userConn))
	personClient := personsclient.NewPersonClient(pbpersons.NewPersonServiceClient(personsConn))

	personsHandler, err := personshttp.NewHandler(personClient, appLogger)
	if err != nil {
		return fmt.Errorf("create Persons handler: %w", err)
	}

	personCatalogHandler, err := personshttp.NewCatalogHandler(personClient, filmsHandler, appLogger)
	if err != nil {
		return fmt.Errorf("create Person catalog handler: %w", err)
	}

	authHandler := authhttp.NewAuthHandler(authClient, userClient, appLogger)
	authHandler.CookieSecure = cfg.HTTP.CookieSecure
	profileHandler := userhttp.NewProfileHandler(userClient, appLogger)
	avatarHandler := userhttp.NewAvatarHandler(userClient, appLogger)
	authMiddleware := middleware.NewAuthMiddleware(authClient, userClient, appLogger)

	handler, err := router.NewRouter(router.Handlers{
		Auth:           authHandler,
		Files:          avatarHandler,
		Profile:        profileHandler,
		Persons:        personsHandler,
		PersonCatalog:  personCatalogHandler,
		Films:          filmsHandler,
		Ratings:        ratingsHandler,
		PersonalRating: personalRatingHandler,
		Library:        libraryHandler,
		Posters:        postersHandler,
	}, authMiddleware, cfg.HTTP.AllowedOrigins, appLogger)
	if err != nil {
		return fmt.Errorf("create Gateway router: %w", err)
	}

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		appLogger.Info("Gateway HTTP server starting", map[string]any{"address": server.Addr})
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve Gateway HTTP: %w", err)
		}

		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		return server.Shutdown(shutdownCtx)
	}
}
