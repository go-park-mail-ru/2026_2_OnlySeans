package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type RegisterUserUseCase struct {
	users    UserRepository
	sessions SessionStore
	hasher   domain.PasswordHasher
}

func NewRegisterUserUseCase(users UserRepository, sessions SessionStore, hasher domain.PasswordHasher) (*RegisterUserUseCase, error) {
	if users == nil {
		return nil, domain.ErrInvalidUserRepo
	}
	if sessions == nil {
		return nil, domain.ErrInvalidSessionStoe
	}
	if hasher == nil {
		return nil, domain.ErrInvalidPasswordHasher
	}
	return &RegisterUserUseCase{
		users:    users,
		sessions: sessions,
		hasher:   hasher,
	}, nil
}

func (uc *RegisterUserUseCase) Run(ctx context.Context, input RegisterUserInput) (*AuthOutput, error) {
	if err := domain.ValidatePassword(input.Password); err != nil {
		return nil, err
	}
	if err := domain.ValidateEmail(input.Email); err != nil {
		return nil, fmt.Errorf("validate email: %w", err)
	}
	if err := domain.ValidateUsername(input.Username); err != nil {
		return nil, fmt.Errorf("validate username: %w", err)
	}

	passwordHash, err := uc.hasher.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	id, err := uc.users.Create(ctx, input.Email, input.Username, passwordHash)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	createdUser, err := uc.users.GetUser(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get created user: %w", err)
	}
	if createdUser == nil {
		return nil, fmt.Errorf("get created user: repository returned nil user")
	}

	session, err := uc.sessions.Create(ctx, createdUser.ID)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	output := NewAuthOutput(
		NewUserDTO(createdUser.ID, createdUser.Email, createdUser.Username),
		NewSessionDTO(session.ID, session.ExpiresAt),
	)
	return &output, nil
}
