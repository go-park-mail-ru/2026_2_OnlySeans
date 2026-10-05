package auth

type UserID int

type User struct {
	ID           UserID `json:"id"`
	Email        string `json:"email"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
}
