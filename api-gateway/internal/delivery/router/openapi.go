package router

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/openapi"
)

type openAPIAdapter struct {
	next http.Handler
}

func newOpenAPIHandler(next http.Handler) http.Handler {
	routes := mux.NewRouter()
	routes.NotFoundHandler = next
	routes.MethodNotAllowedHandler = next

	return openapi.HandlerWithOptions(&openAPIAdapter{
		next: next,
	}, openapi.GorillaServerOptions{
		BaseRouter: routes,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, _ error) {
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

func (a *openAPIAdapter) FilmReleases(w http.ResponseWriter, r *http.Request, id int64, params openapi.FilmReleasesParams) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) MyNotifications(w http.ResponseWriter, r *http.Request, params openapi.MyNotificationsParams) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) ReadMyNotification(w http.ResponseWriter, r *http.Request, id int64) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) MyReleaseSubscriptions(w http.ResponseWriter, r *http.Request, params openapi.MyReleaseSubscriptionsParams) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) UnsubscribeFromRelease(w http.ResponseWriter, r *http.Request, id int64) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) SubscribeToRelease(w http.ResponseWriter, r *http.Request, id int64) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) ListReleases(w http.ResponseWriter, r *http.Request, params openapi.ListReleasesParams) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) FeaturedReleases(w http.ResponseWriter, r *http.Request, params openapi.FeaturedReleasesParams) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) UpdateRelease(w http.ResponseWriter, r *http.Request, id int64) {
	a.next.ServeHTTP(w, r)
}

func (a *openAPIAdapter) RefreshSession(w http.ResponseWriter, r *http.Request) {
	a.next.ServeHTTP(w, r)
}
