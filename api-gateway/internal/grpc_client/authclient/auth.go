package authclient

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/utils"
	pbAuth "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/auth"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrSessionNotFound    = errors.New("session not found or expired")
	ErrInternal           = errors.New("internal server error")
	ErrInvalidEmail       = utils.ErrInvalidEmail
	ErrInvalidPassword    = utils.ErrInvalidPassword
	ErrInvalidArgument    = errors.New("invalid login data")
)

type AuthClient interface {
	IssueSession(ctx context.Context, userID int64, role string) (*pbAuth.Session, error)
	Login(ctx context.Context, email, password string) (*pbAuth.Session, error)
	Logout(ctx context.Context, sessionID string) error
	CheckSession(ctx context.Context, sessionID string) (SessionInfo, error)
	RefreshSession(ctx context.Context, sessionID string) (*pbAuth.Session, error)
}

type SessionInfo struct {
	ExpiresAt       time.Time
	LastRefreshedAt time.Time
	UserID          int64
	Role            string
}

type authClient struct {
	client pbAuth.AuthServiceClient
}

func NewAuthClient(cl pbAuth.AuthServiceClient) AuthClient {
	return &authClient{
		client: cl,
	}
}

func (c *authClient) IssueSession(ctx context.Context, userID int64, role string) (*pbAuth.Session, error) {
	resp, err := c.client.IssueSession(ctx, &pbAuth.IssueSessionRequest{
		UserId: userID,
		Role:   role,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInternal, err)
	}

	return resp.Session, nil
}

func (c *authClient) Login(ctx context.Context, email, password string) (*pbAuth.Session, error) {
	resp, err := c.client.Login(ctx, &pbAuth.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		st := status.Convert(err)
		switch st.Code() {
		case codes.Unauthenticated:
			return nil, ErrInvalidCredentials
		case codes.InvalidArgument:
			for _, fieldErr := range []error{ErrInvalidEmail, ErrInvalidPassword} {
				if st.Message() == fieldErr.Error() {
					return nil, fieldErr
				}

				if reason, ok := strings.CutPrefix(st.Message(), fieldErr.Error()+": "); ok {
					return nil, fmt.Errorf("%w: %s", fieldErr, reason)
				}
			}

			return nil, ErrInvalidArgument
		}

		return nil, fmt.Errorf("%w: %w", ErrInternal, err)
	}

	if resp.GetSession() == nil {
		return nil, ErrInternal
	}

	return resp.GetSession(), nil
}

func (c *authClient) Logout(ctx context.Context, sessionID string) error {
	_, err := c.client.Logout(ctx, &pbAuth.LogoutRequest{
		SessionId: sessionID,
	})

	return err
}

func (c *authClient) CheckSession(ctx context.Context, sessionId string) (SessionInfo, error) {
	resp, err := c.client.CheckSession(ctx, &pbAuth.CheckSessionRequest{
		SessionId: sessionId,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.NotFound, codes.Unauthenticated, codes.InvalidArgument:
			return SessionInfo{}, ErrSessionNotFound
		default:
			return SessionInfo{}, ErrInternal
		}
	}

	return SessionInfo{
		UserID:          resp.GetUserId(),
		ExpiresAt:       resp.GetExpiresAt().AsTime(),
		LastRefreshedAt: resp.GetLastRefreshedAt().AsTime(),
		Role:            resp.GetRole(),
	}, nil
}

func (c *authClient) RefreshSession(ctx context.Context, sessionID string) (*pbAuth.Session, error) {
	response, err := c.client.RefreshSession(ctx, &pbAuth.RefreshSessionRequest{SessionId: sessionID})
	if err != nil {
		switch status.Code(err) {
		case codes.NotFound, codes.Unauthenticated, codes.InvalidArgument:
			return nil, ErrSessionNotFound
		default:
			return nil, fmt.Errorf("%w: %w", ErrInternal, err)
		}
	}

	session := response.GetSession()
	if session == nil || session.GetId() != sessionID || session.GetExpiresAt() == nil || session.GetExpiresAt().CheckValid() != nil {
		return nil, ErrInternal
	}

	return session, nil
}
