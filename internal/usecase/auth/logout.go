package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type LogoutUserUseCase struct {
	sessions domain.SessionStore
}

func NewLogoutUserUseCase(sessions domain.SessionStore) (*LogoutUserUseCase, error) {
	if sessions == nil {
		return nil, errors.New("sessions store is nil")
	}

	return &LogoutUserUseCase{sessions: sessions}, nil
}

func (uc *LogoutUserUseCase) Run(ctx context.Context, input LogoutInput) error {
	if err := uc.sessions.Delete(ctx, input.SessionID); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
