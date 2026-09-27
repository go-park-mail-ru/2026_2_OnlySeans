package auth

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrUserExists   = errors.New("пользователь с таким email уже существует")
	ErrUserNotFound = errors.New("пользователь не найден")
)

type UserRepository interface {
	Create(email, username, passwordHash string) (*User, error)
	FindByEmail(email string) (*User, error)
}

type InMemoryUserRepo struct {
	mu     sync.Mutex
	users  map[string]*User // ключ — email
	nextID int64
}

func NewInMemoryUserRepo() *InMemoryUserRepo {
	return &InMemoryUserRepo{
		users:  make(map[string]*User),
		nextID: 1,
	}
}

func (r *InMemoryUserRepo) Create(email, username, passwordHash string) (*User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[email]; exists {
		return nil, ErrUserExists
	}

	now := time.Now().UTC()
	user := &User{
		ID:           r.nextID,
		Email:        email,
		Username:     username,
		PasswordHash: passwordHash,
		Role:         "user",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	r.users[email] = user
	r.nextID++
	return user, nil
}

func (r *InMemoryUserRepo) FindByEmail(email string) (*User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[email]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}
