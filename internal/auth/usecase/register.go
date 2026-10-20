package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth/domain"
)

type PasswordHasher interface {
	Hash(password string) (string, error)
}

type RegisterUserUseCase struct {
	users    UserRepository
	sessions SessionStore
	hasher   PasswordHasher
}

func NewRegisterUserUseCase(users UserRepository, sessions SessionStore, hasher PasswordHasher) (*RegisterUserUseCase, error) {
	if users == nil {
		return nil, errors.New("users repository is nil")
	}
	if sessions == nil {
		return nil, errors.New("sessions store is nil")
	}
	if hasher == nil {
		return nil, errors.New("password hasher is nil")
	}
	return &RegisterUserUseCase{
		users:    users,
		sessions: sessions,
		hasher:   hasher,
	}, nil
}

// ДОБАВИТЬ РЕПОЗИТОРИИ В ЮЗКЕЙС/либо подумать, надо ли это тут
func (uc *RegisterUserUseCase) Run(ctx context.Context, input RegisterUserInput) (AuthOutput, error) {
	if err := domain.ValidatePassword(input.Password); err != nil {
		return AuthOutput{}, err
	}
	passwordHash, err := uc.hasher.Hash(input.Password)
	if err != nil {
		return AuthOutput{}, fmt.Errorf("Hash password: %w", err)
	}

	user, err := domain.NewUser(input.Email, input.Username, passwordHash)
	if err != nil {
		return AuthOutput{}, fmt.Errorf("Create user: %w", err)
	}

	id, err := uc.users.Create(ctx, user.Email, user.Username, user.PasswordHash)
	if err != nil {
		return AuthOutput{}, fmt.Errorf("Create user: %w", err)
	}

	createdUser, err := uc.users.GetUser(ctx, id)
	if err != nil {
		return AuthOutput{}, fmt.Errorf("get created user: %w", err)
	}
	if createdUser == nil {
		return AuthOutput{}, fmt.Errorf("get created user: repository returned nil user")
	}
	user = *createdUser

	session, err := uc.sessions.Create(ctx, user.ID)
	if err != nil {
		return AuthOutput{}, fmt.Errorf("Create session: %w", err)
	}

	return AuthOutput{
		User:    UserDTO{ID: user.ID, Email: user.Email, Username: user.Username},
		Session: SessionDTO{ID: session.ID, ExpiresAt: session.ExpiresAt},
	}, nil
}
