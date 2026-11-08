package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
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
	uc.logger.Info("registration started", nil)

	if err := utils.ValidatePassword(input.Password); err != nil {
		logging.Validation(uc.logger, "registration rejected: invalid input", domain.ErrInvalidPassword, nil)
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidPassword, err)
	}

	if err := utils.ValidateEmail(input.Email); err != nil {
		logging.Validation(uc.logger, "registration rejected: invalid input", domain.ErrInvalidEmail, nil)
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidEmail, err)
	}

	if err := utils.ValidateUsername(input.Username); err != nil {
		logging.Validation(uc.logger, "registration rejected: invalid input", domain.ErrInvalidUsername, nil)
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidUsername, err)
	}
	uc.logger.Debug("registration input validated", nil)

	passwordHash, err := uc.hasher.Hash(input.Password)
	if err != nil {
		logging.Failure(uc.logger, "password hashing failed", err, map[string]any{})

		return nil, fmt.Errorf("hash password: %w", err)
	}

	uc.logger.Debug("password hashed", nil)
	uc.logger.Debug("creating user", nil)

	id, err := uc.users.Create(ctx, input.Email, input.Username, passwordHash)
	if err != nil {
		logging.Failure(uc.logger, "user creation failed", err, map[string]any{})
		return nil, fmt.Errorf("create user: %w", err)
	}
	uc.logger.Info("user created", map[string]any{"user_id": id})

	createdUser, err := uc.users.GetUser(ctx, id)
	if err != nil {
		logging.Failure(uc.logger, "get created user failed", err, map[string]any{
			"user_id": id,
		})
		return nil, fmt.Errorf("get created user: %w", err)
	}
	if createdUser == nil {
		err := fmt.Errorf("repository returned nil user")
		logging.Failure(uc.logger, "get created user failed", err, map[string]any{
			"user_id": id,
		})
		return nil, fmt.Errorf("get created user: %w", err)
	}
	uc.logger.Debug("created user loaded", map[string]any{"user_id": createdUser.ID})

	session, err := uc.sessions.Create(ctx, createdUser.ID)
	if err != nil {
		logging.Failure(uc.logger, "session creation failed", err, map[string]any{
			"user_id": createdUser.ID,
		})
		return nil, fmt.Errorf("create session: %w", err)
	}
	uc.logger.Info("session created", map[string]any{
		"user_id":    createdUser.ID,
		"expires_at": session.ExpiresAt,
	})

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
