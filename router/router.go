package router

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/films"
)

func NewRouter(authHandler *auth.Handler, filmsHandler *films.Handler, allowedOrigin string) http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/api/register", authHandler.Register).Methods(http.MethodPost)
	r.HandleFunc("/api/login", authHandler.Login).Methods(http.MethodPost)
	r.HandleFunc("/api/logout", RequireAuth(authHandler, authHandler.Logout)).Methods(http.MethodPost)

	r.HandleFunc("/api/films", filmsHandler.ListFilms).Methods(http.MethodGet)
	r.HandleFunc("/api/films/{id}", filmsHandler.GetFilm).Methods(http.MethodGet)

	r.HandleFunc("/api/collections", filmsHandler.ListCollections).Methods(http.MethodGet)
	r.HandleFunc("/api/collections/{id}", filmsHandler.GetCollection).Methods(http.MethodGet)

	r.HandleFunc("/api/authorised", RequireAuth(authHandler, authHandler.Authorised)).Methods(http.MethodGet)

	return withCORS(r, allowedOrigin)
}
