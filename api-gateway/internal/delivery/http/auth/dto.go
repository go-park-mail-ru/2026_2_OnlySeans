package auth

import (
	pbUser "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
)

//easyjson:json
type registerRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

//easyjson:json
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

//easyjson:json
type userResponse struct {
	ID             int64   `json:"id"`
	Email          string  `json:"email"`
	Username       string  `json:"username"`
	AvatarFilename *string `json:"avatar_filename,omitempty"`
}

//easyjson:json
type authResponse struct {
	User userResponse `json:"user"`
}

func toAuthResponse(user *pbUser.User) authResponse {
	return authResponse{
		User: userResponse{
			ID:             user.Id,
			Email:          user.Email,
			Username:       user.Username,
			AvatarFilename: user.AvatarFilename,
		},
	}
}
