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
	users     domain.UserRepository
	sessions  domain.SessionStore
	hasher    domain.PasswordHasher
	appLogger logger.Logger
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
		users:     users,
		sessions:  sessions,
		hasher:    hasher,
		appLogger: appLogger,
	}, nil
}

func (uc *RegisterUserUseCase) Run(ctx context.Context, input RegisterUserInput) (*RegisterOutput, error) {
	uc.appLogger.InfoContext(ctx, "registration started", nil)

	if err := utils.ValidatePassword(input.Password); err != nil {
		logging.Validation(ctx, uc.appLogger, "registration rejected: invalid input", domain.ErrInvalidPassword, nil)
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidPassword, err)
	}

	if err := utils.ValidateEmail(input.Email); err != nil {
		logging.Validation(ctx, uc.appLogger, "registration rejected: invalid input", domain.ErrInvalidEmail, nil)
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidEmail, err)
	}

	if err := utils.ValidateUsername(input.Username); err != nil {
		logging.Validation(ctx, uc.appLogger, "registration rejected: invalid input", domain.ErrInvalidUsername, nil)
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidUsername, err)
	}
	uc.appLogger.DebugContext(ctx, "registration input validated", nil)

	passwordHash, err := uc.hasher.Hash(input.Password)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "password hashing failed", err, map[string]any{})

		return nil, fmt.Errorf("hash password: %w", err)
	}

	uc.appLogger.DebugContext(ctx, "password hashed", nil)
	uc.appLogger.DebugContext(ctx, "creating user", nil)

	id, err := uc.users.Create(ctx, input.Email, input.Username, passwordHash)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "user creation failed", err, map[string]any{})
		return nil, fmt.Errorf("create user: %w", err)
	}
	uc.appLogger.InfoContext(ctx, "user created", map[string]any{"user_id": id})

	createdUser, err := uc.users.GetUser(ctx, id)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "get created user failed", err, map[string]any{
			"user_id": id,
		})
		return nil, fmt.Errorf("get created user: %w", err)
	}
	if createdUser == nil {
		err := fmt.Errorf("repository returned nil user")
		logging.Failure(ctx, uc.appLogger, "get created user failed", err, map[string]any{
			"user_id": id,
		})
		return nil, fmt.Errorf("get created user: %w", err)
	}
	uc.appLogger.DebugContext(ctx, "created user loaded", map[string]any{"user_id": createdUser.ID})

	session, err := uc.sessions.Create(ctx, createdUser.ID)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "session creation failed", err, map[string]any{
			"user_id": createdUser.ID,
		})
		return nil, fmt.Errorf("create session: %w", err)
	}
	uc.appLogger.InfoContext(ctx, "session created", map[string]any{
		"user_id":    createdUser.ID,
		"expires_at": session.ExpiresAt,
	})

	output := NewRegisterOutput(
		NewUserDTO(createdUser.ID, createdUser.Email, createdUser.Username),
		NewSessionDTO(session.ID, session.ExpiresAt),
	)

	output.User.AvatarFilename = createdUser.AvatarFilename

	uc.appLogger.InfoContext(ctx, "user registered", map[string]any{
		"user_id": createdUser.ID,
	})

	return &output, nil
}
