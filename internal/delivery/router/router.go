package router

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/assets"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/library"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/persons"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/ratings"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type Handlers struct {
	Auth           *auth.Handler
	Films          *films.Handler
	Reviews        *reviews.Handler
	Persons        *persons.Handler
	Files          *files.Handler
	Posters        *files.PosterHandler
	Ratings        *ratings.Handler
	Library        *library.Handler
	Profile        *auth.ProfileHandler
	PersonalRating *ratings.MyRatingHandler
	PersonCatalog  *persons.CatalogHandler
	Assets         *assets.Handler
}

func NewRouter(handlers Handlers, allowedOrigins []string, appLogger logger.Logger) (http.Handler, error) {
	authHandler := handlers.Auth
	filmsHandler := handlers.Films
	reviewsHandler := handlers.Reviews
	personsHandler := handlers.Persons
	filesHandler := handlers.Files
	postersHandler := handlers.Posters
	ratingsHandler := handlers.Ratings
	r := mux.NewRouter()
	authenticatedMiddlewares := []Middleware{RequireAuth(authHandler)}
	adminMiddlewares := []Middleware{RequireAuth(authHandler), authHandler.RequireAdmin}
	catalogEditorMiddlewares := []Middleware{RequireAuth(authHandler), RequireCatalogEditor(authHandler, appLogger)}

	r.HandleFunc("/api/register", authHandler.Register).Methods(http.MethodPost)
	r.HandleFunc("/api/login", authHandler.Login).Methods(http.MethodPost)
	r.HandleFunc("/api/logout", WrapMiddlewares(authHandler.Logout, authenticatedMiddlewares)).Methods(http.MethodPost)
	r.HandleFunc("/api/me/avatar", WrapMiddlewares(filesHandler.GetMyAvatarURL, authenticatedMiddlewares)).Methods(http.MethodGet)
	r.HandleFunc("/api/me/avatar", WrapMiddlewares(filesHandler.UpdateMyAvatar, authenticatedMiddlewares)).Methods(http.MethodPut)
	r.HandleFunc("/api/me/avatar", WrapMiddlewares(filesHandler.DeleteMyAvatar, authenticatedMiddlewares)).Methods(http.MethodDelete)
	r.HandleFunc("/api/avatars", WrapMiddlewares(filesHandler.UploadAvatar, authenticatedMiddlewares)).Methods(http.MethodPost)
	r.HandleFunc("/api/avatars/{filename}", filesHandler.GetAvatarURL).Methods(http.MethodGet)

	r.HandleFunc("/api/films/{id}/poster", WrapMiddlewares(postersHandler.UploadPoster, adminMiddlewares)).Methods(http.MethodPost)
	r.HandleFunc("/api/films/{id}/poster", postersHandler.GetPoster).Methods(http.MethodGet)
	r.HandleFunc("/api/films/{id}/poster", WrapMiddlewares(postersHandler.DeletePoster, adminMiddlewares)).Methods(http.MethodDelete)

	r.HandleFunc("/api/films", filmsHandler.ListFilms).Methods(http.MethodGet)
	r.HandleFunc("/api/films/{id}", filmsHandler.GetFilm).Methods(http.MethodGet)
	r.HandleFunc("/api/films", WrapMiddlewares(filmsHandler.CreateFilm, adminMiddlewares)).Methods(http.MethodPost)
	r.HandleFunc("/api/films/{id}", WrapMiddlewares(filmsHandler.UpdateFilm, adminMiddlewares)).Methods(http.MethodPut)
	r.HandleFunc("/api/films/{id}", WrapMiddlewares(filmsHandler.DeleteFilm, adminMiddlewares)).Methods(http.MethodDelete)

	r.HandleFunc("/api/collections", filmsHandler.ListCollections).Methods(http.MethodGet)
	r.HandleFunc("/api/collections/{id}", filmsHandler.GetCollection).Methods(http.MethodGet)

	r.HandleFunc("/api/authorised", WrapMiddlewares(authHandler.Authorised, authenticatedMiddlewares)).Methods(http.MethodGet)

	r.HandleFunc("/api/films/{id}/reviews", reviewsHandler.ListFilmReviews).Methods(http.MethodGet)
	r.HandleFunc("/api/films/{id}/reviews", WrapMiddlewares(reviewsHandler.CreateReview, authenticatedMiddlewares)).Methods(http.MethodPost)
	r.HandleFunc("/api/reviews/{id}", WrapMiddlewares(reviewsHandler.UpdateReview, authenticatedMiddlewares)).Methods(http.MethodPut)
	r.HandleFunc("/api/reviews/{id}", WrapMiddlewares(reviewsHandler.DeleteReview, authenticatedMiddlewares)).Methods(http.MethodDelete)

	r.HandleFunc("/api/films/{id}/rating", ratingsHandler.GetFilmRating).Methods(http.MethodGet)
	r.HandleFunc("/api/films/{id}/rating", WrapMiddlewares(ratingsHandler.SetFilmRating, authenticatedMiddlewares)).Methods(http.MethodPut)
	r.HandleFunc("/api/films/{id}/rating", WrapMiddlewares(ratingsHandler.DeleteFilmRating, authenticatedMiddlewares)).Methods(http.MethodDelete)
	r.HandleFunc("/api/collections/{id}/ratings", ratingsHandler.GetRatingsForCollection).Methods(http.MethodGet)

	r.HandleFunc("/api/persons", personsHandler.ListPersons).Methods(http.MethodGet)
	r.HandleFunc("/api/persons/{id}", personsHandler.GetPerson).Methods(http.MethodGet)
	r.HandleFunc("/api/persons", WrapMiddlewares(personsHandler.CreatePerson, catalogEditorMiddlewares)).Methods(http.MethodPost)
	r.HandleFunc("/api/persons/{id}", WrapMiddlewares(personsHandler.UpdatePerson, catalogEditorMiddlewares)).Methods(http.MethodPut)
	r.HandleFunc("/api/persons/{id}", WrapMiddlewares(personsHandler.DeletePerson, catalogEditorMiddlewares)).Methods(http.MethodDelete)

	if h := handlers.Library; h != nil {
		r.HandleFunc("/api/me/library", WrapMiddlewares(h.Get, authenticatedMiddlewares)).Methods(http.MethodGet)
		r.HandleFunc("/api/me/library/lists/{list}/{film}", WrapMiddlewares(h.SetSaved, authenticatedMiddlewares)).Methods(http.MethodPut)
		r.HandleFunc("/api/me/preferences", WrapMiddlewares(h.Preferences, authenticatedMiddlewares)).Methods(http.MethodGet)
		r.HandleFunc("/api/me/preferences", WrapMiddlewares(h.SavePreference, authenticatedMiddlewares)).Methods(http.MethodPut)
	}
	if h := handlers.Profile; h != nil {
		r.HandleFunc("/api/me/profile", WrapMiddlewares(h.Get, authenticatedMiddlewares)).Methods(http.MethodGet)
		r.HandleFunc("/api/me/profile", WrapMiddlewares(h.Update, authenticatedMiddlewares)).Methods(http.MethodPut)
		r.HandleFunc("/api/me", WrapMiddlewares(h.Update, authenticatedMiddlewares)).Methods(http.MethodPut)
	}
	if h := handlers.PersonalRating; h != nil {
		r.HandleFunc("/api/films/{id}/rating/me", WrapMiddlewares(h.GetPersonalRating, authenticatedMiddlewares)).Methods(http.MethodGet)
	}
	if h := handlers.PersonCatalog; h != nil {
		r.HandleFunc("/api/films/{id}/people", h.GetFilmPersons).Methods(http.MethodGet)
		r.HandleFunc("/api/people/{id}", h.GetPersonCatalog).Methods(http.MethodGet)
	}
	if h := handlers.Assets; h != nil {
		r.HandleFunc("/static/{key:.*}", h.GetAsset).Methods(http.MethodGet)
	}

	csrf, err := newCSRFMiddleware(allowedOrigins)
	if err != nil {
		return nil, err
	}

	protected := csrf.withCSRF(r)
	return WithRequestID(WithLogging(WithRecover(withSecurityHeaders(withCORS(protected, allowedOrigins)), appLogger), appLogger)), nil
}
