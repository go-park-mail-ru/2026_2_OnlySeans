package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type AuthenticateUserUseCase struct {
	sessions domain.SessionStore
	logger   logger.Logger
}

func NewAuthenticateUserUseCase(sessions domain.SessionStore, appLogger logger.Logger) (*AuthenticateUserUseCase, error) {
	if sessions == nil {
		return nil, errors.New("sessions store is nil")
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &AuthenticateUserUseCase{
		sessions: sessions,
		logger:   appLogger,
	}, nil
}

func (uc *AuthenticateUserUseCase) Run(ctx context.Context, input AuthenticateInput) (*AuthenticateOutput, error) {
	uc.logger.DebugContext(ctx, "authenticate started", nil)

	session, err := uc.sessions.Get(ctx, input.SessionID)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			uc.logger.WarnContext(ctx, "authentication rejected", nil)
			return nil, domain.ErrUnauthorized
		}

		logging.Failure(ctx, uc.logger, "get session failed", err, map[string]any{})
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session == nil {
		err := fmt.Errorf("store returned nil session")
		logging.Failure(ctx, uc.logger, "get session failed", err, map[string]any{})
		return nil, fmt.Errorf("get session: %w", err)
	}

	output := NewAuthenticateOutput(session.UserID)

	uc.logger.DebugContext(ctx, "user authenticated", map[string]any{
		"user_id": session.UserID,
	})

	return &output, nil
}
