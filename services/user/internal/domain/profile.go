package domain

type Profile struct {
	User      User
	Favorites []ProfileFilm
	Ratings   []ProfileFilm
	Reviews   int64
}

type ProfileFilm struct {
	ID               int64
	Title            string
	Year             int32
	Type             string
	AgeLimit         int32
	Genres           string
	PosterURL        *string
	PosterStorageKey *string
	Average          float64
	Score            *int32
}
