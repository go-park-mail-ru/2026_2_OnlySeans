package router

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/reviews"
	staticassets "github.com/go-park-mail-ru/2026_2_OnlySeans/static"
)

func NewRouter(authHandler *auth.Handler, filmsHandler *films.Handler, reviewsHandler *reviews.Handler, allowedOrigin string) http.Handler {
	r := mux.NewRouter()

	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static", http.FileServer(http.FS(staticassets.Files))),
	)

	r.HandleFunc("/api/register", authHandler.Register).Methods(http.MethodPost)
	r.HandleFunc("/api/login", authHandler.Login).Methods(http.MethodPost)
	r.HandleFunc("/api/logout", RequireAuth(authHandler, authHandler.Logout)).Methods(http.MethodPost)

	r.HandleFunc("/api/films", filmsHandler.ListFilms).Methods(http.MethodGet)
	r.HandleFunc("/api/films/{id}", filmsHandler.GetFilm).Methods(http.MethodGet)

	r.HandleFunc("/api/collections", filmsHandler.ListCollections).Methods(http.MethodGet)
	r.HandleFunc("/api/collections/{id}", filmsHandler.GetCollection).Methods(http.MethodGet)

	r.HandleFunc("/api/authorised", RequireAuth(authHandler, authHandler.Authorised)).Methods(http.MethodGet)

	r.HandleFunc("/api/films/{id}/reviews", reviewsHandler.ListByFilm).Methods(http.MethodGet)
	r.HandleFunc("/api/films/{id}/reviews", RequireAuth(authHandler, reviewsHandler.Create)).Methods(http.MethodPost)
	r.HandleFunc("/api/reviews/{id}", RequireAuth(authHandler, reviewsHandler.Update)).Methods(http.MethodPut)
	r.HandleFunc("/api/reviews/{id}", RequireAuth(authHandler, reviewsHandler.Delete)).Methods(http.MethodDelete)

	return withCORS(r, allowedOrigin)
}
