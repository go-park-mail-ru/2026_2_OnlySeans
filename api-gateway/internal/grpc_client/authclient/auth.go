package authclient

import (
	"context"
	"errors"

	pbAuth "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrSessionNotFound    = errors.New("session not found or expired")
	ErrInternal           = errors.New("internal server error")
)

type AuthClient interface {
	IssueSession(ctx context.Context, userID int64, role string) (*pbAuth.Session, error)
	Login(ctx context.Context, email, password string) (*pbAuth.Session, error)
	Logout(ctx context.Context, sessionID string) error
	CheckSession(ctx context.Context, sessionID string) (SessionInfo, error)
}

type SessionInfo struct {
	UserID int64
	Role   string
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
		return nil, ErrInternal
	}

	return resp.Session, nil
}

func (c *authClient) Login(ctx context.Context, email, password string) (*pbAuth.Session, error) {
	resp, err := c.client.Login(ctx, &pbAuth.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.Unauthenticated {
			return nil, ErrInvalidCredentials
		}

		return nil, ErrInternal
	}

	return resp.Session, nil
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
		UserID: resp.GetUserId(),
		Role:   resp.GetRole(),
	}, nil
}
