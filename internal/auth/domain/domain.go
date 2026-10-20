package domain

import "time"

type UserID int

// УБРАТЬ ДЖСОН ТЕГИ, ТК ДОМЕННЫЙ СЛОЙ НЕ ДОЛЖЕН ЗНАТЬ О ДАННЫХ ИЗ БД И ТД
// Следовательно добавить DTO

type User struct {
	ID           UserID
	Email        string
	Username     string
	PasswordHash string
}

func NewUser(email, username, passwordHash string) (User, error) {
	if err := ValidateEmail(email); err != nil {
		return User{}, err
	}
	if err := ValidateUsername(username); err != nil {
		return User{}, err
	}
	if passwordHash == "" {
		return User{}, ErrInvalidPasswordHash
	}

	return User{
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
