package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type AuthenticateUserUseCase struct {
	sessions domain.SessionStore
}

func NewAuthenticateUserUseCase(sessions domain.SessionStore) (*AuthenticateUserUseCase, error) {
	if sessions == nil {
		return nil, errors.New("sessions store is nil")
	}

	return &AuthenticateUserUseCase{sessions: sessions}, nil
}

func (uc *AuthenticateUserUseCase) Run(ctx context.Context, input AuthenticateInput) (*AuthenticateOutput, error) {
	session, err := uc.sessions.Get(ctx, input.SessionID)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return nil, domain.ErrUnauthorized
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("get session: store returned nil session")
	}

	output := NewAuthenticateOutput(session.UserID)
	return &output, nil
}
