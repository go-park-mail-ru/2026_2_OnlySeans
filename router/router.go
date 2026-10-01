package router

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/films"
)

func NewRouter(authHandler *auth.Handler, filmsHandler *films.Handler, allowedOrigin string) http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/api/register", authHandler.Register)
	r.HandleFunc("/api/login", authHandler.Login)
	r.HandleFunc("/api/logout", authHandler.Logout)

	filmsHandler.RegisterRoutes(r)

	return withCORS(r, allowedOrigin)
}
