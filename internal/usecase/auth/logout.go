package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type LogoutUserUseCase struct {
	sessions domain.SessionStore
	logger   logger.Logger
}

func NewLogoutUserUseCase(sessions domain.SessionStore, appLogger logger.Logger) (*LogoutUserUseCase, error) {
	if sessions == nil {
		return nil, errors.New("sessions store is nil")
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &LogoutUserUseCase{sessions: sessions, logger: appLogger}, nil
}

func (uc *LogoutUserUseCase) Run(ctx context.Context, input LogoutInput) error {
	uc.logger.Debug("logout started", nil)

	if err := uc.sessions.Delete(ctx, input.SessionID); err != nil {
		uc.logger.Error("session deletion failed", map[string]any{
			"error": err,
		})

		return fmt.Errorf("delete session: %w", err)
	}

	uc.logger.Info("user logged out", nil)

	return nil
}
