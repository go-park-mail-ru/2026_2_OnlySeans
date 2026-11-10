package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type LogoutUserUseCase struct {
	sessions  domain.SessionStore
	appLogger logger.Logger
}

func NewLogoutUserUseCase(sessions domain.SessionStore, appLogger logger.Logger) (*LogoutUserUseCase, error) {
	if sessions == nil {
		return nil, errors.New("sessions store is nil")
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &LogoutUserUseCase{sessions: sessions, appLogger: appLogger}, nil
}

func (uc *LogoutUserUseCase) Run(ctx context.Context, input LogoutInput) error {
	uc.appLogger.DebugContext(ctx, "logout started", nil)

	if err := uc.sessions.Delete(ctx, input.SessionID); err != nil {
		logging.Failure(ctx, uc.appLogger, "session deletion failed", err, map[string]any{})

		return fmt.Errorf("delete session: %w", err)
	}

	uc.appLogger.InfoContext(ctx, "user logged out", nil)

	return nil
}
