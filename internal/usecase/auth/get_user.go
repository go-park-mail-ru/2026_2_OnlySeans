package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type GetUserUseCase struct {
	users UserRepository
}

func NewGetUserUseCase(users UserRepository) (*GetUserUseCase, error) {
	if users == nil {
		return nil, errors.New("users repository is nil")
	}

	return &GetUserUseCase{users: users}, nil
}

func (uc *GetUserUseCase) Run(ctx context.Context, input GetUserInput) (*GetUserOutput, error) {
	user, err := uc.users.GetUser(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}

	output := NewGetUserOutput(user.ID, user.Email, user.Username)
	return &output, nil
}
