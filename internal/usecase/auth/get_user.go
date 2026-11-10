package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type GetUserUseCase struct {
	users     domain.UserRepository
	appLogger logger.Logger
}

func NewGetUserUseCase(users domain.UserRepository, appLogger logger.Logger) (*GetUserUseCase, error) {
	if users == nil {
		return nil, errors.New("users repository is nil")
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &GetUserUseCase{
		users:     users,
		appLogger: appLogger,
	}, nil
}

func (uc *GetUserUseCase) Run(ctx context.Context, input GetUserInput) (*GetUserOutput, error) {
	uc.appLogger.DebugContext(ctx, "get user started", map[string]any{
		"user_id": input.UserID,
	})

	user, err := uc.users.GetUser(ctx, input.UserID)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "get user failed", err, map[string]any{
			"user_id": input.UserID,
		})

		return nil, fmt.Errorf("get user: %w", err)
	}

	if user == nil {
		uc.appLogger.WarnContext(ctx, "user not found", map[string]any{
			"user_id": input.UserID,
		})

		return nil, domain.ErrUserNotFound
	}

	output := NewGetUserOutput(user.ID, user.Email, user.Username)
	output.Role = user.Role
	output.AvatarFilename = user.AvatarFilename

	uc.appLogger.InfoContext(ctx, "user retrieved", map[string]any{
		"user_id": user.ID,
	})

	return &output, nil
}
