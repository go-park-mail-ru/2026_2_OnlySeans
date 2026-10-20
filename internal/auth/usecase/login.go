package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth/domain"
)

type PasswordVerifier interface {
	Verify(password, hash string) error
}

type LoginUserUseCase struct {
	users    UserRepository
	sessions SessionStore
	verifier PasswordVerifier
}

func NewLoginUserUseCase(users UserRepository, sessions SessionStore, verifier PasswordVerifier) (*LoginUserUseCase, error) {
	if users == nil {
		return nil, errors.New("users repository is nil")
	}

	if sessions == nil {
		return nil, errors.New("sessions store is nil")
	}

	if verifier == nil {
		return nil, errors.New("password verifier is nil")
	}

	return &LoginUserUseCase{
		users:    users,
		sessions: sessions,
		verifier: verifier,
	}, nil
}

func (uc *LoginUserUseCase) Run(ctx context.Context, input LoginUserInput) (AuthOutput, error) {
	if err := domain.ValidateEmail(input.Email); err != nil {
		return AuthOutput{}, err
	}

	if input.Password == "" {
		return AuthOutput{}, domain.ErrInvalidPassword
	}

	user, err := uc.users.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return AuthOutput{}, domain.ErrInvalidCredentials
		}
		return AuthOutput{}, fmt.Errorf("get user by email: %w", err)
	}

	if user == nil {
		return AuthOutput{}, domain.ErrInvalidCredentials
	}

	if err := uc.verifier.Verify(input.Password, user.PasswordHash); err != nil {
		if errors.Is(err, ErrPasswordMismatch) {
			return AuthOutput{}, domain.ErrInvalidCredentials
		}
		return AuthOutput{}, fmt.Errorf("verify password: %w", err)
	}

	session, err := uc.sessions.Create(ctx, user.ID)
	if err != nil {
		return AuthOutput{}, fmt.Errorf("create session: %w", err)
	}

	return AuthOutput{
		User:    UserDTO{ID: user.ID, Email: user.Email, Username: user.Username},
		Session: SessionDTO{ID: session.ID, ExpiresAt: session.ExpiresAt},
	}, nil
}
