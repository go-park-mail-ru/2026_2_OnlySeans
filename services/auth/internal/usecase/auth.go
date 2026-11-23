package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/domain"
	passwordutil "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/password"
	"github.com/google/uuid"
)

type UserClient interface {
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
}

type AuthUseCase interface {
	IssueSession(ctx context.Context, userID int64, role string) (*domain.Session, error)
	Login(ctx context.Context, email, password string) (*domain.Session, error)
	Logout(ctx context.Context, sessionID uuid.UUID) error
	CheckSession(ctx context.Context, sessionID uuid.UUID) (UserInfo, error)
}

type UserInfo struct {
	UserID int64
	Role   string
}

type authUseCase struct {
	userClient       UserClient
	sessionUC        SessionUseCase
	passwordVerifier domain.PasswordVerifier
}

func NewAuthUseCase(uc UserClient, suc SessionUseCase, verifier domain.PasswordVerifier) AuthUseCase {
	return &authUseCase{
		userClient:       uc,
		sessionUC:        suc,
		passwordVerifier: verifier,
	}
}

func (u *authUseCase) IssueSession(ctx context.Context, userID int64, role string) (*domain.Session, error) {
	createdSession, err := u.sessionUC.Create(ctx, domain.UserID(userID), role)
	if err != nil {
		return nil, fmt.Errorf("failed to issue session: %w", err)
	}

	return &createdSession, nil
}

func (u *authUseCase) Login(ctx context.Context, email, password string) (*domain.Session, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	currUser, err := u.userClient.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if err := u.passwordVerifier.Verify(password, currUser.PasswordHash); err != nil {
		if errors.Is(err, passwordutil.ErrPasswordMismatch) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("verify password: %w", err)
	}

	createdSession, err := u.sessionUC.Create(ctx, currUser.ID, string(currUser.Role))
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &createdSession, nil
}

func (u *authUseCase) Logout(ctx context.Context, sessionID uuid.UUID) error {
	err := u.sessionUC.Delete(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to destroy session: %w", err)
	}

	return nil
}

func (u *authUseCase) CheckSession(ctx context.Context, sessionID uuid.UUID) (UserInfo, error) {
	session, err := u.sessionUC.Check(ctx, sessionID)
	if err != nil {
		return UserInfo{}, err
	}

	return UserInfo{
		UserID: int64(session.UserID),
		Role:   session.Role,
	}, nil
}
