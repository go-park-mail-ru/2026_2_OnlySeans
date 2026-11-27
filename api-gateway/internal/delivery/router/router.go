package router

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/user"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

type Handlers struct {
	Releases       releasesHandler
	Auth           *auth.AuthHandler
	Films          filmsHandler
	Reviews        reviewsHandler
	Persons        personsHandler
	Files          filesHandler
	Posters        postersHandler
	Ratings        ratingsHandler
	Library        libraryHandler
	Profile        *user.ProfileHandler
	PersonalRating personalRatingHandler
	PersonCatalog  personCatalogHandler
	Assets         assetsHandler
}

func NewRouter(handlers Handlers, authMiddleware *middleware.AuthMiddleware, allowedOrigins []string, appLogger logger.Logger) (http.Handler, error) {
	authHandler := handlers.Auth
	r := mux.NewRouter()
	authenticatedMiddlewares := []middleware.Middleware{authMiddleware.Authenticate}
	adminMiddlewares := []middleware.Middleware{authMiddleware.Authenticate, authMiddleware.RequireAdmin}
	catalogEditorMiddlewares := []middleware.Middleware{authMiddleware.Authenticate, authMiddleware.RequireCatalogEditor}

	r.HandleFunc("/api/register", authHandler.Register).Methods(http.MethodPost)
	r.HandleFunc("/api/login", authHandler.Login).Methods(http.MethodPost)
	r.HandleFunc("/api/refresh", authHandler.Refresh).Methods(http.MethodPost)
	r.HandleFunc("/api/logout", middleware.WrapMiddlewares(authHandler.Logout, authenticatedMiddlewares)).Methods(http.MethodPost)

	if filesHandler := handlers.Files; filesHandler != nil {
		r.HandleFunc("/api/me/avatar", middleware.WrapMiddlewares(filesHandler.GetMyAvatarURL, authenticatedMiddlewares)).Methods(http.MethodGet)
		r.HandleFunc("/api/me/avatar", middleware.WrapMiddlewares(filesHandler.UpdateMyAvatar, authenticatedMiddlewares)).Methods(http.MethodPut)
		r.HandleFunc("/api/me/avatar", middleware.WrapMiddlewares(filesHandler.DeleteMyAvatar, authenticatedMiddlewares)).Methods(http.MethodDelete)
		r.HandleFunc("/api/avatars", middleware.WrapMiddlewares(filesHandler.UploadAvatar, authenticatedMiddlewares)).Methods(http.MethodPost)
		r.HandleFunc("/api/avatars/{filename}", filesHandler.GetAvatarURL).Methods(http.MethodGet)
	}

	if postersHandler := handlers.Posters; postersHandler != nil {
		r.HandleFunc("/api/films/{id}/poster", middleware.WrapMiddlewares(postersHandler.UploadPoster, adminMiddlewares)).Methods(http.MethodPost)
		r.HandleFunc("/api/films/{id}/poster", postersHandler.GetPoster).Methods(http.MethodGet)
		r.HandleFunc("/api/films/{id}/poster", middleware.WrapMiddlewares(postersHandler.DeletePoster, adminMiddlewares)).Methods(http.MethodDelete)
	}

	if filmsHandler := handlers.Films; filmsHandler != nil {
		r.HandleFunc("/api/films", filmsHandler.ListFilms).Methods(http.MethodGet)
		r.HandleFunc("/api/films/{id}", filmsHandler.GetFilm).Methods(http.MethodGet)
		r.HandleFunc("/api/films", middleware.WrapMiddlewares(filmsHandler.CreateFilm, adminMiddlewares)).Methods(http.MethodPost)
		r.HandleFunc("/api/films/{id}", middleware.WrapMiddlewares(filmsHandler.UpdateFilm, adminMiddlewares)).Methods(http.MethodPut)
		r.HandleFunc("/api/films/{id}", middleware.WrapMiddlewares(filmsHandler.DeleteFilm, adminMiddlewares)).Methods(http.MethodDelete)

		r.HandleFunc("/api/collections", filmsHandler.ListCollections).Methods(http.MethodGet)
		r.HandleFunc("/api/collections/{id}", filmsHandler.GetCollection).Methods(http.MethodGet)
	}

	if h := handlers.Releases; h != nil {
		r.HandleFunc("/api/releases", h.List).Methods(http.MethodGet)
		r.HandleFunc("/api/releases/featured", h.Featured).Methods(http.MethodGet)
		r.HandleFunc("/api/films/{id}/releases", h.FilmReleases).Methods(http.MethodGet)
		r.HandleFunc("/api/films/{id}/releases", middleware.WrapMiddlewares(h.Update, catalogEditorMiddlewares)).Methods(http.MethodPatch)
		r.HandleFunc("/api/me/release-subscriptions", middleware.WrapMiddlewares(h.Subscriptions, authenticatedMiddlewares)).Methods(http.MethodGet)
		r.HandleFunc("/api/me/release-subscriptions/{id}", middleware.WrapMiddlewares(h.Subscribe, authenticatedMiddlewares)).Methods(http.MethodPut)
		r.HandleFunc("/api/me/release-subscriptions/{id}", middleware.WrapMiddlewares(h.Unsubscribe, authenticatedMiddlewares)).Methods(http.MethodDelete)
		r.HandleFunc("/api/me/notifications", middleware.WrapMiddlewares(h.Notifications, authenticatedMiddlewares)).Methods(http.MethodGet)
		r.HandleFunc("/api/me/notifications/{id}/read", middleware.WrapMiddlewares(h.ReadNotification, authenticatedMiddlewares)).Methods(http.MethodPut)
	}

	r.HandleFunc("/api/authorised", middleware.WrapMiddlewares(authHandler.Authorised, authenticatedMiddlewares)).Methods(http.MethodGet)

	if reviewsHandler := handlers.Reviews; reviewsHandler != nil {
		r.HandleFunc("/api/films/{id}/reviews", reviewsHandler.ListFilmReviews).Methods(http.MethodGet)
		r.HandleFunc("/api/films/{id}/reviews", middleware.WrapMiddlewares(reviewsHandler.CreateReview, authenticatedMiddlewares)).Methods(http.MethodPost)
		r.HandleFunc("/api/reviews/{id}", middleware.WrapMiddlewares(reviewsHandler.UpdateReview, authenticatedMiddlewares)).Methods(http.MethodPut)
		r.HandleFunc("/api/reviews/{id}", middleware.WrapMiddlewares(reviewsHandler.DeleteReview, authenticatedMiddlewares)).Methods(http.MethodDelete)
	}

	if ratingsHandler := handlers.Ratings; ratingsHandler != nil {
		r.HandleFunc("/api/films/{id}/rating", ratingsHandler.GetFilmRating).Methods(http.MethodGet)
		r.HandleFunc("/api/films/{id}/rating", middleware.WrapMiddlewares(ratingsHandler.SetFilmRating, authenticatedMiddlewares)).Methods(http.MethodPut)
		r.HandleFunc("/api/films/{id}/rating", middleware.WrapMiddlewares(ratingsHandler.DeleteFilmRating, authenticatedMiddlewares)).Methods(http.MethodDelete)
		r.HandleFunc("/api/collections/{id}/ratings", ratingsHandler.GetRatingsForCollection).Methods(http.MethodGet)
	}

	if personsHandler := handlers.Persons; personsHandler != nil {
		r.HandleFunc("/api/persons", personsHandler.ListPersons).Methods(http.MethodGet)
		r.HandleFunc("/api/persons/{id}", personsHandler.GetPerson).Methods(http.MethodGet)
		r.HandleFunc("/api/persons", middleware.WrapMiddlewares(personsHandler.CreatePerson, catalogEditorMiddlewares)).Methods(http.MethodPost)
		r.HandleFunc("/api/persons/{id}", middleware.WrapMiddlewares(personsHandler.UpdatePerson, catalogEditorMiddlewares)).Methods(http.MethodPut)
		r.HandleFunc("/api/persons/{id}", middleware.WrapMiddlewares(personsHandler.DeletePerson, catalogEditorMiddlewares)).Methods(http.MethodDelete)
	}

	if h := handlers.Library; h != nil {
		r.HandleFunc("/api/me/library", middleware.WrapMiddlewares(h.Get, authenticatedMiddlewares)).Methods(http.MethodGet)
		r.HandleFunc("/api/me/library/lists/{list}/{film}", middleware.WrapMiddlewares(h.SetSaved, authenticatedMiddlewares)).Methods(http.MethodPut)
		r.HandleFunc("/api/me/preferences", middleware.WrapMiddlewares(h.Preferences, authenticatedMiddlewares)).Methods(http.MethodGet)
		r.HandleFunc("/api/me/preferences", middleware.WrapMiddlewares(h.SavePreference, authenticatedMiddlewares)).Methods(http.MethodPut)
	}

	if h := handlers.Profile; h != nil {
		r.HandleFunc("/api/me/profile", middleware.WrapMiddlewares(h.Get, authenticatedMiddlewares)).Methods(http.MethodGet)
		r.HandleFunc("/api/me/profile", middleware.WrapMiddlewares(h.Update, authenticatedMiddlewares)).Methods(http.MethodPut)
		r.HandleFunc("/api/me", middleware.WrapMiddlewares(h.Update, authenticatedMiddlewares)).Methods(http.MethodPut)
	}

	if h := handlers.PersonalRating; h != nil {
		r.HandleFunc("/api/films/{id}/rating/me", middleware.WrapMiddlewares(h.GetPersonalRating, authenticatedMiddlewares)).Methods(http.MethodGet)
	}

	if h := handlers.PersonCatalog; h != nil {
		r.HandleFunc("/api/films/{id}/people", h.GetFilmPersons).Methods(http.MethodGet)
		r.HandleFunc("/api/people/{id}", h.GetPersonCatalog).Methods(http.MethodGet)
	}

	if h := handlers.Assets; h != nil {
		r.HandleFunc("/static/{key:.*}", h.GetAsset).Methods(http.MethodGet)
	}

	handler, err := middleware.WithCORSAndCSRF(authMiddleware.OptionalAuthenticate(newOpenAPIHandler(r)), allowedOrigins)
	if err != nil {
		return nil, err
	}

	handler = middleware.WithSecurityHeaders(handler)
	handler = middleware.WithRecover(handler, appLogger)
	handler = middleware.WithLogging(handler, appLogger)

	return middleware.WithRequestID(handler), nil
}
