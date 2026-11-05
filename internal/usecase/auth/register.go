package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/utils"
)

type RegisterUserUseCase struct {
	users    domain.UserRepository
	sessions domain.SessionStore
	hasher   domain.PasswordHasher
	logger   logger.Logger
}

func NewRegisterUserUseCase(users domain.UserRepository, sessions domain.SessionStore, hasher domain.PasswordHasher, appLogger logger.Logger) (*RegisterUserUseCase, error) {
	if users == nil {
		return nil, domain.ErrInvalidUserRepo
	}
	if sessions == nil {
		return nil, domain.ErrInvalidSessionStoe
	}
	if hasher == nil {
		return nil, domain.ErrInvalidPasswordHasher
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}
	return &RegisterUserUseCase{
		users:    users,
		sessions: sessions,
		hasher:   hasher,
		logger:   appLogger,
	}, nil
}

func (uc *RegisterUserUseCase) Run(ctx context.Context, input RegisterUserInput) (*RegisterOutput, error) {
	uc.logger.Debug("register started", nil)

	if err := utils.ValidatePassword(input.Password); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidPassword, err)
	}

	if err := utils.ValidateEmail(input.Email); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidEmail, err)
	}

	if err := utils.ValidateUsername(input.Username); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidUsername, err)
	}

	passwordHash, err := uc.hasher.Hash(input.Password)
	if err != nil {
		uc.logger.Error("password hashing failed", map[string]any{
			"error": err,
		})

		return nil, fmt.Errorf("hash password: %w", err)
	}

	uc.logger.Debug("creating user", map[string]any{
		"email": input.Email,
	})

	id, err := uc.users.Create(ctx, input.Email, input.Username, passwordHash)
	if err != nil {
		uc.logger.Error("user creation failed", map[string]any{
			"error": err,
		})
		return nil, fmt.Errorf("create user: %w", err)
	}

	createdUser, err := uc.users.GetUser(ctx, id)
	if err != nil {
		uc.logger.Error("get created user failed", map[string]any{
			"user_id": id,
			"error":   err,
		})
		return nil, fmt.Errorf("get created user: %w", err)
	}
	if createdUser == nil {
		err := fmt.Errorf("repository returned nil user")
		uc.logger.Error("get created user failed", map[string]any{
			"user_id": id,
			"error":   err,
		})
		return nil, fmt.Errorf("get created user: %w", err)
	}

	session, err := uc.sessions.Create(ctx, createdUser.ID)
	if err != nil {
		uc.logger.Error("session creation failed", map[string]any{
			"user_id": createdUser.ID,
			"error":   err,
		})
		return nil, fmt.Errorf("create session: %w", err)
	}

	output := NewRegisterOutput(
		NewUserDTO(createdUser.ID, createdUser.Email, createdUser.Username),
		NewSessionDTO(session.ID, session.ExpiresAt),
	)

	output.User.AvatarFilename = createdUser.AvatarFilename

	uc.logger.Info("user registered", map[string]any{
		"user_id": createdUser.ID,
	})

	return &output, nil
}
