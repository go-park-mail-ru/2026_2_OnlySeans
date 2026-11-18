package domain

import (
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/utils"
)

type UserID int64

type User struct {
	ID             UserID
	Role           Role
	Email          string
	Username       string
	PasswordHash   string
	AvatarFilename *string
}

func NewUser(email, username, passwordHash string) (*User, error) {
	if err := utils.ValidateEmail(email); err != nil {
		return nil, err
	}
	if err := utils.ValidateUsername(username); err != nil {
		return nil, err
	}
	if passwordHash == "" {
		return nil, ErrInvalidPasswordHash
	}

	return &User{
		Role:         RoleReader,
		Email:        email,
		Username:     username,
		PasswordHash: passwordHash,
	}, nil
}
