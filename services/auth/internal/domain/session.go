package domain

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        uuid.UUID `json:"id"`
	UserID    UserID    `json:"user_id"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewSession(id uuid.UUID, userID UserID, role string, expiresAt time.Time) *Session {
	return &Session{
		ID:        id,
		UserID:    userID,
		Role:      role,
		ExpiresAt: expiresAt,
	}
}

type PasswordHasher interface {
	Hash(password string) (string, error)
}

type PasswordVerifier interface {
	Verify(password, hash string) error
}
