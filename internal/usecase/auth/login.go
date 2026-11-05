package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/utils"
)

type LoginUserUseCase struct {
	users    domain.UserRepository
	sessions domain.SessionStore
	verifier domain.PasswordVerifier
	logger   logger.Logger
}

func NewLoginUserUseCase(users domain.UserRepository, sessions domain.SessionStore, verifier domain.PasswordVerifier, appLogger logger.Logger) (*LoginUserUseCase, error) {
	if users == nil {
		return nil, domain.ErrInvalidUserRepo
	}

	if sessions == nil {
		return nil, domain.ErrInvalidSessionStoe
	}

	if verifier == nil {
		return nil, domain.ErrInvalidPasswordVer
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &LoginUserUseCase{
		users:    users,
		sessions: sessions,
		verifier: verifier,
		logger:   appLogger,
	}, nil
}

func (uc *LoginUserUseCase) Run(ctx context.Context, input LoginUserInput) (*LoginOutput, error) {
	uc.logger.Debug("login started", nil)

	if err := utils.ValidateEmail(input.Email); err != nil {
		uc.logger.Warn("login rejected", map[string]any{
			"reason": "invalid email",
		})

		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidEmail, err)
	}

	if input.Password == "" {
		uc.logger.Warn("login rejected", map[string]any{
			"reason": "empty password",
		})

		return nil, domain.ErrInvalidPassword
	}

	user, err := uc.users.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			uc.logger.Warn("login rejected", nil)
			return nil, domain.ErrInvalidCredentials
		}

		uc.logger.Error("get user by email failed", map[string]any{
			"error": err,
		})

		return nil, fmt.Errorf("get user by email: %w", err)
	}

	if user == nil {
		uc.logger.Warn("login rejected", nil)
		return nil, domain.ErrInvalidCredentials
	}

	if err := uc.verifier.Verify(input.Password, user.PasswordHash); err != nil {
		if errors.Is(err, ErrPasswordMismatch) {
			uc.logger.Warn("login rejected", nil)
			return nil, domain.ErrInvalidCredentials
		}

		uc.logger.Error("password verification failed", map[string]any{
			"error": err,
		})

		return nil, fmt.Errorf("verify password: %w", err)
	}

	session, err := uc.sessions.Create(ctx, user.ID)
	if err != nil {
		uc.logger.Error("session creation failed", map[string]any{
			"user_id": user.ID,
			"error":   err,
		})
		return nil, fmt.Errorf("create session: %w", err)
	}

	output := NewLoginOutput(
		NewUserDTO(user.ID, user.Email, user.Username),
		NewSessionDTO(session.ID, session.ExpiresAt),
	)

	output.User.AvatarFilename = user.AvatarFilename

	uc.logger.Info("user logged in", map[string]any{
		"user_id": user.ID,
	})

	return &output, nil
}
