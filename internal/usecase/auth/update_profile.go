package usecase

import (
	"context"
	"strings"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/utils"
)

type UpdateProfileUseCase struct{ users domain.UserRepository }

func NewUpdateProfileUseCase(users domain.UserRepository) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{users: users}
}

func (uc *UpdateProfileUseCase) Run(ctx context.Context, id domain.UserID, email, username string) (*domain.User, error) {
	if id <= 0 {
		return nil, domain.ErrUnauthorized
	}
	email, username = strings.TrimSpace(email), strings.TrimSpace(username)
	if err := utils.ValidateEmail(email); err != nil {
		return nil, domain.ErrInvalidEmail
	}
	if err := utils.ValidateUsername(username); err != nil {
		return nil, domain.ErrInvalidUsername
	}
	return uc.users.UpdateProfile(ctx, id, email, username)
}
