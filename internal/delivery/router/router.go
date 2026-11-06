package router

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/reviews"
	staticassets "github.com/go-park-mail-ru/2026_2_OnlySeans/static"
)

func NewRouter(authHandler *auth.Handler, filmsHandler *films.Handler, reviewsHandler *reviews.Handler, allowedOrigins []string) (http.Handler, error) {
	r := mux.NewRouter()
	authenticatedMiddlewares := []Middleware{RequireAuth(authHandler)}
	adminMiddlewares := []Middleware{RequireAuth(authHandler), authHandler.RequireAdmin}

	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static", http.FileServer(http.FS(staticassets.Files))),
	)

	r.HandleFunc("/api/register", authHandler.Register).Methods(http.MethodPost)
	r.HandleFunc("/api/login", authHandler.Login).Methods(http.MethodPost)
	r.HandleFunc("/api/logout", WrapMiddlewares(authHandler.Logout, authenticatedMiddlewares)).Methods(http.MethodPost)

	r.HandleFunc("/api/films", filmsHandler.ListFilms).Methods(http.MethodGet)
	r.HandleFunc("/api/films/{id}", filmsHandler.GetFilm).Methods(http.MethodGet)

	r.HandleFunc("/api/films", WrapMiddlewares(filmsHandler.CreateFilm, adminMiddlewares)).Methods(http.MethodPost)

	r.HandleFunc("/api/films/{id}", WrapMiddlewares(filmsHandler.UpdateFilm, adminMiddlewares)).Methods(http.MethodPut)

	r.HandleFunc("/api/films/{id}", WrapMiddlewares(filmsHandler.DeleteFilm, adminMiddlewares)).Methods(http.MethodDelete)

	r.HandleFunc("/api/collections", filmsHandler.ListCollections).Methods(http.MethodGet)
	r.HandleFunc("/api/collections/{id}", filmsHandler.GetCollection).Methods(http.MethodGet)

	r.HandleFunc("/api/authorised", WrapMiddlewares(authHandler.Authorised, authenticatedMiddlewares)).Methods(http.MethodGet)

	r.HandleFunc("/api/films/{id}/reviews", reviewsHandler.ListByFilm).Methods(http.MethodGet)
	r.HandleFunc("/api/films/{id}/reviews", WrapMiddlewares(reviewsHandler.Create, authenticatedMiddlewares)).Methods(http.MethodPost)
	r.HandleFunc("/api/reviews/{id}", WrapMiddlewares(reviewsHandler.Update, authenticatedMiddlewares)).Methods(http.MethodPut)
	r.HandleFunc("/api/reviews/{id}", WrapMiddlewares(reviewsHandler.Delete, authenticatedMiddlewares)).Methods(http.MethodDelete)

	csrf, err := newCSRFMiddleware(allowedOrigins)
	if err != nil {
		return nil, err
	}

	protected := csrf.withCSRF(r)

	return withSecurityHeaders(withCORS(protected, allowedOrigins)), nil
}
