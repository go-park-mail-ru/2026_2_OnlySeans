package auth

import (
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
)

type registerRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type authResponse struct {
	User userResponse `json:"user"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func toRegisterUserInput(request registerRequest) usecase.RegisterUserInput {
	return usecase.NewRegisterUserInput(request.Email, request.Username, request.Password)
}

func toLoginUserInput(request loginRequest) usecase.LoginUserInput {
	return usecase.NewLoginUserInput(request.Email, request.Password)
}

func toAuthResponse(user usecase.UserDTO) authResponse {
	return authResponse{
		User: userResponse{
			ID:       int(user.ID),
			Email:    user.Email,
			Username: user.Username,
		},
	}
}
