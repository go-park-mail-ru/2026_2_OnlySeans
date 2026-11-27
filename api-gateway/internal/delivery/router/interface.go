package router

import "net/http"

type filmsHandler interface {
	ListFilms(http.ResponseWriter, *http.Request)
	GetFilm(http.ResponseWriter, *http.Request)
	CreateFilm(http.ResponseWriter, *http.Request)
	UpdateFilm(http.ResponseWriter, *http.Request)
	DeleteFilm(http.ResponseWriter, *http.Request)
	ListCollections(http.ResponseWriter, *http.Request)
	GetCollection(http.ResponseWriter, *http.Request)
}

type reviewsHandler interface {
	ListFilmReviews(http.ResponseWriter, *http.Request)
	CreateReview(http.ResponseWriter, *http.Request)
	UpdateReview(http.ResponseWriter, *http.Request)
	DeleteReview(http.ResponseWriter, *http.Request)
}

type personsHandler interface {
	ListPersons(http.ResponseWriter, *http.Request)
	GetPerson(http.ResponseWriter, *http.Request)
	CreatePerson(http.ResponseWriter, *http.Request)
	UpdatePerson(http.ResponseWriter, *http.Request)
	DeletePerson(http.ResponseWriter, *http.Request)
}

type filesHandler interface {
	GetMyAvatarURL(http.ResponseWriter, *http.Request)
	UpdateMyAvatar(http.ResponseWriter, *http.Request)
	DeleteMyAvatar(http.ResponseWriter, *http.Request)
	UploadAvatar(http.ResponseWriter, *http.Request)
	GetAvatarURL(http.ResponseWriter, *http.Request)
}

type postersHandler interface {
	UploadPoster(http.ResponseWriter, *http.Request)
	GetPoster(http.ResponseWriter, *http.Request)
	DeletePoster(http.ResponseWriter, *http.Request)
}

type ratingsHandler interface {
	GetFilmRating(http.ResponseWriter, *http.Request)
	SetFilmRating(http.ResponseWriter, *http.Request)
	DeleteFilmRating(http.ResponseWriter, *http.Request)
	GetRatingsForCollection(http.ResponseWriter, *http.Request)
}

type libraryHandler interface {
	Get(http.ResponseWriter, *http.Request)
	SetSaved(http.ResponseWriter, *http.Request)
	Preferences(http.ResponseWriter, *http.Request)
	SavePreference(http.ResponseWriter, *http.Request)
}

type personalRatingHandler interface {
	GetPersonalRating(http.ResponseWriter, *http.Request)
}

type personCatalogHandler interface {
	GetFilmPersons(http.ResponseWriter, *http.Request)
	GetPersonCatalog(http.ResponseWriter, *http.Request)
}

type assetsHandler interface {
	GetAsset(http.ResponseWriter, *http.Request)
}

type releasesHandler interface {
	List(http.ResponseWriter, *http.Request)
	Featured(http.ResponseWriter, *http.Request)
	FilmReleases(http.ResponseWriter, *http.Request)
	Update(http.ResponseWriter, *http.Request)
	Subscriptions(http.ResponseWriter, *http.Request)
	Subscribe(http.ResponseWriter, *http.Request)
	Unsubscribe(http.ResponseWriter, *http.Request)
	Notifications(http.ResponseWriter, *http.Request)
	ReadNotification(http.ResponseWriter, *http.Request)
}
