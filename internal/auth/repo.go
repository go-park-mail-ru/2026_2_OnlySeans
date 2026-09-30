package auth

import (
	"context"
	"sync"
)

type UserRepository interface {
	Create(ctx context.Context, email, username, passwordHash string) (UserID, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetUser(ctx context.Context, id UserID) (*User, error)
}

type InMemoryUserRepo struct {
	mu      sync.Mutex
	byEmail map[string]*User
	byID    map[int]*User
	nextID  int
}

func NewInMemoryUserRepo() *InMemoryUserRepo {
	return &InMemoryUserRepo{
		byEmail: make(map[string]*User),
		byID:    make(map[int]*User),
		nextID:  1,
	}
}

func (r *InMemoryUserRepo) Create(ctx context.Context, email, username, passwordHash string) (UserID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byEmail[email]; exists {
		return 0, ErrUserExists
	}

	id := UserID(r.nextID)
	user := &User{
		ID:           id,
		Email:        email,
		Username:     username,
		PasswordHash: passwordHash,
	}

	r.byEmail[email] = user
	r.byID[int(id)] = user
	r.nextID++

	return id, nil
}

func (r *InMemoryUserRepo) GetByEmail(ctx context.Context, email string) (*User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.byEmail[email]
	if !exists {
		return nil, ErrUserNotFound
	}

	return user, nil
}

func (r *InMemoryUserRepo) GetUser(ctx context.Context, id UserID) (*User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.byID[int(id)]
	if !exists {
		return nil, ErrUserNotFound
	}

	return user, nil
}
