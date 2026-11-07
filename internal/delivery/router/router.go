package router

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/persons"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/profile"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/ratings"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

func NewRouter(
	authHandler *auth.Handler,
	filmsHandler *films.Handler,
	reviewsHandler *reviews.Handler,
	personsHandler *persons.Handler,
	filesHandler *files.Handler,
	ratingsHandler *ratings.Handler,
	allowedOrigins []string,
	appLogger logger.Logger,
	profileHandlers ...*profile.Handler,
) (http.Handler, error) {
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

	r.HandleFunc("/api/films", filmsHandler.ListFilms).Methods(http.MethodGet)
	r.HandleFunc("/api/films/{id}", filmsHandler.GetFilm).Methods(http.MethodGet)
	r.HandleFunc("/api/films", WrapMiddlewares(filmsHandler.CreateFilm, adminMiddlewares)).Methods(http.MethodPost)
	r.HandleFunc("/api/films/{id}", WrapMiddlewares(filmsHandler.UpdateFilm, adminMiddlewares)).Methods(http.MethodPut)
	r.HandleFunc("/api/films/{id}", WrapMiddlewares(filmsHandler.DeleteFilm, adminMiddlewares)).Methods(http.MethodDelete)

	r.HandleFunc("/api/collections", filmsHandler.ListCollections).Methods(http.MethodGet)
	r.HandleFunc("/api/collections/{id}", filmsHandler.GetCollection).Methods(http.MethodGet)

	r.HandleFunc("/api/authorised", WrapMiddlewares(authHandler.Authorised, authenticatedMiddlewares)).Methods(http.MethodGet)
	if len(profileHandlers) > 0 && profileHandlers[0] != nil {
		profileHandler := profileHandlers[0]
		r.HandleFunc("/api/me/profile", WrapMiddlewares(profileHandler.Overview, authenticatedMiddlewares)).Methods(http.MethodGet)
		r.HandleFunc("/api/me/profile", WrapMiddlewares(profileHandler.UpdateAccount, authenticatedMiddlewares)).Methods(http.MethodPut)
		r.HandleFunc("/api/me/password", WrapMiddlewares(profileHandler.ChangePassword, authenticatedMiddlewares)).Methods(http.MethodPut)
		r.HandleFunc("/api/me/favorites/{id}", WrapMiddlewares(profileHandler.SetFavorite, authenticatedMiddlewares)).Methods(http.MethodPut, http.MethodDelete)
	}

	r.HandleFunc("/api/films/{id}/reviews", reviewsHandler.ListByFilm).Methods(http.MethodGet)
	r.HandleFunc("/api/films/{id}/reviews", WrapMiddlewares(reviewsHandler.Create, authenticatedMiddlewares)).Methods(http.MethodPost)
	r.HandleFunc("/api/reviews/{id}", WrapMiddlewares(reviewsHandler.Update, authenticatedMiddlewares)).Methods(http.MethodPut)
	r.HandleFunc("/api/reviews/{id}", WrapMiddlewares(reviewsHandler.Delete, authenticatedMiddlewares)).Methods(http.MethodDelete)

	r.HandleFunc("/api/films/{id}/rating", ratingsHandler.GetFilmRating).Methods(http.MethodGet)
	r.HandleFunc("/api/films/{id}/rating", WrapMiddlewares(ratingsHandler.SetFilmRating, authenticatedMiddlewares)).Methods(http.MethodPut)
	r.HandleFunc("/api/films/{id}/rating", WrapMiddlewares(ratingsHandler.DeleteFilmRating, authenticatedMiddlewares)).Methods(http.MethodDelete)
	r.HandleFunc("/api/collections/{id}/ratings", ratingsHandler.GetRatingsForCollection).Methods(http.MethodGet)

	r.HandleFunc("/api/persons", personsHandler.ListPersons).Methods(http.MethodGet)
	r.HandleFunc("/api/persons/{id}", personsHandler.GetPerson).Methods(http.MethodGet)
	r.HandleFunc("/api/persons", WrapMiddlewares(personsHandler.CreatePerson, catalogEditorMiddlewares)).Methods(http.MethodPost)
	r.HandleFunc("/api/persons/{id}", WrapMiddlewares(personsHandler.UpdatePerson, catalogEditorMiddlewares)).Methods(http.MethodPut)
	r.HandleFunc("/api/persons/{id}", WrapMiddlewares(personsHandler.DeletePerson, catalogEditorMiddlewares)).Methods(http.MethodDelete)

	csrf, err := newCSRFMiddleware(allowedOrigins)
	if err != nil {
		return nil, err
	}

	protected := csrf.withCSRF(r)
	return withLogging(withRecover(withSecurityHeaders(withCORS(protected, allowedOrigins)), appLogger), appLogger), nil
}
