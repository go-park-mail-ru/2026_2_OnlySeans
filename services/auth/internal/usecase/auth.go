package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/domain"
	passwordutil "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/password"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/utils"
)

const (
	issueSessionErrorFormat   = "failed to issue session: %w"
	getUserByEmailErrorFormat = "get user by email: %w"
	verifyPasswordErrorFormat = "verify password: %w"
	createSessionErrorFormat  = "failed to create session: %w"
	destroySessionErrorFormat = "failed to destroy session: %w"
	refreshSessionErrorFormat = "refresh session: %w"
)

type UserClient interface {
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
}

type AuthUseCase interface {
	IssueSession(ctx context.Context, userID int64, role string) (*domain.Session, error)
	Login(ctx context.Context, email, password string) (*domain.Session, error)
	Logout(ctx context.Context, sessionID uuid.UUID) error
	CheckSession(ctx context.Context, sessionID uuid.UUID) (UserInfo, error)
	RefreshSession(ctx context.Context, sessionID uuid.UUID) (*domain.Session, error)
}

type UserInfo struct {
	ExpiresAt       time.Time
	LastRefreshedAt time.Time
	UserID          int64
	Role            string
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
		return nil, fmt.Errorf(issueSessionErrorFormat, err)
	}

	return &createdSession, nil
}

func (u *authUseCase) Login(ctx context.Context, email, password string) (*domain.Session, error) {
	email = strings.TrimSpace(email)

	if err := utils.ValidateEmail(email); err != nil {
		return nil, err
	}

	if err := utils.ValidatePasswordLength(password); err != nil {
		return nil, err
	}

	currUser, err := u.userClient.GetUserByEmail(ctx, email)
	if errors.Is(err, domain.ErrUserNotFound) {
		return nil, domain.ErrInvalidCredentials
	}

	if err != nil {
		return nil, fmt.Errorf(getUserByEmailErrorFormat, err)
	}

	if err := u.passwordVerifier.Verify(password, currUser.PasswordHash); err != nil {
		if errors.Is(err, passwordutil.ErrPasswordMismatch) {
			return nil, domain.ErrInvalidCredentials
		}

		return nil, fmt.Errorf(verifyPasswordErrorFormat, err)
	}

	createdSession, err := u.sessionUC.Create(ctx, currUser.ID, string(currUser.Role))
	if err != nil {
		return nil, fmt.Errorf(createSessionErrorFormat, err)
	}

	return &createdSession, nil
}

func (u *authUseCase) Logout(ctx context.Context, sessionID uuid.UUID) error {
	err := u.sessionUC.Delete(ctx, sessionID)
	if err != nil {
		return fmt.Errorf(destroySessionErrorFormat, err)
	}

	return nil
}

func (u *authUseCase) CheckSession(ctx context.Context, sessionID uuid.UUID) (UserInfo, error) {
	session, err := u.sessionUC.Check(ctx, sessionID)
	if err != nil {
		return UserInfo{}, err
	}

	return UserInfo{
		UserID:          int64(session.UserID),
		ExpiresAt:       session.ExpiresAt,
		LastRefreshedAt: session.LastRefreshedAt,
		Role:            session.Role,
	}, nil
}

func (u *authUseCase) RefreshSession(ctx context.Context, sessionID uuid.UUID) (*domain.Session, error) {
	session, err := u.sessionUC.Refresh(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf(refreshSessionErrorFormat, err)
	}

	return &session, nil
}
