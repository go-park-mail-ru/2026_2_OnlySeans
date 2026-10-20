package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth/domain"
)

type AuthenticateInput struct {
	SessionID string
}

type AuthenticateUserUseCase struct {
	sessions SessionStore
}

func NewAuthenticateUserUseCase(sessions SessionStore) (*AuthenticateUserUseCase, error) {
	if sessions == nil {
		return nil, errors.New("sessions store is nil")
	}

	return &AuthenticateUserUseCase{sessions: sessions}, nil
}

func (uc *AuthenticateUserUseCase) Run(ctx context.Context, input AuthenticateInput) (domain.UserID, error) {
	session, err := uc.sessions.Get(ctx, input.SessionID)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return 0, domain.ErrUnauthorized
		}
		return 0, fmt.Errorf("get session: %w", err)
	}
	if session == nil {
		return 0, fmt.Errorf("get session: store returned nil session")
	}

	return session.UserID, nil
}
