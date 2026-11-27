package router

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/openapi"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

type searchHandler interface {
	GetSearchSuggestions(http.ResponseWriter, *http.Request, openapi.GetSearchSuggestionsParams)
	SearchFilms(http.ResponseWriter, *http.Request, openapi.SearchFilmsParams)
	SearchPersons(http.ResponseWriter, *http.Request, openapi.SearchPersonsParams)
}

type openAPIAdapter struct {
	next   http.Handler
	search searchHandler
}

func newOpenAPIHandler(next http.Handler, search searchHandler, appLogger logger.Logger) http.Handler {
	routes := mux.NewRouter()
	routes.NotFoundHandler = next
	routes.MethodNotAllowedHandler = next

	return openapi.HandlerWithOptions(&openAPIAdapter{
		next:   next,
		search: search,
	}, openapi.GorillaServerOptions{
		BaseRouter: routes,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			if search != nil && strings.HasPrefix(r.URL.Path, "/api/search/") {
				httpresponse.WriteErrorContext(r.Context(), w, http.StatusBadRequest, err.Error(), appLogger)
				return
			}

			next.ServeHTTP(w, r)
		},
	})
}

func (a *openAPIAdapter) GetAuthorised(w http.ResponseWriter, r *http.Request) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) GetAvatarURL(w http.ResponseWriter, r *http.Request, filename string) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) ListCollections(w http.ResponseWriter, r *http.Request, params openapi.ListCollectionsParams) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) GetCollectionById(w http.ResponseWriter, r *http.Request, id int64, params openapi.GetCollectionByIdParams) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) GetCollectionFilmRatings(w http.ResponseWriter, r *http.Request, id int64, params openapi.GetCollectionFilmRatingsParams) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) ListFilms(w http.ResponseWriter, r *http.Request, params openapi.ListFilmsParams) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) CreateFilm(w http.ResponseWriter, r *http.Request) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) DeleteFilm(w http.ResponseWriter, r *http.Request, id int64) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) GetFilmById(w http.ResponseWriter, r *http.Request, id int64) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) UpdateFilm(w http.ResponseWriter, r *http.Request, id int64) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) GetFilmPersons(w http.ResponseWriter, r *http.Request, id int64) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) DeleteFilmPoster(w http.ResponseWriter, r *http.Request, id int64) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) GetFilmPosterURL(w http.ResponseWriter, r *http.Request, id int64) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) UploadFilmPoster(w http.ResponseWriter, r *http.Request, id int64) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) DeleteFilmRating(w http.ResponseWriter, r *http.Request, id int64) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) GetFilmRating(w http.ResponseWriter, r *http.Request, id int64) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) SetFilmRating(w http.ResponseWriter, r *http.Request, id int64) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) GetMyFilmRating(w http.ResponseWriter, r *http.Request, id int64) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) LoginUser(w http.ResponseWriter, r *http.Request) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) LogoutUser(w http.ResponseWriter, r *http.Request) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) UpdateMyProfileLegacy(w http.ResponseWriter, r *http.Request) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) DeleteMyAvatar(w http.ResponseWriter, r *http.Request) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) GetMyAvatarURL(w http.ResponseWriter, r *http.Request) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) UpdateMyAvatar(w http.ResponseWriter, r *http.Request) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) GetMyLibrary(w http.ResponseWriter, r *http.Request) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) SetLibraryFilm(w http.ResponseWriter, r *http.Request, list string, film int64) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) GetMyPreferences(w http.ResponseWriter, r *http.Request) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) SaveMyPreference(w http.ResponseWriter, r *http.Request) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) GetPersonCatalog(w http.ResponseWriter, r *http.Request, id int64) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) ListPersons(w http.ResponseWriter, r *http.Request, params openapi.ListPersonsParams) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) CreatePerson(w http.ResponseWriter, r *http.Request) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) DeletePerson(w http.ResponseWriter, r *http.Request, id openapi.PersonID) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) GetPerson(w http.ResponseWriter, r *http.Request, id openapi.PersonID) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) UpdatePerson(w http.ResponseWriter, r *http.Request, id openapi.PersonID) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) RegisterUser(w http.ResponseWriter, r *http.Request) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) GetSearchSuggestions(w http.ResponseWriter, r *http.Request, params openapi.GetSearchSuggestionsParams) {
	if a.search == nil {
		a.next.ServeHTTP(w, r)
		return
	}

	a.search.GetSearchSuggestions(w, r, params)
}

func (a *openAPIAdapter) SearchFilms(w http.ResponseWriter, r *http.Request, params openapi.SearchFilmsParams) {
	if a.search == nil {
		a.next.ServeHTTP(w, r)
		return
	}

	a.search.SearchFilms(w, r, params)
}

func (a *openAPIAdapter) SearchPersons(w http.ResponseWriter, r *http.Request, params openapi.SearchPersonsParams) {
	if a.search == nil {
		a.next.ServeHTTP(w, r)
		return
	}

	a.search.SearchPersons(w, r, params)
}
