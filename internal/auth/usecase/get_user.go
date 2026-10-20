package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth/domain"
)

type GetUserInput struct {
	UserID domain.UserID
}

type GetUserUseCase struct {
	users UserRepository
}

func NewGetUserUseCase(users UserRepository) (*GetUserUseCase, error) {
	if users == nil {
		return nil, errors.New("users repository is nil")
	}

	return &GetUserUseCase{users: users}, nil
}

func (uc *GetUserUseCase) Run(ctx context.Context, input GetUserInput) (UserDTO, error) {
	user, err := uc.users.GetUser(ctx, input.UserID)
	if err != nil {
		return UserDTO{}, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return UserDTO{}, domain.ErrUserNotFound
	}

	return UserDTO{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
	}, nil
}
