package auth

import (
	"context"
	"strings"
	"sync"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type InMemoryUserRepo struct {
	mu      sync.Mutex
	byEmail map[string]*domain.User
	byID    map[int]*domain.User
	avatars map[domain.UserID]domain.AvatarFile
	nextID  int
}

func NewInMemoryUserRepo() *InMemoryUserRepo {
	return &InMemoryUserRepo{
		byEmail: make(map[string]*domain.User),
		byID:    make(map[int]*domain.User),
		avatars: make(map[domain.UserID]domain.AvatarFile),
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
		Role:         domain.RoleReader,
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

func (r *InMemoryUserRepo) SaveAvatar(_ context.Context, id domain.UserID, file domain.AvatarFile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.byID[int(id)]
	if !ok {
		return domain.ErrUserNotFound
	}
	r.avatars[id] = file
	filename := file.Filename
	user.AvatarFilename = &filename
	return nil
}

func (r *InMemoryUserRepo) GetAvatar(_ context.Context, id domain.UserID) (*domain.AvatarFile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[int(id)]; !ok {
		return nil, domain.ErrUserNotFound
	}
	file, ok := r.avatars[id]
	if !ok || strings.TrimSpace(file.Filename) == "" {
		return nil, domain.ErrFileNotFound
	}
	return &file, nil
}

func (r *InMemoryUserRepo) DeleteAvatar(_ context.Context, id domain.UserID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.byID[int(id)]
	if !ok {
		return domain.ErrUserNotFound
	}
	user.AvatarFilename = nil
	delete(r.avatars, id)
	return nil
}

func (r *InMemoryUserRepo) UpdateProfile(_ context.Context, id domain.UserID, email, username string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.byID[int(id)]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	for otherID, other := range r.byID {
		if otherID == int(id) {
			continue
		}
		if other.Email == email {
			return nil, domain.ErrUserExists
		}
		if other.Username == username {
			return nil, domain.ErrUsernameTaken
		}
	}
	delete(r.byEmail, user.Email)
	user.Email, user.Username = email, username
	r.byEmail[email] = user
	result := *user
	return &result, nil
}
