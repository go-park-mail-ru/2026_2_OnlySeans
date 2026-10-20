package usecase

import (
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth/domain"
)

type RegisterUserInput struct {
	Email    string
	Username string
	Password string
}

type LoginUserInput struct {
	Email    string
	Password string
}

type AuthOutput struct {
	User    UserDTO
	Session SessionDTO
}

type UserDTO struct {
	ID       domain.UserID
	Email    string
	Username string
}

type SessionDTO struct {
	ID        string
	ExpiresAt time.Time
}
