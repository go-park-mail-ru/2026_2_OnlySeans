package router

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/persons"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

func NewRouter(authHandler *auth.Handler, filmsHandler *films.Handler, reviewsHandler *reviews.Handler, personsHandler *persons.Handler, filesHandler *files.Handler, allowedOrigin string, appLogger logger.Logger) http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/api/register", authHandler.Register).Methods(http.MethodPost)
	r.HandleFunc("/api/login", authHandler.Login).Methods(http.MethodPost)
	r.HandleFunc("/api/logout", RequireAuth(authHandler, authHandler.Logout)).Methods(http.MethodPost)
	r.HandleFunc("/api/me/avatar", RequireAuth(authHandler, filesHandler.GetMyAvatarURL)).Methods(http.MethodGet)
	r.HandleFunc("/api/me/avatar", RequireAuth(authHandler, filesHandler.UpdateMyAvatar)).Methods(http.MethodPut)
	r.HandleFunc("/api/me/avatar", RequireAuth(authHandler, filesHandler.DeleteMyAvatar)).Methods(http.MethodDelete)
	r.HandleFunc("/api/avatars", RequireAuth(authHandler, filesHandler.UploadAvatar)).Methods(http.MethodPost)
	r.HandleFunc("/api/avatars/{filename}", filesHandler.GetAvatarURL).Methods(http.MethodGet)

	r.HandleFunc("/api/films", filmsHandler.ListFilms).Methods(http.MethodGet)
	r.HandleFunc("/api/films/{id}", filmsHandler.GetFilm).Methods(http.MethodGet)

	r.HandleFunc("/api/collections", filmsHandler.ListCollections).Methods(http.MethodGet)
	r.HandleFunc("/api/collections/{id}", filmsHandler.GetCollection).Methods(http.MethodGet)

	r.HandleFunc("/api/authorised", RequireAuth(authHandler, authHandler.Authorised)).Methods(http.MethodGet)

	r.HandleFunc("/api/films/{id}/reviews", reviewsHandler.ListByFilm).Methods(http.MethodGet)
	r.HandleFunc("/api/films/{id}/reviews", RequireAuth(authHandler, reviewsHandler.Create)).Methods(http.MethodPost)
	r.HandleFunc("/api/reviews/{id}", RequireAuth(authHandler, reviewsHandler.Update)).Methods(http.MethodPut)
	r.HandleFunc("/api/reviews/{id}", RequireAuth(authHandler, reviewsHandler.Delete)).Methods(http.MethodDelete)

	r.HandleFunc("/api/persons", personsHandler.ListPersons).Methods(http.MethodGet)
	r.HandleFunc("/api/persons/{id}", personsHandler.GetPerson).Methods(http.MethodGet)

	r.HandleFunc("/api/persons", RequireAuth(authHandler, RequireCatalogEditor(authHandler, personsHandler.CreatePerson, appLogger))).Methods(http.MethodPost)
	r.HandleFunc("/api/persons/{id}", RequireAuth(authHandler, RequireCatalogEditor(authHandler, personsHandler.UpdatePerson, appLogger))).Methods(http.MethodPut)
	r.HandleFunc("/api/persons/{id}", RequireAuth(authHandler, RequireCatalogEditor(authHandler, personsHandler.DeletePerson, appLogger))).Methods(http.MethodDelete)

	return withLogging(withRecover(withCORS(r, allowedOrigin), appLogger), appLogger)
}
