package domain

import "time"

type ProfileUser struct {
	ID             UserID
	Email          string
	Username       string
	AvatarFilename *string
	CreatedAt      time.Time
}

type ProfileFilm struct {
	ID               FilmID
	Title            string
	Year             int16
	Type             FilmType
	AgeLimit         int16
	Genres           string
	PosterURL        *string
	PosterStorageKey *string
	Average          float64
	Score            *int
}

type Profile struct {
	User      ProfileUser
	Favorites []ProfileFilm
	Ratings   []ProfileFilm
	Reviews   int
}
