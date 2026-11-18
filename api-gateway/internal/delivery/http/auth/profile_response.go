package auth

import (
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type profileUserResponse struct {
	userResponse
	CreatedAt time.Time `json:"created_at"`
}

type profileFilmResponse struct {
	ID       domain.FilmID   `json:"id"`
	Title    string          `json:"title"`
	Year     int16           `json:"year"`
	Type     domain.FilmType `json:"type"`
	AgeLimit int16           `json:"age_limit"`
	Genres   string          `json:"genres"`
	Poster   *string         `json:"poster"`
	Average  float64         `json:"average"`
	Score    *int            `json:"score,omitempty"`
}

type profileResponse struct {
	User      profileUserResponse   `json:"user"`
	Favorites []profileFilmResponse `json:"favorites"`
	Ratings   []profileFilmResponse `json:"ratings"`
	Reviews   int                   `json:"reviews"`
}

func toProfileResponse(profile *domain.Profile) profileResponse {
	return profileResponse{
		User: profileUserResponse{
			userResponse: userResponse{ID: int(profile.User.ID), Email: profile.User.Email, Username: profile.User.Username, AvatarFilename: profile.User.AvatarFilename},
			CreatedAt:    profile.User.CreatedAt,
		},
		Favorites: toProfileFilms(profile.Favorites),
		Ratings:   toProfileFilms(profile.Ratings),
		Reviews:   profile.Reviews,
	}
}

func toProfileFilms(films []domain.ProfileFilm) []profileFilmResponse {
	result := make([]profileFilmResponse, len(films))
	for i, film := range films {
		result[i] = profileFilmResponse{ID: film.ID, Title: film.Title, Year: film.Year, Type: film.Type, AgeLimit: film.AgeLimit, Genres: film.Genres, Poster: film.PosterURL, Average: film.Average, Score: film.Score}
	}
	return result
}
