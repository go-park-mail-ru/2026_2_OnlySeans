package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type GetUserUseCase struct {
	users  domain.UserRepository
	logger logger.Logger
}

func NewGetUserUseCase(users domain.UserRepository, appLogger logger.Logger) (*GetUserUseCase, error) {
	if users == nil {
		return nil, errors.New("users repository is nil")
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &GetUserUseCase{
		users:  users,
		logger: appLogger,
	}, nil
}

func (uc *GetUserUseCase) Run(ctx context.Context, input GetUserInput) (*GetUserOutput, error) {
	user, err := uc.users.GetUser(ctx, input.UserID)
	if err != nil {
		uc.logger.Error("get user failed", map[string]any{
			"user_id": input.UserID,
			"error":   err,
		})
		return nil, fmt.Errorf("get user: %w", err)
	}

	if user == nil {
		uc.logger.Warn("user not found", map[string]any{
			"user_id": input.UserID,
		})
		return nil, domain.ErrUserNotFound
	}

	output := NewGetUserOutput(user.ID, user.Email, user.Username)

	uc.logger.Info("user retrieved", map[string]any{
		"user_id": user.ID,
	})

	return &output, nil
}
