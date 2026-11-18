package domain

type ProfileFilm struct {
	ID               int64
	Title            string
	Year             int16
	Type             string
	AgeLimit         int16
	Genres           string
	PosterURL        *string
	PosterStorageKey *string
	Average          float64
	Score            *int
}

type Profile struct {
	User      User
	Favorites []ProfileFilm
	Ratings   []ProfileFilm
	Reviews   int
}
