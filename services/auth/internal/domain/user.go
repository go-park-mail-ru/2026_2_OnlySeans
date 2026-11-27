package domain

type UserID int64

type User struct {
	ID           UserID
	Role         Role
	PasswordHash string
}
