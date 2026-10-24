package usecase

import (
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type RegisterUserInput struct {
	Email    string
	Username string
	Password string
}

func NewRegisterUserInput(email, username, password string) RegisterUserInput {
	return RegisterUserInput{
		Email:    email,
		Username: username,
		Password: password,
	}
}

type LoginUserInput struct {
	Email    string
	Password string
}

func NewLoginUserInput(email, password string) LoginUserInput {
	return LoginUserInput{
		Email:    email,
		Password: password,
	}
}

type AuthOutput struct {
	User    UserDTO
	Session SessionDTO
}

func NewAuthOutput(user UserDTO, session SessionDTO) AuthOutput {
	return AuthOutput{
		User:    user,
		Session: session,
	}
}

type UserDTO struct {
	ID       domain.UserID
	Email    string
	Username string
}

func NewUserDTO(id domain.UserID, email, username string) UserDTO {
	return UserDTO{
		ID:       id,
		Email:    email,
		Username: username,
	}
}

type SessionDTO struct {
	ID        string
	ExpiresAt time.Time
}

func NewSessionDTO(id string, expiresAt time.Time) SessionDTO {
	return SessionDTO{
		ID:        id,
		ExpiresAt: expiresAt,
	}
}

type GetUserInput struct {
	UserID domain.UserID
}

func NewGetUserInput(userID domain.UserID) GetUserInput {
	return GetUserInput{UserID: userID}
}

type GetUserOutput struct {
	ID       domain.UserID
	Email    string
	Username string
}

func NewGetUserOutput(id domain.UserID, email, username string) GetUserOutput {
	return GetUserOutput{
		ID:       id,
		Email:    email,
		Username: username,
	}
}

type AuthenticateInput struct {
	SessionID string
}

func NewAuthenticateInput(sessionID string) AuthenticateInput {
	return AuthenticateInput{SessionID: sessionID}
}

type AuthenticateOutput struct {
	ID domain.UserID
}

func NewAuthenticateOutput(id domain.UserID) AuthenticateOutput {
	return AuthenticateOutput{ID: id}
}

type LogoutInput struct {
	SessionID string
}

func NewLogoutInput(sessionID string) LogoutInput {
	return LogoutInput{SessionID: sessionID}
}
