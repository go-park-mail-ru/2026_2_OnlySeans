package user

import (
	"time"

	pbUser "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
)

//easyjson:json
type userResponse struct {
	ID             int64   `json:"id"`
	Email          string  `json:"email"`
	Username       string  `json:"username"`
	AvatarFilename *string `json:"avatar_filename,omitempty"`
}

//easyjson:json
type profileUpdateResponse struct {
	User userResponse `json:"user"`
}

//easyjson:json
type profileUserResponse struct {
	userResponse
	CreatedAt time.Time `json:"created_at"`
}

//easyjson:json
type profileFilmResponse struct {
	ID       int64   `json:"id"`
	Title    string  `json:"title"`
	Year     int32   `json:"year"`
	Type     string  `json:"type"`
	AgeLimit int32   `json:"age_limit"`
	Genres   string  `json:"genres"`
	Poster   *string `json:"poster"`
	Average  float64 `json:"average"`
	Score    *int32  `json:"score,omitempty"`
}

//easyjson:json
type profileResponse struct {
	User      profileUserResponse   `json:"user"`
	Favorites []profileFilmResponse `json:"favorites"`
	Ratings   []profileFilmResponse `json:"ratings"`
	Reviews   int64                 `json:"reviews"`
}

func toUserResponse(user *pbUser.User) userResponse {
	return userResponse{
		ID:             user.GetId(),
		Email:          user.GetEmail(),
		Username:       user.GetUsername(),
		AvatarFilename: user.AvatarFilename,
	}
}

func toProfileResponse(profile *pbUser.GetProfileResponse) profileResponse {
	return profileResponse{
		User: profileUserResponse{
			userResponse: toUserResponse(profile.GetUser()),
			CreatedAt:    profile.GetCreatedAt().AsTime(),
		},
		Favorites: toProfileFilms(profile.GetFavorites()),
		Ratings:   toProfileFilms(profile.GetRatings()),
		Reviews:   profile.GetReviews(),
	}
}

func toProfileFilms(films []*pbUser.ProfileFilm) []profileFilmResponse {
	result := make([]profileFilmResponse, len(films))
	for i, film := range films {
		result[i] = profileFilmResponse{
			ID:       film.GetId(),
			Title:    film.GetTitle(),
			Year:     film.GetYear(),
			Type:     film.GetType(),
			AgeLimit: film.GetAgeLimit(),
			Genres:   film.GetGenres(),
			Poster:   film.Poster,
			Average:  film.GetAverage(),
			Score:    film.Score,
		}
	}
	return result
}
