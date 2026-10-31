package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/utils"
)

type LoginUserUseCase struct {
	users    domain.UserRepository
	sessions domain.SessionStore
	verifier domain.PasswordVerifier
}

func NewLoginUserUseCase(users domain.UserRepository, sessions domain.SessionStore, verifier domain.PasswordVerifier) (*LoginUserUseCase, error) {
	if users == nil {
		return nil, domain.ErrInvalidUserRepo
	}

	if sessions == nil {
		return nil, domain.ErrInvalidSessionStoe
	}

	if verifier == nil {
		return nil, domain.ErrInvalidPasswordVer
	}

	return &LoginUserUseCase{
		users:    users,
		sessions: sessions,
		verifier: verifier,
	}, nil
}

func (uc *LoginUserUseCase) Run(ctx context.Context, input LoginUserInput) (*LoginOutput, error) {
	if err := utils.ValidateEmail(input.Email); err != nil {
		return nil, err
	}

	if input.Password == "" {
		return nil, domain.ErrInvalidPassword
	}

	user, err := uc.users.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	if user == nil {
		return nil, domain.ErrInvalidCredentials
	}

	if err := uc.verifier.Verify(input.Password, user.PasswordHash); err != nil {
		if errors.Is(err, ErrPasswordMismatch) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("verify password: %w", err)
	}

	session, err := uc.sessions.Create(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	output := NewLoginOutput(
		NewUserDTO(user.ID, user.Email, user.Username),
		NewSessionDTO(session.ID, session.ExpiresAt),
	)
	return &output, nil
}
