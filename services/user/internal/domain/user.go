package domain

import "time"

type User struct {
	ID             int64
	Email          string
	Username       string
	PasswordHash   string
	Role           string
	AvatarFilename *string
	CreatedAt      time.Time
}
