package auth

type UserID int64

type User struct {
	ID           UserID `json:"id" db:"id"`
	Email        string `json:"email" db:"email"`
	Username     string `json:"username" db:"username"`
	PasswordHash string `json:"-" db:"password_hash"`
}
