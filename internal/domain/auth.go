package domain

import (
	"time"
)

type UserID int

type User struct {
	ID           UserID
	Email        string
	Username     string
	PasswordHash string
}

func NewUser(email, username, passwordHash string) (*User, error) {
	if err := ValidateEmail(email); err != nil {
		return &User{}, err
	}
	if err := ValidateUsername(username); err != nil {
		return &User{}, err
	}
	if passwordHash == "" {
		return &User{}, ErrInvalidPasswordHash
	}

	return &User{
		Email:        email,
		Username:     username,
		PasswordHash: passwordHash,
	}, nil
}

type Session struct {
	ID        string
	UserID    UserID
	ExpiresAt time.Time
}

func NewSession(id string, userID UserID, expiresAt time.Time) *Session {
	return &Session{
		ID:        id,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}
}

type PasswordHasher interface {
	Hash(password string) (string, error)
}

type PasswordVerifier interface {
	Verify(password, hash string) error
}
