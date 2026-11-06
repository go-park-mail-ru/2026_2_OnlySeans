package domain

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/utils"
)

type Role string

const (
	RoleReader Role = "reader"
	RoleEditor Role = "editor"
	RoleAdmin  Role = "admin"
)

func (role Role) CanEditPersons() bool {
	return role == RoleAdmin || role == RoleEditor
}

type UserID int

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

type SessionStore interface {
	Create(ctx context.Context, userID UserID) (*Session, error)
	Get(ctx context.Context, sessionID string) (*Session, error)
	Delete(ctx context.Context, sessionID string) error
}

type UserRepository interface {
	Create(ctx context.Context, email, username, passwordHash string) (UserID, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetUser(ctx context.Context, id UserID) (*User, error)
}
