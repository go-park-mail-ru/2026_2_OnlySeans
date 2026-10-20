package auth

import (
	"context"
	"sync"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth/domain"
)

type UserID = domain.UserID
type User = domain.User

type InMemoryUserRepo struct {
	mu      sync.Mutex
	byEmail map[string]*domain.User
	byID    map[int]*domain.User
	nextID  int
}

func NewInMemoryUserRepo() *InMemoryUserRepo {
	return &InMemoryUserRepo{
		byEmail: make(map[string]*domain.User),
		byID:    make(map[int]*domain.User),
		nextID:  1,
	}
}

func (r *InMemoryUserRepo) Create(_ context.Context, email, username, passwordHash string) (domain.UserID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byEmail[email]; exists {
		return 0, domain.ErrUserExists
	}

	id := domain.UserID(r.nextID)
	stored := &domain.User{
		ID:           id,
		Email:        email,
		Username:     username,
		PasswordHash: passwordHash,
	}

	r.byEmail[email] = stored
	r.byID[int(id)] = stored
	r.nextID++

	return id, nil
}

func (r *InMemoryUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.byEmail[email]
	if !exists {
		return nil, domain.ErrUserNotFound
	}

	return user, nil
}

func (r *InMemoryUserRepo) GetUser(_ context.Context, id domain.UserID) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.byID[int(id)]
	if !exists {
		return nil, domain.ErrUserNotFound
	}

	return user, nil
}
